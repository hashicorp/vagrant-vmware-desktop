# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

require "vagrant"
require "ipaddr"

module HashiCorp
  module VagrantVMwareDesktop
    class Config < Vagrant.plugin("2", :config)

      # Valid list of values for allowlist verified
      VALID_ALLOWLIST_VERIFIED_VALUES = [true, false, :disable_warning].freeze

      # Defaults for connecting to the utility service
      DEFAULT_UTILITY_HOST = "127.0.0.1".freeze
      DEFAULT_UTILITY_PORT = 9922

      # VMware organizationally unique identifier. MAC needs to start with this value to work.
      VMWARE_MAC_OUI = "00:50:56".freeze
      # Regexp pattern for matching valid VMware OUI
      VMWARE_MAC_PATTERN = /^#{Regexp.escape(VMWARE_MAC_OUI)}/
      # Regexp pattern for matching valid MAC addresses
      MAC_ADDRESS_PATTERN = /^([0-9A-F]{2}:){5}[0-9A-F]{2}$/

      # Boolean value to flag if this VMware box has been properly configured
      # for allowed VMX settings
      #
      # @return [Boolean, Symbol] true if verified, false if unverified,
      #   :disable_warning if silenced
      # @note deprecated in favor of `:allowlist_verified`
      attr_reader :allowlist_verified

      # Set a specific address for the guest which should be reserved
      # from the DHCP server. Requires the `base_mac` to be set.
      #
      # @return [String]
      attr_accessor :base_address

      # Set a custom MAC address for the default NAT interface
      #
      # @return [String]
      attr_accessor :base_mac

      # If set, the VM will be cloned to this directory rather than
      # in the ".vagrant" directory. This is useful for backup reasons (to
      # specifically NOT back up the VM).
      #
      # @return [String]
      attr_accessor :clone_directory

      # If set to `true`, then Vagrant attempts to use `vmrun
      # getGuestIPAddress` to look up the guest's IP, and will fall back to VMX
      # + DHCP lease file parsing only if that fails. This is the default.
      #
      # If set to `false`, then Vagrant will skip the `getGuestIPAddress`
      # attempt.
      #
      # @return [Boolean]
      attr_accessor :enable_vmrun_ip_lookup

      # Can be used to override the user VMware license information
      # detected by the Vagrant VMware utility. This should not be
      # visible to the public, but should be accessible to allow
      # for workarounds and debugging.
      #
      # @return [String]
      attr_accessor :force_vmware_license

      # If set, VMware synced folders will be attempted.
      #
      # @return [Boolean]
      attr_accessor :functional_hgfs

      # If set to `true`, then VMware VM will be launched with a GUI.
      #
      # @return [Boolean]
      attr_accessor :gui

      # If set to `true`, then VMware VM will be cloned using a linked clone
      #
      # @return [Boolean]
      attr_accessor :linked_clone

      # Device to use for NAT interface. By default the NAT interface will
      # be detected automatically.
      #
      # @return [String]
      attr_accessor :nat_device

      # The defined network adapters for the VMware machine.
      #
      # @return [Hash]
      attr_reader :network_adapters

      # Integer value of the number of seconds to pause after applying port
      # forwarding configuration. This gives time for the guest to re-aquire
      # a DHCP address if it detects a lost connection and drops its current
      # IP address when the VMware network service is restarted
      #
      # @return [Integer]
      attr_accessor :port_forward_network_pause

      # This is the character that will be used to replace any '/' characters
      # within shared folders to build the ID. WARNING: Modifying this is
      # for ADVANCED usage only and can very easily cause the provider to
      # break.
      #
      # @return [String]
      attr_accessor :shared_folder_special_char

      # Use the public IP address and port for connecting to the guest VM. By
      # default this is false which directs the SSH connection through the
      # forwarded SSH port on the localhost.
      #
      # @return [Boolean]
      attr_accessor :ssh_info_public

      # If set, the default mount point used by open-vm-tools will
      # be unmounted.
      #
      # @return [Boolean]
      attr_accessor :unmount_default_hgfs

      # Host for connecting to the utility service
      #
      # @return [String]
      attr_accessor :utility_host

      # Port for connecting to the utility service
      #
      # @return [Integer]
      attr_accessor :utility_port

      # Path to the certificate used when connecting to the utility service
      #
      # @return [String]
      attr_accessor :utility_certificate_path

      # If set to `true`, then Vagrant verifies whether the vmnet devices are
      # healthy before using them. This is the default behavior.
      #
      # Setting this to `false` skips the verify behavior, which might in some
      # cases allow Vagrant to boot machines in a mostly-working state,
      # skipping code that would proactively bail out.
      #
      # @return [Boolean]
      attr_accessor :verify_vmnet

      # Hash of VMX key/values to set or unset. The keys should be strings.
      # If the value is nil then the key will be deleted.
      #
      # @return [Hash<String, String>]
      attr_reader :vmx

      def initialize
        @allowlist_verified         = UNSET_VALUE
        @base_address               = UNSET_VALUE
        @base_mac                   = UNSET_VALUE
        @clone_directory            = UNSET_VALUE
        @functional_hgfs            = UNSET_VALUE
        @unmount_default_hgfs       = UNSET_VALUE
        @enable_vmrun_ip_lookup     = UNSET_VALUE
        @gui                        = UNSET_VALUE
        @force_vmware_license       = UNSET_VALUE
        @linked_clone               = UNSET_VALUE
        @nat_device                 = UNSET_VALUE
        @network_adapters           = {}
        @shared_folder_special_char = UNSET_VALUE
        @verify_vmnet               = UNSET_VALUE
        @vmx                        = {}
        @port_forward_network_pause = UNSET_VALUE
        @ssh_info_public            = UNSET_VALUE
        @utility_host               = UNSET_VALUE
        @utility_port               = UNSET_VALUE
        @utility_certificate_path   = UNSET_VALUE

        @logger = Log4r::Logger.new("hashicorp::provider::vmware::config")

        # Setup a NAT adapter by default
        network_adapter(0, :nat, :auto_config => false)
      end

      def merge(other)
        super.tap do |result|
          vmx = {}
          vmx.merge!(@vmx) if @vmx
          vmx.merge!(other.vmx) if other.vmx
          result.instance_variable_set(:@vmx, vmx)
        end
      end

      # Shortcut for setting CPU count for the virtual machine.
      #
      # @param [Integer, String] count
      def cpus=(count)
        vmx["numvcpus"] = count.to_s
      end

      # Sets the memory (in megabytes). This is shorthand for
      # setting the VMX property directly.
      #
      # @param [String] size
      def memory=(size)
        vmx["memsize"] = size.to_s
      end

      # This defines a network adapter for the VM in order to
      # provide networking access to the machine.
      #
      # @param [Integer] slot
      # @param [Symbol] type
      # @param [Hash] options The options for this network adapter.
      def network_adapter(slot, type, options=nil)
        @network_adapters[slot] = [type, options || {}]
      end

      # Boolean value to flag if this VMware box has been properly configured
      # for whitelisted VMX settings
      #
      # @param [Boolean, Symbol] value
      # @return [Boolean, Symbol] true if verified, false if unverified, :disable_warning if silenced
      # @note deprecated for `#allowlist_verified=`
      def whitelist_verified=(value)
        self.allowlist_verified = value
      end

      # Boolean value to flag if this VMware box has been properly configured
      # for whitelisted VMX settings
      #
      # @return [Boolean, Symbol] true if verified, false if unverified, :disable_warning if silenced
      # @note deprecated for `#allowlist_verified`
      def whitelist_verified
        allowlist_verified
      end

      # Boolean value to flag if this VMware box has been properly configured
      # for allowlisted VMX settings
      #
      # @param [Boolean, Symbol] value
      # @return [Boolean, Symbol] true if verified, false if unverified, :disable_warning if silenced
      # @note deprecated for `allowlist_verified`
      def allowlist_verified=(value)
        value = value.to_sym if value.is_a?(String)
        @allowlist_verified = value
      end

      # This is the hook that is called to finalize the object before it
      # is put into use.
      def finalize!
        @clone_directory = nil if @clone_directory == UNSET_VALUE
        @clone_directory ||= ENV["VAGRANT_VMWARE_CLONE_DIRECTORY"]

        @enable_vmrun_ip_lookup = true if @enable_vmrun_ip_lookup == UNSET_VALUE

        @functional_hgfs = true if @functional_hgfs == UNSET_VALUE
        @unmount_default_hgfs = true if @unmount_default_hgfs == UNSET_VALUE

        # Default is to not show a GUI
        @gui = false if @gui == UNSET_VALUE

        if @shared_folder_special_char == UNSET_VALUE
          @shared_folder_special_char = '-'
        end

        if @nat_device == UNSET_VALUE
          @nat_device = nil
        else
          @nat_device = @nat_device.to_s
        end

        @verify_vmnet = true if @verify_vmnet == UNSET_VALUE
        @linked_clone = true if @linked_clone == UNSET_VALUE
        @allowlist_verified = false if @allowlist_verified == UNSET_VALUE
        @port_forward_network_pause = 0 if @port_forward_network_pause == UNSET_VALUE
        @ssh_info_public = false if @ssh_info_public == UNSET_VALUE
        @utility_host = DEFAULT_UTILITY_HOST if @utility_host == UNSET_VALUE
        @utility_port = DEFAULT_UTILITY_PORT if @utility_port == UNSET_VALUE
        if @utility_certificate_path == UNSET_VALUE
          @utility_certificate_path = detect_certificate_path
        end

        if @base_mac == UNSET_VALUE
          @base_mac = nil
        else
          @base_mac = @base_mac.to_s.upcase
          if !@base_mac.include?(":")
            @base_mac = @base_mac.scan(/../).join(":")
          end
          network_adapters[0].last[:mac_address] = @base_mac
        end
        @base_address = nil if @base_address == UNSET_VALUE
        @force_vmware_license = nil if @force_vmware_license == UNSET_VALUE
        @clone_directory = VagrantVMwareDesktop.wsl_to_windows_path(@clone_directory) if @clone_directory
      end

      # This is called to validate the configuration for the VMware
      # adapter. This is only called if we are actually booting up a VMware
      # machine.
      def validate(machine)
        errors = _detected_errors

        if @network_adapters[0][0] != :nat
          errors << I18n.t("hashicorp.vagrant_vmware_desktop.config.non_nat_adapter_zero")
        end

        # If the base_mac or base_address is set within the vm configuration
        # and has not been set within the provider config set them here.
        if !@base_mac && machine.config.vm.base_mac
          @base_mac = machine.config.vm.base_mac.to_s.upcase
          if !@base_mac.include?(":")
            @base_mac = @base_mac.scan(/../).join(":")
          end
          network_adapters[0].last[:mac_address] = @base_mac
        end

        if !@base_address && machine.config.vm.respond_to?(:base_address) && machine.config.vm.base_address
          @base_address = machine.config.vm.base_address
        end

        if @base_mac
          if @base_mac !~ MAC_ADDRESS_PATTERN
            errors << I18n.t("hashicorp.vagrant_vmware_desktop.config.base_mac_invalid",
              mac: @base_mac)
          end

          if @base_mac !~ VMWARE_MAC_PATTERN
            @logger.warn("Base MAC address is set but is not using the VMWare Organizationally Unique " \
              "Identifier (#{VMWARE_MAC_OUI}). If networking problems persist, update the MAC address.")
          end
        end

        if @base_address
          begin
            IPAddr.new(@base_address)
          rescue IPAddr::InvalidAddressError
            errors << I18n.t("hashicorp.vagrant_vmware_desktop.config.base_address_invalid",
              address: @base_address)
          end
        end

        if @base_address && !@base_mac
          errors << I18n.t("hashicorp.vagrant_vmware_desktop.config.base_address_without_mac")
        end

        if @allowlist_verified && !VALID_ALLOWLIST_VERIFIED_VALUES.include?(@allowlist_verified)
          errors << I18n.t("hashicorp.vagrant_vmware_desktop.config.allowlist_verify_value_invalid",
            valid_values: VALID_ALLOWLIST_VERIFIED_VALUES.map(&:inspect).join(', '))
        end

        { "VMware Desktop Provider" => errors }
      end

      # Locate directory with utility service certificate files. Error if
      # path cannot be located
      def detect_certificate_path
        if Vagrant::Util::Platform.windows? || VagrantVMwareDesktop.wsl?
          sysdrv = ENV.fetch("SYSTEMDRIVE", "C:")
          spath = ["HashiCorp", "vagrant-vmware-desktop", "certificates"]
          path = nil
          # NOTE: This directory is created by the utility service during
          # certificate installation. While the HashiCorp subdirectory is
          # specified as camel cased on creation, it ends up as all lower.
          # This presents a problem within the WSL as the path becomes
          # case sensitive. Both regular and all lower subdirectory
          # paths are checked from the ProgramData directory to prevent
          # future issues where the directory may be properly camel cased.
          [spath, spath.map(&:downcase)].each do |path_parts|
            path = VagrantVMwareDesktop.windows_to_wsl_path(
              File.join(sysdrv, "ProgramData", *path_parts)
            )
            break if File.exist?(path)
          end
          path
        else
          "/opt/vagrant-vmware-desktop/certificates"
        end
      end
    end
  end
end
