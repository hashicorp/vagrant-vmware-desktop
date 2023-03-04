# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      class PrepareSyncedFolderCleanup
        include Common

        def initialize(app, env)
          @app = app
        end

        def call(env)
          env[:nfs_valid_ids] = env[:machine].provider.driver.read_running_vms

          @app.call(env)
        end
      end
    end
  end
end
