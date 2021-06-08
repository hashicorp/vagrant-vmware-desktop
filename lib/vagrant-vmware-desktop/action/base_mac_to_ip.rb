require "log4r"
require "ipaddr"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This action maps an IP address to a MAC address in the DHCP
      # server for the default NAT interface
      class BaseMacToIp
        include Common

        def initialize(app, env)
          @app    = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::basemactoip")
        end

        def call(env)
          if env[:machine].provider_config.base_address
            ip = env[:machine].provider_config.base_address
            mac = env[:machine].provider_config.base_mac

            validate_address!(env[:machine].provider.driver,
              ip, env[:machine].provider_config.nat_device)

            env[:machine].provider.driver.reserve_dhcp_address(ip, mac, env[:machine].provider_config.nat_device)

            env[:ui].info(I18n.t("hashicorp.vagrant_vmware_desktop.mac_to_ip_mapping",
              address: ip, mac: mac))
          end
          @app.call(env)
        end

        def validate_address!(driver, address, nat_device)
          devices = driver.read_vmnet_devices
          vmnet = devices.detect { |v| v[:name] == nat_device }
          if !vmnet
            raise Errors::MissingNATDevice
          end
          # NOTE: VMware dhcpd is configured as follows for addressing:
          # .1          -> host machine
          # .2 - .127   -> static address (valid here)
          # .128 - .253 -> DHCP assigned
          # .254        -> DHCP server
          dev_subnet = "#{vmnet[:hostonly_subnet]}/25"
          dev_addr = IPAddr.new(dev_subnet)
          if address.end_with?(".1") || !dev_addr.include?(address)
            raise Errors::BaseAddressRange,
              range_start: dev_addr.to_range.first.succ.succ.to_s,
              range_end: dev_addr.to_range.last.to_s
          end
        end
      end
    end
  end
end
