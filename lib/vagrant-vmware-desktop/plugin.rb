# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

begin
  require "vagrant"
rescue LoadError
  raise "The Vagrant VMware #{HashiCorp::VagrantVMwareDesktop::PRODUCT_NAME} provider must be run within Vagrant."
end

# This is a sanity check to make sure no one is attempting to install
# this into an early Vagrant version.
if Vagrant::VERSION < "1.2.0"
  raise "VMware #{HashiCorp::VagrantVMwareDesktop::PRODUCT_NAME} provider is only compatible with Vagrant 1.2+"
end

require "vagrant-vmware-desktop/setup_plugin"

module HashiCorp
  module VagrantVMwareDesktop
    class Plugin < Vagrant.plugin("2")
      name "VMware #{PRODUCT_NAME.capitalize} Provider"
      description <<-DESC
      This plugin installs a provider which allows Vagrant to manage
      VMware #{PRODUCT_NAME.capitalize} machines.
      DESC

      def self.provider_name
        "vmware_#{PRODUCT_NAME}".to_sym
      end

      def self.provider_options
        {
          box_format: ["vmware_desktop", "vmware_fusion", "vmware_workstation"],
          priority: 10,
        }
      end

      #--------------------------------------------------------------
      # Action Hooks
      #--------------------------------------------------------------
      # Plugin activation/licensing
      action_hook(:activation_start, :environment_load) do |h|
        h.append(SetupPlugin)
      end

      #--------------------------------------------------------------
      # VMware Provider
      #--------------------------------------------------------------

      # We register two providers now, vmware_desktop which is named after
      # this plugin's now current name, and vmware_PRODUCT_NAME which is
      # the legacy naming covering vmware_workstation and vmware_fusion.
      # This provides backwards compatibility ensuring things still work
      # after upgrading

      [:vmware_desktop, provider_name].each do |p_name|
        config(p_name, :provider) do
          require File.expand_path("../config", __FILE__)
          Config
        end

        provider_capability(p_name, :snapshot_list) do
          require File.expand_path("../cap/snapshot", __FILE__)
          Cap::Snapshot
        end

        provider_capability(p_name, :delete_all_snapshots) do
          require File.expand_path("../cap/snapshot", __FILE__)
          Cap::Snapshot
        end

        provider_capability(p_name, :delete_snapshot) do
          require File.expand_path("../cap/snapshot", __FILE__)
          Cap::Snapshot
        end

        provider(p_name, provider_options) do
          require File.expand_path("../provider", __FILE__)
          Provider
        end

        provider_capability(p_name, :forwarded_ports) do
          require File.expand_path("../cap/provider", __FILE__)
          Cap::Provider
        end

        provider_capability(p_name, :public_address) do
          require File.expand_path("../cap/provider", __FILE__)
          Cap::Provider
        end

        provider_capability(p_name, :nic_mac_addresses) do
          require File.expand_path("../cap/provider", __FILE__)
          Cap::Provider
        end

        provider_capability(p_name, :scrub_forwarded_ports) do
          require File.expand_path("../cap/provider", __FILE__)
          Cap::Provider
        end

        Vagrant::Util::Experimental.guard_with(:disks) do
          provider_capability(p_name, :set_default_disk_ext) do
            require File.expand_path("../cap/disk", __FILE__)
            Cap::Disk
          end

          provider_capability(p_name, :default_disk_exts) do
            require File.expand_path("../cap/disk", __FILE__)
            Cap::Disk
          end

          provider_capability(p_name, :configure_disks) do
            require File.expand_path("../cap/disk", __FILE__)
            Cap::Disk
          end

          provider_capability(p_name, :cleanup_disks) do
            require File.expand_path("../cap/disk", __FILE__)
            Cap::Disk
          end
        end
      end

      #--------------------------------------------------------------
      # Synced Folder
      #--------------------------------------------------------------

      synced_folder(:vmware) do
        require File.expand_path("../synced_folder", __FILE__)
        SyncedFolder
      end

      #--------------------------------------------------------------
      # Capabilities introduced by VMware provider
      #--------------------------------------------------------------

      guest_capability("linux", "verify_vmware_hgfs") do
        require_relative "guest_cap/linux/verify_vmware_hgfs"
        GuestCap::Linux::VerifyVMwareHGFS
      end

      guest_capability("linux", "mount_vmware_shared_folder") do
        require_relative "guest_cap/linux/mount_vmware_shared_folder"
        GuestCap::Linux::MountVMwareSharedFolder
      end

      autoload :Action, File.expand_path("../action", __FILE__)
    end
  end
end
