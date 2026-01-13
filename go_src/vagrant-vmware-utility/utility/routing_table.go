// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package utility

import (
	"net"

	hclog "github.com/hashicorp/go-hclog"
)

type NetworkInterface struct {
	name  string
	addrs []net.Addr
}

func (ni *NetworkInterface) Addrs() []net.Addr { return ni.addrs }
func (ni *NetworkInterface) Name() string      { return ni.name }

type InterfacesGetter func() ([]NetworkInterface, error)

type RoutingTable struct {
	Devices []*RoutingDevice

	interfaces InterfacesGetter
	logger     hclog.Logger
}

type RoutingDevice struct {
	Name    string
	Address net.IP
	Netmask net.IPMask
}

func (r *RoutingDevice) Match(network string) bool {
	_, pNet, err := net.ParseCIDR(network)
	if err != nil {
		return false
	}
	if !r.Address.Equal(pNet.IP) {
		return false
	}
	if r.Netmask.String() == pNet.Mask.String() {
		return true
	}
	return false
}

func LoadRoutingTable(igetter InterfacesGetter, logger hclog.Logger) (table *RoutingTable, err error) {
	if logger == nil {
		logger = hclog.New(&hclog.LoggerOptions{
			Output: hclog.DefaultOutput,
			Level:  hclog.Error,
			Name:   "vagrant-vmware-routing-table"})
	} else {
		logger = logger.Named("routing-table")
	}
	if igetter == nil {
		igetter = getLocalInterfaces
	}
	table = &RoutingTable{
		interfaces: igetter,
		logger:     logger,
	}
	err = table.Load()
	if err != nil {
		return nil, err
	}
	return table, nil
}

func (r *RoutingTable) Load() error {
	interfaces, err := r.interfaces()
	if err != nil {
		r.logger.Debug("load failure", "error", err)
		return err
	}
	for _, iface := range interfaces {
		rdev := r.deviceFromIpv4(iface.Addrs())
		if rdev != nil {
			rdev.Name = iface.Name()
			r.logger.Trace("discovered device", "name", rdev.Name, "address",
				rdev.Address, "mask", rdev.Netmask)
			r.Devices = append(r.Devices, rdev)
		} else {
			r.logger.Trace("no IPv4 information", "name", iface.Name)
		}
	}
	r.logger.Debug("loaded routing table", "count", len(r.Devices))
	return nil
}

func (r *RoutingTable) DeviceByName(name string) *RoutingDevice {
	for _, rDev := range r.Devices {
		if rDev.Name == name {
			return rDev
		}
	}
	return nil
}

func getLocalInterfaces() (nifs []NetworkInterface, err error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range interfaces {
		iaddrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		ni := NetworkInterface{
			name:  iface.Name,
			addrs: iaddrs,
		}
		nifs = append(nifs, ni)
	}
	return nifs, err
}

func (r *RoutingTable) deviceFromIpv4(addrs []net.Addr) *RoutingDevice {
	for _, addr := range addrs {
		_, pNet, err := net.ParseCIDR(addr.String())
		if err != nil {
			r.logger.Trace("address parse failure", "address", addr.String(), "error", err)
			continue
		}
		if pNet.IP.To4() == nil {
			r.logger.Trace("invalid IPv4 address", "address", addr.String())
			continue
		}
		return &RoutingDevice{
			Address: pNet.IP,
			Netmask: pNet.Mask}
	}
	return nil
}
