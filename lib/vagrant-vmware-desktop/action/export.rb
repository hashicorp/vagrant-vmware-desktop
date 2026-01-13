# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      class Export
        def initialize(app, env)
          @app = app
        end

        def call(env)
          @env = env

          raise Vagrant::Errors::VMPowerOffToPackage if \
            @env[:machine].state.id != :not_running

          path = File.join(@env["export.temp_dir"], "box.vmx")

          if Vagrant::Util::Platform.respond_to?(:wsl?) && Vagrant::Util::Platform.wsl?
            path = Vagrant::Util::Platform.wsl_to_windows_path(path)
          end

          @env[:ui].info I18n.t("vagrant.actions.vm.export.exporting")
          @env[:machine].provider.driver.export(path)

          @app.call(env)
        end
      end
    end
  end
end
