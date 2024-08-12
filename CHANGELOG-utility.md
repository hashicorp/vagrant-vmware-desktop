## 1.0.23 (Unreleased)

- Recover from invalid NAT settings file [[GH-124]](https://github.com/hashicorp/vagrant-vmware-desktop/pull/124)
- Update license check to prevent downgrade [[GH-125]](https://github.com/hashicorp/vagrant-vmware-desktop/pull/125)
- Fix lease lookup behavior with multiple matches [[GH-120]](https://github.com/hashicorp/vagrant-vmware-desktop/pull/120)

## 1.0.22 (May 5, 2023)

- Updated dependency libraries [[GH-73]](https://github.com/hashicorp/vagrant-vmware-desktop/pull/73)
- macOS universal build [[GH-58]](https://github.com/hashicorp/vagrant-vmware-desktop/pull/58)

## 1.0.21 (September 29, 2021)

- macOS networking updates and experimental builds support [[GH-16]](https://github.com/hashicorp/vagrant-vmware-desktop/pull/16)
- Add UDP support to internal port forwarding service [[GH-17]](https://github.com/hashicorp/vagrant-vmware-desktop/pull/17)
- Update max open files soft limit at startup on macOS [[GH-19]](https://github.com/hashicorp/vagrant-vmware-desktop/pull/19)

## 1.0.20

- Fix debian packaging to prevent installation error on post install script

## 1.0.19

- Fix crash when enabling internal port forwarding service on downgraded driver
- Support internal port forwarding service on all drivers

## 1.0.18

- Provide port forwarding functionality on Big Sur
- Validate network options and force error on Big Sur when unsupported
- Return unsupported error when attempting DHCP reservations on Big Sur

## 1.0.17

- Only use vmrest when running on macOS Big Sur or greater

## 1.0.16

- Resolve path lookup issues on Workstation
- Track service and configuration files within package metadata

## 1.0.15

- Fix address detection issue on macOS pre Big Sur versions
- Allow service initialization when networking file is not present

## 1.0.14

- Ensure vmrest process is reaped to prevent zombie processes
- Updates for changes introduced in macOS Big Sur

## 1.0.13

- Update license detection to handle combined license format
- Add `-license-override` option to service to force specific license type
- Add extra validation for working vmrest during process initialization
- Add support for configuring process via configuration file

## 1.0.12

- Add VMware license detection
- Disable advanced networking on standard VMware license
- Properly recover from forward conflicts in persisted nat.json

## 1.0.11

- Fix vmrest configuration setup on Windows

## 1.0.10

- Utilize vmrest service when available

## 1.0.9

- Fix service setup on macOS

## 1.0.8

- Fix flag handling
- Add notarization to dmg

## 1.0.7

- Update launchd configuration to listen on localhost only
- Prevent VMware installation check during uninstall on Windows
- Add support for faster additions of port forwards
- Optimize port forward pruning behavior

## 1.0.6

- Fix permission issue with macOS installer

## 1.0.5

- Retain network device ownership and permissions when configuring
  networks on Linux platforms
- Add support for DHCP reservations

## 1.0.4

- Automatically fix registry permissions when state is incorrect
- Ignore invalid port forward description fields

## 1.0.3

- Fix service install command for sysv
- Provide better DHCP lease handling on service re-configure

## 1.0.2

- Reduce Windows event logging noise
- Add vmnet verification to help ensure running vmnet processes
- Fix port forwarding behavior in Fusion 8

## 1.0.1

- Wrap Fusion services process when modifying networking settings
- Use isolated settings for NAT tracking to prevent data loss

## 1.0.0

- Initial utility release
