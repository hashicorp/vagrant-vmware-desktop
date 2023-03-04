// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package driver

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
	"golang.org/x/sys/windows/registry"
)

const VMWARE_REGISTRY_PATH = `SOFTWARE\VMware, Inc.`
const VMNETLIB_REGISTRY_PATH = `SOFTWARE\VMware, Inc.\VMnetLib`
const POWERSHELL_PATH = `%systemdrive%\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`
const REGINI_PATH = `%systemdrive%\Windows\System32\regini.exe`
const REGISTRY_OWNERSHIP_SCRIPT = `
param(
    [Parameter(Mandatory=$true)]
    [string]$RootKey,
    [Parameter(Mandatory=$true)]
    [string]$RegKey,
    [System.Security.Principal.SecurityIdentifier]$OwnerSID="S-1-5-18"
)

$ErrorActionPreference = "Stop"

# Escalate privileges to allow taking ownership
$import = '[DllImport("ntdll.dll")] public static extern int RtlAdjustPrivilege(ulong a, bool b, bool c, ref bool d);'
$ntdll = Add-Type -Member $import -Name NtDll -PassThru
$null = $ntdll::RtlAdjustPrivilege(9, 1, 0, [ref]0)
$null = $ntdll::RtlAdjustPrivilege(17, 1, 0, [ref]0)
$null = $ntdll::RtlAdjustPrivilege(18, 1, 0, [ref]0)

$key = [Microsoft.Win32.Registry]::$RootKey.OpenSubKey($RegKey, 'ReadWriteSubTree', 'TakeOwnership')
$acl = New-Object System.Security.AccessControl.RegistrySecurity
$acl.SetOwner($OwnerSID)
$key.SetAccessControl($acl)

[System.Security.Principal.SecurityIdentifier]$AdminsSID = "S-1-5-32-544"
[System.Security.Principal.SecurityIdentifier]$EveryoneSID = "S-1-1-0"

$key = [Microsoft.Win32.Registry]::$RootKey.OpenSubKey($RegKey, 'ReadWriteSubTree', 'ChangePermissions')
# Give everyone read access
$rule = New-Object System.Security.AccessControl.RegistryAccessRule($EveryoneSID, 'ReadKey', 'ContainerInherit', 'None', 'Allow')
$acl.ResetAccessRule($rule)
# Allow owner full access
$rule = New-Object System.Security.AccessControl.RegistryAccessRule($OwnerSID, 'FullControl', 'ContainerInherit', 'None', 'Allow')
$acl.ResetAccessRule($rule)
# Allow administrators full access
$rule = New-Object System.Security.AccessControl.RegistryAccessRule($AdminsSID, 'FullControl', 'ContainerInherit', 'None', 'Allow')
$acl.ResetAccessRule($rule)
$key.SetAccessControl($acl)
`

