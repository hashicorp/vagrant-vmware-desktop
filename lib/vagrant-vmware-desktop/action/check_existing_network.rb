# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

require "log4r"

require "vagrant-vmware-desktop/helper/lock"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This class checks that the existing network interfaces are all
      # properly running and are happy, and shows an error otherwise.
      class CheckExistingNetwork
        include Common

        def initialize(app, env)
          @app    = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::check_existing_network")
        end

        def call(env)
          if env[:machine].provider_config.verify_vmnet
            @logger.info("Checking if the vmnet devices are healthy...")
            Helper::Lock.lock(env[:machine], "vmware-network") do
              env[:ui].info(I18n.t("hashicorp.vagrant_vmware_desktop.verifying_vmnet"))
              env[:machine].provider.driver.verify_vmnet!
            end
          else
            @logger.info("Skipping vmnet device verificiation, vmnet_verify option is false.")
            env[:ui].info(I18n.t("hashicorp.vagrant_vmware_desktop.skipping_vmnet_verify"))
          end

          @app.call(env)
        end
      end
    end
  end
end
