// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package service

import (
	"errors"
	"fmt"

	hclog "github.com/hashicorp/go-hclog"

	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

type Vnetlib interface {
	CreateDevice(newName string) (devName string, err error)
	DeleteDevice(devName string) (err error)
	SetSubnetAddress(devName string, addr string) (err error)
	SetSubnetMask(devName string, mask string) (err error)
	SetNAT(devName string, enable bool) error
	UpdateDeviceNAT(devName string) (err error)
	StatusNAT(devName string) bool
	StartNAT(devName string) (err error)
	StopNAT(devName string) (err error)
	SetDHCP(devName string, enable bool) error
	StatusDHCP(devName string) bool
	StartDHCP(devName string) (err error)
	StopDHCP(devName string) (err error)
	LookupReservedAddress(device, mac string) (addr string, err error)
	ReserveAddress(device, mac, ip string) (err error)
	EnableDevice(devName string) (err error)
	DisableDevice(devName string) (err error)
	UpdateDevice(devName string) (err error)
	DeletePortFwd(device, protocol, hostPort string) (err error)
	GetUnusedDevice() (devName string, err error)
}

type VnetlibExe struct {
	ExePath  string
	Services VmwareServices
	logger   hclog.Logger
}

func NewVnetlib(path string, services VmwareServices, logger hclog.Logger) (Vnetlib, error) {
	if !utility.RootOwned(path, true) {
		return nil, errors.New("Failed to locate valid vnetlib executable")
	}
	logger = logger.Named("vnetlib")
	return &VnetlibExe{
		ExePath:  path,
		Services: services,
		logger:   logger}, nil
}

// Device modifications
func (v *VnetlibExe) CreateDevice(newName string) (devName string, err error) {
	if newName == "" {
		devName, err = v.GetUnusedDevice()
		if err != nil {
			return devName, err
		}
	} else {
		devName = newName
	}
	v.logger.Debug("create new device", "device", devName)
	v.Services.WrapOpenServices(func() {
		exitCode, out := v.addDevice(devName)
		if exitCode == 0 {
			v.logger.Debug("create device failed", "device", devName, "exitcode", exitCode)
			v.logger.Trace("create device failed", "device", devName, "output", out)
			err = errors.New("Failed to create new device")
		}
	})
	return devName, err
}

func (v *VnetlibExe) DeleteDevice(devName string) (err error) {
	v.logger.Debug("delete device", "device", devName)
	v.Services.WrapOpenServices(func() {
		exitCode, out := v.removeDevice(devName)
		if exitCode == 0 {
			v.logger.Debug("delete device failed", "device", devName, "exitcode", exitCode)
			v.logger.Debug("delete device failed", "device", devName, "output", out)
			err = errors.New("Failed to delete device")
		}
	})
	return err
}

func (v *VnetlibExe) SetSubnetAddress(devName string, addr string) (err error) {
	v.logger.Debug("set subnet address", "device", devName, "address", addr)
	v.Services.WrapOpenServices(func() {
		exitCode, out := v.setSubnetAddr(devName, addr)
		if exitCode == 0 {
			v.logger.Debug("set subnet address failed", "device", devName, "address", addr, "exitcode", exitCode)
			v.logger.Trace("set subnet address failed", "device", devName, "address", addr, "output", out)
			err = errors.New("Failed to set subnet address")
		}
	})
	return err
}

func (v *VnetlibExe) SetSubnetMask(devName string, mask string) (err error) {
	v.logger.Debug("set subnet mask", "device", devName, "mask", mask)
	v.Services.WrapOpenServices(func() {
		exitCode, out := v.setSubnetMask(devName, mask)
		if exitCode == 0 {
			v.logger.Debug("set subnet mask failed", "device", devName, "mask", mask, "exitcode", exitCode)
			v.logger.Trace("set subnet mask failed", "device", devName, "mask", mask, "output", out)
			err = errors.New("Failed to set subnet mask")
		}
	})
	return err
}

func (v *VnetlibExe) SetNAT(devName string, enable bool) error {
	v.logger.Debug("set NAT", "device", devName, "enable", enable)
	var exitCode int
	var out string
	v.Services.WrapOpenServices(func() {
		if enable {
			exitCode, out = v.enableNat(devName)
		} else {
			exitCode, out = v.disableNat(devName)
		}
	})
	if exitCode == 0 {
		v.logger.Debug("set NAT failed", "device", devName, "enable", enable, "exitcode", exitCode)
		v.logger.Trace("set NAT failed", "device", devName, "output", out)
		return errors.New("Failed to set NAT")
	}
	return nil
}

func (v *VnetlibExe) SetDHCP(devName string, enable bool) error {
	v.logger.Debug("set DHCP", "device", devName, "enable", enable)
	var exitCode int
	var out string
	v.Services.WrapOpenServices(func() {
		if enable {
			exitCode, out = v.enableDhcp(devName)
		} else {
			exitCode, out = v.disableDhcp(devName)
		}
	})
	if exitCode == 0 {
		v.logger.Debug("set DHCP failed", "device", devName, "enable", enable, "exitcode", exitCode)
		v.logger.Trace("set DHCP failed", "device", devName, "output", out)
		return errors.New("Failed to set DHCP")
	}
	return nil
}

func (v *VnetlibExe) LookupReservedAddress(device, mac string) (addr string, err error) {
	v.logger.Debug("looking up dhcp reserved address", "device", device, "mac", mac)
	exitCode, output := v.lookupReservedAddress(device, mac)
	if exitCode != 0 {
		v.logger.Debug("dhcp address lookup failed", "device", device, "mac", mac, "error", output)
		err = errors.New(fmt.Sprintf("No entry found for MAC %s", mac))
	} else {
		addr = output
	}
	return addr, err
}

func (v *VnetlibExe) ReserveAddress(device, mac, ip string) (err error) {
	v.logger.Debug("reserve dhcp address", "device", device, "mac", mac, "address", ip)
	v.Services.WrapOpenServices(func() {
		exitCode, out := v.reserveAddress(device, mac, ip)
		if exitCode == 0 {
			v.logger.Debug("reserve dhcp address failed", "device", device,
				"mac", mac, "address", ip)
			v.logger.Trace("reserve dhcp address failed", "device", device,
				"output", out)
			err = errors.New("Failed to reserve DHCP IP address")
		}
	})
	return err
}

func (v *VnetlibExe) EnableDevice(devName string) (err error) {
	v.logger.Debug("enable device", "device", devName)
	v.Services.WrapOpenServices(func() {
		exitCode, out := v.enableDevice(devName)
		if exitCode == 0 {
			v.logger.Debug("enable device failed", "device", devName, "exitcode", exitCode)
			v.logger.Trace("enable device failed", "device", devName, "output", out)
			err = errors.New("Failed to enable device")
		}
	})
	return err
}

func (v *VnetlibExe) DisableDevice(devName string) (err error) {
	v.logger.Debug("disable device", "device", devName)
	v.Services.WrapOpenServices(func() {
		exitCode, out := v.disableDevice(devName)
		if exitCode == 0 {
			v.logger.Debug("disable device failed", "device", devName, "exitcode", exitCode)
			v.logger.Trace("disable device failed", "device", devName, "output", out)
			err = errors.New("Failed to disable device")
		}
	})
	return err
}

func (v *VnetlibExe) UpdateDevice(devName string) (err error) {
	v.logger.Debug("update device", "device", devName)
	v.Services.WrapOpenServices(func() {
		exitCode, out := v.updateDevice(devName)
		if exitCode == 0 {
			v.logger.Debug("update device failed", "device", devName, "exitcode", exitCode)
			v.logger.Trace("update device failed", "device", devName, "output", out)
			err = errors.New("Failed to update device")
		}
	})
	return err
}

func (v *VnetlibExe) UpdateDeviceNAT(devName string) (err error) {
	v.logger.Debug("update device NAT", "device", devName)
	v.Services.WrapOpenServices(func() {
		exitCode, out := v.updateNat(devName)
		if exitCode == 0 {
			v.logger.Debug("update device NAT failed", "device", devName, "exitcode", exitCode)
			v.logger.Trace("update device NAT failed", "device", devName, "output", out)
			err = errors.New("Failed to update device NAT")
		}
	})
	return err
}

func (v *VnetlibExe) DeletePortFwd(device, protocol, hostPort string) (err error) {
	v.logger.Debug("delete port fwd", "device", device, "port", hostPort, "protocol", protocol)
	v.Services.WrapOpenServices(func() {
		exitCode, out := v.deletePortFwd(device, protocol, hostPort)
		if exitCode == 0 {
			v.logger.Debug("delete port fwd", "device", device, "host-port", hostPort, "exitcode", exitCode)
			v.logger.Trace("delete port fwd", "device", device, "host-port", hostPort, "output", out)
			err = errors.New("Failed to delete port forward")
		}
	})
	return err
}

// Service modfications
func (v *VnetlibExe) StatusNAT(devName string) bool {
	v.logger.Debug("service NAT status")
	exitCode, _ := v.statusNat(devName)
	v.logger.Trace("service NAT status", "exitcode", exitCode)
	return exitCode == 1
}

func (v *VnetlibExe) StatusDHCP(devName string) bool {
	v.logger.Debug("service DHCP status")
	exitCode, _ := v.statusDhcp(devName)
	v.logger.Trace("service DHCP status", "exitcode", exitCode)
	return exitCode == 1
}

func (v *VnetlibExe) StartNAT(devName string) (err error) {
	v.logger.Debug("service NAT start")
	v.Services.WrapOpenServices(func() {
		exitCode, out := v.startNat(devName)
		if exitCode == 0 {
			v.logger.Debug("service NAT start failed", "device", devName, "exitcode", exitCode)
			v.logger.Trace("service NAT start failed", "device", devName, "output", out)
			err = errors.New("Failed to start NAT service")
		}
	})
	return nil
}

func (v *VnetlibExe) StartDHCP(devName string) (err error) {
	v.logger.Debug("service DHCP start")
	v.Services.WrapOpenServices(func() {
		exitCode, out := v.startDhcp(devName)
		if exitCode == 0 {
			v.logger.Debug("service DHCP start failed", "device", devName, "exitcode", exitCode)
			v.logger.Trace("service DHCP start failed", "device", devName, "output", out)
			err = errors.New("Failed to start DHCP service")
		}
	})
	return err
}

func (v *VnetlibExe) StopNAT(devName string) (err error) {
	v.logger.Debug("service NAT stop")
	v.Services.WrapOpenServices(func() {
		exitCode, out := v.stopNat(devName)
		if exitCode == 0 {
			v.logger.Debug("service NAT stop failed", "device", devName, "exitcode", exitCode)
			v.logger.Trace("service NAT stop failed", "device", devName, "output", out)
			err = errors.New("Failed to stop NAT service")
		}
	})
	return err
}

func (v *VnetlibExe) StopDHCP(devName string) (err error) {
	v.logger.Debug("service DHCP stop")
	v.Services.WrapOpenServices(func() {
		exitCode, out := v.stopDhcp(devName)
		if exitCode == 0 {
			v.logger.Debug("service DHCP stop failed", "device", devName, "exitcode", exitCode)
			v.logger.Trace("service DHCP stop failed", "device", devName, "output", out)
			err = errors.New("Failed to stop DHCP service")
		}
	})
	return err
}

// Helpers
func (v *VnetlibExe) GetUnusedDevice() (devName string, err error) {
	v.logger.Debug("request unused device name")
	exitCode, devName := v.getUnusedDevice()
	if exitCode == 0 {
		v.logger.Debug("unused device name request failed", "exitcode", exitCode, "output", devName)
		return devName, errors.New("Failed to generate new device name")
	}
	return devName, err
}

func (v *VnetlibExe) runcmd(args ...string) (exitCode int, output string) {
	return utility.ExecuteWithOutput(v.buildCommand(args...))
}
