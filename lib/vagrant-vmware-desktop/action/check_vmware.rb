# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This class checks to verify that VMware is properly
      # installed and configured in such a way that Vagrant can use it.
      class CheckVMware
        include Common

        def initialize(app, env)
          @app = app
        end

        def call(env)
          if !env[:check_vmware_complete]
            # Tell the provider to verify itself.
            env[:machine].provider.driver.verify!

            # Mark that we completed it so we only do that once
            env[:check_vmware_complete] = true
          end

          # Carry on
          @app.call(env)
        end
      end
    end
  end
end
