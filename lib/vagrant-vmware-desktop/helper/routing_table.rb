# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

require "ipaddr"

require "vagrant/util/platform"
require "vagrant/util/subprocess"
require "vagrant/util/which"

require "vagrant-vmware-desktop/errors"

module HashiCorp
  module VagrantVMwareDesktop
    module Helper
      # This class reads the TCP/IP routing table for the current host.
      # On Mac OS X and Linux machines, `netstat` is used. Windows is
      # not currently implemented, but similar applications are installed
      # by default.
      class RoutingTable
        def initialize
          if Vagrant::Util::Platform.darwin?
            @table = read_table_darwin
          elsif Vagrant::Util::Platform.linux?
            @table = read_table_linux
          elsif Vagrant::Util::Platform.windows?
            @table = read_table_windows
          else
            raise Errors::RoutingTableUnsupportedOS
          end
        end

        # This will return the device that the given IP would be routed
        # to, or `nil` if it would just route to the default route.
        #
        # @param [String] ip The IP address.
        def device_for_route(ip)
          return @table[ip] if @table.has_key?(ip)

          smallest = nil
          smallest_size = nil
          @table.each do |destination, interface|
            if destination.include?(IPAddr.new(ip))
              # Ruby 2.0 adds a "size" operator to Range, which we should
              # use when we switch to it.
              ip_range = destination.to_range
              ip_size = ip_range.max.to_i - ip_range.min.to_i
              if smallest_size.nil? || ip_size < smallest_size
                smallest = interface
                smallest_size = ip_size
              end
            end
          end

          smallest
        end

        protected

        def read_table_darwin
          netstat_path = Vagrant::Util::Which.which("netstat")
          raise Errors::RoutingTableCommandNotFound if !netstat_path

          r = Vagrant::Util::Subprocess.execute(netstat_path, "-nr", "-f", "inet")
          raise Errors::RoutingTableLoadError, :output => r.stderr if r.exit_code != 0

          result = {}
          r.stdout.split("\n").each do |line|
            # If the line doesn't start with a number, it isn't worth
            # looking at.
            next if line !~ /^\d/

            # Split by whitespace
            parts = line.split

            # Get out the destination and device, since these are
            # the only fields we actually care about.
            destination = parts[0]
            device      = parts[5]

            # Convert the destination into an IPAddr. This involves splitting
            # out the mask and figuring out the proper IP address formatting...
            ip_parts = destination.split("/")
            mask     = ip_parts[1]
            ip_parts = ip_parts[0].split(".")
            mask     ||= Array.new(ip_parts.length, "255")

            while ip_parts.length < 4
              ip_parts << "0"
              mask     << "0" if mask.is_a?(Array)
            end

            # If mask is an array we turn it into a real mask here
            mask = mask.join(".") if mask.is_a?(Array)

            # Build the IP and final destination
            ip_parts = ip_parts.join(".")
            destination = IPAddr.new("#{ip_parts}/#{mask}")

            # Map the destination to the device
            result[destination] = device
          end

          result
        end

        def read_table_linux
          netstat_path = Vagrant::Util::Which.which("netstat")
          raise Errors::RoutingTableCommandNotFound if !netstat_path

          r = Vagrant::Util::Subprocess.execute(netstat_path, "-nr")
          raise Errors::RoutingTableLoadError, :output => r.stderr if r.exit_code != 0

          result = {}
          r.stdout.split("\n").each do |line|
            # If the line doesn't start with a number, it isn't worth
            # looking at.
            next if line !~ /^\d/

            # Split by whitespace
            parts = line.split

            # Grab the pieces
            destination = parts[0]
            mask        = parts[2]
            device      = parts[7]

            # If the destination is default, then ignore
            next if destination == "0.0.0.0"

            # Build the IP
            ip = IPAddr.new("#{destination}/#{mask}")

            # Map the destination to the device
            result[ip] = device
          end

          result
        end

        def read_table_windows
          netsh_path = Vagrant::Util::Which.which("netsh.exe")
          raise Errors::RoutingTableCommandNotFound if !netsh_path

          r = Vagrant::Util::Subprocess.execute(
            netsh_path, "interface", "ip", "show", "route")
          raise Errors::RoutingTableLoadError, :output => r.stderr if r.exit_code != 0

          # Just use Unix-style line endings
          r.stdout.gsub!("\r\n", "\n")

          result = {}
          r.stdout.split("\n").each do |line|
            # Split by whitespace
            parts = line.split

            # If there aren't enough parts, ignore it
            next if parts.length < 6

            # If we didn't get numbers for the metrics, then ignore
            next if parts[2] !~ /^\d+$/

            # Grab the pieces
            ip     = parts[3]
            device = parts[5..-1].join(" ")

            # If we're working with a VMware device, parse out the vmnet
            match = /^VMware Network Adapter (.+?)$/.match(device)
            device = match[1].downcase if match

            # If the destination is default, then ignore
            next if ip == "0.0.0.0/0"

            # Build the IP
            ip = IPAddr.new(ip)

            # Map the destination to the device
            result[ip] = device
          end

          result
        end
      end
    end
  end
end
