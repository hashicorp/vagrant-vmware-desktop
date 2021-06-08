require "log4r"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This stops the VMware machine.
      class Halt
        include Common

        def initialize(app, env)
          @app = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::halt")
        end

        def call(env)
          if env[:machine].provider.state.id == :running
            env[:ui].info(I18n.t("hashicorp.vagrant_vmware_desktop.stopping"))
            stop_mode = env[:force_halt] ? "hard" : "soft"
            env[:machine].provider.driver.stop(stop_mode)
          end

          @app.call(env)
        end
      end
    end
  end
end
