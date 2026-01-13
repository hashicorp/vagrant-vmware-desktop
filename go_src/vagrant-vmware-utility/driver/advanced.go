// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package driver

import (
	"fmt"
	"strconv"

	hclog "github.com/hashicorp/go-hclog"

	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/service"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

// Advanced driver is used where the vnet lib
// is public allowing targeted networking service
// modification and service interaction (Workstation
// Windows and Fusion)

type AdvancedDriver struct {
	BaseDriver
	vnetlib service.Vnetlib
}

func NewAdvancedDriver(vmxPath *string, b *BaseDriver, logger hclog.Logger) (a *AdvancedDriver, err error) {
	logger = logger.Named("advanced")
	if b == nil {
		b, err = NewBaseDriver(vmxPath, "", logger)
		if err != nil {
			return nil, err
		}
	}
	vnetlib, err := service.NewVnetlib(
		b.vmwarePaths.Vnetlib,
		b.VmwareServices,
		logger)
	if err != nil {
		logger.Error("vnetlib creation failed", "error", err)
		return
	}
	a = &AdvancedDriver{
		BaseDriver: *b,
		vnetlib:    vnetlib}
	return
}

func (a *AdvancedDriver) AddVmnet(vmnet *Vmnet) error {
	device, err := a.vnetlib.CreateDevice(vmnet.Name)
	if err != nil {
		a.logger.Debug("device creation failed", "device-name", vmnet.Name, "error", err)
		return err
	}
	// Configure any hostonly options
	if vmnet.Mask != "" {
		if err = a.vnetlib.SetSubnetMask(device, vmnet.Mask); err != nil {
			a.logger.Debug("device subnet mask set failure", "device-name", vmnet.Name,
				"mask", vmnet.Mask, "error", err)
			return err
		}
	}
	if vmnet.Subnet != "" {
		if err = a.vnetlib.SetSubnetAddress(device, vmnet.Subnet); err != nil {
			a.logger.Debug("device subnet set failure", "device-name", vmnet.Name,
				"subnet", vmnet.Subnet, "error", err)
			return err
		}
	}
	// Enable the device
	if err = a.vnetlib.EnableDevice(device); err != nil {
		a.logger.Debug("device enable failure", "device-name", vmnet.Name, "error", err)
		return err
	}
	// Enable any required services
	if vmnet.Dhcp == "yes" {
		if err = a.vnetlib.SetDHCP(device, true); err != nil {
			a.logger.Debug("device DHCP enable failure", "device-name", vmnet.Name,
				"error", err)
			return err
		}
		if err = a.vnetlib.StartDHCP(device); err != nil {
			a.logger.Debug("device DHCP start failure", "device-name", vmnet.Name,
				"error", err)
			return err
		}
	}
	if vmnet.Type == "nat" {
		if err = a.vnetlib.SetNAT(device, true); err != nil {
			a.logger.Debug("device NAT enable failure", "device-name", vmnet.Name,
				"error", err)
			return err
		}
		if err = a.vnetlib.StartNAT(device); err != nil {
			a.logger.Debug("device NAT start failure", "device-name", vmnet.Name,
				"error", err)
			return err
		}
	}
	vmnet.Name = device
	a.logger.Debug("vmnet create", "name", device, "dhcp", vmnet.Dhcp,
		"type", vmnet.Type, "subnet", vmnet.Subnet, "mask",
		vmnet.Mask)
	return nil
}

func (a *AdvancedDriver) UpdateVmnet(vmnet *Vmnet) error {
	device := vmnet.Name
	// Configure any hostonly options
	if vmnet.Mask != "" {
		if err := a.vnetlib.SetSubnetMask(device, vmnet.Mask); err != nil {
			a.logger.Debug("device subnet mask set failure", "device-name", vmnet.Name,
				"mask", vmnet.Mask, "error", err)
			return err
		}
	}
	if vmnet.Subnet != "" {
		if err := a.vnetlib.SetSubnetAddress(device, vmnet.Subnet); err != nil {
			a.logger.Debug("device subnet set failure", "device-name", vmnet.Name,
				"subnet", vmnet.Subnet, "error", err)
			return err
		}
	}
	// Apply new configuration
	if err := a.vnetlib.UpdateDevice(device); err != nil {
		a.logger.Debug("device update failure", "device-name", vmnet.Name,
			"error", err)
		return err
	}

	// Enable/disable any required services
	if a.vnetlib.StatusDHCP(device) {
		if vmnet.Dhcp == "no" {
			if err := a.vnetlib.SetDHCP(device, false); err != nil {
				a.logger.Debug("device DHCP disable failure", "device-name", vmnet.Name,
					"error", err)
				return err
			}
			if err := a.vnetlib.StopDHCP(device); err != nil {
				a.logger.Debug("device DHCP stop failure", "device-name", vmnet.Name,
					"error", err)
				return err
			}
		}
	} else {
		if vmnet.Dhcp == "yes" {
			if err := a.vnetlib.SetDHCP(device, true); err != nil {
				a.logger.Debug("device DHCP enable failure", "device-name", vmnet.Name,
					"error", err)
				return err
			}
			if err := a.vnetlib.StartDHCP(device); err != nil {
				a.logger.Debug("device DHCP start failure", "device-name", vmnet.Name,
					"error", err)
				return err
			}
		}
	}

	if a.vnetlib.StatusNAT(device) {
		if vmnet.Type != "nat" {
			if err := a.vnetlib.SetNAT(device, false); err != nil {
				a.logger.Debug("device NAT disable failure", "device-name", vmnet.Name,
					"error", err)
				return err
			}
			if err := a.vnetlib.StopNAT(device); err != nil {
				a.logger.Debug("device NAT stop failure", "device-name", vmnet.Name,
					"error", err)
				return err
			}
		}
	} else {
		if vmnet.Type == "nat" {
			if err := a.vnetlib.SetNAT(device, true); err != nil {
				a.logger.Debug("device NAT enable failure", "device-name", vmnet.Name,
					"error", err)
				return err
			}
			if err := a.vnetlib.StartNAT(device); err != nil {
				a.logger.Debug("device NAT start failure", "device-name", vmnet.Name,
					"error", err)
				return err
			}
		}
	}
	a.logger.Debug("vmnet update", "name", device, "dhcp", vmnet.Dhcp,
		"type", vmnet.Type, "subnet", vmnet.Subnet, "mask",
		vmnet.Mask)
	return nil
}

func (a *AdvancedDriver) DeleteVmnet(vmnet *Vmnet) error {
	device := vmnet.Name
	// First disable the device
	if err := a.vnetlib.DisableDevice(device); err != nil {
		a.logger.Debug("device disable failure", "device-name", device, "error", err)
		return err
	}
	// Now remove the device
	if err := a.vnetlib.DeleteDevice(device); err != nil {
		a.logger.Debug("device delete failure", "device-name", device, "error", err)
		return err
	}
	return nil
}

// Lookup reserved DHCP address for MAC
func (a *AdvancedDriver) LookupDhcpAddress(device, mac string) (addr string, err error) {
	leases, err := utility.LoadDhcpLeaseFile(a.vmwarePaths.DhcpLeaseFile(device), a.logger)
	if err != nil {
		a.logger.Debug("dhcp leases file load failure", "error", err)
		return addr, err
	}
	paddr, err := leases.IpForMac(mac)
	if err == nil {
		return paddr, err
	}
	return a.vnetlib.LookupReservedAddress(device, mac)
}

func (a *AdvancedDriver) ReserveDhcpAddress(slot int, mac, ip string) error {
	device := fmt.Sprintf("vmnet%d", slot)
	if err := a.vnetlib.ReserveAddress(device, mac, ip); err != nil {
		a.logger.Debug("dhcp reservation failure", "device", device,
			"mac", mac, "address", ip, "error", err)
		return err
	}
	a.logger.Trace("restarting DHCP service to apply update", "device", device)
	_ = a.vnetlib.StopDHCP(device)
	err := a.vnetlib.StartDHCP(device)
	if err != nil {
		a.logger.Error("dhcp service restart failure", "device", device, "error", err)
		return err
	}
	return nil
}

// For deletion of the port forward we can just use the vnetlib
// CLI directly as we no longer care about the description
func (a *AdvancedDriver) DeletePortFwd(pfwds []*PortFwd) error {
	devices := []string{}
	for _, pfwd := range pfwds {
		if a.InternalPortForwarding() {
			if err := a.DeleteInternalPortForward(pfwd); err != nil {
				return err
			}
		} else {
			deviceName := fmt.Sprintf("vmnet%d", pfwd.SlotNumber)
			if err := a.vnetlib.DeletePortFwd(deviceName, pfwd.Protocol, strconv.Itoa(pfwd.Port)); err != nil {
				a.logger.Debug("port forward delete failure", "error", err)
				return err
			}
			rfwd := &utility.PortFwd{HostPort: pfwd.Port, Protocol: pfwd.Protocol}
			if err := a.settings.NAT.Remove(rfwd); err != nil {
				a.logger.Debug("failed to remove forward from settings", "error", err)
				return err
			}
			if err := a.settings.NAT.Save(); err != nil {
				a.logger.Debug("failed to save nat settings", "error", err)
				return err
			}
			found := false
			for _, name := range devices {
				if name == deviceName {
					found = true
				}
			}
			if !found {
				devices = append(devices, deviceName)
			}
		}
	}
	for _, deviceName := range devices {
		if err := a.restartNAT(deviceName); err != nil {
			return err
		}
	}
	return nil
}

func (a *AdvancedDriver) restartNAT(device string) error {
	if err := a.vnetlib.StopNAT(device); err != nil {
		a.logger.Debug("NAT stop failure", "device", device, "error", err)
		return err
	}
	if err := a.vnetlib.UpdateDeviceNAT(device); err != nil {
		a.logger.Debug("device NAT update failure", "device", device, "error", err)
		return err
	}
	if err := a.vnetlib.StartNAT(device); err != nil {
		a.logger.Debug("device NAT start failure", "device", device, "error", err)
		return err
	}
	return nil
}

func (a *AdvancedDriver) savePortFwd(pfwd *PortFwd) error {
	newPf := &utility.PortFwd{
		Device:      strconv.Itoa(pfwd.SlotNumber),
		Protocol:    pfwd.Protocol,
		Description: pfwd.Description,
		HostPort:    pfwd.Port,
		GuestIp:     pfwd.Guest.Ip,
		GuestPort:   pfwd.Guest.Port,
	}
	if err := a.settings.NAT.Add(newPf); err != nil {
		a.logger.Debug("port forward settings failure", "error", err)
		return err
	}
	if err := a.settings.Save(); err != nil {
		a.logger.Debug("settings save failure", "error", err)
		return err
	}
	return nil
}
