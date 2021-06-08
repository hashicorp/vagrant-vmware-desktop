require "log4r"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This prepares some variables for the environment hash so that
      # forwarded port collision detection works properly with VMware.
      class PrepareForwardedPortCollisionParams
        include Common

        def initialize(app, env)
          @app = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::prepare_fp_collision")
        end

        def call(env)
          # Get the forwared ports for all NAT devices and mark them as in use.
          env[:port_collision_extra_in_use] = env[:machine].provider.driver.all_forwarded_ports

          # Repair the forwarded port collisions
          env[:port_collision_repair] = true

          @app.call(env)
        end
      end
    end
  end
end
