# A sample Gemfile
source "https://rubygems.org"

group :development do
  if File.exist?(File.expand_path("../../vagrant", __FILE__))
    gem "vagrant", path: "../vagrant"
  elsif ENV["VAGRANT_PATH"]
    gem "vagrant", path: ENV["VAGRANT_PATH"]
  else
    gem "vagrant", git: "git://github.com/mitchellh/vagrant.git"
  end

  gem "rake"
  gem "rspec", "~> 3.1"
  gem "rspec-its", "~> 1.1"
  gem "webmock", "~> 1.9.3"
  gem "pry-byebug"

  #gem "debugger", "~> 1.3.1"
  #gem "vagrant-spec", :git => "git://github.com/mitchellh/vagrant-spec.git"
end

group :plugins do
  gem "vagrant-vmware-desktop",
    :path => ".", :require => "vagrant-vmware-desktop"
end

gem "fpm"
