# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      module Common
        # We don't want to expose the entire path to the middleware
        # class, so we just give ourselves the class name.
        #
        # @return [String]
        def to_s
          class_name = self.class.to_s.split("::").last
          "VMware Middleware: #{class_name}"
        end
      end
    end
  end
end
