// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utility

import (
	"errors"
	"fmt"
	"strings"
)

type NetworkingFileMock struct {
	Path             string
	Devices          []*Device
	DhcpReservations []*DhcpReservation
	PortFwds         []*PortFwd
}

// Loads the networking file at the given path and parses
// the network adapters defined within the file
func LoadNetworkingFileMock(path string, devices []*Device, dhcp []*DhcpReservation, portfwds []*PortFwd) (*NetworkingFileMock, error) {
	nFile := &NetworkingFileMock{
		Path:             path,
		Devices:          devices,
		DhcpReservations: dhcp,
		PortFwds:         portfwds,
	}
	return nFile, nil
}

func (n *NetworkingFileMock) GetPortFwds() []*PortFwd {
	return n.PortFwds
}

func (n *NetworkingFileMock) GetPath() string {
	return n.Path
}

func (n *NetworkingFileMock) GetDevices() []*Device {
	return n.Devices
}

func (n *NetworkingFileMock) Merge(netF NetworkingFile) error {
	return nil
}

func (n *NetworkingFileMock) MergeFwds(fwds []*PortFwd) error {
	return nil
}

func (n *NetworkingFileMock) Save() (string, error) {
	return n.Path, nil
}

func (n *NetworkingFileMock) AddDhcpReservation(device int, mac, address string) error {
	return nil
}

func (n *NetworkingFileMock) LookupDhcpReservation(device int, mac string) (addr string, err error) {
	seekMac := strings.ToLower(mac)
	for _, res := range n.DhcpReservations {
		if strings.ToLower(res.Mac) == seekMac && res.Device == device {
			return res.Address, err
		}
	}
	return addr, errors.New(fmt.Sprintf("No entry found for MAC %s", mac))
}

// Add a new port forward
func (n *NetworkingFileMock) AddPortFwd(fwd *PortFwd) error {
	return nil
}

// Remove a port forward
func (n *NetworkingFileMock) RemovePortFwd(fwd *PortFwd) error {
	return nil
}

// Remove a defined adapter with given name
func (n *NetworkingFileMock) RemoveDeviceByName(devName string) error {
	return nil
}

// Remove a defined adapter with given slot number
func (n *NetworkingFileMock) RemoveDeviceBySlot(slotNumber int) error {
	return nil
}

// Find defined adapter with given name
func (n *NetworkingFileMock) GetDeviceByName(devName string) (foundDevice *Device) {
	for _, adapter := range n.Devices {
		if adapter.Name == devName {
			foundDevice = adapter
			break
		}
	}
	return foundDevice
}

// Find defined adapter with given slot number
func (n *NetworkingFileMock) GetDeviceBySlot(slotNumber int) (foundDevice *Device) {
	for _, adapter := range n.Devices {
		if adapter.Number == slotNumber {
			foundDevice = adapter
			break
		}
	}
	return foundDevice
}

func (n *NetworkingFileMock) CreateDevice(params ...string) *Device {
	var netmask *string
	var subnetIp *string
	if len(params) > 1 {
		netmask = &params[0]
		subnetIp = &params[1]
	}
	slot := 1
	adapter := &Device{
		Name:           fmt.Sprintf("vmnet%d", slot),
		Number:         slot,
		Dhcp:           true,
		VirtualAdapter: true,
	}
	if netmask != nil && subnetIp != nil {
		adapter.HostonlyNetmask = *netmask
		adapter.HostonlySubnet = *subnetIp
	}
	n.Devices = append(n.Devices, adapter)
	return adapter
}

func (n *NetworkingFileMock) Load() error {
	return nil
}

func (n *NetworkingFileMock) PortfwdExists(fwd *PortFwd) bool {
	for _, eFwd := range n.PortFwds {
		if eFwd == fwd {
			return true
		}
	}
	return false
}

func (n *NetworkingFileMock) HostPortFwd(port int, protocol string) *PortFwd {
	for _, fwd := range n.PortFwds {
		if fwd.HostPort == port && fwd.Protocol == protocol {
			return fwd
		}
	}
	return nil
}
