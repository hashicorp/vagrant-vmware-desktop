# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

require "log4r"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This fixes old-style machine IDs that were set in early versions of
      # the VMware providers.
      class FixOldMachineID
        include Common

        def initialize(app, env)
          @app = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::fix_old_machine_id")
        end

        def call(env)
          driver     = env[:machine].provider.driver
          machine_id = env[:machine].id
          if machine_id && driver.vmx_path.to_s != machine_id
            @logger.warn("Old-style ID found. Resetting to VMX path.")
            env[:machine].id = driver.vmx_path.to_s
          end

          @app.call(env)
        end
      end
    end
  end
end
