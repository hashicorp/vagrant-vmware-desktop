## 3.0.1

- Fix bug with outputting a snapshot tree 

## 3.0.0 

- Open source plugin, remove encoding

## 2.1.5

- Enable support for experimental disks support
- Fix DHCP reservation requests

## 2.1.4

- Prefer default vmnet8 for NAT device when multiple matches detected
- Include provider option `nat_device` to force specific device

## 2.1.3

- Disable NAT device detection for standard license installations
- Automatically disable linked clones for player

## 2.1.2

- Log warning on MAC OUI mismatches
- Driver support for vmrest utility

## 2.1.1

- Do not include base_mac value when packaging box
- Correct MAC addresses with invalid format (missing `:` characters)
- Halt guest immediately when destroying

## 2.1.0

- Support for Vagrant 2.2.8

## 2.0.3

- Fix synced folder UID and GID parsing
- Fix state check on guest stop validation
- Support faster port forwarding when available

## 2.0.2

- Properly validate provider configuration
- Fix race condition on VM halt action

## 2.0.1

- Timeout soft VM stops to prevent hanging
- Add support for defining MAC address for NAT interface
- Add support for reserving IP address for NAT interface

## 2.0.0

- Updates for Fusion 11 and Workstation 15

## 1.0.4

- Support running within Windows Subsystem for Linux
- Request vmnet verification if supported by utility

## 1.0.3

- Fix utility checkpoint check
- Only run license check once per command invocation

## 1.0.2

- Fix Vagrantfile configuration loading race condition

## 1.0.1

- Detect and remove any previously installed workstation or fusion plugins

## 1.0.0

- Initial release with unified VMware Fusion and Workstation support
- Initial integration with utility service
- Introduce `vmware_desktop` provider for generic provider configuration
- Add support for Vagrant `package` command
- Only print VMX warning messages on initial `up`
- Add checkpoint support

## 0.8.x

* Shared folders are now mounted as opposed to symlinked.
* The special character that replaces '/' in shared folders can now be
  configured for advanced users.
* Shared folder directory is removed on the guest if it is a symbolic link.
* Provider verifies HGFS is properly installed on the guest VM.

## 0.6.3

* Big change to how IDs work internally with VMware VMs. Fixes major
  issues with multiple NFS shares across VMware VMs.

## 0.6.1

* Ability to set the mac address of bridged networks.

## 0.6.0 (unreleased)

* Human-friendly error when permissions don't allow clone of VM.
* Fix issue with mounting shared folder at a path that already exists.

## 0.5.1 (April 16, 2013)

* `vagrant ssh -c` now works with the Fusion provider.

## 0.5.0 (April 16, 2013)

* Retry starting the service starter in case it crashes early.
* Retry enabling shared folders a few times, since there can be race
  conditions here.

## 0.4.2 (March 29, 2013)

* Delete lock files when cloning to avoid "file already in use" errors.

## 0.4.1 (March 27, 2013)

* Set VAGRANT_VMWARE_FUSION_APP to tell Vagrant where VMware Fusion is.

## 0.4.0 (March 22, 2013)

* Don't stomp forwarded port config of every other Vagrant VMware
  Fusion VM on boot.

## 0.3.10 (March 15, 2013)

* Check permissions on the networking file and show an error if they're
  incorrect.
* If there are no enabled shared folders, we won't request enabling
  shared folders from the VM.

## 0.3.9 (March 14, 2013)

* Look for "VMware Fusion.app", not "VMWare Fusion.app", so that the app
  can be properly found on case sensitive file systems.
* Look for VMware Fusion in the "~/Applications" directory as well.

## 0.3.8 (March 14, 2013)

* Fix nil error involved with hostonly network collision detection. [GH-1]

## 0.3.7 (March 13, 2013)

* Run VMX modifications later so that network configurations don't
  overwrite them.

## 0.3.6

* Allow disabling shared folder if the "disabled" option is set.

## 0.3.5

* Report SSH not ready if we can't read an IP.

## 0.3.4

* Support the "https_proxy" environmental variable for activation.

## 0.3.3

* Compile the sudo wrapper to be compatible with OS X back to 10.5
* Fix issue where hostonly collision would fail for existing network devices.
* If vmnet devices aren't healthy, try to reconfigure them before erroring.

## 0.3.2

* Fix an issue with routing table collision detection getting false
  negatives with IP prefixes.

## 0.3.1

* Fix an issue with activation when there is no internet connection.
* Halt will properly discard suspended state if there is any.
* Host only network collision detection is improved.
* Detect when a vmnet device's routes collide with another device and
  show an error.

## 0.3.0

* Detect issues with vmnet services early in `up` process and attempt
  to fix them.
* Huge performance improvements with forwarding ports.
* Improved error detection all around with forwarded port collisions.
* Forwarded port collisions are now corrected if they can be.
* `config.vm.hostname` works with VMware.

## 0.2.1

* Try to "hard" stop the VM after gracefully stopping it.

## 0.2.0

* Add `vagrant provision` support
* Fix issue where shared folders that had ids with '/' in it would fail.
* Properly install the sudo helper (setuid) in more cases to avoid
  errors.
* Clear `DYLD/LD_LIBRARY_PATH` env var when invoking the setuid binary
  to avoid Mac OS X warning messages in stderr.
* Retry certain commands in more places, as VMware sometimes randomly fails.
* Support NFS shared folders.
* Auto-answer any dialogs to attempt to avoid VM hanging.

## 0.1.0

* Initial beta release!
