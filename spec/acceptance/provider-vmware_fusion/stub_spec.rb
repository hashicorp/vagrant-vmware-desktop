# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# Load all specs defined in common desktop
desktop_dir = File.dirname(File.dirname(__FILE__))
Dir.new(desktop_dir).each do |desktop_path|
  full_path = File.join(desktop_dir, desktop_path)
  if File.file?(full_path) && full_path.end_with?("_spec.rb")
    require full_path
  end
end
