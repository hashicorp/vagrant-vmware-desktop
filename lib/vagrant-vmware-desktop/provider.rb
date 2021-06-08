require "log4r"
require "thread"
require "vagrant"

module HashiCorp
  module VagrantVMwareDesktop
    class Provider < Vagrant.plugin("2", :provider)
      attr_reader :driver

      def initialize(machine)
        @logger      = Log4r::Logger.new("hashicorp::vagrant::vmware")
        @machine     = machine

        # Force a load of the driver
        machine_id_changed
      end

      def action(name)
        # Attempt to get the action method from the Action class if it
        # exists, otherwise return nil to show that we don't support the
        # given action.
        action_method = "action_#{name}"
        return Action.send(action_method) if Action.respond_to?(action_method)
        nil
      end

      def machine_id_changed
        begin
          id = @machine.id
          id = id.chomp if id
          @logger.debug("Re-initializing driver for new ID: #{id.inspect}")
          @driver = Driver.create(id, @machine.provider_config)
        rescue Errors::DriverMissingVMX
          # Delete the VM. This will trigger a machine_id_changed again that
          # should never fail.
          @machine.id = nil
        end
      end

      # Returns the SSH info for accessing the VMware VM.
      def ssh_info
        # We can't SSH if it isn't running
        return nil if state.id != :running

        # Try to read the IP of the machine so we can access it. If
        # this returns nil then we report that we're not ready for SSH.
        # We retry this a few times because sometimes VMware doesn't have
        # an IP ready right away.
        machine_ip = nil
        10.times do |i|
          machine_ip = @driver.read_ip(
            @machine.provider_config.enable_vmrun_ip_lookup
          )
          break if machine_ip
          sleep i+1
        end

        return nil if !machine_ip

        if !@machine.provider_config.ssh_info_public
          @logger.debug("Using localhost lookup for SSH info.")
          host_port = @driver.host_port_forward(machine_ip, :tcp, @machine.config.ssh.guest_port)
          if host_port
            return {
              :host => "127.0.0.1",
              :port => host_port
            }
          else
            @logger.error("Failed localhost SSH info lookup. Using public address.")
          end
        end
        @logger.debug("Using public address lookup for SSH info.")
        return {
          :host => machine_ip,
          :port => @machine.config.ssh.guest_port
        }
      end

      def state
        state_id = @driver.read_state
        @logger.debug("VM state requested. Current state: #{state_id}")

        # Get the short and long description
        short = I18n.t("hashicorp.vagrant_vmware_desktop.states.short_#{state_id}")
        long  = I18n.t("hashicorp.vagrant_vmware_desktop.states.long_#{state_id}")

        # Return the MachineState object
        Vagrant::MachineState.new(state_id, short, long)
      end

      def to_s
        "VMware #{PRODUCT_NAME.capitalize}"
      end
    end
  end
end
