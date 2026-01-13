# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

require "log4r"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This class removes all shared folders from the VM.
      class ClearSharedFolders
        include Common

        def initialize(app, env)
          @app    = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::clear_shared_folders")
        end

        def call(env)
          @logger.info("Clearing shared folders")
          env[:machine].provider.driver.clear_shared_folders

          # Carry on
          @app.call(env)
        end
      end
    end
  end
end
