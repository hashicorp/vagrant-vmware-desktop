# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

require "json"
require "set"

require "log4r"

require "vagrant/util/scoped_hash_override"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This does NAT port forwarding on the VMware VM.
      class ForwardPorts
        include Common
        include Vagrant::Util::ScopedHashOverride

        def initialize(app, env)
          @app    = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::forward_ports")
        end

        def call(env)
          # Build the definitions for our driver.
          @logger.debug("Building up ports to forward...")
          definitions = []
          env[:machine].config.vm.networks.each do |type, options|
            # Ignore anything but forwarded ports
            next if type != :forwarded_port
            options = scoped_hash_override(options, :vmware)

            # Ignore disabled ports
            next if options[:disabled]

            definitions << {
              device: env[:machine].provider_config.nat_device,
              guest_port: options[:guest],
              host_port: options[:host],
              protocol: options[:protocol],
            }
          end

          # Make sure we're not conflicting with any of the NAT forwarded
          # ports. Note that port collision detection/handling should fix
          # any collisions at a higher level, so this is more of an ASSERT
          # type statement.
          all_ports   = Set.new(env[:machine].provider.driver.all_forwarded_ports)
          all_defined = Set.new(definitions.map { |d| d[:host_port].to_i })
          intersection = all_ports & all_defined
          if !intersection.empty?
            raise Errors::ForwardedPortsCollideWithExistingNAT,
              :ports => intersection.to_a.sort.join(", ")
          end

          # Set the guest IP on all forwarded ports
          guest_ip = nil
          5.times do |_|
            guest_ip = env[:machine].provider.driver.read_ip(
              env[:machine].provider_config.enable_vmrun_ip_lookup
            )
            break if guest_ip
            sleep 2
          end

          if !guest_ip
            raise Errors::ForwardedPortNoGuestIP
          end

          definitions.each do |fp|
            fp[:guest_ip] = guest_ip
          end

          # UI
          env[:ui].info(I18n.t("hashicorp.vagrant_vmware_desktop.forwarding_ports"))
          definitions.each do |fp|
            env[:ui].detail(I18n.t(
              "hashicorp.vagrant_vmware_desktop.forward_port_entry",
              :guest_port => fp[:guest_port],
              :host_port => fp[:host_port]))
          end

          # Forward the ports!
          env[:machine].provider.driver.forward_ports(definitions)

          # Store the forwarded ports for later
          env[:machine].data_dir.join("forwarded_ports").open("w+") do |f|
            ports = {}
            definitions.each do |fp|
              ports[fp[:host_port].to_i] = fp[:guest_port].to_i
            end

            f.write(JSON.dump(ports))
          end

          # Because the network gets restarted when ports are forwarded the
          # guest may see that the network connection has been lost and then
          # regained. NetworkManager will some times see this and drop a
          # current DHCP lease and start the process over again which prevents
          # expected access to the guest. To prevent that, we just wait for a
          # bit until the network is ready.
          port_forward_network_pause = env[:machine].provider_config.port_forward_network_pause.to_i
          if port_forward_network_pause > 0
            env[:ui].info("Pausing for network to stabilize (#{port_forward_network_pause} seconds)")
            sleep(port_forward_network_pause)
          end

          @app.call(env)
        end
      end
    end
  end
end
