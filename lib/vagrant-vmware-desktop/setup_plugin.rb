# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

require "vagrant-vmware-desktop/checkpoint_client"

module HashiCorp
  module VagrantVMwareDesktop
    class SetupPlugin
      def initialize(app, env)
        @app = app
      end

      def call(env)
        # Initialize i18n
        VagrantVMwareDesktop.init_i18n

        # Initialize logging
        VagrantVMwareDesktop.init_logging

        # Start the checks for the plugin and utility
        CheckpointClient.instance.setup(env[:env]).check

        @app.call(env)
      end
    end
  end
end
