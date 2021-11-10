require "fileutils"
require "ipaddr"
require "pathname"

require "log4r"

require "vagrant/util/busy"
require 'vagrant/util/platform'
require 'vagrant/util/retryable'
require "vagrant/util/subprocess"

require "vagrant-vmware-desktop/helper/vagrant_utility"

module HashiCorp
  module VagrantVMwareDesktop
    module Driver
      # This is the base driver for VMware products, which contains
      # some shared common helpers.
      class Base

        # Default NAT device when detection is unavailable
        DEFAULT_NAT_DEVICE = "vmnet8".freeze

        SECTOR_TO_BYTES = 512.freeze

        # Vagrant utility version requirement which must be satisfied to properly
        # work with this version of the plugin. This should be used when new API
        # end points are added to the utility to ensure expected functionality.
        VAGRANT_UTILITY_VERSION_REQUIREMENT = "~> 1.0.14".freeze

        # Enforce VMX ethernet allowlisting. Truthy or falsey values enable/disable. :quiet will
        # enforce the allowlist without printing warning to user
        VMX_ETHERNET_ALLOWLIST_ENFORCE = false

        # allowlisted networking settings that should not be removed from
        # the VMX settings when configuring devices
        VMX_ETHERNET_ALLOWLIST = ["pcislotnumber"].map(&:freeze).freeze

        # Warning template printed to user when a allowlisted setting is
        # detected in the VMX configuration _before_ allowlisting is enabled
        VMX_ETHERNET_ALLOWLIST_DETECTION_PREWARNING = <<-EOW.gsub(/^ {10}/, "").freeze
          The VMX file for this box contains a setting that is automatically overwritten by Vagrant
          when started. Vagrant will stop overwriting this setting in an upcoming release which may
          prevent proper networking setup. Below is the detected VMX setting:

            %VMX_KEY% = "%VMX_VALUE%"

          If networking fails to properly configure, it may require this VMX setting. It can be manually
          applied via the Vagrantfile:

            Vagrant.configure(2) do |config|
              config.vm.provider :vmware_desktop do |vmware|
                vmware.vmx["%VMX_KEY%"] = "%VMX_VALUE%"
              end
            end

          For more information: https://www.vagrantup.com/docs/vmware/boxes.html#vmx-allowlisting
        EOW

        # Customized setting messages to display after white list is enforced to fix
        # boxes that may be broken by allowlisting
        VMX_ETHERNET_ALLOWLIST_POSTFIX = {
          "pcislotnumber" => <<-EOP.gsub(/^ {12}/, "").freeze
            If networking fails to properly configure, it may require adding the following setting to
            the Vagrantfile:

              Vagrant.configure(2) do |config|
                config.vm.provider :vmware_desktop do |vmware|
                  vmware.vmx["%VMX_KEY%"] = "32"
                end
              end

            For more information: https://www.vagrantup.com/docs/vmware/boxes.html#vmx-allowlisting
          EOP
        }

        # Warning template printed to user when a allowlisted setting is
        # detected in the VMX configuration _after_ allowlisting is enabled
        VMX_ETHERNET_ALLOWLIST_DETECTION_WARNING = <<-EOW.gsub(/^ {10}/, "").freeze
          The VMX file for this box contains a setting that is no longer overwritten by Vagrant. This
          may cause networking setup for the VM to fail. Below is the detected VMX setting that is
          no longer overwritten by Vagrant:

            %VMX_KEY% = "%VMX_VALUE%"

          For more information: https://www.vagrantup.com/docs/vmware/boxes.html#vmx-allowlisting
        EOW

        # This is the path to the VM folder. If it is nil then the VM
        # hasn't yet been created.
        #
        # @return [Pathname]
        attr_reader :vm_dir

        # This is the path to the VMX file.
        #
        # @return [Pathname]
        attr_reader :vmx_path

        # Helper utility for interacting with the Vagrant VMware Utility REST API
        #
        # @return [Helper::VagrantUtility]
        attr_reader :vagrant_utility

        # Provider config
        #
        # @return [Vagrant::Config]
        attr_reader :config

        # Vagrant utility version
        #
        # @return [Gem::Version]
        attr_reader :utility_version

        # Include this so we can retry some subprocess stuff
        include Vagrant::Util::Retryable

        def initialize(vmx_path, config)
          @config = config

          @logger   = Log4r::Logger.new("hashicorp::provider::vmware_driver")
          @vmx_path = vmx_path
          @vmx_path = Pathname.new(@vmx_path) if @vmx_path

          if @vmx_path && @vmx_path.directory?
            # The VMX path is a directory, not a VMX. This is probably due to
            # legacy ID of a VM. Early versions of the VMware provider used the
            # directory as the ID rather than the VMX itself.
            @logger.info("VMX path is a directory. Finding VMX file...")

            # Set the VM dir
            @vm_dir = @vmx_path

            # Find the VMX file
            @vmx_path = nil
            @vm_dir.children(true).each do |child|
              if child.basename.to_s =~ /^(.+?)\.vmx$/
                @vmx_path = child
                break
              end
            end

            raise Errors::DriverMissingVMX, :vm_dir => @vm_dir.to_s if !@vmx_path
          end

          # Make sure the vm_dir is properly set always to the directory
          # containing the VMX
          @vm_dir = nil
          @vm_dir = @vmx_path.parent if @vmx_path

          @logger.info("VMX file: #{@vmx_path}")
          @vagrant_utility = Helper::VagrantUtility.new(
            config.utility_host, config.utility_port,
            certificate_path: config.utility_certificate_path
          )

          if config.force_vmware_license
            @logger.warn("overriding VMware license detection with value: #{config.force_vmware_license}")
            @license = config.force_vmware_license
          end

          set_vmware_info
          if config.nat_device.nil?
            if professional?
              detect_nat_device!
            else
              @logger.warn("standard license is in use - forcing default NAT device (#{DEFAULT_NAT_DEVICE})")
              config.nat_device = DEFAULT_NAT_DEVICE
            end
          end
        end

        # @return [Boolean] using standard license
        def standard?
          !professional?
        end

        # @return [Boolean] using professional license
        def professional?
          @pro_license
        end

        # Pull list of vmware devices and detect any NAT types. Use
        # first device discovered
        def detect_nat_device!
          vmnets = read_vmnet_devices
          # First check the default device and if it has NAT enabled, use that
          device = vmnets.detect do |dev|
            dev[:name] == DEFAULT_NAT_DEVICE &&
              dev[:type].to_s.downcase == "nat" &&
              dev[:dhcp] && dev[:hostonly_subnet]
          end
          # If the default device isn't properly configured, now we detect
          if !device
            device = vmnets.detect do |dev|
              @logger.debug("checking device for NAT usage #{dev}")
              dev[:type].to_s.downcase == "nat" &&
                dev[:dhcp] && dev[:hostonly_subnet]
            end
          end
          # If we aren't able to properly detect the device, just use the default
          if device.nil?
            @logger.warn("failed to locate configured NAT device, using default - #{DEFAULT_NAT_DEVICE}")
            config.nat_device = DEFAULT_NAT_DEVICE
            return
          end
          @logger.debug("discovered vmware NAT device: #{device[:name]}")
          config.nat_device = device[:name]
        end

        # Returns an array of all forwarded ports from the VMware NAT
        # configuration files.
        #
        # @return [Array<Integer>]
        def all_forwarded_ports(full_info = false)
          all_fwds = vagrant_utility.get("/vmnet/#{config.nat_device}/portforward")
          if all_fwds.success?
            fwds = all_fwds.get(:content, :port_forwards) || []
            @logger.debug("existing port forwards: #{fwds}")
            full_info ? fwds : fwds.map{|fwd| fwd[:port]}
          else
            raise Errors::DriverAPIPortForwardListError,
              message: all_fwds[:content][:message]
          end
        end

        # Returns port forwards grouped by IP
        #
        # @param [String] ip guest IP address (optional)
        # @return [Hash]
        def forwarded_ports_by_ip(ip = nil)
          all_fwds = vagrant_utility.get("/vmnet/#{config.nat_device}/portforward")
          result = {}
          Array(all_fwds.get(:content, :port_forwards)).each do |fwd|
            g_ip = fwd[:guest][:ip]
            g_port = fwd[:guest][:port]
            h_port = fwd[:port]
            f_proto = fwd[:protocol].downcase.to_sym
            result[g_ip] ||= {:tcp => {}, :udp => {}}
            if f_proto != :tcp && f_proto != :udp
              raise Errors::PortForwardInvalidProtocol,
                guest_ip: g_ip,
                guest_port: g_port,
                host_port: h_port,
                protocol: f_proto
            end
            result[g_ip][f_proto][g_port] = h_port
          end
          if ip
            @logger.debug("port forwards for IP #{ip}: #{result[ip]}")
            result[ip]
          else
            @logger.debug("port forwards by IP: #{result}")
            result
          end
        end

        # Returns host port mapping for the given guest port
        #
        # @param [String] ip guest IP address
        # @param [String, Symbol] proto protocol type (tcp/udp)
        # @param [Integer] guest_port guest port
        # @return [Integer, nil] host port
        def host_port_forward(ip, proto, guest_port)
          proto = proto.to_sym
          if ![:udp, :tcp].include?(proto)
            raise ArgumentError.new("Unsupported protocol provided!")
          end
          fwd_ports = forwarded_ports_by_ip(ip)
          if fwd_ports && fwd_ports[proto]
            fwd_ports[proto][guest_port]
          end
        end

        # This removes all the shared folders from the VM.
        def clear_shared_folders
          @logger.info("Clearing shared folders...")

          vmx_modify do |vmx|
            vmx.keys.each do |key|
              # Delete all specific shared folder configs
              if key =~ /^sharedfolder\d+\./i
                vmx.delete(key)
              end
            end

            # Tell VMware that we have no shared folders
            vmx["sharedfolder.maxnum"] = "0"
          end
        end

        # This clones a VMware VM from one folder to another.
        #
        # @param [Pathname] source_vmx The path to the VMX file of the source.
        # @param [Pathname] destination The path to the directory where the
        #   VM will be placed.
        # @param [Boolean] use linked clone
        # @return [Pathname] The path to the new VMX file.
        def clone(source_vmx, destination, linked=false)
          # If we're prior to Vagrant 1.8, then we don't do any linked
          # clones since we don't have some of the core things in to make
          # this a smoother experience.
          if Gem::Version.new(Vagrant::VERSION) < Gem::Version.new("1.8.0")
            linked = false
          end

          # We can't just check if the user has a standard license since
          # workstation with a standard license (player) will not allow
          # linked cloning, but a standard license on fusion will allow it
          if @license.to_s.downcase == "player"
            @logger.warn("disabling linked clone due to insufficient access based on VMware license")
            linked = false
          end

          if linked
            @logger.info("Cloning machine using VMware linked clones.")
            # The filename of the resulting VMX
            destination_vmx = destination.join(source_vmx.basename)

            begin
              # Do a linked clone!
              vmrun("clone", host_path(source_vmx), host_path(destination_vmx), "linked")
              # Common cleanup
            rescue Errors::VMRunError => e
              # Check if this version of VMware doesn't support linked clones
              # and just super it up.
              stdout = e.extra_data[:stdout] || ""
              if stdout.include?("parameters was invalid")
                @logger.warn("VMware version doesn't support linked clones, falling back")
                destination_vmx = false
              else
                raise
              end
            end
          end

          if !destination_vmx
            @logger.info("Cloning machine using direct copy.")
            # Sanity test
            if !destination.directory?
              raise Errors::CloneFolderNotFolder, path: destination.to_s
            end

            # Just copy over the files within the folder of the source
            @logger.info("Cloning VM to #{destination}")
            source_vmx.parent.children(true).each do |child|
              @logger.debug("Copying: #{child.basename}")
              begin
                FileUtils.cp_r child.to_s, destination.to_s
              rescue Errno::EACCES
                raise Errors::DriverClonePermissionFailure,
                  destination: destination.to_s
              end

              # If we suddenly didn't become a directory, something is
              # really messed up. We should see in the stack trace its
              # from this case.
              if !destination.directory?
                raise Errors::CloneFolderNotFolder, path: destination.to_s
              end
            end

            # Calculate the VMX file of the destination
            destination_vmx = destination.join(source_vmx.basename)
          end

          # Perform the cleanup
          clone_cleanup(destination_vmx)

          # Return the final name
          destination_vmx
        end

        def export(destination_vmx)
          destination_vmx = Pathname.new(destination_vmx)
          @logger.debug("Starting full clone export to: #{destination_vmx}")
          vmrun("clone", host_vmx_path, host_path(destination_vmx), "full")
          clone_cleanup(destination_vmx)
          @logger.debug("Full clone export complete #{vmx_path} -> #{destination_vmx}")
        end

        # This creates a new vmnet device and returns information about that
        # device.
        #
        # @param [Hash] config Configuration for the new vmnet device
        def create_vmnet_device(config)
          result = vagrant_utility.post("/vmnet",
            subnet: config[:subnet_ip],
            mask: config[:netmask]
          )
          if !result.success?
            raise Errors::DriverAPIDeviceCreateError,
              message: result[:content][:message]
          end
          result_device = result.get(:content)
          new_device = {
            name: result_device[:name],
            nummber: result_device[:name].sub('vmnet', '').to_i,
            dhcp: result_device[:dhcp],
            hostonly_netmask: result_device[:mask],
            hostonly_subnet: result_device[:subnet]
          }
        end

        # This deletes the VM.
        def delete
          @logger.info("Deleting VM: #{@vm_dir}")
          begin
            @vm_dir.rmtree
          rescue Errno::ENOTEMPTY
            FileUtils.rm_rf(@vm_dir.to_s)
          end
        end

        # This discards the suspended state of the machine.
        def discard_suspended_state
          Dir.glob("#{@vm_dir}/*.vmss").each do |file|
            @logger.info("Deleting VM state file: #{file}")
            File.delete(file)
          end

          vmx_modify do |vmx|
            # Remove the checkpoint keys
            vmx.delete("checkpoint.vmState")
            vmx.delete("checkpoint.vmState.readOnly")
            vmx.delete("vmotion.checkpointFBSize")
          end
        end

        # This enables shared folders on the VM.
        def enable_shared_folders
          @logger.info("Enabling shared folders...")
          vmrun("enableSharedFolders", host_vmx_path, :retryable => true)
        end

        # This configures a set of forwarded port definitions on the
        # machine.
        #
        # @param [<Array<Hash>] definitions The ports to forward.
        def forward_ports(definitions)
          if Vagrant::Util::Platform.windows?
            vmxpath = @vmx_path.to_s.gsub("/", 92.chr)
          elsif VagrantVMwareDesktop.wsl?
            vmxpath = host_vmx_path
          else
            vmxpath = @vmx_path.to_s
          end
          @logger.debug("requesting ports to be forwarded: #{definitions}")
          # Starting with utility version 1.0.7 we can send all port forward
          # requests at once to be processed. We include backwards compatible
          # support to allow earlier utility versions to remain functional.
          if utility_version > Gem::Version.new("1.0.6")
            @logger.debug("forwarding ports via collection method")
            definitions.group_by{|f| f[:device]}.each_pair do |device, pfwds|
              fwds = pfwds.map do |fwd|
                {
                  :port => fwd[:host_port],
                  :protocol => fwd.fetch(:protocol, "tcp").to_s.downcase,
                  :description => "vagrant: #{vmxpath}",
                  :guest => {
                    :ip => fwd[:guest_ip],
                    :port => fwd[:guest_port]
                  }
                }
              end
              result = vagrant_utility.put("/vmnet/#{device}/portforward", fwds)
              if !result.success?
                raise Errors::DriverAPIPortForwardError,
                  message: result[:content][:message]
              end
            end
          else
            @logger.debug("forwarding ports via individual method")
            definitions.each do |fwd|
              result = vagrant_utility.put("/vmnet/#{fwd[:device]}/portforward",
                :port => fwd[:host_port],
                :protocol => fwd.fetch(:protocol, "tcp").to_s.downcase,
                :description => "vagrant: #{vmxpath}",
                :guest => {
                  :ip => fwd[:guest_ip],
                  :port => fwd[:guest_port]
                }
              )
              if !result.success?
                raise Errors::DriverAPIPortForwardError,
                  message: result.get(:content, :message)
              end
            end
          end
        end

        # This is called to prune the forwarded ports from NAT configurations.
        def prune_forwarded_ports
          @logger.debug("requesting prune of unused port forwards")
          result = vagrant_utility.delete("/portforwards")
          if !result.success?
            raise Errors::DriverAPIPortForwardPruneError,
              message: result.get(:content, :message)
          end
        end

        # This is used to remove all forwarded ports, including those not
        # registered with the plugin
        def scrub_forwarded_ports
          fwds = all_forwarded_ports(true)
          return if fwds.empty?
          if utility_version > Gem::Version.new("1.0.7")
            result = vagrant_utility.delete("/vmnet/#{config.nat_device}/portforward", fwds)
            if !result.success?
              raise Errors::DriverAPIPortForwardPruneError,
                message: result[:content][:message]
            end
          else
            fwds.each do |fwd|
              result = vagrant_utility.delete("/vmnet/#{config.nat_device}/portforward", fwd)
              if !result.success?
                raise Errors::DriverAPIPortForwardPruneError,
                  message: result.get(:content, :message)
              end
            end
          end
        end

        # This returns an IP that can be used to access the machine.
        #
        # @return [String]
        def read_ip(enable_vmrun_ip_lookup=true)
          @logger.info("Reading an accessible IP for machine...")

          # NOTE: Read from DHCP leases first so we can attempt to fetch the address
          # for the vmnet8 device first. If multiple networks are defined on the guest
          # it will return the address of the last device, which will fail when doing
          # port forward lookups

          # Read the VMX data so that we can look up the network interfaces
          # and find a properly accessible IP.
          vmx = read_vmx_data

          0.upto(8).each do |slot|
            # We don't care if this adapter isn't present
            next if vmx["ethernet#{slot}.present"] != "TRUE"

            # Get the type of this adapter. Bail if there is no type.
            type = vmx["ethernet#{slot}.connectiontype"]
            next if !type

            if type != "nat" && type != "custom"
              @logger.warn("Non-NAT interface on slot #{slot}. We can only read IPs of NATs for now.")
              next
            end

            # Get the MAC address
            @logger.debug("Trying to get MAC address for ethernet#{slot}")
            mac = vmx["ethernet#{slot}.address"]
            if !mac || mac == ""
              @logger.debug("No explicitly set MAC, looking or auto-generated one...")
              mac = vmx["ethernet#{slot}.generatedaddress"]

              if !mac
                @logger.warn("Couldn't find MAC, can't determine IP.")
                next
              end
            end

            @logger.debug(" -- MAC: #{mac}")

            # Look up the IP!
            dhcp_ip = read_dhcp_lease(config.nat_device, mac)
            return dhcp_ip if dhcp_ip
          end

          if enable_vmrun_ip_lookup
            # Try to read the IP using vmrun getGuestIPAddress. This
            # won't work if the guest doesn't have guest tools installed or
            # is using an old version of VMware.
            begin
              @logger.info("Trying vmrun getGuestIPAddress...")
              result = vmrun("getGuestIPAddress", host_vmx_path)
              result = result.stdout.chomp

              # If returned address ends with a ".1" do not accept address
              # and allow lookup via VMX.
              # see: https://github.com/vmware/open-vm-tools/issues/93
              if result.end_with?(".1")
                @logger.warn("vmrun getGuestIPAddress returned: #{result}. Result resembles address retrieval from wrong " \
                  "interface. Discarding value and proceeding with VMX based lookup.")
                result = nil
              else
                # Try to parse the IP Address. This will raise an exception
                # if it fails, which will halt our attempt to use it.
                IPAddr.new(result)
                @logger.info("vmrun getGuestIPAddress success: #{result}")
                return result
              end
            rescue Errors::VMRunError
              @logger.info("vmrun getGuestIPAddress failed: VMRunError")
              # Ignore, try the MAC address way.
            rescue IPAddr::InvalidAddressError
              @logger.info("vmrun getGuestIPAddress failed: InvalidAddressError for #{result.inspect}")
              # Ignore, try the MAC address way.
            end
          else
            @logger.info("Skipping vmrun getGuestIPAddress as requested by config.")
          end
          nil
        end

        # This reads all the network adapters that are on the machine and
        # enabled.
        #
        # @return [Array<Hash>]
        def read_network_adapters
          vmx = read_vmx_data

          adapters = []
          vmx.keys.each do |key|
            # We only care about finding present ethernet adapters
            match = /^ethernet(\d+)\.present$/i.match(key)
            next if !match
            next if vmx[key] != "TRUE"

            # We found one, so store it away
            slot    = match[1]
            adapter = {
              :slot => slot,
              :type => vmx["ethernet#{slot}.connectiontype"]
            }

            adapters << adapter
          end

          return adapters
        end

        # This returns an array of paths to all the running VMX files.
        #
        # @return [Array<String>]
        def read_running_vms
          vmrun("list").stdout.split("\n").find_all do |path|
            path !~ /running VMs:/
          end
        end

        # This reads the state of this VM and returns a symbol representing
        # it.
        #
        # @return [Symbol]
        def read_state
          # The VM is not created if we don't have a directory
          return :not_created if !@vm_dir

          # Check to see if the VM is running, which requires shelling out
          vmx_path = nil
          begin
            vmx_path = @vmx_path.realpath.to_s
          rescue Errno::ENOENT
            @logger.info("VMX path doesn't exist, not created: #{@vmx_path}")
            return :not_created
          end

          vmx_path = host_vmx_path

          # Downcase the lines in certain case-insensitive cases
          downcase = VagrantVMwareDesktop.case_insensitive_fs?(vmx_path)

          # OS X is case insensitive so just lowercase everything
          vmx_path = vmx_path.downcase if downcase

          if Vagrant::Util::Platform.windows?
            # Replace any slashes to be unix-style.
            vmx_path.gsub!(92.chr, "/")
          end

          vmrun("list").stdout.split("\n").each do |line|
            if Vagrant::Util::Platform.windows?
              # On Windows, we normalize the paths to unix-style.
              line.gsub!("\\", "/")

              # We also change any drive letter to be lowercased
              line[0] = line[0].downcase
            end

            # Case insensitive filesystems, so we downcase.
            line = line.downcase if downcase
            return :running if line == vmx_path
          end

          # Check to see if the VM is suspended based on whether a file
          # exists in the VM directory
          return :suspended if Dir.glob("#{@vm_dir}/*.vmss").length >= 1

          # I guess it is not running.
          return :not_running
        end

        # This reads all the network adapters and returns
        # their assigned MAC addresses
        def read_mac_addresses
          vmx = read_vmx_data
          macs = {}
          vmx.keys.each do |key|
            # We only care about finding present ethernet adapters
            match = /^ethernet(\d+)\.present$/i.match(key)
            next if !match
            next if vmx[key] != "TRUE"

            slot = match[1].to_i

            # Vagrant's calling code assumes the MAC is all caps no colons
            mac = vmx["ethernet#{slot}.generatedaddress"].to_s
            mac = mac.upcase.gsub(/:/, "")

            # Vagrant's calling code assumes these will be 1-indexed
            slot += 1

            macs[slot] = mac
          end
          macs
        end

        # Reserve an address on the DHCP sever for the given
        # MAC address. Defaults to the NAT device at vmnet8
        #
        # @param [String] ip IP address to reserve
        # @param [String] mac MAC address to associate
        # @return [true]
        def reserve_dhcp_address(ip, mac, vmnet="vmnet8")
          result = vagrant_utility.put("/vmnet/#{vmnet}/dhcpreserve/#{mac}/#{ip}")
          if !result.success?
            raise Errors::DriverAPIAddressReservationError,
              device: vmnet,
              address: ip,
              mac: mac,
              message: result[:content][:message]
          end
          true
        end

        # This reads the vmnet devices and various information about them.
        #
        # @return [Array<Hash>]
        def read_vmnet_devices
          result = vagrant_utility.get("/vmnet")
          if !result.success?
            raise Errors::DriverAPIDeviceListError,
              message: result[:content][:message]
          end
          Array(result.get(:content, :vmnets)).map do |vmnet|
            {
              name: vmnet[:name],
              type: vmnet[:type],
              number: vmnet[:name].sub('vmnet', '').to_i,
              dhcp: vmnet[:dhcp],
              hostonly_netmask: vmnet[:mask],
              hostonly_subnet: vmnet[:subnet],
              virtual_adapter: "yes"
            }
          end
        end

        # This modifies the metadata of the virtual machine to add the
        # given network adapters.
        #
        # @param [Array] adapters
        def setup_adapters(adapters, allowlist_verified=false)
          vmx_modify do |vmx|
            # Remove all previous adapters
            vmx.keys.each do |key|
              check_key = key.downcase
              ethernet_key = key.match(/^ethernet\d\.(?<setting_name>.+)$/)
              if !ethernet_key.nil? && !VMX_ETHERNET_ALLOWLIST.include?(ethernet_key["setting_name"])
                @logger.warn("setup_adapter: Removing VMX key: #{ethernet_key}")
                vmx.delete(key)
              elsif ethernet_key
                if !allowlist_verified
                  display_ethernet_allowlist_warning(key, vmx[key])
                elsif allowlist_verified  == :disable_warning
                  @logger.warn("VMX allowlisting warning message has been disabled via configuration. `#{key}`")
                else
                  @logger.info("VMX allowlisting has been verified via configuration. `#{key}`")
                end
                if !VMX_ETHERNET_ALLOWLIST_ENFORCE && allowlist_verified != true
                  @logger.warn("setup_adapter: Removing allowlisted VMX key: #{ethernet_key}")
                  vmx.delete(key)
                end
              end
            end

            # Go through the adapters to enable and set them up properly
            adapters.each do |adapter|
              key = "ethernet#{adapter[:slot]}"

              vmx["#{key}.present"] = "TRUE"
              vmx["#{key}.connectiontype"] = adapter[:type].to_s
              vmx["#{key}.virtualdev"] = "e1000"

              if adapter[:mac_address]
                vmx["#{key}.addresstype"] = "static"
                vmx["#{key}.address"] = adapter[:mac_address]
              else
                # Dynamically generated MAC address
                vmx["#{key}.addresstype"] = "generated"
              end

              if adapter[:vnet]
                # Some adapters define custom vmnet devices to connect to
                vmx["#{key}.vnet"] = adapter[:vnet]
                vmx["#{key}.connectiontype"] = "custom" if adapter[:type] == :nat
              end
            end
          end
        end

        # This creates a shared folder within the VM.
        def share_folder(id, hostpath)
          @logger.info("Adding a shared folder '#{id}': #{hostpath}")

          vmrun("addSharedFolder", host_vmx_path, id, host_path(hostpath))
          vmrun("setSharedFolderState", host_vmx_path, id, host_path(hostpath), "writable")
        end

        # All the methods below deal with snapshots: taking them, restoring
        # them, deleting them, etc.
        def snapshot_take(name)
          vmrun("snapshot", host_vmx_path, name)
        end

        def snapshot_delete(name)
          vmrun("deleteSnapshot", host_vmx_path, name)
        end

        def snapshot_revert(name)
          vmrun("revertToSnapshot", host_vmx_path, name)
        end

        def snapshot_list
          snapshots = []
          vmrun("listSnapshots", host_vmx_path).stdout.split("\n").each do |line|
            if !line.include?("Total snapshot")
              snapshots << line
            end
          end

          snapshots
        end

        def snapshot_tree
          snapshots = []
          snap_level = 0
          vmrun("listSnapshots", host_vmx_path, "showTree").stdout.split("\n").each do |line|
            if !line.include?("Total snapshot")
              # if the line starts with white space then it is a child
              # of the previous line
              if line.start_with?(/\s/)
                current_level = line.count("\t")
                if current_level > snap_level
                  name = "#{snapshots.last}/#{line.gsub(/\s/, "")}"
                elsif current_level == snap_level
                  path = snapshots.last.split("/")
                  path.pop
                  path << line.gsub(/\s/, "")
                  name = path.join("/")
                else
                  path = snapshots.last.split("/")
                  diff = snap_level - current_level
                  (0..diff).to_a.each { |i| path.pop }
                  path << line.gsub(/\s/, "")
                  name = path.join("/")
                end
                snap_level = current_level
                snapshots << name
              else
                snapshots << line
              end
            end
          end

          snapshots
        end

        # This will start the VMware machine.
        def start(gui=false)
          gui_arg = gui ? "gui" : "nogui"
          vmrun("start", host_vmx_path, gui_arg, retryable: true, timeout: 45)
        rescue Vagrant::Util::Subprocess::TimeoutExceeded
          # Sometimes vmrun just hangs. We give it a generous timeout
          # of 45 seconds, and then throw this.
          raise Errors::StartTimeout
        end

        # This will stop the VMware machine.
        def stop(stop_mode="soft")
          begin
            vmrun("stop", host_vmx_path, stop_mode, :retryable => true, timeout: 15)
          rescue Errors::VMRunError, Vagrant::Util::Subprocess::TimeoutExceeded
            begin
              vmrun("stop", host_vmx_path, "hard", :retryable => true)
            rescue Errors::VMRunError
              # There is a chance that the "soft" stop may have timed out, yet
              # still succeeded which would result in the "hard" stop failing
              # due to the guest not running. Because of this we do a quick
              # state check and only allow the error if the guest is still
              # running
              raise if read_state == :running
            end
          end
        end

        # This will suspend the VMware machine.
        def suspend
          vmrun("suspend", host_vmx_path, :retryable => true)
        end

        # This is called to do any message suppression if we need to.
        def suppress_messages
          if PRODUCT_NAME == "fusion"
            contents = <<-DATA
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>disallowUpgrade</key>
	<true/>
