# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

require "log4r"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This deletes the machine from the system.
      class Destroy
        include Common

        def initialize(app, env)
          @app = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::destroy")
        end

        def call(env)
          if env[:machine].provider.state.id == :running
            raise Errors::DestroyInvalidState
          end

          # As long as the VM exists, we destroy it.
          if env[:machine].provider.state.id != :not_created
            env[:ui].info(I18n.t("hashicorp.vagrant_vmware_desktop.destroying"))
            env[:machine].provider.driver.delete
            env[:machine].id = nil
          end

          @app.call(env)
        end
      end
    end
  end
end
