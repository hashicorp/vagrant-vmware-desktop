require "set"

require "log4r"

require "vagrant/util/network_ip"
require "vagrant/util/scoped_hash_override"

require "vagrant-vmware-desktop/helper/lock"
require "vagrant-vmware-desktop/helper/routing_table"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This action sets up all the network adapters for the machine and
      # also tells the guest to configure the networks.
      class Network
        include Common

        include Vagrant::Util::NetworkIP
        include Vagrant::Util::ScopedHashOverride

        DEFAULT_VMNET_NAT = "vmnet8"

        def initialize(app, env)
          @app    = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::network")
        end

        def call(env)
          # Set this to an ivar so that helper methods have access to it
          @env = env

          # Get the list of network adapters from the configuration
          network_adapters_config = env[:machine].provider_config.network_adapters.dup

          # Assign the adapter slot for each high-level network
          available_slots = Set.new(1..8)
          network_adapters_config.each do |slot, _data|
            available_slots.delete(slot)
          end

          @logger.debug("Available slots for high-level adapters: #{available_slots.inspect}")
          @logger.info("Determining network adapters required for high-level configuration...")
          available_slots = available_slots.to_a.sort
          env[:machine].config.vm.networks.each do |type, options|
            # We only handle private and public networks
            next if type != :private_network && type != :public_network

            scope_key = "vmware_#{PRODUCT_NAME}".to_sym
            options   = scoped_hash_override(options, scope_key)

            # Figure out the slot that this adapter will go into
            slot = options[:adapter]
            if !slot
              if available_slots.empty?
                raise Errors::NetworkingNoSlotsForHighLevel
              end

              slot = available_slots.shift
            end

            # Configure it
            data = nil
            if type == :private_network
              # private_network = hostonly
              data        = [:hostonly, options]
            elsif type == :public_network
              # public_network = bridged
              data        = [:bridged, options]
            end

            # Store it!
            @logger.info(" -- Slot #{slot}: #{data[0]}")
            network_adapters_config[slot] = data
          end

          @logger.info("Determining adapters and compiling network configuration...")
          adapters = []
          networks = []
          network_adapters_config.each do |slot, data|
            type    = data[0]
            options = data[1]

            if slot == 0 && env[:machine].provider_config.nat_device != DEFAULT_VMNET_NAT
              # TODO: what's the device name on windows?
              options[:device] = "/dev/#{env[:machine].provider_config.nat_device}"
            end

            @logger.info("Slot #{slot}. Type: #{type}")

            # Get normalized configuration so we can add/scrub values
            config = send("#{type}_config", options)
            @logger.debug("Normalized configuration: #{config.inspect}")

            # Get the adapter configuration for the driver
            adapter = send("#{type}_adapter", config)
            adapter[:slot] = slot
            adapters << adapter
            @logger.debug("Adapter configuration: #{adapter.inspect}")

            # Get the network configuration for the guest
            network = send("#{type}_network_config", config)
            network[:auto_config] = config[:auto_config]
            networks << network
            @logger.debug("Network configuration: #{network.inspect}")
          end

          if !adapters.empty?
            # Modify the VM metadata to add adapters
            @logger.info("Enabling #{adapters.length} adapters...")
            Helper::Lock.lock(env[:machine], "vmware-network") do
              env[:ui].info(I18n.t("hashicorp.vagrant_vmware_desktop.enabling_adapters"))
              env[:machine].provider.driver.setup_adapters(adapters, env[:machine].provider_config.allowlist_verified)
            end
          end

          @app.call(env)

          if !networks.empty?
            # Assign interface numbers to the networks
            assign_interface_numbers(networks, adapters)

            networks_to_configure = networks.select { |n| n[:auto_config] }
            env[:ui].info(I18n.t("hashicorp.vagrant_vmware_desktop.configuring_networks"))
            env[:machine].guest.capability(:configure_networks, networks_to_configure)
          end
        end

        def nat_config(options)
          return {
            :auto_config => true,
            :type        => :dhcp
          }.merge(options)
        end

        def nat_adapter(config)
          {
            type: :nat,
            mac_address: config[:mac_address],
            vnet: config[:device]
          }.compact
        end

        def nat_network_config(config)
          return {
            :type => :dhcp
          }
        end

        def hostonly_config(options)
          # Get the default configuration built up
          config = {
            :auto_config => true,
            :netmask     => "255.255.255.0",
            :type        => :dhcp
          }.merge(options || {})

          if options[:ip]
            # Check if we are using ipv6, which is not supported
            ip = IPAddr.new(options[:ip])
            if ip.ipv6?
              raise Errors::VMNetNoIPV6
            end

            # We are using static if we have an IP set
            config[:type] = :static

            # Get the static IP and use the static IP + subnet mask to
            # determine the subnet IP.
            static_ip = config[:ip]
            subnet_ip = network_address(static_ip, config[:netmask])
            config[:subnet_ip] = subnet_ip

            # Calculate the actual IP of the adapter itself, which is usually
            # just the network address "+ 1" in the last octet
            ip_parts = subnet_ip.split(".").map { |i| i.to_i }
            adapter_ip    = ip_parts.dup
            adapter_ip[3] += 1
            config[:adapter_ip] ||= adapter_ip.join(".")
          end

          # Make sure the type is a symbol
          config[:type] = config[:type].to_sym

          return config
        end

        def hostonly_adapter(config)
          # If we're just doing normal DHCP, then we just connect to the
          # basic default adapter.
          if config[:type] == :dhcp
            return {
              :type => :hostonly
            }
          end

          # Otherwise we want a static IP. Start by trying to find
          # an existing network that matches our needs.
          vmnet = nil
          @env[:machine].provider.driver.read_vmnet_devices.each do |device|
            if device[:hostonly_subnet] == config[:subnet_ip]
              @logger.info("Found matching vmnet device: #{device[:name]}")
              vmnet = device
              break
            end
          end

          # Check for collisions by checking for if there is another device
          # that the IP would route to. The basic logic is: if there is
          # a device, and it is NOT the vmnet we care about, then it
          # is an error.
          @logger.info("Checking for hostonly network collisions...")
          device = routing_table.device_for_route(config[:ip])
          if device
            if !vmnet || device != vmnet[:name]
              # There is a collision with some other networking device.
              raise Errors::NetworkingHostOnlyCollision,
                :device => device,
                :ip     => config[:ip]
            end
          end

          if !vmnet
            @logger.info("No collisions detected, creating new vmnet device.")
            vmnet = @env[:machine].provider.driver.create_vmnet_device(
              :netmask => config[:netmask],
              :subnet_ip => config[:subnet_ip])
          end

          # Determine MAC address of the adapter
          mac_address = config[:mac]
          mac_address = vmware_mac_format(mac_address) if mac_address

          # Return a more complex configuration to describe what we need
          return {
            :type        => :custom,
            :mac_address => mac_address,
            :vnet        => vmnet[:name]
          }
        end

        def hostonly_network_config(config)
          return {
            :type       => config[:type],
            :adapter_ip => config[:adapter_ip],
            :ip         => config[:ip],
            :netmask    => config[:netmask]
          }
        end

        def bridged_config(options)
          return {
            :auto_config => true,
            :mac         => nil,
            :type        => :dhcp
          }.merge(options || {})
        end

        def bridged_adapter(config)
          mac_address = config[:mac]
          mac_address = vmware_mac_format(mac_address) if mac_address

          return {
            :type        => :bridged,
            :mac_address => mac_address
          }
        end

        def bridged_network_config(config)
          if config[:ip]
            options = {
              auto_config: true,
              mac:         nil,
              netmask:     "255.255.255.0",
            }.merge(config)
            options[:type] = :static
            return options
          end

          return {
            :type => :dhcp
          }
        end

        #-----------------------------------------------------------------
        # Misc. helpers
        #-----------------------------------------------------------------
        # Assigns the actual interface number of a network based on the
        # enabled NICs on the virtual machine.
        #
        # This interface number is used by the guest to configure the
        # NIC on the guest VM.
        #
        # The networks are modified in place by adding an ":interface"
        # field to each.
        def assign_interface_numbers(networks, adapters)
          # First create a mapping of adapter slot to interface number
          # by reading over the existing network adapters.
          slots_in_use = []
          vm_adapters = @env[:machine].provider.driver.read_network_adapters
          vm_adapters.each do |adapter|
            slots_in_use << adapter[:slot].to_i
          end

          slot_to_interface = {}
          slots_in_use.sort.each_index do |i|
            slot_to_interface[slots_in_use[i]] = i
          end

          # Make a pass through the adapters to assign the :interface
          # key to each network configuration.
          adapters.each_index do |i|
            adapter = adapters[i]
            network = networks[i]

            # Figure out the interface number by simple lookup
            network[:interface] = slot_to_interface[adapter[:slot]]
          end
        end

        # This converts the Vagrant configured MAC address format to
        # a typical MAC address format.
        #
        # @param [String] mac
        # @return [String]
        def vmware_mac_format(mac)
          mac.scan(/.{2}/).join(":")
        end

        # This is a lazy loaded {Helper::RoutingTable}.
        #
        # @return [Helper::RoutingTable]
        def routing_table
          @routing_table ||= Helper::RoutingTable.new
        end
      end
    end
  end
end