</dict>
</plist>
          DATA

            if @vmx_path
              filename = @vmx_path.basename.to_s.gsub(".vmx", ".plist")
              @vmx_path.dirname.join(filename).open("w+") do |f|
                f.puts(contents.strip)
              end
            end
          end
        end

        # This method is called to verify that the installation looks good,
        # and raises an exception if it doesn't.
        def verify!
          utility_requirement = Gem::Requirement.new(VAGRANT_UTILITY_VERSION_REQUIREMENT.split(","))
          if !utility_requirement.satisfied_by?(utility_version)
            raise Errors::UtilityUpgradeRequired,
              utility_version: utility_version.to_s,
              utility_requirement: utility_requirement.to_s
          end
        end

        # This method is called to verify whether the vmnet devices are healthy.
        # If not, an exception should be raised.
        def verify_vmnet!
          result = vagrant_utility.post("/vmnet/verify")
          if !result.success?
            # Check if the result was a 404. This indicates a utility service
            # running before the vmnet verification endpoint was introduced
            # and is not really an error
            if result[:code] == 404
              return
            end
            raise Errors::VMNetDevicesWontStart
          end
        end

        # This method returns whether or not the VMX process is still alive.
        def vmx_alive?
          # If we haven't cleanly shut down, then it is alive.
          read_vmx_data["cleanshutdown"] != "TRUE"

          # Note at some point it would be nice to actually track the VMX
          # process itself. But at this point this isn't very feasible.
        end

        # This allows modifications of the VMX file by handling it as
        # a simple hash. By adding/modifying/deleting keys in the hash,
        # the VMX is modified.
        def vmx_modify
          # Read the data once
          vmx_data = read_vmx_data

          # Yield it so that it can be modified
          yield vmx_data

          # And write it back!
          @logger.info("Modifying VMX data...")
          @vmx_path.open("w") do |f|
            vmx_data.keys.sort.each do |key|
              value = vmx_data[key]

              # We skip nil values and remove them from the output
              if value.nil?
                @logger.debug("  - DELETE #{key}")
                next
              end

              # Write the value
              @logger.debug("  - SET #{key} = \"#{value}\"")
              f.puts("#{key} = \"#{value}\"")
            end

            f.fsync
          end
        end

        # Gets the version of the vagrant-vmware-utility currently
        # in use
        #
        # @return [String, nil]
        def vmware_utility_version
          @logger.debug("Getting version from vagrant-vmware-utility")
          result = vagrant_utility.get("/version")
          if result.success?
            result.get(:content, :version)
          end
        end

        # Gets the currently attached disks
        #
        # @param [List<String>] List of disk types to search for
        # @return [Hash] Hash of disks
        def get_disks(types)
          vmx = read_vmx_data
          disks = {}
          vmx.each do |k, v|
            if k.match(/(#{types.map{|t| Regexp.escape(t)}.join("|")})\d+:\d+/)
              key, attribute = k.split(".", 2)
              if disks[key].nil?
                disks[key] = {attribute => v}
              else
                disks[key].merge!({attribute => v})
              end
            end
          end
          disks
        end

        # Create a vmdk disk
        #
        # @params [String] Disk filename
        # @params [Integer] Size of disk in bytes
        # @params [Integer] Disk type (given by vmware-vdiskmanager)
        # @params [String] Disk adapter
        def create_disk(disk_filename, disk_size, disk_type, disk_adapter)
          disk_path = File.join(File.dirname(@vmx_path), disk_filename)
          disk_size = "#{Vagrant::Util::Numeric.bytes_to_megabytes(disk_size).to_s}MB"
          vdiskmanager("-c", "-s", disk_size, "-t", disk_type.to_s, "-a", disk_adapter, disk_path)
          disk_path
        end

        # Make a disk larger
        #
        # @params [String] Path to disk
        # @params [Integer] Size of disk in bytes
        def grow_disk(disk_path, disk_size)
          disk_size = "#{Vagrant::Util::Numeric.bytes_to_megabytes(disk_size).to_s}MB"
          vdiskmanager("-x", disk_size, disk_path)
        end

        # Adds a disk to the vm's vmx file
        #
        # @params [String] filename to insert
        # @params [String] slot to add the disk to
        # @params [Map] (deafults to {}) map of extra options to specify in the vmx file of the form {opt => value}
        def add_disk_to_vmx(filename, slot, extra_opts={})
          root_key, _ = slot.split(":", 2)
          vmx_modify do |vmx|
            vmx["#{root_key}.present"] = "TRUE"
            vmx["#{slot}.filename"] = filename
            vmx["#{slot}.present"] = "TRUE"
            extra_opts.each do |key, value|
              vmx["#{slot}.#{key}"] = value
            end
          end
        end

        # Removes a disk to the vm's vmx file
        #
        # @params [String] filename to remove
        # @params [List] (defaults to []) list of extra options remove from the vmx file
        def remove_disk_from_vmx(filename, extra_opts=[])
          vmx = read_vmx_data
          vmx_disk_entry = vmx.select { |_, v| v == filename }
          if !vmx_disk_entry.empty?
            slot = vmx_disk_entry.keys.first.split(".").first
            vmx_modify do |vmx|
              vmx.delete("#{slot}.filename")
              vmx.delete("#{slot}.present")
              extra_opts.each do |opt|
                vmx.delete("#{slot}.#{opt}")
              end
            end
          end
        end

        # @params [String] disk base filename eg. disk.vmx
        def remove_disk(disk_filename)
          disk_path = File.join(File.dirname(@vmx_path), disk_filename)
          vdiskmanager("-U", disk_path) if File.exist?(disk_path)
          remove_disk_from_vmx(disk_filename)
        end

        # Gets the size of a .vmdk disk
        # Spec for vmdk: https://github.com/libyal/libvmdk/blob/master/documentation/VMWare%20Virtual%20Disk%20Format%20(VMDK).asciidoc
        #
        # @params [String] full path to .vmdk file
        # @return [int, nil] size of disk in bytes, nil if path does not exist
        def get_disk_size(disk_path)
          return nil if !File.exist?(disk_path)

          disk_size = 0
          at_extent_description = false
          File.foreach(disk_path) do |line|
            next if !line.valid_encoding?
            # Look for `# Extent description` header
            if line.include?("# Extent description")
              at_extent_description = true
              next
            end
            if at_extent_description
              # Exit once the "Extent description" section is done
              # signified by the new line
              if line == "\n"
                break
              else
                # Get the 2nd entry on each line - number of sectors
                sectors = line.split(" ")[1].to_i
                # Convert sectors to bytes
                disk_size += (sectors * SECTOR_TO_BYTES)
              end
            end
          end
          disk_size
        end

        protected

        # This reads the latest DHCP lease for a MAC address on the
        # given vmnet device.
        #
        # @param [String] vmnet The name of the vmnet device like "vmnet8"
        # @param [String] mac The MAC address
        # @return [String] The IP, or nil.
        def read_dhcp_lease(vmnet, mac)
          @logger.info("Reading DHCP lease for '#{mac}' on '#{vmnet}'")
          result = vagrant_utility.get("/vmnet/#{vmnet}/dhcplease/#{mac}")
          if result.success?
            result.get(:content, :ip)
          end
        end

        # This returns the VMX data just as a hash.
        #
        # @return [Hash]
        def read_vmx_data
          @logger.info("Reading VMX data...")

          # Read the VMX contents into memory
          contents = @vmx_path.read

          # Convert newlines to unix-style
          contents.gsub!(/\r\n?/, "\n")

          # Parse it out into a hash
          vmx_data = {}
          contents.split("\n").each do |line|
            # If it is a comment then ignore it
            next if line =~ /^#/

            # Parse out the key/value
            match = /^(.+?)\s*=\s*(.*?)\s*$/.match(line)
            if !match
              @logger.warn("Weird value in VMX: '#{line}'")
              next
            end

            # Set the data
            key   = match[1].strip.downcase
            value = match[2]
            value = value[1, value.length-2] if value =~ /^".*?"$/
            @logger.debug("  - #{key} = #{value}")
            vmx_data[key] = value
          end

          # Return it
          vmx_data
        end

        # Executes a given executable with retries
        def vmexec(executable, *command)
          # Get the options hash if it exists
          opts = {}
          opts = command.pop if command.last.is_a?(Hash)

          tries = 0
          tries = 3 if opts[:retryable]

          interrupted = false
          sigint_callback = lambda do
            interrupted = true
          end

          command_opts = { :notify => [:stdout, :stderr] }
          command_opts[:timeout] = opts[:timeout] if opts[:timeout]

          command = command.dup
          command << command_opts


          Vagrant::Util::Busy.busy(sigint_callback) do
            retryable(:on => Errors::VMExecError, :tries => tries, :sleep => 2) do
              r_path = executable.to_s
              if VagrantVMwareDesktop.wsl?
                r_path = VagrantVMwareDesktop.windows_to_wsl_path(r_path)
              end
              result = Vagrant::Util::Subprocess.execute(r_path, *command)
              if result.exit_code != 0
                raise Errors::VMExecError,
                  :executable => executable.to_s,
                  :command => command.inspect,
                  :stdout => result.stdout.chomp,
                  :stderr => result.stderr.chomp
              end

              # Make sure we only have unix-style line endings
              result.stderr.gsub!(/\r\n?/, "\n")
              result.stdout.gsub!(/\r\n?/, "\n")

              return result
            end
          end
        end

        # This executes the "vmrun" command with the given arguments.
        def vmrun(*command)
          # Get the VMware product family
          host_type = "player" # default, plugin is not support "vmplayer".
          case PRODUCT_NAME.to_s
          when "workstation" then
            host_type = "ws"
          when "fusion"
            host_type = "fusion"
          end
           
          # Execute the "vmrun" with host_type parameters
          begin
            vmexec(@vmrun_path, *command)
          rescue Errors::VMExecError => e
            raise Errors::VMRunError,
              :command => e.extra_data[:command],
              :stdout => e.extra_data[:stdout],
              :stderr => e.extra_data[:stderr]
          end
        end

        # This executes the "vmware-vdiskmanager" command with the given arguments.
        def vdiskmanager(*command)
          vmexec(@vmware_vdiskmanager_path, *command)
        end

        # Set VMware information
        def set_vmware_info
          result = vagrant_utility.get("/vmware/info")
          if !result.success?
            raise Errors::DriverAPIVMwareVersionDetectionError,
              message: result[:content][:message]
          end
          if @license.to_s.empty?
            @license = result.get(:content, :license).to_s.downcase
          end
          if (@license.include?("workstation") || @license.include?("pro")) && !@license.include?("vl")
            @pro_license = true
          else
            @pro_license = false
          end
          @version = result.get(:content, :version)
          @product_name = result.get(:content, :product)
          result = vagrant_utility.get("/vmware/paths")
          if !result.success?
            raise Errors::DriverAPIVmwarePathsDetectionError,
              message: result[:content][:message]
          end
          @vmrun_path = result.get(:content, :vmrun)
          @vmware_vmx_path = result.get(:content, :vmx)
          @vmware_vdiskmanager_path = result.get(:content, :vdiskmanager)
          result = vagrant_utility.get("/version")
          @utility_version = Gem::Version.new(result[:content][:version])
          @logger.debug("vagrant vmware utility version detected: #{@utility_version}")
          @logger.debug("vmware product detected: #{@product_name}")
          @logger.debug("vmware license in use: #{@license}")
          if !@pro_license
            @logger.warn("standard VMware license currently in use which may result in degraded networking experience")
          end
          if VagrantVMwareDesktop.wsl?
            @logger.debug("Detected WSL environment, converting paths...")
            rpath = @vmrun_path
            xpath = @vmware_vmx_path
            dpath = @vmware_vdiskmanager_path
            @vmrun_path = VagrantVMwareDesktop.windows_to_wsl_path(@vmrun_path)
            @vmware_vmx_path = VagrantVMwareDesktop.windows_to_wsl_path(@vmware_vmx_path)
            @vmware_vdiskmanager_path = VagrantVMwareDesktop.windows_to_wsl_path(@vmware_vdiskmanager_path)
            @logger.debug("Converted `#{rpath}` -> #{@vmrun_path}")
            @logger.debug("Converted `#{xpath}` -> #{@vmware_vmx_path}")
            @logger.debug("Converted `#{dpath}` -> #{@vmware_vdiskmanager_path}")
          end
        end


        # This performs common cleanup tasks on a cloned machine.
        def clone_cleanup(destination_vmx)
          destination = destination_vmx.parent

          # Delete any lock files
          destination.children(true).each do |child|
            if child.extname == ".lck"
              @logger.debug("Deleting lock file: #{child}")
              child.rmtree
            end
          end

          # Next we make some VMX modifications
          self.class.new(destination_vmx, config).vmx_modify do |vmx|
            # This forces VMware to generate a new UUID which avoids the
            # "It appears you have moved this VM" error.
            vmx["uuid.action"] = "create"

            # Ask VMware to auto-answer any dialogs since we'll be running
            # headless, in general.
            vmx["msg.autoanswer"] = "true"
          end

          # Return the destination VMX file
          destination_vmx
        end

        # Display warning message about allowlisted VMX ethernet settings
        def display_ethernet_allowlist_warning(vmx_key, vmx_val)
          if VMX_ETHERNET_ALLOWLIST_ENFORCE != :quiet
            if create_notification_file(vmx_key)
              if VMX_ETHERNET_ALLOWLIST_ENFORCE
                warning_msg = VMX_ETHERNET_ALLOWLIST_DETECTION_WARNING
                setting_name = vmx_key.slice(vmx_key.rindex(".") + 1, vmx_key.size).downcase
                if VMX_ETHERNET_ALLOWLIST_POSTFIX[setting_name]
                  warning_msg += "\n"
                  warning_msg += VMX_ETHERNET_ALLOWLIST_POSTFIX[setting_name]
                end
              else
                warning_msg = VMX_ETHERNET_ALLOWLIST_DETECTION_PREWARNING
              end
              warning_msg = warning_msg.gsub("%VMX_KEY%", vmx_key).gsub("%VMX_VALUE%", vmx_val)
              warning_msg.split("\n").each do |line|
                $stderr.puts "WARNING: #{line}"
              end
            end
          end
        end

        # Creates a file within the vm directory to flag if the warning has
        # already been provided to the user. This helps prevent warnings from being
        # re-displayed after the initial `up`.
        def create_notification_file(key)
          path = File.join(vm_dir, "vagrant-vmx-warn-#{key}.flg")
          if !File.exist?(path)
            FileUtils.touch(path)
            true
          else
            false
          end
        end

        # This converts the VMX file path to the true
        # host path. At this point it only applies
        # modifications if running within the WSL on
        # Windows. For all other cases, it just forces
        # a String type.
        #
        # @return [String] path to VMX file
        def host_vmx_path
          host_path(@vmx_path)
        end

        # This converts the file path to the true
        # host path. At this point it only applies
        # modifications if running within the WSL on
        # Windows. For all other cases, it just forces
        # a String type.
        #
        # @param [String, Pathname] path
        # @return [String] path to VMX file
        def host_path(path)
          VagrantVMwareDesktop.wsl_to_windows_path(path)
        end
      end
    end
  end
end
