module HashiCorp
  module VagrantVMwareDesktop
    module Action
      class MessageAlreadyRunning
        include Common

        def initialize(app, env)
          @app = app
        end

        def call(env)
          env[:ui].info(I18n.t("hashicorp.vagrant_vmware_desktop.already_running"))
          @app.call(env)
        end
      end
    end
  end
end
