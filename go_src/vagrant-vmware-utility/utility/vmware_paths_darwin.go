package utility

import (
	"errors"
	"os"
	"path/filepath"
)

func (v *VmwarePaths) Load() error {
	v.InstallDir = "/Applications/VMware Fusion.app"
	if _, err := os.Stat(v.InstallDir); err != nil {
		v.logger.Trace("install path does not exist", "path", v.InstallDir)
		return errors.New("Failed to locate VMware installation directory!")
	}
	v.BridgePid = "/var/run/vmnet-bridge.pid"
	v.Networking = "/Library/Preferences/VMware Fusion/networking"
	// Starting on Big Sur, VMware is using the native vmnet framework instead
	// of their internal tools. This means that the dhcpd being used is also
	// provided by the platform and not VMware internal tools
	darwin, err := GetDarwinMajor()
	if err != nil {
		// Assume non-Big Sur by default
		darwin = 19
	}
	if darwin >= 20 {
		v.DhcpLease = "/var/db/dhcpd_leases"
	} else {
		v.DhcpLease = "/var/db/vmware/vmnet-dhcpd-{{device}}.leases"
	}
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
