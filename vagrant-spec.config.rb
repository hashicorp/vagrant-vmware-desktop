Vagrant::Spec::Acceptance.configure do |c|
  c.vagrant_path = "/usr/bin/vagrant"
  c.provider "vmware_fusion",
    box: "/Users/mitchellh/Downloads/package_fusion.box"
end
