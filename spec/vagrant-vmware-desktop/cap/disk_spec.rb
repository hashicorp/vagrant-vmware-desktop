# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

require_relative "../../spec_base"
require "vagrant"

require "vagrant-vmware-desktop/cap/disk"

describe HashiCorp::VagrantVMwareDesktop::Cap::Disk do
  let(:driver) { double("driver", 'is_linked_clone?': linked_clone) }
  let(:linked_clone) { false }
  let(:ui) { double("ui") }
  let(:machine) { double("machine", provider: double("provider", driver: driver),
    env: double("env", ui: ui))}
  let(:defined_disks) { [
    double("disk", id: "12345", name: "vagrant_primary", size: 196608, primary: true,
      type: :disk, disk_ext: "vmdk", file: "#{vm_path}/vagrant_primary.vmdk"),
    double("disk", id: "67890", name: "disk-0", size: 2147483648, primary: false,
      type: :disk, disk_ext: "vmdk", file: "#{vm_path}/disk-0.vmdk"),
    double("disk", id: "abcde", name: "disk-1", size: 196608, primary: false,
      type: :disk, disk_ext: "vmdk", file: "#{vm_path}/disk-1.vmdk"),
    double("dvd", id: "efghi", name: "my-dvd", primary: false, type: :dvd,
      disk_ext: "vmdk", file: dvd_path, provider_config: nil)
  ] }
  let(:vm_path) { "/Users/vagrant/.vagrant/machines/mach/vmware/123-qwe-456" }
  let(:vmx_path) { "#{vm_path}/machine.vmx" }
  let(:dvd_path) { "/some/path/dvd.iso" }

  let(:disks) {[
    {"UUID" => "12345",
      "Path" => "#{vm_path}/vagrant_primary.vmdk",
      "Name" => "vagrant_primary"},
    {"UUID" => "67890",
      "Path" => "#{vm_path}/disk-0.vmdk",
      "Name" => "disk-0"},
    {"UUID" => "324bbb53-d5ad-45f8-9bfa-1f2468b199a8",
      "Path" => "#{vm_path}/disk-1.vmdk",
      "Name" => "disk-1"},
  ]}

  let(:dvds) {[
    {"UUID" => "qwer1234",
      "Path" => dvd_path,
      "Name" => "my-dvd"}
  ]}

  before do
    allow(driver).to receive(:vmx_path).and_return(vmx_path)
  end

  describe "#configure_disks" do
    let(:dummy_disk_data) { {
      "UUID" => "123", "Path" => "/path/to/nowhere.vmdk", "Name" => "nowhere"
    } }

    before do
      allow(driver).to receive(:get_disks).and_return({})
    end

    it "does nothing if there are no defined disks" do
      expect(described_class.configure_disks(machine, [])).to eq({})
    end

    it "sets up disk type defined disks" do
      expect(described_class).to receive(:setup_disk).exactly(3).and_return(dummy_disk_data)
      expect(described_class).to receive(:setup_dvd).exactly(1).and_return(dummy_disk_data)
      configured_disks = described_class.configure_disks(machine, defined_disks)
      expect(configured_disks).to eq(
        {disk: [dummy_disk_data, dummy_disk_data, dummy_disk_data], floppy: [], dvd: [dummy_disk_data]})
    end

    it "sets up disks with provider specifig config" do
      expect(described_class).to receive(:get_slot).with("scsi", anything).
        and_return("scsi0:1").exactly(3)
      expect(described_class).to receive(:get_slot).with("ide", anything).
        and_return("ide0:1").exactly(1)

      expect(driver).to receive(:create_disk).with("disk1.vmdk", 196608, 0, "buslogic").
        and_return("#{vmx_path}/disk1.vmdk")
      expect(driver).to receive(:create_disk).with("disk2.vmdk", 196608, 0, "lsilogic").
        and_return("#{vmx_path}/disk2.vmdk")
      expect(driver).to receive(:add_disk_to_vmx).with("disk1.vmdk", "scsi0:1")
      expect(driver).to receive(:add_disk_to_vmx).with("disk2.vmdk", "scsi0:1")
      expect(driver).to receive(:add_disk_to_vmx).with(dvd_path, "scsi0:1", {"deviceType" => described_class::DEFAULT_DVD_DEVICE_TYPE})
      expect(driver).to receive(:add_disk_to_vmx).with("#{dvd_path}2", "ide0:1", {"deviceType" => "thing"})

      disks = [
        double("disk", id: "12345", name: "disk1", size: 196608, primary: false, type: :disk, disk_ext: "vmdk",
          file: "#{vm_path}/disk1.vmdk", provider_config: {vmware_desktop: {adapter_type: "buslogic"}}),
        double("disk2", id: "22345", name: "disk2", size: 196608, primary: false, type: :disk, disk_ext: "vmdk",
          file: "#{vm_path}/disk2.vmdk", provider_config: {vmware_desktop: {adapter_type: "oops"}}),
        double("dvd", id: "32345", name: "dvd", primary: false, type: :dvd,  file: dvd_path,
          provider_config: {vmware_desktop: {bus_type: "scsi"}}),
        double("dvd2", id: "42345", name: "dvd2", primary: false, type: :dvd,  file: "#{dvd_path}2",
          provider_config: {vmware_desktop: {device_type: "thing"}}),
      ]
      described_class.configure_disks(machine, disks)
    end

    context "ide space full" do
      before do
        allow(driver).to receive(:get_disks).and_return({
          "ide0:0" => {"filename" => "disk-0.vmdk", "present" => "TRUE"},
          "ide0:1" => {"filename" => "disk-1.vmdk", "present" => "TRUE"},
          "ide1:0" => {"filename" => "disk-2.vmdk", "present" => "TRUE"},
        })
      end

      it "configures disks" do
        expect(driver).to receive(:create_disk).with("disk1.vmdk", 196608, 0, "lsilogic").
          and_return("#{vmx_path}/disk1.vmdk")
        expect(driver).to receive(:create_disk).with("disk2.vmdk", 196608, 0, "lsilogic").
          and_return("#{vmx_path}/disk2.vmdk")

        expect(driver).to receive(:add_disk_to_vmx).with("disk1.vmdk", "ide1:1")
        expect(driver).to receive(:add_disk_to_vmx).with("disk2.vmdk", "ide2:0")

        disks = [
          double("disk", id: "12345", name: "disk1", size: 196608, primary: false, type: :disk,
            disk_ext: "vmdk", file: "#{vm_path}/disk1.vmdk", provider_config: {vmware_desktop: {bus_type: "ide"}}),
          double("disk", id: "12345", name: "disk2", size: 196608, primary: false, type: :disk,
            disk_ext: "vmdk", file: "#{vm_path}/disk2.vmdk", provider_config: {vmware_desktop: {bus_type: "ide"}}),
        ]
        described_class.configure_disks(machine, disks)
      end
    end

    context "some disks already attached" do
      before do
        allow(driver).to receive(:get_disks).and_return({
          "scsi0:0" => {"filename" => "vagrant_primary.vmdk", "present" => "TRUE", "redo" => ""}
        })
        allow(driver).to receive(:get_disk_size).with("#{vm_path}/vagrant_primary.vmdk").and_return(196608)
      end

      it "creates new disks" do
        expect(driver).to_not receive(:grow_disk)
        expect(described_class).to receive(:create_disk).and_return("/path/to/nowhere").exactly(2)
        expect(driver).to receive(:add_disk_to_vmx)
        configured_disks = described_class.configure_disks(machine, defined_disks)
        expect(configured_disks[:disk].map { |d| d.flatten.any?(nil) }.any?(true)).to be(false)
      end
    end

    context "shrinking disks" do
      before do
        allow(driver).to receive(:get_disks).and_return({
          "scsi0:0" => {"filename" => "vagrant_primary.vmdk", "present" => "TRUE", "redo" => ""},
          "ide1:0" => {"filename" => "disk-0.vmdk", "present" => "TRUE"},
          "ide2:0" => {"filename" => "disk-1.vmdk", "present" => "TRUE"},
          "ide0:1" => {"filename" => dvd_path, "present" => "TRUE", "deviceType" => "cdrom-image"}
        })
        allow(driver).to receive(:get_disk_size).with("#{vm_path}/vagrant_primary.vmdk").and_return(2147483648)
        allow(driver).to receive(:get_disk_size).with("#{vm_path}/disk-0.vmdk").and_return(2147483648)
        allow(driver).to receive(:get_disk_size).with("#{vm_path}/disk-1.vmdk").and_return(196608)
      end

      it "does not try to shrink disks" do
        expect(driver).to_not receive(:grow_disk)
        expect(described_class).to_not receive(:create_disk)
        expect(ui).to receive(:warn)
        configured_disks = described_class.configure_disks(machine, defined_disks)
        expect(configured_disks[:disk].map { |d| d.flatten.any?(nil) }.any?(true)).to be(false)
      end
    end

    context "growing disks" do
      before do
        allow(driver).to receive(:get_disks).and_return({
          "scsi0:0" => {"filename" => "vagrant_primary.vmdk", "present" => "TRUE", "redo" => ""},
          "ide1:0" => {"filename" => "disk-0.vmdk", "present" => "TRUE"},
          "ide2:0" => {"filename" => "disk-1.vmdk", "present" => "TRUE"},
          "ide0:1" => {"filename" => dvd_path, "present" => "TRUE", "deviceType" => "cdrom-image"},
        })
        allow(driver).to receive(:get_disk_size).with("#{vm_path}/vagrant_primary.vmdk").and_return(19660)
        allow(driver).to receive(:get_disk_size).with("#{vm_path}/disk-0.vmdk").and_return(196608)
        allow(driver).to receive(:get_disk_size).with("#{vm_path}/disk-1.vmdk").and_return(196608)
      end

      it "grows disk" do
        expect(driver).to receive(:grow_disk).exactly(2)
        expect(described_class).to_not receive(:create_disk)
        expect(driver).to_not receive(:add_disk_to_vmx)
        allow(driver).to receive(:snapshot_list).and_return([])
        configured_disks = described_class.configure_disks(machine, defined_disks)
        expect(configured_disks[:disk].map { |d| d.flatten.any?(nil) }.any?(true)).to be(false)
      end

      it "raises an error if a snapshot is defined" do
        expect(driver).to_not receive(:grow_disk)
        expect(described_class).to_not receive(:create_disk)
        expect(driver).to_not receive(:add_disk_to_vmx)
        allow(driver).to receive(:snapshot_list).and_return(["a"])
        expect{
          described_class.configure_disks(machine, defined_disks)
        }.to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::DiskNotResizedSnapshot)
      end

      context "when guest is a linked clone" do
        let(:linked_clone) { true }

        before do
          allow(ui).to receive(:warn)
          allow(driver).to receive(:grow_disk)
          allow(driver).to receive(:snapshot_list).and_return([])
        end

        it "should warn it will not grow linked clone" do
          expect(ui).to receive(:warn).with(/Primary disk of the guest remains/)
          described_class.configure_disks(machine, defined_disks)
        end

        it "should not grow the primary disk" do
          expect(driver).to receive(:grow_disk).once
          expect(described_class).to_not receive(:create_disk)
          expect(driver).to_not receive(:add_disk_to_vmx)
          configured_disks = described_class.configure_disks(machine, defined_disks)
          expect(configured_disks[:disk].map { |d| d.flatten.any?(nil) }.any?(true)).to be(false)
        end
      end

      context "vmx pointing to not root metadata disk" do
        let(:defined_disks) { [
          double("disk", id: "abcde", name: "disk", size: 196608, primary: true,
            type: :disk, disk_ext: "vmdk", file: "#{vm_path}/disk-1.vmdk"),
        ] }

        before do
          allow(driver).to receive(:get_disks).and_return({
            "scsi0:0" => {"filename" => "disk-s001.vmdk", "present" => "TRUE"},
          })
        end

        it "grows root metadata disk" do
          expect(described_class).to_not receive(:create_disk)
          expect(driver).to_not receive(:add_disk_to_vmx)
          allow(driver).to receive(:snapshot_list).and_return([])

          expect(driver).to_not receive(:get_disk_size).with("#{vm_path}/disk-s001.vmdk")
          expect(driver).to receive(:get_disk_size).with("#{vm_path}/disk.vmdk").and_return(19660)
          expect(driver).to receive(:grow_disk).with("#{vm_path}/disk.vmdk", anything)

          described_class.configure_disks(machine, defined_disks)
        end
      end
    end
  end

  describe "#cleanup_disks" do
    it "does nothing if disk_meta is empty" do
      expect(described_class.cleanup_disks(machine, defined_disks, {})).to eq(nil)
    end

    context "some attached disks" do

      let(:disk_meta) { { "disk" => disks, "floppy" => [], "dvd" => dvds} }

      it "does not remove defined disks" do
        expect(driver).to_not receive(:remove_disk)
        described_class.cleanup_disks(machine, defined_disks, disk_meta)
      end

      it "removes undefined disks" do
        expect(driver).to receive(:remove_disk).exactly(3)
        expect(driver).to receive(:remove_disk_from_vmx).exactly(1)

        described_class.cleanup_disks(machine, [], disk_meta)
      end

      context "with primary disk defined" do
        before { disks.first["primary"] = true }

        it "does not remove the primary disk" do
          expect(driver).to receive(:remove_disk).exactly(2)
          expect(driver).to receive(:remove_disk_from_vmx)

          described_class.cleanup_disks(machine, [], disk_meta)
        end
      end
    end
  end
end
