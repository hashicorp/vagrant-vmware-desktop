# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

module HashiCorp
  module VagrantVMwareDesktop
    module Helper
      # The Lock module implements some locking primitives for parallelism
      # that respect the Vagrant version that is available.
      module Lock
        def self.lock(machine, name, **opts, &block)
          # Before version 1.6, we don't have any sort of locking
          return block.call if Vagrant::VERSION < "1.6.0"

          # Set some defaults
          opts = { retry: true }.merge(opts)

          # Lock the environment and yield it.
          begin
            return machine.env.lock(name, &block)
          rescue Vagrant::Errors::EnvironmentLockedError
            raise if !opts[:retry]
            sleep 1
            retry
          end
        end
      end
    end
  end
end
