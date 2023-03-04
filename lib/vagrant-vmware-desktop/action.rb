# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

require "pathname"

require "vagrant/action/builder"

require "vagrant/action/general/package"
require "vagrant/action/general/package_setup_folders"
require "vagrant/action/general/package_setup_files"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # Include the built-in modules so we can use them as top-level things.
      include Vagrant::Action::Builtin
      include Vagrant::Action::General

      autoload :BaseMacToIp, "vagrant-vmware-desktop/action/base_mac_to_ip"
      autoload :Boot, "vagrant-vmware-desktop/action/boot"
      autoload :CheckExistingNetwork, "vagrant-vmware-desktop/action/check_existing_network"
      autoload :CheckVMware, "vagrant-vmware-desktop/action/check_vmware"
      autoload :ClearSharedFolders, "vagrant-vmware-desktop/action/clear_shared_folders"
      autoload :Checkpoint, "vagrant-vmware-desktop/action/checkpoint"
      autoload :Compatibility, "vagrant-vmware-desktop/action/compatibility"
      autoload :Common, "vagrant-vmware-desktop/action/common"
      autoload :Created, "vagrant-vmware-desktop/action/created"
      autoload :Destroy, "vagrant-vmware-desktop/action/destroy"
      autoload :DiscardSuspendedState, "vagrant-vmware-desktop/action/discard_suspended_state"
      autoload :Export, "vagrant-vmware-desktop/action/export"
      autoload :FixOldMachineID, "vagrant-vmware-desktop/action/fix_old_machine_id"
      autoload :ForwardPorts, "vagrant-vmware-desktop/action/forward_ports"
      autoload :Halt, "vagrant-vmware-desktop/action/halt"
      autoload :Import, "vagrant-vmware-desktop/action/import"
      autoload :MachineLock, "vagrant-vmware-desktop/action/machine_lock"
      autoload :MessageAlreadyRunning, "vagrant-vmware-desktop/action/message_already_running"
      autoload :MessageNotCreated, "vagrant-vmware-desktop/action/message_not_created"
      autoload :MessageNotRunning, "vagrant-vmware-desktop/action/message_not_running"
      autoload :Network, "vagrant-vmware-desktop/action/network"
      autoload :PackageVagrantfile, "vagrant-vmware-desktop/action/package_vagrantfile"
      autoload :PrepareNFSSettings, "vagrant-vmware-desktop/action/prepare_nfs_settings"
      autoload :PrepareForwardedPortCollisionParams, "vagrant-vmware-desktop/action/prepare_forwarded_port_collision_params"
      autoload :PrepareSyncedFolderCleanup, "vagrant-vmware-desktop/action/prepare_synced_folder_cleanup"
      autoload :PruneForwardedPorts, "vagrant-vmware-desktop/action/prune_forwarded_ports"
      autoload :PruneNFSExports, "vagrant-vmware-desktop/action/prune_nfs_exports"
      autoload :Running, "vagrant-vmware-desktop/action/running"
      autoload :SetDisplayName, "vagrant-vmware-desktop/action/set_display_name"
      autoload :ShareFolders, "vagrant-vmware-desktop/action/share_folders"
      autoload :SnapshotDelete, "vagrant-vmware-desktop/action/snapshot_delete"
      autoload :SnapshotRestore, "vagrant-vmware-desktop/action/snapshot_restore"
      autoload :SnapshotSave, "vagrant-vmware-desktop/action/snapshot_save"
      autoload :Suspend, "vagrant-vmware-desktop/action/suspend"
      autoload :Suspended, "vagrant-vmware-desktop/action/suspended"
      autoload :VMXModify, "vagrant-vmware-desktop/action/vmx_modify"
      autoload :WaitForAddress, "vagrant-vmware-desktop/action/wait_for_address"
      autoload :WaitForCommunicator, "vagrant-vmware-desktop/action/wait_for_communicator_compat"
      autoload :WaitForVMXHalt, "vagrant-vmware-desktop/action/wait_for_vmx_halt"

      # This action is called to destroy a VM.
      def self.action_destroy
        Vagrant::Action::Builder.new.tap do |b|
          b.use Compatibility

          if Vagrant::VERSION < "1.6.0"
            b.use MachineLock
          end

          b.use FixOldMachineID
          b.use Call, Created do |env1, b2|
            if !env1[:result]
              b2.use MessageNotCreated
              next
            end

            b2.use Call, DestroyConfirm do |env2, b3|
              if env2[:result]
                b3.use ConfigValidate
                b3.use ProvisionerCleanup, :before if defined?(ProvisionerCleanup)
                b3.use EnvSet, :force_halt => env2.key?(:force_halt) ? env2[:force_halt] : true
                b3.use action_halt
                b3.use Destroy
                b3.use PruneForwardedPorts

                if Vagrant::VERSION < "1.4.0"
                  b3.use PruneNFSExports
                else
                  b3.use PrepareSyncedFolderCleanup
                  b3.use SyncedFolderCleanup
                end
              end
            end
          end
          b.use Checkpoint
        end
      end

      # This action is called to stop a running VM.
      def self.action_halt
        Vagrant::Action::Builder.new.tap do |b|
          b.use Compatibility
          b.use ConfigValidate
          b.use FixOldMachineID
          b.use CheckVMware

          if Vagrant::VERSION < "1.6.0"
            b.use MachineLock
          end

          b.use Call, Created do |env1, b2|
            if !env1[:result]
              b2.use MessageNotCreated
              next
            end

            b2.use DiscardSuspendedState
            b2.use Call, Running do |env2, b3|
              if env2[:result]
                b3.use Call, GracefulHalt, :not_running, :running do |env3, b4|
                  if !env3[:result]
                    b4.use DiscardSuspendedState
                    b4.use Halt
                  end

                  b4.use WaitForVMXHalt
                end
              end
            end
          end
          b.use Checkpoint
        end
      end

      # This action is called to package a VM.
      def self.action_package
        Vagrant::Action::Builder.new.tap do |b|
          b.use Compatibility
          b.use ConfigValidate
          b.use FixOldMachineID

          b.use Call, Created do |env1, b2|
            if !env1[:result]
              b2.use MessageNotCreated
              next
            end

            b2.use PackageSetupFolders
            b2.use PackageSetupFiles
            b2.use action_halt
            b2.use PruneForwardedPorts
            b2.use PrepareSyncedFolderCleanup
            b2.use SyncedFolderCleanup
            b2.use Package
            b2.use Export
            b2.use PackageVagrantfile
          end
          b.use Checkpoint
        end
      end

      # This action is called to provision a VM.
      def self.action_provision
        Vagrant::Action::Builder.new.tap do |b|
          b.use Compatibility
          b.use ConfigValidate
          b.use FixOldMachineID

          b.use Call, Created do |env1, b2|
            if !env1[:result]
              b2.use MessageNotCreated
              next
            end

            b2.use Call, Running do |env2, b3|
              if !env2[:result]
                raise Vagrant::Errors::VMNotRunningError
              end

              b3.use Provision
            end
          end
          b.use Checkpoint
        end
      end

      # This action is called when the VM is to be stopped then started.
      def self.action_reload
        Vagrant::Action::Builder.new.tap do |b|
          b.use Compatibility
          b.use FixOldMachineID
          b.use Call, Created do |env1, b2|
            if !env1[:result]
              b2.use MessageNotCreated
              next
            end

            b2.use action_halt
            b2.use action_start
          end
          b.use Checkpoint
        end
      end

      # This action is called when the VM is to be resumed.
      def self.action_resume
        Vagrant::Action::Builder.new.tap do |b|
          b.use Compatibility
          b.use ConfigValidate
          b.use FixOldMachineID
          b.use CheckVMware
          b.use Call, Created do |env1, b2|
            if !env1[:result]
              b2.use MessageNotCreated
              next
            end

            b2.use action_start
          end
          b.use Checkpoint
        end
      end

      # This action is called to delete a snapshot.
      def self.action_snapshot_delete
        Vagrant::Action::Builder.new.tap do |b|
          b.use Compatibility
          b.use ConfigValidate
          b.use FixOldMachineID
          b.use CheckVMware

          b.use Call, Created do |env1, b2|
            if !env1[:result]
              b2.use MessageNotCreated
              next
            end

            b2.use SnapshotDelete
          end
          b.use Checkpoint
        end
      end

      # This action is called to restore a snapshot.
      def self.action_snapshot_restore
        Vagrant::Action::Builder.new.tap do |b|
          b.use Compatibility
          b.use ConfigValidate
          b.use FixOldMachineID
          b.use CheckVMware

          b.use Call, Created do |env1, b2|
            if !env1[:result]
              b2.use MessageNotCreated
              next
            end

            b2.use SnapshotRestore
            b2.use action_start
          end
          b.use Checkpoint
        end
      end

      # This action is called to save a snapshot.
      def self.action_snapshot_save
        Vagrant::Action::Builder.new.tap do |b|
          b.use Compatibility
          b.use ConfigValidate
          b.use FixOldMachineID
          b.use CheckVMware

          b.use Call, Created do |env1, b2|
            if !env1[:result]
              b2.use MessageNotCreated
              next
            end

            b2.use SnapshotSave
          end
          b.use Checkpoint
        end
      end

      # This action is called to SSH into the machine.
      def self.action_ssh
        Vagrant::Action::Builder.new.tap do |b|
          b.use Compatibility
          b.use ConfigValidate
          b.use FixOldMachineID
          b.use CheckVMware
          b.use SSHExec
          b.use Checkpoint
        end
      end

      # This action is called that will run a single SSH command.
      def self.action_ssh_run
        Vagrant::Action::Builder.new.tap do |b|
          b.use Compatibility
          b.use ConfigValidate
          b.use FixOldMachineID
          b.use CheckVMware
          b.use Call, Created do |env1, b2|
            if !env1[:result]
              b2.use MessageNotCreated
              next
            end

            b2.use Call, Running do |env2, b3|
              if !env2[:result]
                raise Vagrant::Errors::VMNotRunningError
              end

              b3.use SSHRun
            end
          end
          b.use Checkpoint
        end
      end

      # This action starts the VM, from whatever state it may be.
      def self.action_start
        Vagrant::Action::Builder.new.tap do |b|
          b.use Compatibility
          b.use FixOldMachineID

          if Vagrant::VERSION >= "1.5.0"
            b.use BoxCheckOutdated
          end

          b.use Call, Running do |env, b2|
            if env[:result]
              b2.use MessageAlreadyRunning
              next
            end

            b2.use CheckExistingNetwork
            b2.use PruneForwardedPorts
            b2.use Call, Suspended do |env2, b3|
              # If it is suspended then the following have no effect
              if !env2[:result]
                if Vagrant::VERSION < "1.6.0"
                  b3.use MachineLock
                end

                b3.use Provision

                if Vagrant::VERSION < "1.4.0"
                  b3.use PruneNFSExports
                  b3.use NFS
                  b3.use ClearSharedFolders
                  b3.use ShareFolders
                else
                  b3.use PrepareSyncedFolderCleanup
                  b3.use SyncedFolderCleanup
                  b3.use SyncedFolders
                end

                b3.use PrepareNFSSettings
                b3.use Network
                b3.use BaseMacToIp
                b3.use SetHostname
              end

              Vagrant::Util::Experimental.guard_with(:disks) do
                b3.use CleanupDisks
                b3.use Disk
              end
              b3.use VMXModify
              b3.use PrepareForwardedPortCollisionParams
              b3.use HandleForwardedPortCollisions
              b3.use Boot
              b3.use WaitForAddress
              b3.use ForwardPorts

              if Vagrant::VERSION < "1.3.0"
                b3.use WaitForCommunicatorCompat
              else
                b3.use WaitForCommunicator
              end
            end
          end
          b.use Checkpoint
        end
      end

      # This action is called to stop a running VM.
      def self.action_suspend
        Vagrant::Action::Builder.new.tap do |b|
          b.use Compatibility
          b.use ConfigValidate
          b.use FixOldMachineID
          b.use CheckVMware
          b.use Call, Created do |env1, b2|
            if !env1[:result]
              b2.use MessageNotCreated
              next
            end

            b2.use Suspend
          end
          b.use Checkpoint
        end
      end

      # This action is called to bring the box up from nothing.
      def self.action_up
        Vagrant::Action::Builder.new.tap do |b|
          b.use Compatibility
          b.use ConfigValidate
          b.use FixOldMachineID
          b.use CheckVMware

          if Vagrant::VERSION < "1.6.0"
            b.use MachineLock
          end


          b.use Call, Created do |env1, b2|
            if !env1[:result]
              # If it is not created, then we need to grab the box,
              # import it, and so on.
              if Vagrant::VERSION < "1.5.0"
                b2.use HandleBoxUrl
              else
                b2.use HandleBox
              end

              # Vagrant 1.8 added config.vm.clone. We do some things
              # to get ready for it here.
              if Vagrant::VERSION >= "1.8.0"
                b2.use PrepareClone
              end

              b2.use Import
              b2.use SetDisplayName
            end

            b2.use action_start
          end
          b.use Checkpoint
        end
      end
    end
  end
end
