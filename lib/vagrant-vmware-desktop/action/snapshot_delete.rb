require "log4r"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This snapshots the VMware machine.
      class SnapshotDelete
        include Common

        def initialize(app, env)
          @app = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::snapshot_delete")
        end

        def call(env)
          env[:ui].info(I18n.t(
            "hashicorp.vagrant_vmware_desktop.snapshot_deleting",
            name: env[:snapshot_name]))
          env[:machine].provider.driver.snapshot_delete(env[:snapshot_name])

          @app.call(env)
        end
      end
    end
  end
end
