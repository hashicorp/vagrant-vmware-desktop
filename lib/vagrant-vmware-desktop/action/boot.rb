# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

require "log4r"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This class boots the actual VMware VM. It also waits
      # for it to complete booting.
      class Boot
        include Common

        def initialize(app, env)
          @app = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::boot")
        end

        def call(env)
          env[:ui].info(I18n.t("hashicorp.vagrant_vmware_desktop.booting"))
          env[:machine].provider.driver.start(
            env[:machine].provider_config.gui)

          @app.call(env)
        end
      end
    end
  end
end
