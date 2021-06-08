require "json"

module HashiCorp
  module VagrantVMwareDesktop
    module Cap
      module Provider

        def self.forwarded_ports(machine)
          path = machine.data_dir.join("forwarded_ports")
          return JSON.parse(path.read) if path.file?
          {}
        end

        def self.public_address(machine)
          guest_ip = nil
          5.times do |_|
            guest_ip = machine.provider.driver.read_ip(
              machine.provider_config.enable_vmrun_ip_lookup
            )
            break if guest_ip
            sleep 2
          end

          guest_ip
        end

        def self.nic_mac_addresses(machine)
          machine.provider.driver.read_mac_addresses
        end

        def self.scrub_forwarded_ports(machine)
          machine.provider.driver.scrub_forwarded_ports
        end
      end
    end
  end
end
