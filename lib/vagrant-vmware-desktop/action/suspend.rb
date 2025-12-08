# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

require "log4r"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This suspends the VMware machine.
      class Suspend
        include Common

        def initialize(app, env)
          @app = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::suspend")
        end

        def call(env)
          if env[:machine].provider.state.id == :running
            env[:ui].info(I18n.t("hashicorp.vagrant_vmware_desktop.suspending"))
            env[:machine].provider.driver.suspend
          end

          @app.call(env)
        end
      end
    end
  end
end
