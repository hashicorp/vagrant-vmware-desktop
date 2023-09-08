# Vagrant VMware Desktop Providers

This is the common codebase for the official providers for VMware
desktop products: Fusion, Player, and Workstation. This therefore works
on Windows, Mac, and Linux.

## Box Format

All the desktop plugins share a common box format known as `vmware_desktop`.
The plugins also all support the formats `vmware_fusion`, `vmware_workstation`,
and `vmware_player`.

## Developing

There are two separate parts which work together to provide the vagrant-vmware-desktop
functionality. The first is the vagrant-vmware-desktop RubyGem. This does the bulk of
the work for the plugin. The second part is a vagrant-vmware-utility service that the
vagrant-vmware-desktop plugin interacts with. The purpose of this part of the plugin
is to do operations which require privleged access on the host. This includes network
operations and verification of fusion/workstation.

### RubyGem - Desktop plugin

Using bundler allows for local development. If you need to test the RubyGem plugin
on another system you can build a gem by building directly:

```shell
gem build vagrant-vmware-desktop.gemspec
```

### Utility Service

This part of the plugin lives in the `go_src` directory and is required to be
running when using the vagrant-vmware-desktop plugin. To build and start it:


#### Linux / macOS

```shell
cd go_src/vagrant-vmware-utility
go build
./vagrant-vmware-utility certificate generate
sudo ./vagrant-vmware-utility api
```

#### Windows

In a user console, run:

```shell
cd go_src\vagrant-vmware-utility
go build
.\vagrant-vmware-utility certificate generate
```

In an administrator console, run:

```shell
cd go_src\vagrant-vmware-utility
.\vagrant-vmware-utility api
```

#### Certificates

The plugin interacts with the utility service via a REST API. The utility service creates
these certificates with the `certificate generate` command:

```shell
./vagrant-vmware-utility certificate generate
```

This will output the path to certificates directory. Because a development build is in
use, the plugin will be unable to locate the certificates to communicate with the API.
The plugin can be configured with the directory path:

```ruby
Vagrant.configure("2") do |config|
  config.vm.provider :vmware_desktop do |vmware|
    vmware.utility_certificate_path = "PATH"
  end
end
```

Another option is to link the certificates directory to the location the plugin
expects to find them:

```shell
$ sudo mkdir /opt/vagrant-vmware-desktop
$ sudo ln -s PATH /opt/vagrant-vmware-desktop/certificates
```
