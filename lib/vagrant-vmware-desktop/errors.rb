require "vagrant"

module HashiCorp
  module VagrantVMwareDesktop
    module Errors
      # This is the base class for all errors within the VMware Fusion
      # plugin.
      class Base < Vagrant::Errors::VagrantError
        error_namespace("hashicorp.vagrant_vmware_desktop.errors")

        def initialize(message=nil, *args)
          message ||= {}
          message[:product] = PRODUCT_NAME.capitalize

          super(message, *args)
        end
      end

      class FeatureNotSupported < Base
        error_key(:feature_not_supported)
      end

      class BoxVMXFileNotFound < Base
        error_key(:box_vmx_file_not_found)
      end

      class CannotGetSSHInfo < Base
        error_key(:cannot_get_ssh_info)
      end

      class CloneFolderExists < Base
        error_key(:clone_folder_exists)
      end

      class CloneFolderNotFolder < Base
        error_key(:clone_folder_not_folder)
      end

      class DiskNotCreated < Base
        error_key(:disk_not_created)
      end

      class DiskNotResizedSnapshot < Base
        error_key(:disk_not_resized_snapshot)
      end

      class MissingNATDevice < Base
        error_key(:missing_nat_device)
      end

      class BaseAddressRange < Base
        error_key(:base_address_range)
      end

      class DestroyInvalidState < Base
        error_key(:destroy_invalid_state)
      end

      class DriverInvalidResponse < Base
        error_key(:driver_invalid_response)
      end

      class DriverClonePermissionFailure < Base
        error_key(:driver_clone_permission_failure)
      end

      class DriverDHCPLeasesReadPerms < Base
        error_key(:driver_dhcp_leases_read_perms)
      end

      class DriverMissingDHCPLeasesFile < Base
        error_key(:driver_missing_dhcp_leases_file)
      end

      class DriverMissingFPNatConf < Base
        error_key(:driver_missing_fp_nat_conf)
      end

      class DriverMissingNetworkingFile < Base
        error_key(:driver_missing_networking_file)
      end

      class DriverMissingServiceStarter < Base
        error_key(:driver_missing_service_starter)
      end

      class DriverMissingVMNetCLI < Base
        error_key(:driver_missing_vmnet_cli)
      end

      class DriverMissingVMX < Base
        error_key(:driver_missing_vmx)
      end

      class DriverMissingVMXCLI < Base
        error_key(:driver_missing_vmx_cli)
      end

      class DriverNetworkingFileNotFound < Base
        error_key(:driver_networking_file_not_found)
      end

      class DriverNetworkingFileBadPermissions < Base
        error_key(:driver_networking_file_bad_permissions)
      end

      class DriverReadVersionFailed < Base
        error_key(:driver_read_version_failed)
      end

      class DriverVMNetCommandFailed < Base; end

      class DriverVMNetConfigureFailed < DriverVMNetCommandFailed
        error_key(:driver_vmnet_configure_failed)
      end

      class DriverVMNetStartFailed < DriverVMNetCommandFailed
        error_key(:driver_vmnet_start_failed)
      end

      class DriverVMNetStopFailed < DriverVMNetCommandFailed
        error_key(:driver_vmnet_stop_failed)
      end

      class ForwardedPortsCollideWithExistingNAT < Base
        error_key(:forwarded_ports_collide_with_existing_nat)
      end

      class ForwardedPortNoGuestIP < Base
        error_key(:forwarded_port_no_guest_ip)
      end

      class FusionUpgradeRequired < Base
        error_key(:fusion_upgrade_required)
      end

      class GuestMissingHGFS < Base
        error_key(:guest_missing_hgfs)
      end

      class HelperFailed < Base
        error_key(:helper_failed)
      end

      class HelperInstallFailed < Base
        error_key(:helper_install_failed)
      end

      class HelperInvalidCommand < Base
        error_key(:helper_invalid_command)
      end

      class HelperNotRoot < Base
        error_key(:helper_not_root)
      end

      class HelperRequiresCommand < Base
        error_key(:helper_requires_command)
      end

      class HelperRequiresDataFile < Base
        error_key(:helper_requires_data_file)
      end

      class HelperRequiresReinstall < Base
        error_key(:helper_requires_reinstall)
      end

      class HelperWrapperMissing < Base
        error_key(:helper_wrapper_missing)
      end

      class LinuxMountGIDFailed < Base
        error_key(:linux_mount_gid_failed)
      end

      class LinuxMountUIDFailed < Base
        error_key(:linux_mount_uid_failed)
      end

      class LinuxServiceInitScriptMissing < Base
        error_key(:linux_service_init_script_missing)
      end

      class NetworkingFileMissingVersion < Base
        error_key(:networking_file_missing_version)
      end

      class NetworkingFileUnsupportedVersion < Base
        error_key(:networking_file_unsupported_version)
      end

      class NetworkingHostOnlyCollision < Base
        error_key(:networking_host_only_collision)
      end

      class NetworkingNoSlotsForHighLevel < Base
        error_key(:networking_no_slots_for_high_level)
      end

      class NFSNoNetwork < Base
        error_key(:nfs_no_network)
      end

      class PackageNotSupported < Base
        error_key(:package_not_supported)
      end

      class RoutingTableError < Base; end

      class RoutingTableCommandNotFound < RoutingTableError
        error_key(:routing_table_command_not_found)
      end

      class RoutingTableLoadError < RoutingTableError
        error_key(:routing_table_load_error)
      end

      class RoutingTableUnsupportedOS < RoutingTableError
        error_key(:routing_table_unsupported_os)
      end

      class SharedFolderSymlinkFailed < Base
        error_key(:shared_folder_symlink_failed)
      end

      class SingleMachineLock < Base
        error_key(:single_machine_lock)
      end

      class StartTimeout < Base
        error_key(:start_timeout)
      end

      class UtilityUpgradeRequired < Base
        error_key(:utility_upgrade_required)
      end

      class VMNetDeviceCreateFailed < Base
        error_key(:vmnet_device_create_failed)
      end

      class VMNetDeviceRouteConflict < Base
        error_key(:vmnet_device_route_conflict)
      end

      class VMNetDevicesWontStart < Base
        error_key(:vmnet_devices_wont_start)
      end

      class VMNetSlotsFull < Base
        error_key(:vmnet_slots_full)
      end

      class VMNetNoIPV6 < Base
        error_key(:vmnet_no_ipv6)
      end

      class VMExecError < Base
        error_key(:vmexec_error)
      end

      class VMRunError < Base
        error_key(:vmrun_error)
      end

      class VMRunNotFound < Base
        error_key(:vmrun_not_Found)
      end

      class VMwareLinuxServiceWontStart < Base
        error_key(:vmware_linux_service_wont_start)
      end

      class VnetLibError < Base
        error_key(:vnetlib_error)
      end

      class VnetLibNotFound < Base
        error_key(:vnetlib_not_found)
      end

      class WorkstationUpgradeRequired < Base
        error_key(:workstation_upgrade_required)
      end

      ## Driver API

      class DriverAPICertificateError < Base
        error_key(:driver_api_certificate_error)
      end

      class DriverAPIKeyError < Base
        error_key(:driver_api_key_error)
      end

      class DriverAPIConnectionFailed < Base
        error_key(:driver_api_connection_failed)
      end

      class DriverAPIRequestUnexpectedError < Base
        error_key(:driver_api_request_unexpected_error)
      end

      class DriverAPIInvalidResponse < Base
        error_key(:driver_api_invalid_response)
      end

      class DriverAPIPortForwardListError < Base
        error_key(:driver_api_port_forward_list_error)
      end

      class DriverAPIDeviceCreateError < Base
        error_key(:driver_api_device_create_error)
      end

      class DriverAPIPortForwardError < Base
        error_key(:driver_api_port_forward_error)
      end

      class DriverAPIPortForwardPruneError < Base
        error_key(:driver_api_port_forward_prune_error)
      end

      class DriverAPIDeviceListError < Base
        error_key(:driver_api_device_list_error)
      end

      class DriverAPIVMwareVersionDetectionError < Base
        error_key(:driver_api_vmware_version_detection_error)
      end

      class DriverAPIVMwarePathsDetectionError < Base
        error_key(:driver_api_vmware_paths_detection_error)
      end

      class DriverAPIAddressReservationError < Base
        error_key(:driver_api_address_reservation_error)
      end
    end
  end
end
