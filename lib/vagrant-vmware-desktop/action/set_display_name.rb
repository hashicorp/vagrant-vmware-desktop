# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

require "log4r"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This class sets the default displayName for the VM.
      class SetDisplayName
        include Common

        def initialize(app, env)
          @app    = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::vmx_modify")
        end

        def call(env)
          default_name = "#{env[:root_path].basename.to_s}: #{env[:machine].name}"

          @logger.info("Setting the default display name: #{default_name}")
          env[:machine].provider.driver.vmx_modify do |vmx|
            # Delete any alternate casings
            vmx.keys.each do |key|
              if key.downcase == "displayname"
                vmx.delete(key)
              end
            end

            # Set the displayName
            vmx["displayname"] = default_name
          end

          # Carry on
          @app.call(env)
        end
      end
    end
  end
end