// Generate current list of vmnets
func (b *BaseDriver) Vmnets() (*Vmnets, error) {
	b.logger.Info("collecting vmnets")
	access := b.registryAccess(registry.QUERY_VALUE | registry.ENUMERATE_SUB_KEYS)
	configPath := VMNETLIB_REGISTRY_PATH + `\VMnetConfig`
	regKey, err := registry.OpenKey(registry.LOCAL_MACHINE,
		configPath, access)
	if err != nil {
		b.logger.Trace("vmnet list registry open", "path", configPath,
			"error", err)
		return nil, err
	}
	devices, err := regKey.ReadSubKeyNames(-1)
	if err != nil {
		b.logger.Trace("vmnet list subkeys", "path", configPath,
			"error", err)
		return nil, err
	}
	vmnets := &Vmnets{}
	for _, vmnetName := range devices {
		vn := &Vmnet{Name: vmnetName}
		basePath := VMNETLIB_REGISTRY_PATH + `\VMnetConfig\` + vmnetName
		deviceKey, err := registry.OpenKey(registry.LOCAL_MACHINE,
			basePath, access)
		if err != nil {
			b.logger.Trace("vmnet list registry open", "path", basePath,
				"error", err)
			return nil, err
		}
		natKey, err := registry.OpenKey(registry.LOCAL_MACHINE,
			basePath+`\NAT`, access)
		if err != nil {
			b.logger.Trace("vmnet nat check", "path", basePath+`\NAT`, "error", err)
			vn.Type = "hostOnly"
		} else {
			natEnabled, _, err := natKey.GetIntegerValue("UseNAT")
			if err == nil && natEnabled == 1 {
				vn.Type = "nat"
			} else {
				vn.Type = "hostOnly"
			}
		}
		dhcpKey, err := registry.OpenKey(registry.LOCAL_MACHINE,
			basePath+`\DHCP`, access)
		if err != nil {
			b.logger.Trace("vmnet dhcp check", "path", basePath+`\DHCP`, "error", err)
			vn.Dhcp = "no"
		} else {
			dhcpEnabled, _, err := dhcpKey.GetIntegerValue("UseDHCP")
			if err == nil && dhcpEnabled == 1 {
				vn.Dhcp = "yes"
			} else {
				vn.Dhcp = "no"
			}
		}
		subAddr, _, err := deviceKey.GetStringValue("IPSubnetAddress")
		if err == nil {
			vn.Subnet = subAddr
		} else {
			b.logger.Trace("vmnet subnet ip", "path", basePath, "key", "IPSubnetAddress",
				"error", err)
		}
		subMask, _, err := deviceKey.GetStringValue("IPSubnetMask")
		if err != nil && vn.Subnet != "" {
			b.logger.Trace("vmnet subnet mask", "path", basePath, "key", "IPSubnetMask",
				"error", err)
			vn.Mask = "255.255.255.0"
		} else if err == nil {
			vn.Mask = subMask
		}
		vmnets.Vmnets = append(vmnets.Vmnets, vn)
	}
	vmnets.Num = len(vmnets.Vmnets)
	return vmnets, nil
}

// Generate current list of port forwards for given device
func (b *BaseDriver) PortFwds(device string) (pfwds *PortFwds, err error) {
	pfwds = &PortFwds{}
	if b.InternalPortForwarding() {
		pfwds.PortForwards, err = b.InternalPortFwds()
		return
	}

	var devices []string
	if device != "" {
		device = fmt.Sprintf("vmnet%s", device)
		if err := b.supportPortFwds(device); err != nil {
			b.logger.Trace("portforward check", "device", device, "supported", "false")
			return nil, err
		}
		devices = []string{device}
	} else {
		regKey, err := registry.OpenKey(registry.LOCAL_MACHINE,
			VMNETLIB_REGISTRY_PATH+`\VMnetConfig`,
			b.registryAccess(registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS))
		if err != nil {
			b.logger.Trace("portforward registry open", "path", VMNETLIB_REGISTRY_PATH+`\VMnetConfig`,
				"error", err)
			return nil, err
		}
		allDevices, err := regKey.ReadSubKeyNames(-1)
		if err != nil {
			b.logger.Trace("portforward registry subkeys", "path", VMNETLIB_REGISTRY_PATH+`\VMnetConfig`,
				"error", err)
			return nil, err
		}
		for _, dev := range allDevices {
			if err := b.supportPortFwds(dev); err == nil {
				b.logger.Trace("portforward supported device", "device", dev)
				devices = append(devices, dev)
			}
		}
	}
	fwdList := []*PortFwd{}
	for _, device := range devices {
		tcpFwds, err := b.buildFwdMap(device, "TCPForward")
		if err != nil {
			b.logger.Debug("failed to build tcp forward list", "device", device, "error", err)
			err = nil
		}
		udpFwds, err := b.buildFwdMap(device, "UDPForward")
		if err != nil {
			b.logger.Debug("failed to build udp forward list", "device", device, "error", err)
			err = nil
		}
		slot, err := strconv.Atoi(strings.Replace(device, "vmnet", "", -1))
		if err != nil {
			return nil, err
		}
		for portKey, fwd := range tcpFwds {
			hostPort, err := strconv.Atoi(portKey)
			if err != nil {
				b.logger.Trace("portforward host port conversion", "port", portKey, "error", err)
				return nil, err
			}
			guestPort, err := strconv.Atoi(fwd["port"])
			if err != nil {
				b.logger.Trace("portforward guest port conversion", "port", fwd["port"], "error", err)
				return nil, err
			}

			prtfwd := &PortFwd{
				SlotNumber:  slot,
				Port:        hostPort,
				Protocol:    "tcp",
				Description: fwd["description"],
				Guest: &PortFwdGuest{
					Ip:   fwd["ip"],
					Port: guestPort},
			}
			fwdList = append(fwdList, prtfwd)
		}
		for portKey, fwd := range udpFwds {
			hostPort, err := strconv.Atoi(portKey)
			if err != nil {
				b.logger.Trace("portforward host port conversion", "port", portKey, "error", err)
				return nil, err
			}
			guestPort, err := strconv.Atoi(fwd["port"])
			if err != nil {
				b.logger.Trace("portforward guest port conversion", "port", fwd["port"], "error", err)
				return nil, err
			}

			prtfwd := &PortFwd{
				SlotNumber:  slot,
				Port:        hostPort,
				Protocol:    "udp",
				Description: fwd["description"],
				Guest: &PortFwdGuest{
					Ip:   fwd["ip"],
					Port: guestPort}}
			fwdList = append(fwdList, prtfwd)
		}
	}
	pfwds.PortForwards = fwdList
	pfwds.Num = len(fwdList)
	return pfwds, nil
}

// Find installed VMware product information
func (b *BaseDriver) VmwareInfo() (*VmwareInfo, error) {
	var access uint32
	access = registry.QUERY_VALUE
	if runtime.GOARCH == "amd64" {
		access = access | registry.WOW64_32KEY
	}
	corePath := `SOFTWARE\VMware, Inc.`
	coreKey, err := registry.OpenKey(registry.LOCAL_MACHINE, corePath, access)
	if err != nil {
		b.logger.Trace("vmware core info registry open", "path", corePath, "error", err)
		return nil, err
	}
	product, _, err := coreKey.GetStringValue("Core")
	if err != nil {
		b.logger.Trace("vmware core info registry read", "path", corePath, "key", "Core",
			"error", err)
		return nil, err
		product = `VMware Workstation`
	}
	vmwarePath := corePath + `\` + product
	regKey, err := registry.OpenKey(registry.LOCAL_MACHINE, vmwarePath, access)
	if err != nil {
		b.logger.Trace("vmware info registry open", "path", vmwarePath, "error", err)
		return nil, err
	}
	version, _, err := regKey.GetStringValue("ProductVersion")
	if err != nil {
		b.logger.Trace("vmware info registry read", "path", vmwarePath, "key", "ProductVersion",
			"error", err)
		return nil, err
	}
	matches, err := utility.MatchPattern(`^(?P<version>\d+\.\d+\.\d+)(?P<build>\d+)?`, version)
	if err != nil {
		b.logger.Trace("vmware info version match", "version", version, "error", err)
		return nil, err
	}
	info := strings.Split(product, " ")
	license := strings.ToLower(info[len(info)-1])

	v := &VmwareInfo{
		Product: "Workstation",
		Version: matches["version"],
		Build:   matches["build"],
		Type:    "Release",
		License: license}
	b.vmwareInfo = v
	return v, nil
}

// Windows always validates to true as the utility
// will not install without a valid Workstation installation
func (b *BaseDriver) Validate() bool {
	b.validated = true
	b.validationReason = "Valid VMware installation"
	return true
}

func (b *BaseDriver) registryAccess(access uint32) uint32 {
	if runtime.GOARCH == "amd64" {
		access = access | registry.WOW64_32KEY
	}
	return access
}

func (b *BaseDriver) registryTakeOwnership(root registry.Key, path string) bool {
	var o_prefix string
	switch root {
	case registry.CLASSES_ROOT:
		o_prefix = "ClassesRoot"
	case registry.LOCAL_MACHINE:
		o_prefix = "LocalMachine"
	case registry.USERS:
		o_prefix = "Users"
	}
	cmd := exec.Command(utility.ExpandPath(POWERSHELL_PATH),
		"-ExecutionPolicy", "Unrestricted",
		"-NoProfile", "-Noninteractive",
		"-Command", "& {"+REGISTRY_OWNERSHIP_SCRIPT+"}",
		"-RootKey", o_prefix, "-RegKey", `"`+path+`"`)
	exitCode, output := utility.ExecuteWithOutput(cmd)
	if exitCode != 0 {
		b.logger.Warn("failed to change registry ownership", "prefix", o_prefix, "key", path, "output", output)
		return false
	}
	b.logger.Warn("registry ownership executed", "exitcode", exitCode, "output", output)
	return true
}

func (b *BaseDriver) supportPortFwds(device string) error {
	access := b.registryAccess(registry.QUERY_VALUE)
	// First check that we can get to the requested device
	_, err := registry.OpenKey(registry.LOCAL_MACHINE, VMNETLIB_REGISTRY_PATH+`\VMnetConfig\`+device, access)
	if err != nil {
		b.logger.Trace("portfwd invalid device", "device", device, "error", err)
		return errors.New(fmt.Sprintf(
			"Device does not exist: %s", device))
	}
	// Next check that NAT is enabled on the device
	_, err = registry.OpenKey(registry.LOCAL_MACHINE, VMNETLIB_REGISTRY_PATH+`\VMnetConfig\`+device+`\NAT`, access)
	if err != nil {
		b.logger.Trace("portfwd device nat disabled", "device", device, "error", err)
		return errors.New(fmt.Sprintf(
			"Device does not have NAT service enabled: %s", device))
	}
	return nil
}

func (b *BaseDriver) buildFwdMap(device, key string) (map[string]map[string]string, error) {
	fwdData := map[string]map[string]string{}
	access := b.registryAccess(registry.QUERY_VALUE | registry.ENUMERATE_SUB_KEYS | registry.READ)
	netRegistryPath := VMNETLIB_REGISTRY_PATH + `\VMnetConfig\` + device + `\NAT\` + key
	regKey, err := registry.OpenKey(registry.LOCAL_MACHINE, netRegistryPath, access)
	if err != nil {
		b.logger.Trace("portforward map build registry open", "path", netRegistryPath,
			"error", err)
		return fwdData, nil
	}
	forwards, err := regKey.ReadValueNames(-1)
	if err != nil {
		b.logger.Trace("portforward map build registry key list", "path", netRegistryPath,
			"error", err)
		return nil, err
	}
	b.logger.Trace("portforward mapping list", "path", netRegistryPath, "forwards", forwards)
	for _, fwdKey := range forwards {
		valFwd, _, err := regKey.GetStringValue(fwdKey)
		if err != nil {
			b.logger.Trace("portforward map registry read", "path", netRegistryPath, "key", fwdKey,
				"error", err)
			continue
		}
		b.logger.Trace("portforward mapping raw data", "path", netRegistryPath, "key", fwdKey,
			"value", valFwd)
		if valFwd == "" {
			continue
		}
		// determine if entry or description
		if strings.Contains(fwdKey, "Description") {
			hostPort := strings.Replace(fwdKey, "Description", "", -1)
			if _, ok := fwdData[hostPort]; !ok {
				fwdData[hostPort] = map[string]string{}
			}
			fwdData[hostPort]["description"] = valFwd
		} else {
			if _, ok := fwdData[fwdKey]; !ok {
				fwdData[fwdKey] = map[string]string{}
			}
			parts := strings.Split(valFwd, ":")
			if len(parts) != 2 {
				// log error
				continue
			}
			fwdData[fwdKey]["ip"] = parts[0]
			fwdData[fwdKey]["port"] = parts[1]
		}
	}
	b.logger.Trace("portforward map", "key", key, "device", device, "fwds", fwdData)
	return fwdData, nil
}
