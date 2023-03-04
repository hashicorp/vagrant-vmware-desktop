# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      class MessageNotCreated
        include Common

        def initialize(app, env)
          @app = app
        end

        def call(env)
          env[:ui].info(I18n.t("hashicorp.vagrant_vmware_desktop.not_created"))
          @app.call(env)
        end
      end
    end
  end
end
