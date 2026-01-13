# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      class PruneNFSExports
        include Common

        def initialize(app, env)
          @app = app
        end

        def call(env)
          if env[:host]
            vms = env[:machine].provider.driver.read_running_vms
            env[:host].nfs_prune(vms)
          end

          @app.call(env)
        end
      end
    end
  end
end
