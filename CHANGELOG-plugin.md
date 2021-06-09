## 2.1.6 (UNRELEASED)

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

