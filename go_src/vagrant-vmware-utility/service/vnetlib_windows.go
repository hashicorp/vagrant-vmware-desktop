// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package service

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// Windows service names for VMware services
const VMWARE_NAT_SERVICE = `VMware NAT Service`
const VMWARE_DHCP_SERVICE = `VMnetDHCP`

// Windows registry path to the VMnet configurations
const VMNETCONFIG_REGISTRY_PATH = `SOFTWARE\VMware, Inc.\VMnetLib\VMnetConfig`

func (v *VnetlibExe) deletePortFwd(device, protocol, port string) (int, string) {
	var protoKey string
	if protocol == "tcp" {
		protoKey = "TCPForward"
	} else {
		protoKey = "UDPForward"
	}
	fwdPath := VMNETCONFIG_REGISTRY_PATH + `\` + device + `\NAT\` + protoKey
	regKey, err := registry.OpenKey(registry.LOCAL_MACHINE, fwdPath,
		v.registryAccess(registry.ALL_ACCESS))
	if err != nil {
		v.logger.Trace("portforward delete registry open", "path", fwdPath, "error", err)
		return -1, err.Error()
	}
	defer regKey.Close()
	err = regKey.DeleteValue(port)
	if err != nil {
		v.logger.Trace("portforward delete registry delete", "path", fwdPath, "key", port,
			"error", err)
		return -1, err.Error()
	}
	err = regKey.DeleteValue(port + `Description`)
	if err != nil {
		v.logger.Trace("portforward delete registry description delete", "path", fwdPath,
			"key", port+`Description`, "error", err)
	}
	return 1, ""
}

func (v *VnetlibExe) addDevice(name string) (int, string) {
	return v.runcmd("add", "adapter", name)
}

func (v *VnetlibExe) removeDevice(name string) (int, string) {
	return -1, "not implemented"
}

func (v *VnetlibExe) setSubnetAddr(name, addr string) (int, string) {
	return v.runcmd("set", "vnet", name, "addr", addr)
}

func (v *VnetlibExe) setSubnetMask(name, mask string) (int, string) {
	return v.runcmd("set", "vnet", name, "mask", mask)
}

func (v *VnetlibExe) enableNat(name string) (int, string) {
	return v.runcmd("add", "nat", name)
}

func (v *VnetlibExe) disableNat(name string) (int, string) {
	return v.runcmd("remove", "nat", name)
}

func (v *VnetlibExe) enableDhcp(name string) (int, string) {
	return v.runcmd("add", "dhcp", name)
}

func (v *VnetlibExe) disableDhcp(name string) (int, string) {
	return v.runcmd("remove", "dhcp", name)
}

// TODO: Test these enable/disable
func (v *VnetlibExe) enableDevice(name string) (int, string) {
	return v.runcmd("enable", "adapter", name)
}

func (v *VnetlibExe) disableDevice(name string) (int, string) {
	return v.runcmd("disable", "adapter", name)
}

func (v *VnetlibExe) updateDevice(name string) (int, string) {
	return v.runcmd("update", "adapter", name)
}

func (v *VnetlibExe) updateNat(name string) (int, string) {
	return v.runcmd("update", "nat", name)
}

// Creating an IP to MAC mapping always returns 0 and never
// returns output, so just run the command, hope for the
// best and return success.
// The mapping will be added to the Registry but not actually
// used until the config has been rewritten so force that as
// well before leaving.
func (v *VnetlibExe) reserveAddress(device, mac, ip string) (int, string) {
	_, _ = v.runcmd("set", "dhcp", device, "addipmac", ip, mac)
	_, _ = v.runcmd("update", "dhcp", device)
	return 1, ""
}

// This needs to iterate the registry entries
func (v *VnetlibExe) lookupReservedAddress(device, mac string) (int, string) {
	keyPath := VMNETCONFIG_REGISTRY_PATH + `\` + device + `\DHCP\FixedIPtoMac`
	regKey, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath,
		v.registryAccess(registry.ALL_ACCESS))
	if err != nil {
		v.logger.Trace("reserved address lookup registry open failure", "path", keyPath,
			"error", err)
		return -1, err.Error()
	}
	defer regKey.Close()
	regmacs, err := regKey.ReadValueNames(0)
	if err != nil {
		v.logger.Trace("reserved address lookup registry read failure", "path", keyPath,
			"error", err)
		return -1, err.Error()
	}
	seekMac := strings.ToLower(mac)
	for _, regmac := range regmacs {
		if strings.ToLower(regmac) == seekMac {
			v.logger.Trace("reserved address lookup found matching mac", "mac", mac)
			addr, _, err := regKey.GetStringValue(regmac)
			if err != nil {
				v.logger.Trace("reserved address lookup failed to read registry value", "path",
					keyPath, "mac", mac, "error", err)
				return -1, err.Error()
			}
			return 0, addr
		}
	}
	return -1, "no address found"
}

func (v *VnetlibExe) statusNat(name string) (int, string) {
	running, _ := v.serviceRunning(VMWARE_NAT_SERVICE)
	if running {
		return 1, ""
	}
	return -1, ""
}

func (v *VnetlibExe) statusDhcp(name string) (int, string) {
	running, _ := v.serviceRunning(VMWARE_DHCP_SERVICE)
	if running {
		return 1, ""
	}
	return -1, ""
}

func (v *VnetlibExe) startNat(name string) (int, string) {
	srv, err := v.getService(VMWARE_NAT_SERVICE)
	if err != nil {
		v.logger.Trace("start nat get service", "name", VMWARE_NAT_SERVICE,
			"error", err)
		return -1, err.Error()
	}
	defer srv.Close()
	err = srv.Start()
	if err != nil {
		v.logger.Trace("start nat service", "name", VMWARE_NAT_SERVICE,
			"error", err)
		return -1, err.Error()
	}
	return 1, ""
}

func (v *VnetlibExe) startDhcp(name string) (int, string) {
	srv, err := v.getService(VMWARE_DHCP_SERVICE)
	if err != nil {
		v.logger.Trace("start dhcp get service", "name", VMWARE_DHCP_SERVICE,
			"error", err)
		return -1, err.Error()
	}
	defer srv.Close()
	err = srv.Start()
	if err != nil {
		v.logger.Trace("start dhcp service", "name", VMWARE_DHCP_SERVICE,
			"error", err)
		return -1, err.Error()
	}
	return 1, ""
}

func (v *VnetlibExe) stopNat(name string) (int, string) {
	srv, err := v.getService(VMWARE_NAT_SERVICE)
	if err != nil {
		v.logger.Trace("stop nat get service", "name", VMWARE_NAT_SERVICE,
			"error", err)
		return -1, err.Error()
	}
	defer srv.Close()
	_, err = srv.Control(svc.Stop)
	if err != nil {
		v.logger.Trace("stop nat service", "name", VMWARE_NAT_SERVICE,
			"error", err)
		return -1, err.Error()
	}
	err = v.waitForState(srv, svc.Stopped)
	if err != nil {
		v.logger.Trace("stop nat service", "error", err)
		return -1, "Failed to transition NAT service to stopped state"
	}
	return 1, ""
}

func (v *VnetlibExe) stopDhcp(name string) (int, string) {
	srv, err := v.getService(VMWARE_DHCP_SERVICE)
	if err != nil {
		v.logger.Trace("stop dhcp get service", "name", VMWARE_DHCP_SERVICE,
			"error", err)
		return -1, err.Error()
	}
	defer srv.Close()
	_, err = srv.Control(svc.Stop)
	if err != nil {
		v.logger.Trace("stop dhcp service", "name", VMWARE_DHCP_SERVICE,
			"error", err)
		return -1, err.Error()
	}
	err = v.waitForState(srv, svc.Stopped)
	if err != nil {
		v.logger.Trace("stop dhcp service", "error", err)
		return -1, "Failed to transition DHCP service to stopped state"
	}
	return 1, ""
}

func (v *VnetlibExe) getUnusedDevice() (int, string) {
	regKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE,
		VMNETCONFIG_REGISTRY_PATH, v.registryAccess(registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS))
	if err != nil {
		v.logger.Trace("vnetlib registry open failure", "path", VMNETCONFIG_REGISTRY_PATH,
			"error", err)
		return -1, err.Error()
	}
	allDevices, err := regKey.ReadSubKeyNames(-1)
	if err != nil {
		v.logger.Trace("vnetlib registry subkey list failure", "path", VMNETCONFIG_REGISTRY_PATH,
			"error", err)
		return -1, err.Error()
	}
	var devName string
	for i := 1; i <= len(allDevices); i++ {
		devName = fmt.Sprintf("vmnet%d", i)
		// First check if device path exists. If it does not
		// exist then name is valid
		devPath := VMNETCONFIG_REGISTRY_PATH + `\` + devName
		_, err := registry.OpenKey(registry.LOCAL_MACHINE,
			devPath, v.registryAccess(registry.QUERY_VALUE))
		if err != nil {
			v.logger.Trace("vnetlib device key open failure", "path", devPath,
				"error", err)
			return 1, devName
		}

		// Next check if the adapter path exists for the device. If
		// it does not, then name is valid
		adapterPath := devPath + `\Adapter`
		adaKey, err := registry.OpenKey(registry.LOCAL_MACHINE,
			adapterPath, v.registryAccess(registry.QUERY_VALUE))
		if err != nil {
			v.logger.Trace("vnetlib adapter key open failure", "path", adapterPath,
				"error", err)
			return 1, devName
		}

		// Finally check if this adapter is currently enabled. If it
		// is not, then name is valid
		enabled, _, err := adaKey.GetIntegerValue("UseAdapter")
		if err != nil {
			v.logger.Trace("vnetlib adapter usage read failure", "path", adapterPath,
				"error", err)
			return 1, devName
		}
		if enabled != 1 {
			return 1, devName
		}
		v.logger.Trace("vnetlib check device not available", "name", devName)
	}
	v.logger.Trace("vnetlib unused device detection failure", "error", "did not detect open slot")
	return -1, "unknown"
}

func (v *VnetlibExe) buildCommand(args ...string) *exec.Cmd {
	args = append([]string{"--"}, args...)
	return exec.Command(v.ExePath, args...)
}

func (v *VnetlibExe) getService(name string) (*mgr.Service, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, err
	}
	srv, err := m.OpenService(name)
	if err != nil {
		return nil, err
	}
	return srv, nil
}

func (v *VnetlibExe) serviceRunning(name string) (bool, error) {
	srv, err := v.getService(name)
	if err != nil {
		return false, errors.New(fmt.Sprintf(
			"Failed to locate %s: %s", name, err))
	}
	defer srv.Close()
	status, err := srv.Query()
	if err != nil {
		return false, errors.New(fmt.Sprintf(
			"Failed to query %s: %s", name, err))
	}
	if status.State == svc.Running {
		return true, nil
	}
	return false, nil
}

func (v *VnetlibExe) registryAccess(access uint32) uint32 {
	if runtime.GOARCH == "amd64" {
		access = access | registry.WOW64_32KEY
	}
	return access
}

func (v *VnetlibExe) waitForState(srv *mgr.Service, desiredState svc.State) error {
	waitInterval := 1 * time.Second
	for i := 0; i < SERVICE_STATUS_TIMEOUT; i++ {
		currentStatus, err := srv.Query()
		if err != nil {
			v.logger.Trace("service status query", "name", srv.Name, "error", err)
			return err
		}
		if currentStatus.State == desiredState {
			v.logger.Trace("service reached desired state", "name", srv.Name, "state", desiredState)
			return nil
		}
		v.logger.Trace("service not in desired state", "name", srv.Name, "current", currentStatus.State,
			"desired", desiredState)
		time.Sleep(waitInterval)
	}
	return errors.New("Service failed to reach desired state")
}
