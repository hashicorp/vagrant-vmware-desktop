# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

source "https://rubygems.org"

group :development do
  if File.exist?(File.expand_path("../../vagrant", __FILE__))
    gem "vagrant", path: "../vagrant"
  elsif ENV["VAGRANT_PATH"]
    gem "vagrant", path: ENV["VAGRANT_PATH"]
  else
    gem "vagrant", git: "https://github.com/hashicorp/vagrant.git"
  end
end

group :plugins do
  gemspec
end

# NOTE: Used for packaging
gem "fpm"
