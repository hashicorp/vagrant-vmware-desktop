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
