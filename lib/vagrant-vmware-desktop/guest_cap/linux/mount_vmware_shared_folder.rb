# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

module HashiCorp
  module VagrantVMwareDesktop
    module GuestCap
      module Linux
        class MountVMwareSharedFolder
          # Root location for vagrant generated vmhgfs mounts
          VAGRANT_ROOT_MOUNT_POINT = "/mnt/vagrant-mounts".freeze

          def self.mount_vmware_shared_folder(machine, name, guestpath, options)
            expanded_guest_path = machine.guest.capability(
              :shell_expand_guest_path, guestpath)

            # Determine if we're using the HGFS kernel module or open-vm-tools.
            # We prefer to use the kernel module so we test for that.
            kernel_mod = machine.communicate.test("PATH=\"/sbin:$PATH\" lsmod | grep -i '^vmhgfs'")

            # The user can also override the mount strategy used by specifying a
            # vmware__mount_strategy option (the prefix is removed by Vagrant core).
            kernel_mod = options[:mount_strategy].to_s == "kernel" if options[:mount_strategy]

            # NOTE: This is pulled directly from the linux cap within vagrant proper. Once it is properly
            #   extracted in vagrant, this should just be a method call for consistency.
            if options[:owner].to_i.to_s == options[:owner].to_s
              mount_uid = options[:owner]
            else
              output = ""
              uid_command = "id -u #{options[:owner]}"
              machine.communicate.execute(uid_command, error_check: false) { |type, data|
                output << data if type == :stdout
              }
              mount_uid = output.strip
              if mount_uid.to_i.to_s != mount_uid
                raise Errors::LinuxMountUIDFailed,
                  folder_name: name
              end
            end

            if options[:group].to_i.to_s == options[:group].to_s
              mount_gid = options[:group]
            else
              output = ""
              gid_command = "getent group #{options[:group]}"
              machine.communicate.execute(gid_command, error_check: false) { |type, data|
                output << data if type == :stdout
              }
              mount_gid = output.strip.split(':').at(2)
              if mount_gid.to_i.to_s != mount_gid && options[:owner] == options[:group]
                output = ""
                result = machine.communicate.execute("id -g #{options[:owner]}", error_check: false) { |type, data|
                  output << data if type == :stdout
                }
                mount_gid = output.strip
              end
              if mount_gid.to_i.to_s != mount_gid
                raise Errors::LinuxMountGIDFailed,
                  folder_name: name
              end
            end
            uid = mount_uid
            gid = mount_gid

            # Create the guest path if it doesn't exist
            machine.communicate.sudo(
              "test -L #{expanded_guest_path} && rm -rf #{expanded_guest_path}",
              error_check: false)
            machine.communicate.sudo("mkdir -p #{expanded_guest_path}")

            # If the kernel module is in use, continue using existing mount strategy. If the
            # kernel module is not in use, then we can assume the open-vm-tools are in use
            # which will automatically mount the share into the /mnt/hgfs directory
            if kernel_mod
              # Expand the name to the proper VMware name
              name = ".host:/#{name}"

              # Start building the mount options starting with the basic UID/GID
              mount_options = "-o uid=#{uid},gid=#{gid}"

              # Options
              mount_options += ",#{options[:extra]}" if options[:extra]
              mount_options += ",#{options[:mount_options].join(",")}" if options[:mount_options]

              # Build the full command
              mount_command = "mount -t vmhgfs"
              mount_command = "#{mount_command} #{mount_options} '#{name}' '#{expanded_guest_path}'"
              # Attempt to mount the folder. We retry here a few times because
              # it can fail early on.
              attempts = 0
              while true
                success = true
                machine.communicate.sudo(mount_command) do |type, data|
                  success = false if type == :stderr && data =~ /No such device/i

                  # Sometimes it takes extra time for the `vmhgfs` filesystem
                  # type to become available for use.
                  success = false if type == :stderr && data =~ /unknown filesystem type/i
                end

                break if success

                attempts += 1
                if attempts > 5
                  raise Vagrant::Errors::LinuxMountFailed,
                    command: mount_command
                end
                sleep 3
              end
            else
              # If using vmhgfs-fuse mounting the shared folder directly results in invalid
              # symlink generation. To resolve this we can bind mount the shared folder from
              # the full host shared mount. The open-vm-tools will automatically mount in
              # /mnt/hgfs, but the biggest issue with this mount is that it is fully accessible
              # to all users. Instead, we unmount that point if it is found, and we remount the
              # entire shared host (not just an individual folder) with uid and gid applied. The
              # container directory is only accessible via root, and the bind mounts result in the
              # expected behavior.
              current_mount_point = "#{VAGRANT_ROOT_MOUNT_POINT}/#{uid}-#{gid}"
              hgfs_mount_options = "allow_other,default_permissions,uid=#{uid},gid=#{gid}"
              hgfs_mount_options << ",#{options[:extra]}" if options[:extra]
              hgfs_mount_options << ",#{Array(options[:mount_options]).join(',')}" if !Array(options[:mount_options]).empty?
              hgfs_mount = "vmhgfs-fuse -o #{hgfs_mount_options} .host:/ '#{current_mount_point}'"

              # Allow user to disable unmounting of default vmhgfs-fuse mount point at /mnt/hgfs
              # by setting: `unmount_default_hgfs = false` in the provider config
              if machine.provider_config.unmount_default_hgfs
                machine.communicate.sudo <<-EOH.gsub(/^ */, '')
                  if mount | grep /mnt/hgfs; then
                    umount /mnt/hgfs
                  fi
                EOH
              end

              # Unique mount point based on uid/gid pair
              machine.communicate.sudo <<-EOH.gsub(/^ */, '')
                mount | grep " #{current_mount_point} "
                if test $? -ne 0; then
                  mkdir -p '#{current_mount_point}'
                  chmod 700 '#{VAGRANT_ROOT_MOUNT_POINT}'
                  #{hgfs_mount}
                fi
              EOH

              # Finally bind mount to the expected guest location
              mount_command = "mount --bind '#{current_mount_point}/#{name}' '#{expanded_guest_path}'"
              machine.communicate.sudo(mount_command)
            end

            # Emit an upstart event if we can
            machine.communicate.sudo <<-EOH.gsub(/^ {14}/, '')
              if command -v /sbin/init && /sbin/init --version | grep upstart; then
                /sbin/initctl emit --no-wait vagrant-mounted MOUNTPOINT='#{expanded_guest_path}'
              fi
            EOH
          end
        end
      end
    end
  end
end
