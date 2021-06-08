module HashiCorp
  module VagrantVMwareDesktop
    module Action
      class MessageNotRunning
        include Common

        def initialize(app, env)
          @app = app
        end

        def call(env)
          env[:ui].info(I18n.t("hashicorp.vagrant_vmware_desktop.not_running"))
          @app.call(env)
        end
      end
    end
  end
end
