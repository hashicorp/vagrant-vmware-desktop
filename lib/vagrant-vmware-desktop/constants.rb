module HashiCorp
  module VagrantVMwareDesktop
    version_text_path = File.expand_path(File.join(__dir__, "../../versions/desktop.txt"))
    if File.exist?(version_text_path)
      VERSION = File.read(version_text_path)
    else
      VERSION = "STUB"
    end

    # This is the name of the gem.
    #
    # @return [String]
    PRODUCT_NAME = RbConfig::CONFIG["host_os"].include?("darwin") ? "fusion" : "workstation"
    PLUGIN_NAME = "vagrant-vmware-desktop"
  end
end
