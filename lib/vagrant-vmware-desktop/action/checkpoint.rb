# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This class checks if there is a new release of the Vagrant
      # VMware desktop plugin or the Vagrant VMware utility available
      # and notifies the user
      class Checkpoint
        include Common

        def initialize(app, env)
          @app = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::checkpoint")
        end

        def call(env)
          if !env[:checkpoint_complete]
            if !CheckpointClient.instance.complete?
              @logger.debug("waiting for checkpoint to complete...")
            end
            CheckpointClient.instance.result.each_pair do |name, result|
              next if !result.is_a?(Hash)
              next if result["cached"]
              version_check(env, name, result)
              alerts_check(env, name, result)
            end
            @logger.debug("checkpoint data processing complete")
            env[:checkpoint_complete] = true
          end
          @app.call(env)
        end

        def alerts_check(env, name, result)
          full_name = "vagrant-vmware-#{name}"
          if result["alerts"] && !result["alerts"].empty?
            result["alerts"].group_by{|a| a["level"]}.each_pair do |_, alerts|
              alerts.each do |alert|
                date = nil
                begin
                  date = Time.at(alert["date"])
                rescue
                  date = Time.now
                end
                output = I18n.t("vagrant.hashicorp.vagrant_vmware_desktop.alert",
                  message: alert["message"],
                  date: date,
                  url: alert["url"]
                )
                case alert["level"]
                when "info"
                  alert_ui = Vagrant::UI::Prefixed.new(env.ui, full_name)
                  alert_ui.info(output)
                when "warn"
                  alert_ui = Vagrant::UI::Prefixed.new(env.ui, "#{full_name}-warning")
                  alert_ui.warn(output)
                when "critical"
                  alert_ui = Vagrant::UI::Prefixed.new(env.ui, "#{full_name}-alert")
                  alert_ui.error(output)
                end
              end
              env.ui.info("")
            end
          else
            @logger.debug("no alert notifications to display")
          end
        end

        def version_check(env, name, result)
          name = name.to_s
          latest_version = Gem::Version.new(result["current_version"])
          installed_version = Gem::Version.new(result["installed_version"])
          ui = Vagrant::UI::Prefixed.new(env.ui, "vagrant-vmware-#{name}")
          if latest_version > installed_version
            @logger.info("new version of Vagrant VMware #{name.capitalize} available - #{latest_version}")
            ui.info(I18n.t("hashicorp.vagrant_vmware_desktop.version_upgrade_available",
              name: name.capitalize,
              download_url: download_url["current_download_url"],
              latest_version: latest_version))
            env.ui.info("")
          else
            @logger.debug("Vagrant VMware #{name.capitalize} is currently up to date")
          end
        end
      end
    end
  end
end
