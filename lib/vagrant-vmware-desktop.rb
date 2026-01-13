# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

require "rbconfig"
require "pathname"
require "fileutils"

require "vagrant-vmware-desktop/constants"
require "vagrant-vmware-desktop/plugin"

module HashiCorp
  module VagrantVMwareDesktop

    # List of plugins that are incompatible with this plugin. These
    # are the deprecated versions of the VMware plugin that this
    # plugin replaces.
    CONFLICTING_GEMS = [
      "vagrant-vmware-fusion",
      "vagrant-vmware-workstation"
    ].map(&:freeze).freeze

    autoload :Action, "vagrant-vmware-desktop/action"
    autoload :Cap, "vagrant-vmware-desktop/cap"
    autoload :CheckpointClient, "vagrant-vmware-desktop/checkpoint_client"
    autoload :Config, "vagrant-vmware-desktop/config"
    autoload :Driver, "vagrant-vmware-desktop/driver"
    autoload :Errors, "vagrant-vmware-desktop/errors"
    autoload :Provider, "vagrant-vmware-desktop/provider"
    autoload :SetupPlugin, "vagrant-vmware-desktop/setup_plugin"
    autoload :SyncedFolder, "vagrant-vmware-desktop/synced_folder"

    # This initializes the i18n load path so that the plugin-specific
    # translations work.
    def self.init_i18n
      I18n.load_path << File.expand_path("locales/en.yml", source_root)
      I18n.reload!
    end

    # This initializes the logging so that our logs are outputted at
    # the same level as Vagrant core logs.
    def self.init_logging
      # Initialize logging
      level = nil
      begin
        level = Log4r.const_get(ENV["VAGRANT_LOG"].upcase)
      rescue NameError
        # This means that the logging constant wasn't found,
        # which is fine. We just keep `level` as `nil`. But
        # we tell the user.
        level = nil
      end

      # Some constants, such as "true" resolve to booleans, so the
      # above error checking doesn't catch it. This will check to make
      # sure that the log level is an integer, as Log4r requires.
      level = nil if !level.is_a?(Integer)

      # Set the logging level on all "vagrant" namespaced
      # logs as long as we have a valid level.
      if level
        logger = Log4r::Logger.new("hashicorp")
        logger.outputters = Log4r::Outputter.stderr
        logger.level = level
        logger = nil
      end
    end

    # This returns the path to the source of this plugin.
    #
    # @return [Pathname]
    def self.source_root
      @source_root ||= Pathname.new(File.expand_path("../../", __FILE__))
    end

    # These WSL methods are wrappers so things work
    # gracefully on different versions of Vagrant that
    # may have all, some, or none of the methods available
    # that we need.


    # Detect if inside WSL
    #
    # @return [Boolean]
    def self.wsl?
      Vagrant::Util::Platform.respond_to?(:wsl?) &&
        Vagrant::Util::Platform.wsl?
    end

    # Convert WSL path to a Windows path
    #
    # @param [String, Pathname] path
    # @return [String]
    def self.wsl_to_windows_path(path)
      if wsl?
        oval = ENV["VAGRANT_WSL_ENABLE_WINDOWS_ACCESS"]
        ENV["VAGRANT_WSL_ENABLE_WINDOWS_ACCESS"] = "1"
        path = Vagrant::Util::Platform.wsl_to_windows_path(path.to_s)
        ENV["VAGRANT_WSL_ENABLE_WINDOWS_ACCESS"] = oval
        return "//?/#{path}".tr("/", 92.chr)
      end
      if Vagrant::Util::Platform.windows?
        path.to_s.tr("/", 92.chr)
      else
        path.to_s
      end
    end

    # Convert Windows path to WSL path
    #
    # @param [String, Pathname] path
    # @return [String]
    def self.windows_to_wsl_path(path)
      if wsl?
        if Vagrant::Util::Platform.respond_to?(:windows_to_wsl_path)
          return Vagrant::Util::Platform.windows_to_wsl_path(path.to_s)
        elsif path.match(/^[A-Za-z]:/)
          return "/mnt/#{path[0, 1].downcase}#{path[2..-1].tr(92.chr, '/')}"
        end
      end
      path.to_s
    end

    # Check if path is located on DrvFs file system
    #
    # @param [String, Pathname]
    # @return [Boolean]
    def self.wsl_drvfs_path?(path)
      path = path.to_s
      if Vagrant::Util::Platform.respond_to?(:wsl_drvfs_path?)
        return Vagrant::Util::Platform.wsl_drvfs_path?(path)
      end
      if wsl?
        return Vagrant::Util::Platform.wsl_windows_access_bypass?(path)
      end
      false
    end

    # Check if path is case sensitive
    #
    # @param [String, Pathname] path to check (must exist)
    # @return [Boolean]
    def self.case_sensitive_fs?(path)
      !case_sensitive_fs(path)
    end

    # Check if path is case insensitive
    #
    # @param [String, Pathname] path to check (must exist)
    # @return [Boolean]
    def self.case_insensitive_fs?(path)
      begin
        FileUtils.compare_file(path.to_s, path.to_s.downcase)
      rescue Errno::ENOENT
        false
      end
    end

    # Checks for a valid installation state. Raises exception if
    # invalid state detected.
    #
    # @return [NilClass]
    # @raises [StandardError]
    def self.validate_install!
      CONFLICTING_GEMS.each do |libname|
        if !Gem::Specification.find_all_by_name(libname).empty?
          $stderr.puts <<-EOF
ERROR: Vagrant has detected an incompatible plugin installed. Please uninstall
the `#{libname}` plugin and run the command again. To uninstall the plugin,
run the command shown below:

  vagrant plugin uninstall #{libname}

EOF
          raise "Plugin conflict"
        end
      end
    end

    # Check for any deprecated suid binaries from previous
    # vmware plugin installations and remove them
    def self.orphan_artifact_cleanup!
      if !Vagrant::Util::Platform.windows?
        glob_path = Vagrant.user_data_path.join("gems", "**", "**", "vagrant-vmware-{workstation,fusion}*").to_s
        Dir.glob(glob_path).each do |del_path|
          FileUtils.rm_rf(del_path) if File.directory?(del_path)
        end
      end
    end
  end
end

# Validation the installation
HashiCorp::VagrantVMwareDesktop.validate_install!

# Remove any artifacts that may exist from
# previous plugin installations
HashiCorp::VagrantVMwareDesktop.orphan_artifact_cleanup!
