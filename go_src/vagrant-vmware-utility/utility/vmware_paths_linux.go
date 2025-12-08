// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package utility

import (
	"errors"
	"os"
)

func (v *VmwarePaths) Load() (err error) {
	v.InstallDir = "/usr/lib/vmware"
	if _, err = os.Stat(v.InstallDir); err != nil {
		v.logger.Trace("install path does not exist", "path", v.InstallDir)
		return errors.New("Failed to locate VMware installation directory!")
	}
	v.BridgePid = "/var/run/vmnet-bridge-0.pid"
	v.DhcpLease = "/etc/vmware/{{device}}/dhcpd/dhcpd.leases"
	v.Networking = "/etc/vmware/networking"
	v.NatConf = "/etc/vmware/{{device}}/nat/nat.conf"
	v.VmnetCli = "/usr/bin/vmware-networks"
	v.Services = "/etc/init.d/vmware"
	v.Vmx = "/usr/lib/vmware/bin/vmware-vmx"
	v.Vmrun = "/usr/bin/vmrun"
	v.Vdiskmanager = "/usr/bin/vmware-vdiskmanager"

	if _, err = os.Stat("/bin/vmrest"); err == nil {
		v.Vmrest = "/bin/vmrest"
		return
	}
	if _, err = os.Stat("/usr/bin/vmrest"); err == nil {
		v.Vmrest = "/usr/bin/vmrest"
		return
	}
	// NOTE: This path will likely not work called directly but we can try
	if _, err = os.Stat("/usr/lib/vmware/bin/vmrest"); err == nil {
		v.Vmrest = "/usr/lib/vmware/bin/vmrest"
		return
	}
	v.Vmrest = "/bin/false"
	return
}

func (v *VmwarePaths) UpdateVmwareDhcpLeasePath(version string) error {
	return nil
}
