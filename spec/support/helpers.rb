# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

require "pathname"
require "tempfile"

module HashiCorp
  module VagrantVMwareDesktop
    module Spec
      module Helpers
        # This creates a temporary directory and returns the path to it.
        # The temporary directory is guaranteed to exist only for the duration
        # of a spec.
        #
        # @return [Pathname]
        def tempdir
          path = nil
          while path.nil?
            file = Tempfile.new("tempdir")
            path = file.path
            file.unlink

            begin
              Dir.mkdir(path)
            rescue
              path = nil
            end
          end

          Pathname.new(path)
        end

        # This creates a temporary file with the given contents and returns
        # the path to it. The temporary file is guaranteed to exist only for
        # the duration of a spec.
        #
        # @param [String] contents The contents for the temporary file.
        def tempfile(contents)
          # Create the temporary file
          f = Tempfile.new("vagrant-test")
          f.write(contents)
          f.fsync

          # Cache it so that it doesn't get unlinked
          @tempfiles ||= []
          @tempfiles << f

          # Return the path
          Pathname.new(f.path)
        end
      end
    end
  end
end
