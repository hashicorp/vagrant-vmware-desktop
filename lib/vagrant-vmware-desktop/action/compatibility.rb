# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This modifies the env with some compatibility layers for
      # prior versions of Vagrant.
      class Compatibility
        include Common

        def initialize(app, env)
          @app = app
        end

        def call(env)
          return @app.call(env) if env[:_vmware_compatibility]

          if Vagrant::VERSION < "1.5.0"
            if env[:ui]
              # Create the new UI methods
              ui = env[:ui]
              def ui.detail(*args)
                self.info(*args)
              end

              def ui.output(*args)
                self.info(*args)
              end
            end
          end

          env[:_vmware_compatibility] = true
          @app.call(env)
        end
      end
    end
  end
end
