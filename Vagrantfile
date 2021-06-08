# -*- mode: ruby -*-
# vi: set ft=ruby :

build_boxes = [
  'osx-10.15',
  'win-7'
]

script_env_vars = Hash[
  ENV.map do |key, value|
    if key.start_with?('VAGRANT_RUNNER_')
      [key.sub('VAGRANT_RUNNER_', ''), value]
    end
  end.compact
]

Vagrant.configure("2") do |config|
  build_boxes.each do |box_basename|
    config.vm.define(box_basename) do |box_config|
      if box_basename.start_with?("win")
        provision_script = "package/vagrant/win.ps1"
        box_config.vm.communicator = 'winrm'
        if File.exist?("Win_CodeSigning.p12")
          box_config.vm.provision "file", source: "Win_CodeSigning.p12", destination: "C:/Users/vagrant/Win_CodeSigning.p12"
        end
      else
        provision_script = "package/vagrant/osx.sh"
        box_config.vm.provision 'shell', inline: "sysctl -w net.inet.tcp.win_scale_factor=8\nsysctl " \
                                                 "-w net.inet.tcp.autorcvbufmax=33554432\nsysctl -w " \
                                                 "net.inet.tcp.autosndbufmax=33554432\n"
        ["MacOS_CodeSigning.p12", "MacOS_PackageSigning.cert", "MacOS_PackageSigning.key"].each do |path|
          if File.exist?(path)
            box_config.vm.provision "file", source: path, destination: "/Users/vagrant/#{path}"
          end
        end
      end

      box_config.vm.box = "hashicorp-vagrant/#{box_basename}"
      box_config.vm.provision "shell", path: provision_script, env: script_env_vars

      config.vm.provider :vmware_desktop do |v|
        v.vmx["memsize"] = "4096"
        v.vmx["numvcpus"] = "2"
        v.vmx["cpuid.coresPerSocket"] = "1"
      end
    end
  end
end
