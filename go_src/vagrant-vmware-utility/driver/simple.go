// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package driver

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	hclog "github.com/hashicorp/go-hclog"

	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

// Simple driver is used where the vnet lib
// is not public requiring full network service
// modifications (Workstation Linux)
type SimpleDriver struct {
	BaseDriver
}

type fileBackup struct {
	Path       string
	BackupPath string
}

func NewSimpleDriver(vmxPath *string, b *BaseDriver, logger hclog.Logger) (s *SimpleDriver, err error) {
	logger = logger.Named("simple")
	if b == nil {
		b, err = NewBaseDriver(vmxPath, "", logger)
		if err != nil {
			return nil, err
		}
	}
	s = &SimpleDriver{
		BaseDriver: *b}
	return
}

func (s *SimpleDriver) AddVmnet(vmnet *Vmnet) error {
	netF, err := s.LoadNetworkingFile()
	if err != nil {
		return err
	}
	var device *utility.Device
	if vmnet.Mask != "" {
		device = netF.CreateDevice(vmnet.Mask, vmnet.Subnet)
	} else {
		device = netF.CreateDevice()
	}
	device.Dhcp = vmnet.Dhcp == "yes"
	device.Nat = vmnet.Type == "nat"
	s.logger.Debug("vmnet create", "name", device.Name, "dhcp", device.Dhcp,
		"nat", device.Nat, "subnet", device.HostonlySubnet, "mask",
		device.HostonlyNetmask)
	vmnet.Name = device.Name
	return s.saveAndRestart(netF)
}

func (s *SimpleDriver) UpdateVmnet(vmnet *Vmnet) error {
	netF, err := s.LoadNetworkingFile()
	if err != nil {
		return err
	}
	device := netF.GetDeviceByName(vmnet.Name)
	if device == nil {
		return errors.New(fmt.Sprintf(
			"Device does not exist %s", vmnet.Name))
	}
	device.Dhcp = vmnet.Dhcp == "yes"
	device.Nat = vmnet.Type == "nat"
	device.HostonlyNetmask = vmnet.Mask
	device.HostonlySubnet = vmnet.Subnet
	s.logger.Debug("vmnet update", "name", device.Name, "dhcp", device.Dhcp,
		"nat", device.Nat, "subnet", device.HostonlySubnet, "mask",
		device.HostonlyNetmask)
	return s.saveAndRestart(netF)
}

func (s *SimpleDriver) DeleteVmnet(vmnet *Vmnet) error {
	netF, err := s.LoadNetworkingFile()
	if err != nil {
		return err
	}
	err = netF.RemoveDeviceByName(vmnet.Name)
	if err != nil {
		return err
	}
	return s.saveAndRestart(netF)
}

// Lookup reserved DHCP address for MAC
func (s *SimpleDriver) LookupDhcpAddress(device, mac string) (addr string, err error) {
	leases, err := utility.LoadDhcpLeaseFile(s.vmwarePaths.DhcpLeaseFile(device), s.logger)
	if err != nil {
		s.logger.Debug("dhcp leases file load failure", "error", err)
		return addr, err
	}
	paddr, err := leases.IpForMac(mac)
	if err == nil {
		return *paddr, err
	}
	netF, err := s.LoadNetworkingFile()
	if err != nil {
		return addr, err
	}
	slotNumber := strings.Replace(device, "vmnet", "", -1)
	slot, _ := strconv.Atoi(slotNumber)
	return netF.LookupDhcpReservation(slot, mac)
}

func (s *SimpleDriver) ReserveDhcpAddress(slot int, mac, ip string) error {
	netF, err := s.LoadNetworkingFile()
	if err != nil {
		return err
	}
	err = netF.AddDhcpReservation(slot, mac, ip)
	if err != nil {
		return err
	}
	return s.saveAndRestart(netF)
}

func (s *SimpleDriver) AddPortFwd(pfwds []*PortFwd) error {
	netF, err := s.LoadNetworkingFile()
	if err != nil {
		return err
	}
	for _, pfwd := range pfwds {
		description, err := s.validatePortFwdDescription(pfwd.Description)
		if err != nil {
			return err
		}
		newPf := &utility.PortFwd{
			Device:      strconv.Itoa(pfwd.SlotNumber),
			Protocol:    pfwd.Protocol,
			Description: description,
			HostPort:    pfwd.Port,
			GuestIp:     pfwd.Guest.Ip,
			GuestPort:   pfwd.Guest.Port,
		}
		if err := netF.AddPortFwd(newPf); err != nil {
			return err
		}
	}
	return s.saveAndRestart(netF)
}

