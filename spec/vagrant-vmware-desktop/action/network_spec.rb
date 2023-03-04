# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

require "vagrant-vmware-desktop/config"
require "vagrant-vmware-desktop/errors"

require "spec_base"

describe HashiCorp::VagrantVMwareDesktop::Action::Network do
  let(:app) { double('app') }
  let(:machine) { instance_double(Vagrant::Machine) }
  let(:provider_config) { instance_double(HashiCorp::VagrantVMwareDesktop::Config) }
  let(:ui) { instance_double(Vagrant::UI::Basic) }
  let(:env) {{
    machine: machine,
    ui: ui,
  }}
  let(:environment) { instance_double(Vagrant::Environment) }
  let(:networks) {[]}
  let(:network_adapters) { [] }

  subject { described_class.new(app, env) }

  before do
    allow(machine).to receive(:provider_config) { provider_config }
    allow(machine).to receive(:env) { environment }
    allow(environment).to receive(:lock)
    allow(provider_config).to receive(:network_adapters) {{}}
    allow(machine).to receive_message_chain('config.vm.networks') { networks }
    allow(machine).to receive_message_chain('provider.driver.read_network_adapters') { network_adapters }
    allow(machine).to receive_message_chain('guest.capability')
    allow(app).to receive(:call)

    allow(ui).to receive(:info)
  end

  it "stubs its way through using base values" do
    subject.call(env)
  end

  context "with a bridged network that has an IP set" do
    let(:networks) {[
      [:public_network, {ip: "192.168.1.30"}]
    ]}

    it "results in a static network config" do
      subject.call(env)

      expect(machine.guest).to have_received(:capability).with(:configure_networks, [a_hash_including(type: :static)])
    end
  end

  context "#hostonly_config" do
    context "with IPv6 configuration provided" do
      it "should raise an exception" do
        expect{ subject.hostonly_config(ip: "fe00::0") }.to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::VMNetNoIPV6)
      end
    end
  end
end
