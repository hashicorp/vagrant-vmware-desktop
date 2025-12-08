// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package utility

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (v *VmwarePaths) Load() error {
	v.InstallDir = "/Applications/VMware Fusion.app"
	if _, err := os.Stat(v.InstallDir); err != nil {
		v.logger.Trace("install path does not exist", "path", v.InstallDir)
		return errors.New("Failed to locate VMware installation directory!")
	}
	v.BridgePid = "/var/run/vmnet-bridge.pid"
	v.Networking = "/Library/Preferences/VMware Fusion/networking"
	v.NatConf = "/Library/Preferences/VMware Fusion/{{device}}/nat.conf"
	v.VmnetCli = filepath.Join(v.InstallDir, "Contents/Library/vmnet-cli")
	v.Vnetlib = filepath.Join(v.InstallDir, "Contents/Library/vmnet-cfgcli")
	v.Services = filepath.Join(v.InstallDir, "Contents/Library/services/Open VMware Fusion Services")
	v.Vmrun = filepath.Join(v.InstallDir, "Contents/Library/vmrun")
	v.Vmx = filepath.Join(v.InstallDir, "Contents/Library/vmware-vmx")
	v.Vmrest = filepath.Join(v.InstallDir, "Contents/Library/vmrest")
	v.Vdiskmanager = filepath.Join(v.InstallDir, "Contents/Library/vmware-vdiskmanager")
	return nil
}

func (v *VmwarePaths) UpdateVmwareDhcpLeasePath(version string) error {
	// default to using VMware dhcp lease file
	v.DhcpLease = "/var/db/vmware/vmnet-dhcpd-{{device}}.leases"

	// In Big Sur, VMware started using the native vmnet framework instead
	// of their internal tools. This means that the dhcpd being used is also
	// provided by the platform and not VMware internal tools.
	// However, issues with transiting VPNs required migrating back to the
	// VMware DHCP implementation.

	// New experimental pre-releases use VMware DHCP
	if version == "e.x.p" {
		return nil
	}

	darwin, err := GetDarwinMajor()
	if darwin != 20 {
		return nil
	}

	// Big Sur and Fusion 12.0 or 12.1 used the native framework
	verParts := strings.Split(version, ".")
	if len(verParts) < 2 {
		return errors.New("Invalid version string")
	}
	major, err := strconv.Atoi(verParts[0])
	if err != nil {
		return fmt.Errorf("Invalid major version number: %s", verParts[0])
	}
	minor, err := strconv.Atoi(verParts[1])
	if err != nil {
		return fmt.Errorf("Invalid minor version number: %s", verParts[1])
	}
	if major == 12 && (minor == 0 || minor == 1) {
		v.DhcpLease = "/var/db/dhcpd_leases"
	}

	return nil
}
