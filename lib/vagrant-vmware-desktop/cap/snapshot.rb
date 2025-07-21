# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

module HashiCorp
  module VagrantVMwareDesktop
    module Cap
      module Snapshot

        @@logger = Log4r::Logger.new("hashicorp::provider::vmware::cap::snapshot")

        # List snapshots
        #
        # @param [Vagrant::Machine] machine - the current machine
        # @return [List<String>] - snapshot names
        def self.snapshot_list(machine)
          machine.provider.driver.snapshot_list
        end

        # Delete all snapshots for the machine
        #
        # @param [Vagrant::Machine] machine - the current machine
        def self.delete_all_snapshots(machine)
          # To delete a snapshot with children of the same name, use the 
          # full path to the snapshot. eg. /clone/clone if the machine
          # has 2 snapshots called "clone"
          snapshots = machine.provider.driver.snapshot_tree
          snapshots.sort {|x, y| y.length <=> x.length}.each do |snapshot|
            @@logger.info("Deleting snapshot #{snapshot}")
            machine.provider.driver.snapshot_delete(snapshot)
          end
        end

        # Delete a given snapstho
        #
        # @param [Vagrant::Machine] machine - the current machine
        # @param [String] snapshot_name - name of the snapshot to delete
        def self.delete_snapshot(machine, snapshot_name)
          @@logger.info("Deleting snapshot #{snapshot_name}")
          machine.provider.driver.snapshot_delete(snapshot_name)
        end
      end
    end
  end
end
