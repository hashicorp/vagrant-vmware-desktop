# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      class WaitForAddress
        include Common

        def initialize(app, env)
          @app = app
        end

        def call(env)
          env[:ui].info(
            I18n.t("hashicorp.vagrant_vmware_desktop.waiting_for_address"))
          while true
            # If we're interrupted then just back out
            return if env[:interrupted]

            # If we have an IP we are don
            break if env[:machine].provider.driver.read_ip(
              env[:machine].provider_config.enable_vmrun_ip_lookup
            )

            sleep 1
          end

          @app.call(env)
        end
      end
    end
  end
end
