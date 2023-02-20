# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

require "vagrant-vmware-desktop/config"

require "spec_base"

describe HashiCorp::VagrantVMwareDesktop::Action::BaseMacToIp do
  let(:app) { double("app") }
  let(:machine) { instance_double(Vagrant::Machine) }
  let(:provider_config) {
    double("provider_config",
      base_mac: base_mac,
      base_address: base_address,
      nat_device: "vmnet8")
  }
  let(:base_address) { nil }
  let(:base_mac) { double("base_mac") }

  let(:ui) { instance_double(Vagrant::UI::Basic) }
  let(:env) {{
    machine: machine,
    ui: ui,
  }}
  let(:provider) { double("provider") }
  let(:driver) { double("driver") }

  let(:vmnet_devices) {
    [{name: "vmnet8",
      hostonly_subnet: "192.168.33.0"},
    {name: "vmnet2",
      hostonly_subnet: "10.0.1.0"}]
  }

  subject { described_class.new(app, env) }

  before do
    allow(app).to receive(:call)
    allow(machine).to receive(:provider_config).and_return(provider_config)
    allow(machine).to receive(:provider).and_return(provider)
    allow(provider).to receive(:driver).and_return(driver)
    allow(driver).to receive(:reserve_dhcp_address)
    allow(driver).to receive(:read_vmnet_devices).and_return(vmnet_devices)

    allow(ui).to receive(:info)
  end

  describe "#call" do
    after { subject.call(env) }

    context "when base_address is unset" do
      it "should not do any validation" do
        expect(subject).not_to receive(:validate_address!)
      end

      it "should call the next action" do
        expect(app).to receive(:call)
      end
    end

    context "when base_address is set" do
      before { expect(app).to receive(:call) }

      context "with a valid address" do
        let(:base_address) { "192.168.33.22" }

        it "should validate the address" do
          expect(subject).to receive(:validate_address!)
        end

        it "should request the address reservation" do
          expect(driver).to receive(:reserve_dhcp_address).with(base_address, base_mac, "vmnet8")
        end

        it "should notify the user of the reservation" do
          expect(ui).to receive(:info).with(/#{base_address}/)
        end

        context "when a nat device is set to non default" do
          let(:base_address) { "10.0.1.2" }

          let(:provider_config) {
            double("provider_config",
              base_mac: base_mac,
              base_address: base_address,
              nat_device: "vmnet2")
          }
    
          it "should request the address reservation" do
            expect(driver).to receive(:reserve_dhcp_address).with(base_address, base_mac, "vmnet2")
          end
        end
      end
    end
  end

  describe "#validate_address!" do
    let(:address) { "192.168.33.22" }
    let(:device) { "vmnet8" }

    it "should not raise error when address is valid" do
      expect { subject.validate_address!(driver, address, device) }.not_to raise_error
    end

    context "when vmnet8 does not exist" do
      let(:vmnet_devices) { [] }

      it "should raise error" do
        expect { subject.validate_address!(driver, address, device) }.
          to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::MissingNATDevice)
      end
    end

    context "when address outside configured subnet" do
      let(:address) { "192.168.22.4" }

      it "should raise error" do
        expect { subject.validate_address!(driver, address, device) }.
          to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::BaseAddressRange)
      end
    end

    context "when address is reserved for host machine" do
      let(:address) { "192.168.33.1" }

      it "should raise error" do
        expect { subject.validate_address!(driver, address, device) }.
          to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::BaseAddressRange)
      end
    end

    context "when address is reserved for DHCP leases" do
      let(:address) { "192.168.33.130" }

      it "should raise error" do
        expect { subject.validate_address!(driver, address, device) }.
          to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::BaseAddressRange)
      end
    end
  end
end