func (s *SimpleDriver) DeletePortFwd(pfwds []*PortFwd) error {
	netF, err := s.LoadNetworkingFile()
	if err != nil {
		return err
	}
	for _, pfwd := range pfwds {
		newPf := &utility.PortFwd{
			Device:      strconv.Itoa(pfwd.SlotNumber),
			Protocol:    pfwd.Protocol,
			Description: pfwd.Description,
			HostPort:    pfwd.Port,
			GuestIp:     pfwd.Guest.Ip,
			GuestPort:   pfwd.Guest.Port,
		}
		err = s.clearNatConfPortFwd(
			fmt.Sprintf("vmnet%d", pfwd.SlotNumber), pfwd.Protocol, pfwd.Port)
		if err != nil {
			return err
		}
		if err := netF.RemovePortFwd(newPf); err != nil {
			return err
		}
	}
	return s.saveAndRestart(netF)
}

func (s *SimpleDriver) clearNatConfPortFwd(device, protocol string, iport int) error {
	port := strconv.Itoa(iport)
	natconf, err := s.Natfile(device)
	if err != nil {
		return err
	}
	sectionName := strings.ToLower("incoming" + protocol)
	section := natconf.GetSection(sectionName)
	if section == nil {
		s.logger.Debug("failed to locate section in nat.conf file", "section", sectionName)
		return errors.New(
			fmt.Sprintf("Invalid NAT section name: %s", sectionName))
	}
	for _, entry := range section.Entries {
		if entry.Match(port) {
			s.logger.Debug("removing forward from nat.conf", "section", sectionName, "port", port)
			if err := section.DeleteEntry(entry); err != nil {
				return err
			}
			if err := natconf.Save(); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

func (s *SimpleDriver) saveAndRestart(netF utility.NetworkingFile) error {
	path, err := netF.Save()
	if err != nil {
		return err
	}
	s.settings.NAT.Clear()
	if err := s.settings.NAT.MultiAdd(netF.GetPortFwds()); err != nil {
		return err
	}
	if err := s.settings.Save(); err != nil {
		return err
	}
	backups, err := s.backupDhcpLeases(netF)
	if err != nil {
		s.logger.Warn("failed to restore DHCP leases", "error", err)
	}
	if err := s.vmnet.Configure(path); err != nil {
		s.logger.Debug("vmnet configure failed", "error", err)
		return err
	}
	if err := s.vmnet.Stop(); err != nil {
		s.logger.Debug("vmnet service stop failed (non-fatal)", "error", err)
	}
	err = s.restoreDhcpLeases(backups)
	if err != nil {
		s.logger.Warn("failed to restore DHCP leases", "error", err)
	}
	if err := s.vmnet.Start(); err != nil {
		s.logger.Debug("vmnet service start failed", "error", err)
		return err
	}
	return nil
}

func (s *SimpleDriver) backupDhcpLeases(netF utility.NetworkingFile) (backups []*fileBackup, err error) {
	for _, dev := range netF.GetDevices() {
		if !dev.Dhcp {
			continue
		}
		leasePath := s.VmwarePaths().DhcpLeaseFile(dev.Name)
		if utility.FileExists(leasePath) {
			leaseFile, err := os.Open(leasePath)
			if err != nil {
				s.logger.Trace("failed to open lease file", "path", leasePath, "error", err)
				return nil, err
			}
			defer leaseFile.Close()
			s.logger.Trace("creating dhcp lease file backup", "device", dev.Name, "path", leasePath)
			tmpFile, err := ioutil.TempFile("", "vagrant-vmware")
			if err != nil {
				s.logger.Trace("failed to create backup file", "error", err)
				return nil, err
			}
			defer tmpFile.Close()
			_, err = io.Copy(tmpFile, leaseFile)
			if err != nil {
				s.logger.Trace("lease backup copy failed", "lease-path", leasePath, "backup-path", tmpFile.Name(),
					"error", err)
				return nil, err
			}
			backups = append(backups, &fileBackup{Path: leasePath, BackupPath: tmpFile.Name()})
		}
	}
	return backups, nil
}

func (s *SimpleDriver) restoreDhcpLeases(backups []*fileBackup) error {
	for _, backup := range backups {
		src, err := os.Open(backup.BackupPath)
		if err != nil {
			s.logger.Trace("failed to open backup file for lease restore", "path",
				backup.BackupPath, "error", err)
			return err
		}
		defer src.Close()
		defer os.Remove(src.Name())
		dst, err := os.OpenFile(backup.Path, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			s.logger.Trace("failed to open lease file for restore", "path",
				backup.Path, "error", err)
			return err
		}
		defer dst.Close()
		_, err = io.Copy(dst, src)
		if err != nil {
			s.logger.Trace("failed to write lease restore", "path", backup.Path,
				"error", err)
			return err
		}
	}
	return nil
}
