# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

require "log4r"

require "vagrant/util/scoped_hash_override"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This takes the configured synced folders on the VM and shares them
      # into the VMware guest.
      class ShareFolders
        include Common

        include Vagrant::Util::ScopedHashOverride

        def initialize(app, env)
          @app    = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::shared_folders")
        end

        def call(env)
          # Move on first so that the machine is powered on
          @app.call(env)

          shared_folders = {}
          env[:machine].config.vm.synced_folders.each do |id, data|
            data      = scoped_hash_override(data, :vmware)

            # Ignore disabled shared folders
            if data[:disabled]
              @logger.info("Disabled shared folder, ignoring: #{id}")
              next
            end

            # Ignore NFS shared folders as well
            next if data[:nfs]

            # Use it!
            shared_folders[id] = data
          end

          if shared_folders.empty?
            @logger.info("No shared folders. Doing nothing.")
            return
          end

          # Verify that the machine can actually support sharing folders
          # if we can.
          if env[:machine].guest.capability?(:verify_vmware_hgfs)
            @logger.info("Verifying HGFS is on the guest...")
            if !env[:machine].guest.capability(:verify_vmware_hgfs)
              raise Errors::GuestMissingHGFS
            end
          end

          # Get the SSH info which we'll use later
          ssh_info = env[:machine].ssh_info

          # short guestpaths first, so we don't step on ourselves
          shared_folders = shared_folders.dup.sort_by do |id, data|
            if data[:guestpath]
              data[:guestpath].length
            else
              # A long enough path to just do this at the end.
              10000
            end
          end

          @logger.info("Preparing shared folders with VMX...")
          env[:ui].info I18n.t("hashicorp.vagrant_vmware_desktop.sharing_folders")
          env[:machine].provider.driver.enable_shared_folders
          shared_folders.each do |id, data|
            id        = id.gsub('/', env[:machine].provider_config.shared_folder_special_char)
            path      = File.expand_path(data[:hostpath], env[:root_path])
            guestpath = data[:guestpath]

            env[:ui].info(I18n.t("hashicorp.vagrant_vmware_desktop.sharing_folder_single",
                                 hostpath: path,
                                 guestpath: guestpath))

            env[:machine].provider.driver.share_folder(id, path)

            # Remove trailing slashes
            guestpath = guestpath.gsub(/\/+$/, "")

            # Calculate the owner and group
            data[:owner] ||= ssh_info[:username]
            data[:group] ||= ssh_info[:username]

            # Mount it!
            env[:machine].guest.capability(
              :mount_vmware_shared_folder, id, data[:guestpath], data)
          end
        end
      end
    end
  end
end
