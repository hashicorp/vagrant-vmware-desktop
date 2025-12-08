# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

require "vagrant/errors"
require "vagrant-vmware-desktop"
require "vagrant-vmware-desktop/guest_cap/linux/mount_vmware_shared_folder"

describe HashiCorp::VagrantVMwareDesktop::GuestCap::Linux::MountVMwareSharedFolder do
  let(:machine) { double("machine", communicate: communicator, guest: guest, provider_config: provider_config) }
  let(:communicator) { double("communicator") }
  let(:guest) { double("guest") }
  let(:provider_config) { double("provider_config", unmount_default_hgfs: unmount_default_hgfs) }
  let(:unmount_default_hgfs) { true }

  before do
    allow_any_instance_of(HashiCorp::VagrantVMwareDesktop::Errors::Base).to receive(:translate_error)
  end

  describe ".mount_vmware_shared_folder" do
    let(:name) { "VMWARE_SHARE_NAME" }
    let(:guestpath) { "/guest/path" }
    let(:options) { {owner: 1, group: 1} }

    let(:expanded_path) { "/expanded/guest/path" }

    before do
      allow(communicator).to receive(:test)
      allow(guest).to receive(:capability).and_return(expanded_path)
      allow(communicator).to receive(:execute).
        and_return(0)
      allow(communicator).to receive(:sudo)
    end

    it "should expand the guest path" do
      expect(guest).to receive(:capability).with(:shell_expand_guest_path, guestpath).
        and_return(expanded_path)
      described_class.mount_vmware_shared_folder(machine, name, guestpath, options)
    end

    it "should test for the kernel module" do
      expect(communicator).to receive(:test).with(/lsmod/)
      described_class.mount_vmware_shared_folder(machine, name, guestpath, options)
    end

    context "with no owner provided" do
      let(:options) { {group: 1} }

      it "should fetch uid from the guest" do
        expect(communicator).to receive(:execute).with(/id -u/, any_args).
          and_yield(:stdout, "1").and_return(0)
        described_class.mount_vmware_shared_folder(machine, name, guestpath, options)
      end

      it "should remove all whitespace from uid" do
        expect(communicator).to receive(:execute).with(/id -u/, any_args).
          and_yield(:stdout, "1\n\n").and_return(0)
        expect(communicator).to receive(:sudo).with(/uid=1/)
        described_class.mount_vmware_shared_folder(machine, name, guestpath, options)
      end

      it "should raise error when failed to discover uid" do
        expect(communicator).to receive(:execute).with(/id -u/, any_args).
          and_yield(:stdout, "\n\n").and_return(0)
        expect { described_class.mount_vmware_shared_folder(machine, name, guestpath, options) }.
          to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::LinuxMountUIDFailed)
      end
    end

    context "with no group provided" do
      let(:options) { {owner: 1} }

      it "should fetch gid from the guest" do
        expect(communicator).to receive(:execute).with(/getent group/, any_args).
          and_yield(:stdout, "user:x:1:").and_return(0)
        described_class.mount_vmware_shared_folder(machine, name, guestpath, options)
      end

      it "should remove all whitespace from gid" do
        expect(communicator).to receive(:execute).with(/getent group/, any_args).
          and_yield(:stdout, "user:x:1:\n\n").and_return(0)
        expect(communicator).to receive(:sudo).with(/gid=1/)
        described_class.mount_vmware_shared_folder(machine, name, guestpath, options)
      end

      it "should raise error when failed to discover gid" do
        expect(communicator).to receive(:execute).with(/getent group/, any_args).
          and_yield(:stdout, "error\n\n").and_return(0)
        expect { described_class.mount_vmware_shared_folder(machine, name, guestpath, options) }.
          to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::LinuxMountGIDFailed)
      end

      context "when owner and group name in options are the same" do
        let(:options) { {owner: "vagrant", group: "vagrant"} }

        before do
          allow(communicator).to receive(:execute).with(/id -u/, any_args).
            and_yield(:stdout, "1").and_return(0)
        end

        it "should fetch effective id when getent fails" do
          expect(communicator).to receive(:execute).with(/getent group/, any_args).
            and_yield(:stdout, "error").and_return(0)
          expect(communicator).to receive(:execute).with(/id -g/, any_args).
            and_yield(:stdout, "1\n\n").and_return(0)
          expect(communicator).to receive(:sudo).with(/gid=1/)
          described_class.mount_vmware_shared_folder(machine, name, guestpath, options)
        end

        it "should raise error when failed to discover gid" do
          expect(communicator).to receive(:execute).with(/getent group/, any_args).
            and_yield(:stdout, "error").and_return(0)
          expect(communicator).to receive(:execute).with(/id -g/, any_args).
            and_yield(:stdout, "error").and_return(0)
          expect { described_class.mount_vmware_shared_folder(machine, name, guestpath, options) }.
            to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::LinuxMountGIDFailed)
        end
      end
    end

    it "should remove guest path before mount" do
      expect(communicator).to receive(:sudo).with(/rm -rf #{expanded_path}/, any_args)
      described_class.mount_vmware_shared_folder(machine, name, guestpath, options)
    end

    it "should create the guest path before mount" do
      expect(communicator).to receive(:sudo).with(/mkdir -p #{expanded_path}/, any_args)
      described_class.mount_vmware_shared_folder(machine, name, guestpath, options)
    end

    context "when kernel module is found" do
      before do
        allow(communicator).to receive(:test).with(/lsmod/).and_return(true)
      end

      it "should mount synced folders using the mount command" do
        expect(communicator).to receive(:sudo).with(/mount -t vmhgfs/)
        described_class.mount_vmware_shared_folder(machine, name, guestpath, options)
      end

      it "should include uid and gid settings" do
        expect(communicator).to receive(:sudo).with(/uid=1,gid=1/)
        described_class.mount_vmware_shared_folder(machine, name, guestpath, options)
      end

      context "when mount options are provided" do
        before { options[:mount_options] = ["custom", "opts"] }

        it "should append custom mount options" do
          expect(communicator).to receive(:sudo).with(/uid=1,gid=1,custom,opts/)
          described_class.mount_vmware_shared_folder(machine, name, guestpath, options)
        end
      end
    end

    it "should unmount default shared folders mount" do
      expect(communicator).to receive(:sudo).with(%r{umount /mnt/hgfs})
      described_class.mount_vmware_shared_folder(machine, name, guestpath, options)
    end

    context "when provider config sets unmount_default_hgfs to false" do
      let(:unmount_default_hgfs) { false }

      it "should not unmount default shared folders mount" do
        expect(communicator).not_to receive(:sudo).with(%r{umount /mnt/hgfs})
        described_class.mount_vmware_shared_folder(machine, name, guestpath, options)
      end
    end

    it "should mount base share using vmhgfs-fuse" do
      expect(communicator).to receive(:sudo).with(/vmhgfs-fuse/)
      described_class.mount_vmware_shared_folder(machine, name, guestpath, options)
    end

    it "should use bind mount to complete the mount" do
      expect(communicator).to receive(:sudo).with(/mount --bind/)
      described_class.mount_vmware_shared_folder(machine, name, guestpath, options)
    end
  end
end
