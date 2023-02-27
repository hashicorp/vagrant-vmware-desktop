# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

require_relative "../spec_base"
require "pathname"
require "vagrant-vmware-desktop/config"

describe HashiCorp::VagrantVMwareDesktop::Config do
  let(:machine) { double("machine") }
  let(:instance) { described_class.new }

  it "should automatically add nat network" do
    expect(instance.network_adapters[0].first).to eq(:nat)
  end

  describe "#cpus=" do
    it "should set the vmx numvcpus entry" do
      instance.cpus = 2
      instance.finalize!
      expect(instance.vmx["numvcpus"]).to eq("2")
    end
  end

  describe "#memory=" do
    it "should set the vmx memsize entry" do
      instance.memory = 512
      instance.finalize!
      expect(instance.vmx["memsize"]).to eq("512")
    end
  end

  describe "#whitelist_verified=" do
    it "should symbolize value if string" do
      instance.whitelist_verified = "disable_warning"
      instance.finalize!
      expect(instance.whitelist_verified).to eq(:disable_warning)
    end

    it "should set allowlist_verified" do
      instance.whitelist_verified = "disable_warning"
      instance.finalize!
      expect(instance.allowlist_verified).to eq(:disable_warning)
    end
  end

  describe "#allowlist_verified=" do
    it "should symbolize value if string" do
      instance.allowlist_verified = "disable_warning"
      instance.finalize!
      expect(instance.allowlist_verified).to eq(:disable_warning)
    end
  end

  describe "#validate" do
    let(:base_mac) { nil }
    let(:base_address) { nil }

    before do
      allow(machine).to receive_message_chain(:config, :vm, :base_mac).and_return(base_mac)
      allow(machine).to receive_message_chain(:config, :vm, :base_address).and_return(base_address)
    end

    it "should generate error if first network adapter is not nat" do
      instance.network_adapter(0, :static)
      instance.finalize!
      errors = instance.validate(machine).values.first
      expect(errors.first).to include("NAT device")
    end

    context "base_mac" do
      it "should validate successfully if valid MAC address" do
        instance.base_mac = "00:50:56:00:00:00"
        instance.finalize!
        errors = instance.validate(machine).values.first
        expect(errors).to be_empty
      end

      it "should error if MAC address does not start with VMware OUI" do
        instance.base_mac = "00:50:33:00:00:00"
        instance.finalize!
        errors = instance.validate(machine).values.first
        expect(errors).to be_empty
      end

      it "should generate error if not a valid MAC address" do
        instance.base_mac = "INVALID"
        instance.finalize!
        errors = instance.validate(machine).values.first
        expect(errors.first).to match(/MAC.+invalid/)
      end

      context "when unset and defined in vm config" do
        let(:base_mac) { "00:50:56:00:00:00" }

        it "should be set after validation" do
          instance.finalize!
          errors = instance.validate(machine).values.first
          expect(errors).to be_empty
          expect(instance.base_mac).to eq(base_mac)
        end
      end
    end

    context "base_address" do
      it "should generate error with valid IP address and no base_mac set" do
        instance.base_address = "127.0.0.1"
        instance.finalize!
        errors = instance.validate(machine).values.first
        expect(errors.first).to match(/base_mac/)
      end

      it "should validate successfully if valid IP address and base_mac is set" do
        instance.base_mac = "00:50:56:00:00:00"
        instance.base_address = "127.0.0.1"
        instance.finalize!
        errors = instance.validate(machine).values.first
        expect(errors).to be_empty
      end

      it "should generate error if base_address is not a valid IP address" do
        instance.base_address = "INVALID"
        instance.finalize!
        errors = instance.validate(machine).values.first
        expect(errors.first).to match(/IP.+invalid/)
      end

      context "when unset and defined in vm config" do
        let(:base_address) { "127.0.0.1" }

        it "should be set after validation" do
          instance.base_mac = "00:50:56:00:00:00"
          instance.finalize!
          errors = instance.validate(machine).values.first
          expect(errors).to be_empty
          expect(instance.base_address).to eq(base_address)
        end
      end
    end
  end

  context "after finalize" do
    before {
      instance.finalize!
    }
    it "defaults verify_vmnet to true" do
      expect(instance.verify_vmnet).to eq(true)
    end
  end
end
