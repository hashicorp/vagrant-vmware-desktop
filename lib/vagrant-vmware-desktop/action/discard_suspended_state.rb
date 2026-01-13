# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

require "log4r"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This discards the suspended state of the machine.
      class DiscardSuspendedState
        include Common

        def initialize(app, env)
          @app = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::discard_suspended_state")
        end

        def call(env)
          machine_state = env[:machine].provider.state.id
          if machine_state == :suspended
            env[:ui].info(I18n.t("hashicorp.vagrant_vmware_desktop.discarding_suspended_state"))
          end

          if machine_state != :not_created
            # We don't care if the machine is suspended or not, just always
            # delete the suspended state if it is created, since that is our job
            env[:machine].provider.driver.discard_suspended_state
          end

          @app.call(env)
        end
      end
    end
  end
end
