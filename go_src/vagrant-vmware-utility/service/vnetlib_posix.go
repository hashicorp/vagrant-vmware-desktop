// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// +build !windows

package service

import (
	"os/exec"

	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

const DHCP_RESERVED_ADDRESS = `IP:\s+(?P<address>[^\s]+)\s`
const UNUSED_VNET_PATTERN = `vmnet:\s+(?P<device_name>[^\s]+)\s`

func (v *VnetlibExe) deletePortFwd(device, protocol, port string) (int, string) {
	return v.runcmd("setnatportfwd", device, protocol, port)
}

func (v *VnetlibExe) addDevice(name string) (int, string) {
	return v.runcmd("addadapter", name)
}

func (v *VnetlibExe) removeDevice(name string) (int, string) {
	return v.runcmd("removeadapter", name)
}

func (v *VnetlibExe) setSubnetAddr(name, addr string) (int, string) {
	return v.runcmd("setsubnetaddr", name, addr)
}

func (v *VnetlibExe) setSubnetMask(name, mask string) (int, string) {
	return v.runcmd("setsubnetmask", name, mask)
}

func (v *VnetlibExe) enableNat(name string) (int, string) {
	return v.runcmd("setnatusage", name, "yes")
}

func (v *VnetlibExe) disableNat(name string) (int, string) {
	return v.runcmd("setnatusage", name, "no")
}

func (v *VnetlibExe) enableDhcp(name string) (int, string) {
	return v.runcmd("setdhcpusage", name, "yes")
}

func (v *VnetlibExe) disableDhcp(name string) (int, string) {
	return v.runcmd("setdhcpusage", name, "no")
}

func (v *VnetlibExe) enableDevice(name string) (int, string) {
	return v.runcmd("enablehostonlyadap", name)
}

func (v *VnetlibExe) disableDevice(name string) (int, string) {
	return v.runcmd("disablehostonlyadap", name)
}

func (v *VnetlibExe) updateDevice(name string) (int, string) {
	return v.runcmd("udpateadapterfromconfig", name)
}

func (v *VnetlibExe) updateNat(name string) (int, string) {
	return v.runcmd("updatenatfromconfig", name)
}

func (v *VnetlibExe) statusNat(name string) (int, string) {
	return v.runcmd("servicestatus", name, "nat")
}

func (v *VnetlibExe) statusDhcp(name string) (int, string) {
	return v.runcmd("servicestatus", name, "dhcp")
}

func (v *VnetlibExe) startNat(name string) (int, string) {
	return v.runcmd("servicestart", name, "nat")
}

func (v *VnetlibExe) startDhcp(name string) (int, string) {
	return v.runcmd("servicestart", name, "dhcp")
}

func (v *VnetlibExe) stopNat(name string) (int, string) {
	return v.runcmd("servicestop", name, "nat")
}

func (v *VnetlibExe) stopDhcp(name string) (int, string) {
	return v.runcmd("servicestop", name, "dhcp")
}

func (v *VnetlibExe) reserveAddress(device, mac, ip string) (int, string) {
	return v.runcmd("setdhcpmac2ip", device, mac, ip)
}

func (v *VnetlibExe) lookupReservedAddress(device, mac string) (int, string) {
	exitCode, out := v.runcmd("getdhcpmac2ip", device, mac)
	if exitCode != 0 {
		return exitCode, out
	}
	matches, err := utility.MatchPattern(DHCP_RESERVED_ADDRESS, out)
	if err != nil {
		return 1, err.Error()
	}
	v.logger.Trace("reserved address lookup", "device", device, "mac",
		mac, "address", matches["address"])
	return 0, matches["address"]
}

func (v *VnetlibExe) getUnusedDevice() (int, string) {
	cmd := v.buildCommand("getunusedvnet")
	exitCode, out := utility.ExecuteWithOutput(cmd)
	if exitCode != 1 {
		return exitCode, out
	}
	matches, err := utility.MatchPattern(UNUSED_VNET_PATTERN, out)
	if err != nil {
		return -1, err.Error()
	}
	v.logger.Trace("unused device name", "device", matches["device_name"])
	return 1, matches["device_name"]
}

func (v *VnetlibExe) buildCommand(args ...string) *exec.Cmd {
	return exec.Command(v.ExePath, args...)
}
