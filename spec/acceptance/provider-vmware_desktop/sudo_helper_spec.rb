# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

shared_examples 'provider/sudo_helper' do |provider, options|
  [:box, :plugin].each do |option_key|
    if !options[:box]
      raise ArgumentError,
        "#{option_key} option must be specified for provider: #{provier}"
    end
  end

  include_context "acceptance"

  let(:vmware_install_dir) { Dir.glob("#{environment.homedir}/gems/*/gems/vagrant-vmware-*").first.to_s }
  let(:wrapper_path) {
    os = RbConfig::CONFIG["host_os"]
    os = "darwin" if os =~ /darwin/
    os = "linux" if os =~ /linux/
    cpu = RbConfig::CONFIG["host_cpu"]
    cpu = "amd64" if cpu == "x86_64"
    File.join(vmware_install_dir, "bin", "vagrant_vmware_desktop_sudo_helper_wrapper_#{os}_#{cpu}")
  }
  let(:helper_path) { File.join(vmware_install_dir, "bin", "vagrant_vmware_desktop_sudo_helper") }
  let(:wrapper_installed_path) { File.join(vmware_install_dir, "bin", "wrapper_installed") }
  let(:plugin_provider) { provider.to_s.split("_").last }

  before do
    environment.skeleton("sudo_helper")
    assert_execute("vagrant", "box", "add", "box", options[:box])
    assert_execute("vagrant", "plugin", "install", options[:plugin])
    expect(File.exist?(vmware_install_dir)).to be_truthy
  end

  context "with successfully created VM" do
    after do
      assert_execute("vagrant", "destroy", "--force")
    end

    it "modifies permission/ownership of files used as root" do
      status("Test: machine is created successfully")
      result = execute("vagrant", "up", "--provider=#{provider}")
      expect(result).to exit_with(0)

      status("Test: bin files are owned by root")
      wrapper_stat = File.stat(wrapper_path)
      helper_stat = File.stat(helper_path)
      wrapper_installed_stat = File.stat(wrapper_installed_path)

      expect(wrapper_stat.uid).to eq(0)
      expect(wrapper_installed_stat.uid).to eq(0)
      expect(helper_stat.uid).to eq(0)

      expect(wrapper_stat.mode).to eq(0104755)
      expect(wrapper_installed_stat.mode).to eq(0100644)
      expect(helper_stat.mode).to eq(0100644)

      status("Test: ruby lib files are owned by root")
      rb_files = Dir.glob(File.join(vmware_install_dir, "lib", "**", "**", "*.rb"))
      expect(rb_files.count).not_to eq(0)
      rb_files.each do |rb_path|
        rb_stat = File.stat(rb_path)
        expect(rb_stat.uid).to eq(0)
        expect(rb_stat.mode).to eq(0100644)
      end

      status("Test: all wrapper binaries are owned by root")
      wp_files = Dir.glob(File.join(vmware_install_dir, "bin", "vagrant_vmware_desktop_sudo_helper_wrapper*"))
      expect(wp_files.count).not_to eq(0)
      wp_files.each do |wp_path|
        wp_stat = File.stat(wp_path)
        expect(wp_stat.uid).to eq(0)
      end
    end
  end

  it "will fail to start on invalid file ownership" do
    status("Test: machine is created successfully")
    result = execute("vagrant", "up", "--provider=#{provider}")
    expect(result).to exit_with(0)
    status("Test: replace sudo wrapper with modified contents")
    FileUtils.mv(helper_path, "#{helper_path}.bak")
    File.write(helper_path, "`touch #{environment.homedir}/escalation.txt`\nexit 0")
    result = execute("vagrant", "reload")
    expect(result).not_to exit_with(0)
    expect(result.stderr).to match(/improperly setup/)
    expect(File.exist?("#{environment.homedir}/escalation.txt")).not_to be_truthy
  end
end
