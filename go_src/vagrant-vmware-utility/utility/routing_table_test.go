// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utility

import (
	"fmt"
	"net"
	"testing"
)

type MockAddr struct {
	addr string
}

func (ma *MockAddr) String() string  { return ma.addr }
func (ma *MockAddr) Network() string { return "mock" }

func TestRoutingTableLoad(t *testing.T) {
	_, err := LoadRoutingTable(nil, defaultUtilityLogger())
	if err != nil {
		t.Errorf("Failed to load system routing table: %s", err)
	}
}

func TestSingleInterface(t *testing.T) {
	rt := &RoutingTable{
		interfaces: func() ([]NetworkInterface, error) {
			return generateFakeInterfaces(1), nil
		},
		logger: defaultUtilityLogger()}
	err := rt.Load()
	if err != nil {
		panic(fmt.Sprintf(
			"Failed to load interfaces: %s", err))
	}
	if len(rt.Devices) != 1 {
		t.Errorf("Unexpected number of devices loaded 1 != %d", len(rt.Devices))
	}
}

func TestMultipleInterfaces(t *testing.T) {
	rt := &RoutingTable{
		interfaces: func() ([]NetworkInterface, error) {
			return generateFakeInterfaces(4), nil
		},
		logger: defaultUtilityLogger()}
	err := rt.Load()
	if err != nil {
		panic(fmt.Sprintf(
			"Failed to load interfaces: %s", err))
	}
	if len(rt.Devices) != 4 {
		t.Errorf("Unexpected number of devices loaded 4 != %d", len(rt.Devices))
	}
}

func TestInvalidInterface(t *testing.T) {
	rt := &RoutingTable{
		interfaces: func() ([]NetworkInterface, error) {
			ifaces := generateFakeInterfaces(4)
			ifaces = append(ifaces, NetworkInterface{
				name: "invalid",
				addrs: []net.Addr{
					&MockAddr{addr: "mock failure"}}})
			return ifaces, nil
		},
		logger: defaultUtilityLogger()}
	err := rt.Load()
	if err != nil {
		panic(fmt.Sprintf(
			"Failed to load interfaces: %s", err))
	}
	if len(rt.Devices) != 4 {
		t.Errorf("Unexpected number of devices loaded 4 != %d", len(rt.Devices))
	}
	for _, dev := range rt.Devices {
		if dev.Name == "invalid" {
			t.Errorf("Device named 'invalid' should not be included.")
		}
	}
}

func TestMultipleAddrsOnInterface(t *testing.T) {
	rt := &RoutingTable{
		interfaces: func() ([]NetworkInterface, error) {
			ifaces := generateFakeInterfaces(4)
			ifaces = append(ifaces, NetworkInterface{
				name: "multi",
				addrs: []net.Addr{
					&MockAddr{addr: "non-address-value"},
					&MockAddr{addr: "172.12.33.0/32"}}})
			ifaces = append(ifaces, NetworkInterface{
				name: "invalid",
				addrs: []net.Addr{
					&MockAddr{addr: "mock failure"},
					&MockAddr{addr: "172.12.33.3"}}})
			return ifaces, nil
		},
		logger: defaultUtilityLogger()}
	err := rt.Load()
	if err != nil {
		panic(fmt.Sprintf(
			"Failed to load interfaces: %s", err))
	}
	if len(rt.Devices) != 5 {
		t.Errorf("Unexpected number of devices loaded 5 != %d", len(rt.Devices))
	}
	for _, dev := range rt.Devices {
		if dev.Name == "invalid" {
			t.Errorf("Device named 'invalid' should not be included.")
		}
	}
}

func TestGetDeviceByName(t *testing.T) {
	rt := &RoutingTable{
		interfaces: func() ([]NetworkInterface, error) {
			return generateFakeInterfaces(4), nil
		},
		logger: defaultUtilityLogger()}
	err := rt.Load()
	if err != nil {
		panic(fmt.Sprintf(
			"Failed to load interfaces: %s", err))
	}
	rd := rt.DeviceByName("dev1")
	if rd == nil {
		t.Errorf("Failed to get valid device by name 'dev1'")
	}
}

func TestRoutingDeviceMatch(t *testing.T) {
	rt := &RoutingTable{
		interfaces: func() ([]NetworkInterface, error) {
			return generateFakeInterfaces(1), nil
		},
		logger: defaultUtilityLogger()}
	err := rt.Load()
	if err != nil {
		panic(fmt.Sprintf(
			"Failed to load interfaces: %s", err))
	}
	rd := rt.DeviceByName("dev0")
	if rd == nil {
		panic("Failed to get valid device by name 'dev0'")
	}
	if !rd.Match("192.168.0.0/24") {
		t.Errorf("Device failed to match valid CIDR")
	}
}

func generateFakeInterfaces(num int) (nifs []NetworkInterface) {
	for i := 0; i < num; i++ {
		nif := NetworkInterface{
			name: fmt.Sprintf("dev%d", i),
			addrs: []net.Addr{
				&MockAddr{
					addr: fmt.Sprintf("192.168.%d.0/24", i)}}}
		nifs = append(nifs, nif)
	}
	return nifs
}
