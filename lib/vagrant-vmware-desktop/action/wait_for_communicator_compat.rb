module HashiCorp
  module VagrantVMwareDesktop
    module Action
      class WaitForCommunicatorCompat
        include Common

        def initialize(app, env)
          @app = app
        end

        def call(env)
          env[:ui].info(
            I18n.t("hashicorp.vagrant_vmware_desktop.waiting_for_boot"))
          while true
            # If we're interrupted then just back out
            return if env[:interrupted]

            if env[:machine].communicate.ready?
              env[:ui].info(I18n.t(
                "hashicorp.vagrant_vmware_desktop.booted_and_ready"))
              break
            end

            sleep 2
          end

          @app.call(env)
        end
      end
    end
  end
end
