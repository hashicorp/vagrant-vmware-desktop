# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

require "i18n"
require "rspec/its"
# webmock currently fails if this require isn't here. can
# likely be removed at some point in the near future
require "time"
require "webmock/rspec"

require "support/helpers"

# Create a temporary directory where test vagrant will run. The reason we save
# this to a constant is so we can clean it up later.
VAGRANT_TEST_CWD = Dir.mktmpdir("vagrant-vmware-test-cwd")
VAGRANT_TEST_HOME = Dir.mktmpdir("vagrant-vmware-test-home")

# Configure VAGRANT_CWD so that the tests never find an actual
# Vagrantfile anywhere, or at least this minimizes those chances.
ENV["VAGRANT_CWD"] = VAGRANT_TEST_CWD
ENV["VAGRANT_HOME"] = VAGRANT_TEST_HOME
FileUtils.mkdir_p(File.join(VAGRANT_TEST_HOME, "data"))

RSpec.configure do |c|
  c.include HashiCorp::VagrantVMwareDesktop::Spec::Helpers

  c.after(:suite) do
    FileUtils.rm_rf(VAGRANT_TEST_CWD)
    FileUtils.rm_rf(VAGRANT_TEST_HOME)
  end
end

FixturesPath = Pathname(File.dirname(__FILE__)).join('fixtures')

I18n.load_path << File.expand_path("../../locales/en.yml", __FILE__)
I18n.reload!

require "vagrant-vmware-desktop"
HashiCorp::VagrantVMwareDesktop.init_i18n
