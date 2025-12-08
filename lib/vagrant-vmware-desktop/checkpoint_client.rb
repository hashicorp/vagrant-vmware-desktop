# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

require "log4r"
require "singleton"
require "vagrant-vmware-desktop/config"
require "vagrant-vmware-desktop/constants"

module HashiCorp
  module VagrantVMwareDesktop
    class CheckpointClient

      include Singleton

      # Maximum number of seconds to wait for check to complete
      CHECKPOINT_TIMEOUT = 10

      # @return [Log4r::Logger]
      attr_reader :logger

      # @return [Boolean]
      attr_reader :enabled

      # @return [Hash]
      attr_reader :files

      # @return [Vagrant::Environment]
      attr_reader :env

      def initialize
        @logger = Log4r::Logger.new("hashicorp::provider::vmware::checkpoint_client")
        @enabled = false
      end

      # Setup will attempt to load the checkpoint library and log if
      # it is not found. Checkpoint should be around as it is a
      # dependency of Vagrant, but if it's not, it shouldn't be a
      # show stopper
      #
      # @param [Vagrant::Environment] env
      # @return [self]
      def setup(env)
        begin
          require "checkpoint"
          @enabled = true
        rescue LoadError
          @logger.debug("checkpoint library not found. disabling.")
        end
        if ENV["VAGRANT_CHECKPOINT_DISABLE"]
          @logger.debug("checkpoint disabled via explicit request with environment variable")
          @enabled = false
        end
        @files = {
          plugin_signature: env.data_dir.join("checkpoint_signature-vmp"),
          plugin_cache: env.data_dir.join("checkpoint_cache-vmp"),
          utility_signature: env.data_dir.join("checkpoint_signature-vmu"),
          utility_cache: env.data_dir.join("checkpoint_cache-vmu")
        }
        @checkpoint_threads = {}
        @env = env
        self
      end

      # Start checkpoint checks for both the plugin and utility
      #
      # @return [self]
      def check
        check_plugin
        check_utility
        self
      end

      # All checks complete
      def complete?
        complete_plugin? && complete_utility?
      end

      # Plugin check complete
      def complete_plugin?
        if @checkpoint_threads.key?(:plugin)
          !@checkpoint_threads[:plugin].alive?
        else
          true
        end
      end

      # Utility check complete
      def complete_utility?
        if @checkpoint_threads.key?(:utility)
          !@checkpoint_threads[:utility].alive?
        else
          true
        end
      end

      # Result of all checks
      #
      # @return [Hash]
      def result
        {
          desktop: result_plugin,
          utility: result_utility
        }
      end

      # Get result from plugin check, wait if not complete
      def result_plugin
        if !enabled || !@checkpoint_threads.key?(:plugin)
          nil
        elsif !defined?(@result_plugin)
          @checkpoint_threads[:plugin].join(CHECKPOINT_TIMEOUT)
          @result_plugin = @checkpoint_threads[:result]
        else
          @result_plugin
        end
      end

      # Get result from utility check, wait if not complete
      def result_utility
        if !enabled || !@checkpoint_threads.key?(:utility)
          nil
        elsif !defined?(@result_utility)
          @checkpoint_threads[:utility].join(CHECKPOINT_TIMEOUT)
          @result_utility = @checkpoint_threads[:result]
        else
          @result_utility
        end
      end

      # Run check for plugin
      #
      # @return [self]
      def check_plugin
        if enabled && !@checkpoint_threads.key?(:plugin)
          logger.debug("starting plugin check")
          @checkpoint_threads[:plugin] = Thread.new do
            Thread.current.abort_on_exception = false
            Thread.current.report_on_exception = false
            begin
              Thread.current[:result] = Checkpoint.check(
                product: "vagrant-vmware-desktop",
                version: VERSION,
                signature_file: files[:plugin_signature],
                cache_file: files[:plugin_cache]
              )
              if Thread.current[:result].is_a?(Hash)
                Thread.current[:result].merge!("installed_version" => VERSION)
              else
                Thread.current[:result] = nil
              end
              logger.debug("plugin check complete")
            rescue => e
              logger.debug("plugin check failure - #{e}")
            end
          end
        end
        self
      end

      # Run check for utility
      #
      # @return [self]
      def check_utility
        if enabled && !@checkpoint_threads.key?(:utility)
          logger.debug("starting utility check")
          @checkpoint_threads[:utility] = Thread.new do
            Thread.current.abort_on_exception = false
            Thread.current.report_on_exception = false
            begin
              tmp_config = Config.new
              tmp_config.finalize!
              driver = Driver.create(nil, tmp_config)
              @logger.debug("getting utility version")
              utility_version = driver.vmware_utility_version
              @logger.debug("got utility version: #{utility_version.inspect}")
              if utility_version
                begin
                  @logger.debug("running utility checkpoint check")
                  Thread.current[:result] = Checkpoint.check(
                    product: "vagrant-vmware-utility",
                    version: utility_version,
                    signature_file: files[:utility_signature],
                    cache_file: files[:utility_cache]
                  )
                  if Thread.current[:result].is_a?(Hash)
                    Thread.current[:result].merge!("installed_version" => utility_version)
                  else
                    Thread.current[:result] = nil
                  end
                  logger.debug("utility check complete")
                rescue => e
                  logger.debug("utility check failure - #{e}")
                end
              else
                logger.debug("skipping utility checkpoint, unable to determine version")
              end
            rescue => e
              logger.debug("utility communication error - #{e}")
            end
          end
        end
        nil
      end
    end
  end
end
