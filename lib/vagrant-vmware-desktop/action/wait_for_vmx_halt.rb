# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

require "log4r"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This waits for the actual "vmx" process to finish running.
      class WaitForVMXHalt
        include Common

        def initialize(app, env)
          @app = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::wait_for_vmx_halt")
        end

        def call(env)
          @logger.info("Waiting for VMX process to go away...")
          if env[:machine].provider.driver.vmx_alive?
            env[:ui].info(I18n.t("hashicorp.vagrant_vmware_desktop.waiting_for_vmx_halt"))

            60.times do |i|
              if !env[:machine].provider.driver.vmx_alive?
                @logger.info("VMX process went away!")
                break
              end

              sleep 1
            end
          end

          @app.call(env)
        end
      end
    end
  end
end
