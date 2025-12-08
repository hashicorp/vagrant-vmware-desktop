# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

require_relative "../../acceptance/base"

Vagrant::Spec::Acceptance.configure do |c|
  c.component_paths << File.expand_path("../../../acceptance", __FILE__)
  c.skeleton_paths << File.expand_path("../../../acceptance/skeletons", __FILE__)

  options = {
    box: ENV["VAGRANT_SPEC_BOX"],
    plugin: ENV["VAGRANT_VMWARE_PLUGIN_FILE"]
  }

  c.provider "vmware_workstation", options
  c.provider "vmware_fusion", options
end
