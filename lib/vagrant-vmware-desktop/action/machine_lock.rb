# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

require "vagrant/action/builtin/lock"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This class locks a single machine so that operations can only be done
      # on one machine at a time.
      class MachineLock < Vagrant::Action::Builtin::Lock
        include Common

        def initialize(app, outer_env)
          options = {}
          options[:path] = lambda do |env|
            env[:machine].data_dir.join("lock")
          end

          options[:exception] = lambda do |env|
            Errors::SingleMachineLock.new(:machine => env[:machine].name)
          end

          super(app, outer_env, options)
        end
      end
    end
  end
end
