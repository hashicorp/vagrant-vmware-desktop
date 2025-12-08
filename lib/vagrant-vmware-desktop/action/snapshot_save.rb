# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

require "log4r"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This snapshots the VMware machine.
      class SnapshotSave
        include Common

        def initialize(app, env)
          @app = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::snapshot_save")
        end

        def call(env)
          env[:ui].info(I18n.t(
            "hashicorp.vagrant_vmware_desktop.snapshot_saving",
            name: env[:snapshot_name]))
          env[:machine].provider.driver.snapshot_take(env[:snapshot_name])

          @app.call(env)
        end
      end
    end
  end
end
