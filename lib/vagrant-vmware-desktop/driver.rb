# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

require "vagrant/util/platform"

require "vagrant-vmware-desktop/driver/base"

module HashiCorp
  module VagrantVMwareDesktop
    module Driver
      # This returns a new driver for the given VM directory, using the
      # proper underlying platform driver.
      def self.create(vm_dir, config)
        Base.new(vm_dir, config)
      end
    end
  end
end
