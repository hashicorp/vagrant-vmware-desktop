# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

require 'vagrant/util/template_renderer'

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      class PackageVagrantfile
        # For TemplateRenderer
        include Vagrant::Util

        def initialize(app, env)
          @app = app
        end

        def call(env)
          @env = env
          create_metadata
          @app.call(env)
        end

        # This method creates the auto-generated Vagrantfile at the root of the
        # box. This Vagrantfile contains the MAC address so that the user doesn't
        # have to worry about it.
        #
        # @note This is deprecated as the base mac is no longer required to be set.
        #       The method (and template) are preserved to easily allow enabling
        #       this functionality in the future for a different purpose if requried.
        def create_vagrantfile
          mac_addresses = @env[:machine].provider.driver.read_mac_addresses
          base_mac = mac_addresses[mac_addresses.keys.min]
          File.open(File.join(@env["export.temp_dir"], "Vagrantfile"), "w") do |f|
            f.write(TemplateRenderer.render("package_Vagrantfile", {
              base_mac: base_mac
            }))
          end
        end

        # Creates a metadata.json file which includes provider information
        def create_metadata
          File.open(File.join(@env["export.temp_dir"], "metadata.json"), "w") do |f|
            f.write({provider: "vmware_desktop"}.to_json)
          end
        end
      end
    end
  end
end
