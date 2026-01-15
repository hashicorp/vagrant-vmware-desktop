# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

require "vagrant/util/experimental"
require "vagrant/util/numeric"

module HashiCorp
  module VagrantVMwareDesktop
    module Cap
      module Disk

        @@logger = Log4r::Logger.new("hashicorp::provider::vmware::cap::disk")

        DEFAULT_DISK_EXT = "vmdk".freeze
        BUS_TYPES = ["nvme", "sata", "ide", "scsi"].map(&:freeze).freeze
        DEFAULT_BUS = "scsi".freeze
        DEFAULT_DVD_BUS = "ide".freeze
        DEFAULT_DVD_DEVICE_TYPE = "cdrom-image"
        # Adapter types (from vmware-vdiskmanager -h)
        DISK_ADAPTER_TYPES = ["ide", "buslogic", "lsilogic", "pvscsi"].map(&:freeze).freeze
        DEFAULT_ADAPTER_TYPE = "lsilogic".freeze
        # Disk types (from vmware-vdiskmanager -h)
        # 0 : single growable virtual disk
        # 1 : growable virtual disk split into multiple files
        # 2 : preallocated virtual disk
        # 3 : preallocated virtual disk split into multiple files
        # 4 : preallocated ESX-type virtual disk
        # 5 : compressed disk optimized for streaming
        # 6 : thin provisioned virtual disk - ESX 3.x and above
        DEFAULT_DISK_TYPE = 0.freeze
        PRIMARY_DISK_SLOTS = ["nvme0:0", "scsi0:0", "sata0:0", "ide0:0"].map(&:freeze).freeze

        def self.set_default_disk_ext(machine)
          DEFAULT_DISK_EXT
        end

        def self.default_disk_exts(machine)
          [DEFAULT_DISK_EXT].freeze
        end

        # @param [Vagrant::Machine] machine
        # @param [VagrantPlugins::Kernel_V2::VagrantConfigDisk] defined_disks
        # @return [Hash] configured_disks - A hash of all the current configured disks
        def self.configure_disks(machine, defined_disks)
          return {} if defined_disks.empty?

          attached_disks = machine.provider.driver.get_disks(BUS_TYPES)
          configured_disks = {disk: [], floppy: [], dvd: []}

          defined_disks.each do |disk|
            case disk.type
            when :disk
              disk_data = setup_disk(machine, disk, attached_disks)
              if !disk_data.empty?
                configured_disks[:disk] << disk_data
              end
            when :floppy
              # TODO: Write me
              machine.ui.info(I18n.t("hashicorp.vagrant_vmware_desktop.cap.disks.floppy_not_supported", name: disk.name))
            when :dvd
              disk_data = setup_dvd(machine, disk, attached_disks)
              if !disk_data.empty?
                configured_disks[:dvd] << disk_data
              end
            else
              @@logger.info("Invalid disk type #{disk.type}, carrying on.")
            end
          end

          configured_disks
        end

        # @param [Vagrant::Machine] machine
        # @param [VagrantPlugins::Kernel_V2::VagrantConfigDisk] defined_disks
        # @param [Hash] disk_meta - A hash of all the previously defined disks
        #                           from the last configure_disk action
        # @return [nil]
        def self.cleanup_disks(machine, defined_disks, disk_meta)
          return if disk_meta.values.flatten.empty?

          ["disk", "dvd"].each do |k|
            if !disk_meta[k].is_a?(Array)
              raise TypeError, "Expected `Array` but received `#{disk_meta[k].class}`"
            end
          end

          # TODO: floppy
          disk_meta["disk"].each do |d|
            # If disk is defined or the primary disk, don't remove
            if d["primary"]
              @@logger.warn("Vagrant will not clean up the primary disk! Primary disk no longer tracked in Vagrantfile")
              next
            end
            next if defined_disks.any? { |dsk| dsk.name == d["Name"]}
            machine.provider.driver.remove_disk(File.basename(d["Path"]))
          end

          disk_meta["dvd"].each do |d|
            # If disk is defined, don't remove
            next if defined_disks.any? { |dsk| dsk.name == d["Name"]}
            machine.provider.driver.remove_disk_from_vmx(d["Path"], ["deviceType"])
          end
          nil
        end

        protected

        # Retrieves a disk from a Hash of all the disks
        #
        # @param [Config::Disk] disk - the current disk to configure
        # @param [Hash] all_disks - A hash of all currently defined disks
        # @return [Hash, nil] - A hash of the current disk, nil if not found
        def self.get_disk(disk, all_disks)
          if disk.primary
            PRIMARY_DISK_SLOTS.each do |primary_slot|
              disk_info = all_disks[primary_slot]
              if disk_info
                @@logger.debug("disk info for primary slot #{primary_slot} - #{disk_info}")
                return disk_info if disk_info["present"].to_s.upcase == "TRUE"
              end
            end

            nil
          else
            if disk.type == :dvd
              all_disks.values.detect { |v| v["filename"] == disk.file }
            else
              all_disks.values.detect do |v|
                m_ext = File.extname(v["filename"])
                m_fil = v["filename"].sub(/#{Regexp.escape(m_ext)}$/, "")
                m_ext.sub!(".", "")
                m_ext == disk.disk_ext &&
                  (m_fil == disk.name || m_fil =~ /^#{Regexp.escape(disk.name)}-\d+$/)
              end
            end
          end
        end

        # Sets up all disk configs of type `:disk`
        #
        # @param [Vagrant::Machine] machine - the current machine
        # @param [Config::Disk] disk - the current disk to configure
        # @param [Array] all_disks - A list of all currently defined disks in VirtualBox
        # @return [Hash] - disk_metadata
        def self.setup_disk(machine, disk, attached_disks)
          current_disk = get_disk(disk, attached_disks)

          if current_disk.nil?
            raise Errors::DiskPrimaryMissing if disk.primary

            disk_path = create_disk(machine, disk, attached_disks)
          else
            # If the path matches the disk name + some extra characters then
            # make sure to remove the extra characters. They may take the form
            # <name>-f.vmdk, <name>-f<num>.vmdk, <name>-s<num>.vmdk, <name>-s.vmdk, <name>-flat.vmdk, <name>-delta.vmdk
            # ref: https://github.com/libyal/libvmdk/blob/master/documentation/VMWare%20Virtual%20Disk%20Format%20(VMDK).asciidoc
            if current_disk["filename"].match(/\w+-(f\d*|s\d*|flat|delta).vmdk/)
              file_name = current_disk["filename"].gsub(/-(f\d*|s\d*|flat|delta)/, "")
              disk_path = File.join(File.dirname(machine.provider.driver.vmx_path), file_name)
            else
              disk_path = File.join(File.dirname(machine.provider.driver.vmx_path), current_disk["filename"])
            end

            # disk.size is in bytes
            if disk.size > machine.provider.driver.get_disk_size(disk_path)
              if disk.primary && machine.provider.driver.is_linked_clone?
                machine.env.ui.warn(I18n.t("hashicorp.vagrant_vmware_desktop.disk_not_growing_linked_primary"))
                @@logger.warn("Not growing primary disk - guest is linked clone")
              else
                grow_disk(machine, disk_path, disk)
              end
            elsif disk.size < machine.provider.driver.get_disk_size(disk_path)
              machine.env.ui.warn(I18n.t("hashicorp.vagrant_vmware_desktop.disk_not_shrinking", path: disk.name))
              @@logger.warn("Not shrinking disk #{disk.name}")
            else
              @@logger.info("Not changing #{disk.name}")
            end
          end
          {UUID: disk.id, Name: disk.name, Path: disk_path, primary: !!disk.primary}
        end

        # Sets up all disk configs of type `:dvd`
        #
        # @param [Vagrant::Machine] machine - the current machine
        # @param [Config::Disk] disk - the current disk to configure
        # @param [Array] all_disks - A list of all currently defined disks in VirtualBox
        # @return [Hash] - disk_metadata
        def self.setup_dvd(machine, disk, attached_disks)
          disk_provider_config = {}
          if disk.provider_config && disk.provider_config.key?(:vmware_desktop)
            disk_provider_config = disk.provider_config[:vmware_desktop]
            if !BUS_TYPES.include?(disk_provider_config[:bus_type])
              @@logger.warn("#{disk_provider_config[:bus_type]} is not valid. Should be one of " \
                "#{BUS_TYPES.join(', ')}. Setting bus type to #{DEFAULT_DVD_BUS}")
              disk_provider_config[:bus_type] = DEFAULT_DVD_BUS
            end
          end

          current_disk = get_disk(disk, attached_disks)
          bus = disk_provider_config.fetch(:bus_type, DEFAULT_DVD_BUS)
          if current_disk.nil?
            # Attach dvd if not already attached
            slot = get_slot(bus, attached_disks)
            dvd_opts = {"deviceType" => disk_provider_config.fetch(:device_type, DEFAULT_DVD_DEVICE_TYPE)}
            machine.provider.driver.add_disk_to_vmx(disk.file, slot, dvd_opts)
            # Add newly attached disk
            attached_disks[slot] = {"filename" => disk.file}
          end
          {UUID: disk.id, Name: disk.name, Path: disk.file, primary: !!disk.primary}
        end

        # Creates a disk and attaches disk by editing the machine's vmx data
        #
        # @param [Vagrant::Machine] machine
        # @param [Kernel_V2::VagrantConfigDisk] disk_config
        # @return [String] full path to disk file
        def self.create_disk(machine, disk_config, attached_disks)
          if disk_config.file.nil? || !File.exist?(disk_config.file)
            # Create a new disk if a file is not provided, or that path does not exist
            disk_filename = "#{disk_config.name}.#{disk_config.disk_ext}"
            disk_provider_config = {}
            if disk_config.provider_config && disk_config.provider_config.key?(:vmware_desktop)
              disk_provider_config = disk_config.provider_config[:vmware_desktop]

              if !DISK_ADAPTER_TYPES.include?(disk_provider_config[:adapter_type])
                @@logger.warn("#{disk_provider_config[:adapter_type]} is not valid. Should be one " \
                  "of #{DISK_ADAPTER_TYPES.join(', ')}. Setting adapter type to #{DEFAULT_ADAPTER_TYPE}")
                disk_provider_config[:adapter_type] = DEFAULT_ADAPTER_TYPE
              end
              if !BUS_TYPES.include?(disk_provider_config[:bus_type])
                @@logger.warn("#{disk_provider_config[:bus_type]} is not valid. Should be one of " \
                  "#{BUS_TYPES.join(', ')}. Setting bus type to #{DEFAULT_BUS}")
                disk_provider_config[:bus_type] = DEFAULT_BUS
              end
            end
            disk_type = DEFAULT_DISK_TYPE
            disk_adapter = disk_provider_config.fetch(:adapter_type, DEFAULT_ADAPTER_TYPE)
            bus = disk_provider_config.fetch(:bus_type,  DEFAULT_BUS)
            disk_path = machine.provider.driver.create_disk(disk_filename, disk_config.size, disk_type, disk_adapter)
          else
            # Don't create a new disk if one is provided by `file` config argument
            disk_path = disk_config.file
          end

          slot = get_slot(bus, attached_disks)
          machine.provider.driver.add_disk_to_vmx(File.basename(disk_path), slot)
          # Add newly attached disk
          attached_disks[slot] = { "filename" => disk_filename }
          disk_path
        end

        # Expand an existing disk
        #
        # @param [Vagrant::Machine] machine
        # @param [String] disk_path Path to disk for expansion
        # @param [VagrantPlugins::Kernel_V2::VagrantConfigDisk] disk_config
        # @return [nil]
        def self.grow_disk(machine, disk_path, disk_config)
          if machine.provider.driver.snapshot_list.empty?
            machine.provider.driver.grow_disk(disk_path, disk_config.size)
          else
            raise Errors::DiskNotResizedSnapshot, path: disk_path
          end
          nil
        end

        # Gets the next slot available
        #
        # @param [String] bus_name name, one of BUS_TYPES
        # @param [Hash] attached disks
        def self.get_slot(bus_name, attached_disks)
          buses = Hash.new
          # Always init a zero entry
          buses[0] = Set.new([-1])

          # Populate buses with currently used slots
          attached_disks.keys.each do |k|
            val = k.match(/#{bus_name}(?<data>\d+:\d+)/)
            next if val.nil?
            bus, slot = val[:data].split(":", 2)
            buses[bus.to_i] ||= Set.new([-1])
            buses[bus.to_i].add(slot.to_i)
          end

          # Attempt to find any empty slots on
          # the available buses
          bus_num, slot_num = catch(:found) do
            buses.keys.sort.each do |b_idx|
              bus = buses[b_idx]
              (0..bus.max).each do |slot|
                throw :found, [b_idx, slot] if !bus.include?(slot)
              end
            end

            nil
          end

          # If no empty slots found, add new one
          # to final bus
          if bus_num.nil?
            bus_num = buses.keys.max
            slot_num = buses[bus_num].max + 1
          end

          # If this is an IDE bus and the slot
          # is greater than 1, increment bus and
          # reset the slot
          if bus_name.to_s == "ide" && slot_num > 1
            bus_num += 1
            slot_num = 0
          end

          "#{bus_name}#{bus_num}:#{slot_num}"
        end
      end
    end
  end
end
