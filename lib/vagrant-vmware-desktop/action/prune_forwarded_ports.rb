# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

require "log4r"

require "vagrant-vmware-desktop/helper/lock"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This action prunes out forwarded ports that aren't in use any longer,
      # usually by machines that don't exist anymore.
      class PruneForwardedPorts
        include Common

        def initialize(app, env)
          @app    = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::prune_forwarded_ports")
        end

        def call(env)
          @logger.info("Pruning forwarded ports...")
          Helper::Lock.lock(env[:machine], "vmware-network") do
            env[:machine].provider.driver.prune_forwarded_ports
          end

          # Carry on
          @app.call(env)
        end
      end
    end
  end
end
