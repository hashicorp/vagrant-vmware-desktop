require "log4r"

module HashiCorp
  module VagrantVMwareDesktop
    module Action
      # This can be used with "Call" built-in to check if the machine
      # is suspended.
      class Suspended
        include Common

        def initialize(app, env)
          @app = app
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::suspended")
        end

        def call(env)
          env[:result] = env[:machine].state.id == :suspended
          @logger.debug("result: #{env[:result].inspect}")
          @app.call(env)
        end
      end
    end
  end
end
