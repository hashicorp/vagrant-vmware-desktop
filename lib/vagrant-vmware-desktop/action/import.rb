# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

require "pathname"
require "securerandom"

require "log4r"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This class "imports" a VMware machine by copying the proper
      # files over to the VM folder.
      class Import
        include Common

        def initialize(app, env)
          @app = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::import")
        end

        def call(env)
          # Create the folder to store the VM
          vm_folder = env[:machine].provider_config.clone_directory
          vm_folder ||= env[:machine].data_dir
          if VagrantVMwareDesktop.wsl? && !VagrantVMwareDesktop.wsl_drvfs_path?(vm_folder) #  !Vagrant::Util::Platform.wsl_windows_access_bypass?(vm_folder)
            @logger.info("import folder location cannot be used due to file system type (#{vm_folder})")
            vm_folder = File.join(VagrantVMwareDesktop.windows_to_wsl_path(Vagrant::Util::Platform.wsl_windows_appdata_local), "vagrant-vmware-desktop")
            @logger.info("import folder location has been updated to supported location: #{vm_folder}")
          end
          vm_folder = File.expand_path(vm_folder, env[:machine].env.root_path)
          vm_folder = Pathname.new(vm_folder)
          vm_folder.mkpath if !vm_folder.directory?
          vm_folder = vm_folder.realpath

          # Create a random name for our import. We protect against some
          # weird theoretical realities where this never gets us a unique
          # name.
          found = false
          10.times do |i|
            temp = vm_folder.join(SecureRandom.uuid)
            if !temp.exist?
              vm_folder = temp
              found = true
              break
            end
          end
          raise Errors::CloneFolderExists if !found
          vm_folder.mkpath

          # TODO: If cloning, we need to verify the clone machine is not running

          # Determine the primary VMX file for the box
          vmx_file = nil
          if env[:clone_id]
            vmx_file = Pathname.new(env[:clone_id])
          else
            vmx_file = env[:machine].box.metadata["vmx_file"]
            if vmx_file
              vmx_file = env[:machine].box.directory.join(vmx_file)
            end

            if !vmx_file
              # Not specified by metadata, attempt to discover VMX file
              @logger.info("VMX file not in metadata, attempting to discover...")

              env[:machine].box.directory.children(true).each do |child|
                if child.basename.to_s =~ /^(.+?)\.vmx$/
                  vmx_file = child
                  break
                end
              end
            end
          end

          # If we don't have a VMX file, it is an error
          raise Errors::BoxVMXFileNotFound if !vmx_file || !vmx_file.file?

          # Otherwise, log it out and continue
          @logger.debug("Cloning into: #{vm_folder}")
          @logger.info("VMX file: #{vmx_file}")

          # Clone the VM
          clone_name = nil
          clone_name = env[:machine].config.vm.clone if env[:clone_id]
          clone_name = env[:machine].box.name if !clone_name
          env[:ui].info(I18n.t(
            "hashicorp.vagrant_vmware_desktop.cloning",
            :name => clone_name))
          env[:machine].id = env[:machine].provider.driver.clone(vmx_file, vm_folder, env[:machine].provider_config.linked_clone).to_s

          # If we were interrupted, then undo this
          destroy_import(env) if env[:interrupted]

          # Silence!
          env[:machine].provider.driver.suppress_messages

          # Copy the SSH key from the clone machine if we can
          if env[:clone_machine]
            key_path = env[:clone_machine].data_dir.join("private_key")
            if key_path.file?
              FileUtils.cp(
                key_path,
                env[:machine].data_dir.join("private_key"))
            end
          end

          @app.call(env)
        end

        def recover(env)
          if env[:machine].provider.state.id != :not_created
            # Ignore errors that Vagrant knows about.
            return if env["vagrant.error"].is_a?(Vagrant::Errors::VagrantError)

            # Return if we already tried to destroyimport
            return if env[:import_destroyed]

            # Return if we're not supposed to destroy
            return if !env[:destroy_on_error]

            # Note that we already tried to destroy so we don't infinite loop
            env[:import_destroyed] = true

            # Undo the import
            destroy_import(env)
          end
        end

        # This undoes the import by destroying it.
        def destroy_import(env)
          destroy_env = env.dup
          destroy_env.delete(:interrupted)
          destroy_env[:config_validate] = false
          destroy_env[:force_confirm_destroy] = true
          env[:action_runner].run(Action.action_destroy, destroy_env)
        end
      end
    end
  end
end
