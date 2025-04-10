# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

module HashiCorp
  module VagrantVMwareDesktop
    module GuestCap
      module Linux
        autoload :VerifyVMwareHGFS, "vagrant-vmware-desktop/guest_cap/linux/verify_vmware_hgfs"
        autoload :MountVMwareSharedFolder, "vagrant-vmware-desktop/guest_cap/linux/mount_vmware_shared_folder"
      end
    end
  end
end
