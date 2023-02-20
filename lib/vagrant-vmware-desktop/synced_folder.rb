# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

module HashiCorp
  module VagrantVMwareDesktop
    class SyncedFolder < Vagrant.plugin("2", :synced_folder)
      def initialize(*args)
        super

        @logger = Log4r::Logger.new("hashicorp::provider::vmware::synced_folder")
      end

      def usable?(machine)
        # These synced folders only work if the provider is VMware
        [Plugin.provider_name, :vmware_desktop].include?(machine.provider_name) &&
          machine.provider_config.functional_hgfs
      end

      def prepare(machine, folders, _opts)
        # We don't do anything prior to the machine booting
      end

      def enable(machine, folders, _opts)
        # Verify that the machine can actually support sharing folders
        # if we can.
        if machine.guest.capability?(:verify_vmware_hgfs)
          machine.ui.info I18n.t("hashicorp.vagrant_vmware_desktop.waiting_for_hgfs")
          if !machine.guest.capability(:verify_vmware_hgfs)
            raise Errors::GuestMissingHGFS
          end
        end

        # Get the SSH info which we'll use later. We retry a few times
        # since sometimes it seems to return nil.
        ssh_info = nil
        10.times do |i|
          ssh_info = machine.ssh_info
          break if ssh_info
          sleep 1
        end

        if ssh_info == nil
          raise Errors::CannotGetSSHInfo
        end

        # short guestpaths first, so we don't step on ourselves
        shared_folders = folders.dup.sort_by do |id, data|
          if data[:guestpath]
            data[:guestpath].length
          else
            # A long enough path to just do this at the end.
            10000
          end
        end

        @logger.info("Preparing shared folders with VMX...")
        machine.ui.info I18n.t("hashicorp.vagrant_vmware_desktop.sharing_folders")
        machine.provider.driver.enable_shared_folders
        shared_folders.each do |id, data|
          id        = id.gsub(%r{[:\\/]}, machine.provider_config.shared_folder_special_char)
          path      = data[:hostpath]
          guestpath = data[:guestpath]

          message = I18n.t("hashicorp.vagrant_vmware_desktop.sharing_folder_single",
                           hostpath: path,
                           guestpath: guestpath)
          if Vagrant::VERSION < "1.5.0"
            machine.ui.info(message)
          else
            machine.ui.detail(message)
          end

          machine.provider.driver.share_folder(id, path)

          # Remove trailing slashes
          guestpath = guestpath.gsub(/\/+$/, "")

          # Calculate the owner and group
          data[:owner] ||= ssh_info[:username]
          data[:group] ||= ssh_info[:username]

          # Mount it!
          machine.guest.capability(
            :mount_vmware_shared_folder, id, data[:guestpath], data)
        end
      end

      def cleanup(machine, opts)
        if machine.id && machine.id != ""
          @logger.info("Clearing shared folders")
          machine.provider.driver.clear_shared_folders
        end
      end
    end
  end
end
