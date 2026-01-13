# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

require "log4r"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This class modifies the VMX file according to the advanced
      # provider configuration.
      class VMXModify
        include Common

        def initialize(app, env)
          @app    = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::vmx_modify")
        end

        def call(env)
          vmx_changes = env[:machine].provider_config.vmx
          if vmx_changes.length > 0
            @logger.info("Modifying VMX file according to user config...")
            env[:machine].provider.driver.vmx_modify do |vmx|
              vmx_changes.each do |key, value|
                if value.nil?
                  @logger.info("  - Delete: #{key}")
                  vmx.delete(key)
                else
                  @logger.info("  - Set: #{key} = '#{value}'")
                  vmx[key.to_s.downcase] = value
                end
              end
            end
          end

          # Carry on
          @app.call(env)
        end
      end
    end
  end
end
