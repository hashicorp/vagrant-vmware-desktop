module HashiCorp
  module VagrantVMwareDesktop
    module GuestCap
      module Linux
        class VerifyVMwareHGFS
          def self.verify_vmware_hgfs(machine)
            # Retry a few times since some systems take time to load
            # the VMware kernel modules.
            8.times do |i|
              # Kernel module
              return true if machine.communicate.test(
                "PATH=\"/sbin:$PATH\" lsmod | grep -i '^vmhgfs'")

              # open-vm-tools (FUSE filesystem)
              return true if machine.communicate.test(
                "command -v vmhgfs-fuse")

              sleep(2 ** i)
            end

            return false
          end
        end
      end
    end
  end
end
