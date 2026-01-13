# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This inspects the Vagrantfile provided by the
      # box (if one was provided) and checks if it
      # set the `base_mac` value. If it did, a warning
      # will be generated to the user.
      #
      # NOTE: This action is merely a "best effort" at
      # providing the warning. It is using non-public
      # Vagrant internals to inspect box Vagrantfile.
      # As such, any errors encountered will be ignored
      # with only a debug log.
      class BaseMacConfigWarning
        include Common

        def initialize(app, env)
          @app = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::basemacwarning")
        end

        def call(env)
          catch(:complete) do
            begin
              # Attempt to extract the vagrantfile loader
              loader = env[:machine].vagrantfile.instance_variable_get(:@loader)
              if !loader
                @logger.debug("base_mac check in box vagrantfile failed - cannot access loader")
                throw :complete
              end

              # Attempt to get the Vagrantfile source for the box
              # provided Vagrantfile
              source = loader.instance_variable_get(:@sources)&.keys&.last
              if !source
                @logger.debug("base_mac check in box vagrantfile failed - cannot get box source")
                throw :complete
              end

              begin
                # Attempt to load the partial config
                partial = loader.partial_load(source)

                # Only proceed to display warning if the base_mac value
                # in the partial load matches the base_mac in the final
                # config
                throw :complete if partial.vm.base_mac != env[:machine].config.vm.base_mac

                # Display the warning message
                env[:ui].warn(
                  I18n.t(
                    "hashicorp.vagrant_vmware_desktop.box_base_mac_warning",
                    base_mac: env[:machine].config.vm.base_mac
                  )
                )
              rescue KeyError => err
                @logger.debug("base_mac check in box vagrantfile failed - partial load failure #{err.class}: #{err}")
              end
            rescue => err
              @logger.debug("base_mac check in box vagrantfile failed - #{err.class}: #{err}")
            end
          end

          @app.call(env)
        end
      end
    end
  end
end
