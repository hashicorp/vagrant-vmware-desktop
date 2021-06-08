require "vagrant-vmware-desktop/config"

require "spec_base"

describe HashiCorp::VagrantVMwareDesktop::Action::CheckExistingNetwork do
  let(:app) { double('app') }
  let(:machine) { instance_double(Vagrant::Machine) }
  let(:provider_config) { instance_double(HashiCorp::VagrantVMwareDesktop::Config) }
  let(:ui) { instance_double(Vagrant::UI::Basic) }
  let(:env) {{
    machine: machine,
    ui: ui,
  }}
  let(:environment) { instance_double(Vagrant::Environment) }
  let(:provider) { double('provider') }
  let(:driver) { double('driver') }

  subject { described_class.new(app, env) }

  before do
    allow(machine).to receive(:provider_config) { provider_config }
    allow(machine).to receive(:env) { environment }
    allow(environment).to receive(:lock).and_yield
    allow(provider_config).to receive(:verify_vmnet) { verify_vmnet }
    allow(app).to receive(:call)
    allow(machine).to receive(:provider) { provider }
    allow(provider).to receive(:driver) { driver }
    allow(driver).to receive(:verify_vmnet!)

    allow(ui).to receive(:info)
  end

  context "when verify_vmnet is true" do
    let(:verify_vmnet) { true }

    it "calls verify_vmnet! on the driver" do
      subject.call(env)
      expect(driver).to have_received(:verify_vmnet!)
    end
  end

  context "when verify_vmnet is false" do
    let(:verify_vmnet) { false }

    it "skips the call to verify_vmnet! on the driver" do
      subject.call(env)
      expect(driver).not_to have_received(:verify_vmnet!)
    end
  end
end
