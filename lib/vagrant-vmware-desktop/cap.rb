# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

module HashiCorp
  module VagrantVMwareDesktop
    module Cap
      autoload :Disk, "vagrant-vmware-desktop/cap/disk"
      autoload :Provider, "vagrant-vmware-desktop/cap/provider"
      autoload :Snapshot, "vagrant-vmware-desktop/cap/snapshot"
    end
  end
end
