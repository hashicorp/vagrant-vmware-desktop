// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utility

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"
	"text/template"
)

const VNET_ADAPTER = `
answer VNET_{{.Id}}_DHCP {{.Dhcp}}
answer VNET_{{.Id}}_DHCP_CFG_HASH {{.CfgHash}}
answer VNET_{{.Id}}_HOSTONLY_NETMASK {{.HostonlyNetmask}}
answer VNET_{{.Id}}_HOSTONLY_SUBNET {{.HostonlySubnet}}
answer VNET_{{.Id}}_NAT {{.Nat}}
answer VNET_{{.Id}}_VIRTUAL_ADAPTER {{.VirtAdapter}}
`
const NAT_PORTFWD = `{{.Type}}_nat_portfwd {{.Device}} {{.Protocol}} {{.HostPort}} {{.GuestIp}} {{.GuestPort}}
`
const DHCP_RESERVATION = `add_dhcp_mac_to_ip {{.Device}} {{.Mac}} {{.Address}}
`

type Adapter struct {
	Id, Dhcp, CfgHash, HostonlyNetmask, HostonlySubnet, Nat, VirtAdapter string
}
type NatPortFwd struct {
	Type, Device, Protocol, HostPort, GuestIp, GuestPort string
}
type NatDhcpReservation struct {
	Device, Mac, Address string
}

func TestNetworkingLoadFailure(t *testing.T) {
	_, err := LoadNetworkingFile("/unknown/path/to/file", nil)
	if err == nil {
		t.Errorf("Network loading of unknown file did not fail")
	}
}

func TestNetorkingLoadSuccess(t *testing.T) {
	path := createValidNetworkingFile(1)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		t.Errorf("Failed to load networking file: %s", err)
	}
	if nFile.Path != path {
		t.Errorf("Invalid networking path %s != %s", path, nFile.Path)
	}
}

func TestLoadMultipleDevices(t *testing.T) {
	path := createValidNetworkingFile(5)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	if len(nFile.Devices) != 5 {
		t.Errorf("Wrong number of devices %d != %d", 5, len(nFile.Devices))
	}
}

func TestAddBasicDevice(t *testing.T) {
	path := createValidNetworkingFile(1)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	newDev := nFile.CreateDevice()
	if newDev == nil {
		t.Errorf("Failed to create new device")
	}
	if newDev.Name != "vmnet2" {
		t.Errorf("Unexpected device name vmnet2 != %s", newDev.Name)
	}
	if newDev.Number != 2 {
		t.Errorf("Unexpected device slot 2 != %d", newDev.Number)
	}
	if newDev.HostonlyNetmask != "" {
		t.Errorf("HostonlyNetmask should be blank - %s", newDev.HostonlyNetmask)
	}
	if newDev.HostonlySubnet != "" {
		t.Errorf("HostonlySubnet should be blank - %s", newDev.HostonlySubnet)
	}
	if !newDev.VirtualAdapter {
		t.Errorf("Device should be marked as virtual adapter")
	}
}

func TestAddDeviceWithNetmask(t *testing.T) {
	path := createValidNetworkingFile(1)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	mask := "255.255.255.0"
	newDev := nFile.CreateDevice(mask)
	if newDev.HostonlyNetmask != "" {
		t.Errorf("Netmask should not be set without subnet IP - %s", newDev.HostonlyNetmask)
	}
}

func TestAddDeviceWithNetmaskSubnetIp(t *testing.T) {
	path := createValidNetworkingFile(1)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	mask := "255.255.255.0"
	subIp := "172.32.3.0"
	newDev := nFile.CreateDevice(mask, subIp)
	if newDev.HostonlyNetmask != mask {
		t.Errorf("Unexpected HostonlyNetmask value %s != %s", mask, newDev.HostonlyNetmask)
	}
	if newDev.HostonlySubnet != subIp {
		t.Errorf("Unexpected HostonlySubnet value %s != %s", subIp, newDev.HostonlySubnet)
	}
}

func TestRemoveDevice(t *testing.T) {
	path := createValidNetworkingFile(2)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	err = nFile.RemoveDeviceBySlot(1)
	if err != nil {
		t.Errorf("Error removing device: %s", err)
	}
	if len(nFile.Devices) != 1 {
		t.Errorf("Invalid number of devices 1 != %d", len(nFile.Devices))
	}
	if nFile.Devices[0].Number != 2 {
		t.Errorf("Invalid device slot number 2 != %d", nFile.Devices[0].Number)
	}
}

func TestGetDevice(t *testing.T) {
	path := createValidNetworkingFile(2)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	device := nFile.GetDeviceBySlot(1)
	if device == nil {
		t.Errorf("Failed get defined device")
	}
	if device.Number != 1 {
		t.Errorf("Invalid device returned. Expect slot number 1, not %d", device.Number)
	}
	device = nFile.GetDeviceBySlot(2)
	if device == nil {
		t.Errorf("Failed get defined device")
	}
	if device.Number != 2 {
		t.Errorf("Invalid device returned. Expect slot number 2, not %d", device.Number)
	}
}

func TestFailedGetDevice(t *testing.T) {
	path := createValidNetworkingFile(2)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	device := nFile.GetDeviceBySlot(5)
	if device != nil {
		t.Errorf("Invalid device returned. Slot number 5 requested, got %d", device.Number)
	}
}

func TestReloadFile(t *testing.T) {
	path := createValidNetworkingFile(2)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	nFile.CreateDevice()
	err = nFile.Load()
	if err != nil {
		t.Errorf("Failed to reload file: %s", err)
	}
	if len(nFile.Devices) != 2 {
		t.Errorf("Unexpected number of devices 2 != %d", len(nFile.Devices))
	}
}

func TestBasicPortFwd(t *testing.T) {
	path := createValidNetworkingFile(1)
	defer os.Remove(path)
	err := addPortFwds(generatePortFwds(2), path)
	if err != nil {
		panic(fmt.Sprintf("Failed to add portfwds to file: %s", err))
	}
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	if len(nFile.PortFwds) != 2 {
		t.Errorf("Unexpected number of portfwd rules 2 != %d", len(nFile.PortFwds))
	}
}

func TestAddPortFwd(t *testing.T) {
	path := createValidNetworkingFile(1)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	fwd := &PortFwd{
		Device:    "8",
		Protocol:  "tcp",
		HostPort:  3000,
		GuestPort: 3000,
		GuestIp:   "127.0.1.3",
	}
	nFile.AddPortFwd(fwd)
	if !fwd.Enable {
		t.Errorf("Port forward should be enabled when adding")
	}
}

func TestMergePortFwd(t *testing.T) {
	path := createValidNetworkingFile(1)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	fwd := &PortFwd{
		Device:    "8",
		Protocol:  "tcp",
		HostPort:  3000,
		GuestPort: 3000,
		GuestIp:   "127.0.1.3"}
	if err := nFile.AddPortFwd(fwd); err != nil {
		panic(fmt.Sprintf("Failed to add port forward: %s", err))
	}
	newFwd := &PortFwd{
		Device:      "8",
		Protocol:    "tcp",
		HostPort:    3000,
		GuestPort:   3000,
		GuestIp:     "127.0.1.3",
		Description: "Custom Description",
		Enable:      true}
	if err := nFile.MergeFwds([]*PortFwd{newFwd}); err != nil {
		t.Errorf("Failed to merge port forward: %s", err)
	}
	if nFile.PortFwds[0].Description != "Custom Description" {
		t.Errorf("Merge port forward failed. Description not set.")
	}
}

func TestMergePortFwdNotEnabled(t *testing.T) {
	path := createValidNetworkingFile(1)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	fwd := &PortFwd{
		Device:    "8",
		Protocol:  "tcp",
		HostPort:  3000,
		GuestPort: 3000,
		GuestIp:   "127.0.1.3"}
	if err := nFile.AddPortFwd(fwd); err != nil {
		panic(fmt.Sprintf("Failed to add port forward: %s", err))
	}
	newFwd := &PortFwd{
		Device:      "8",
		Protocol:    "tcp",
		HostPort:    3000,
		GuestPort:   3000,
		GuestIp:     "127.0.1.3",
		Description: "Custom Description",
		Enable:      false}
	if err := nFile.MergeFwds([]*PortFwd{newFwd}); err != nil {
		t.Errorf("Failed to merge port forward: %s", err)
	}
	if nFile.PortFwds[0].Description == "Custom Description" {
		t.Errorf("Merge port forward failed. Description should not be set.")
	}
}

func TestRemovePortFwd(t *testing.T) {
	path := createValidNetworkingFile(1)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	fwd := &PortFwd{
		Device:    "8",
		Protocol:  "tcp",
		HostPort:  3000,
		GuestPort: 3000,
		GuestIp:   "127.0.1.3",
	}
	nFile.RemovePortFwd(fwd)
	if fwd.Enable {
		t.Errorf("Port forward should not be enabled when removing")
	}
}

func TestReusePortFwd(t *testing.T) {
	path := createValidNetworkingFile(1)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	fwd := &PortFwd{
		Device:    "8",
		Protocol:  "tcp",
		HostPort:  3000,
		GuestPort: 3000,
		GuestIp:   "127.0.1.3",
	}
	err = nFile.RemovePortFwd(fwd)
	if err != nil {
		panic(fmt.Sprintf("Failed to load initial port forward: %s", err))
	}
	err = nFile.AddPortFwd(fwd)
	if err == nil {
		t.Errorf("Port forward should not be allowed to be reused")
	}
}

func TestNewDeviceSave(t *testing.T) {
	path := createValidNetworkingFile(2)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	nFile.CreateDevice()
	_, err = nFile.Save()
	if err != nil {
		t.Errorf("Failed to save file: %s", err)
	}
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("Failed to read file: %s", err))
	}
	content := string(buf)
	if !strings.Contains(content, "VNET_2") {
		t.Errorf("New device not found in saved file contents\n%s", content)
	}
}

func TestAddPortFwdSave(t *testing.T) {
	path := createValidNetworkingFile(1)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	fwd := &PortFwd{
		Device:    "8",
		Protocol:  "tcp",
		HostPort:  3000,
		GuestPort: 3000,
		GuestIp:   "127.0.1.3",
	}
	nFile.AddPortFwd(fwd)
	_, err = nFile.Save()
	if err != nil {
		t.Errorf("Failed to save file: %s", err)
	}
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("Failed to read file: %s", err))
	}
	content := string(buf)
	matcher := fmt.Sprintf("%s_nat_portfwd %s %s %d %s %d", "add",
		fwd.Device, fwd.Protocol, fwd.HostPort, fwd.GuestIp, fwd.GuestPort)
	if !strings.Contains(content, matcher) {
		t.Errorf("Port forward not found in saved file contents\n%s", content)
	}
}

func TestRemovePortFwdSave(t *testing.T) {
	path := createValidNetworkingFile(1)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	fwd := &PortFwd{
		Device:    "8",
		Protocol:  "tcp",
		HostPort:  3000,
		GuestPort: 3000,
		GuestIp:   "127.0.1.3",
	}
	nFile.RemovePortFwd(fwd)
	_, err = nFile.Save()
	if err != nil {
		t.Errorf("Failed to save file: %s", err)
	}
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("Failed to read file: %s", err))
	}
	content := string(buf)
	matcher := fmt.Sprintf("%s_nat_portfwd %s %s %d %s %d", "remove",
		fwd.Device, fwd.Protocol, fwd.HostPort, fwd.GuestIp, fwd.GuestPort)
	if !strings.Contains(content, matcher) {
		t.Errorf("Port forward not found in saved file contents\n%s", content)
	}
}

func TestPermissionsSave(t *testing.T) {
	path := createValidNetworkingFile(1)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	_, err = nFile.Save()
	if err != nil {
		t.Errorf("Failed to save file: %s", err)
	}
	checkFile, err := os.Open(nFile.Path)
	if err != nil {
		t.Errorf("Failed to open file: %s", err)
	}
	defer checkFile.Close()
	cInfo, err := checkFile.Stat()
	if err != nil {
		t.Errorf("Failed to stat file: %s", err)
	}
	if cInfo.Mode() != 0644 {
		t.Errorf("Invalid file permissions. Expected 644, got %s", strconv.FormatInt(int64(cInfo.Mode()), 8))
	}
}

func TestAddDhcpReservation(t *testing.T) {
	path := createValidNetworkingFile(1)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	res := generateDhcpReservations(1)
	err = nFile.AddDhcpReservation(8, res[0].Mac, res[0].Address)
	if err != nil {
		t.Errorf("Failed to add dhcp reservation: %s", err)
	}
	_, err = nFile.Save()
	if err != nil {
		t.Errorf("Failed to save file: %s", err)
	}
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("Failed to read file: %s", err))
	}
	content := string(buf)
	matcher := fmt.Sprintf("add_dhcp_mac_to_ip 8 %s %s", res[0].Mac, res[0].Address)
	if !strings.Contains(content, matcher) {
		t.Errorf("DHCP reservation not found in saved contents\n%s", content)
	}
}

func TestAddExistingDhcpReservation(t *testing.T) {
	path := createValidNetworkingFile(1)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	res := generateDhcpReservations(2)
	err = nFile.AddDhcpReservation(8, res[0].Mac, res[0].Address)
	if err != nil {
		t.Errorf("Failed to add dhcp reservation: %s", err)
	}
	err = nFile.AddDhcpReservation(8, res[0].Mac, res[1].Address)
	if err != nil {
		t.Errorf("Failed to add dhcp reservation: %s", err)
	}
	if len(nFile.DhcpReservations) != 1 {
		t.Errorf("Unexpected number of dhcp reservations 1 != %d",
			len(nFile.DhcpReservations))
	}
}

func TestAddMultipleDhcpReservation(t *testing.T) {
	path := createValidNetworkingFile(1)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	reserves := generateDhcpReservations(5)
	for _, res := range reserves {
		err = nFile.AddDhcpReservation(8, res.Mac, res.Address)
		if err != nil {
			t.Errorf("Failed to add dhcp reservation: %s", err)
		}
	}
	actualLen := len(nFile.DhcpReservations)
	if actualLen != 5 {
		t.Errorf("Unexpected number of dhcp reservations 5 != %d",
			actualLen)
	}
}

func TestNoneLookupDhcpReservation(t *testing.T) {
	path := createValidNetworkingFile(1)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	res := generateDhcpReservations(1)
	_, err = nFile.LookupDhcpReservation(8, res[0].Mac)
	if err == nil {
		t.Errorf("Expected error on reservation lookup but received none.")
	}
}

func TestNoMatchLookupDhcpReservation(t *testing.T) {
	path := createValidNetworkingFile(1)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	reserves := generateDhcpReservations(5)
	for _, res := range reserves {
		err = nFile.AddDhcpReservation(8, res.Mac, res.Address)
		if err != nil {
			t.Errorf("Failed to add dhcp reservation: %s", err)
		}
	}
	_, err = nFile.LookupDhcpReservation(8, "00:00:00:00:00:00")
	if err == nil {
		t.Errorf("Expected error on reservation lookup but received none.")
	}
}

func TestLookupDhcpReservation(t *testing.T) {
	path := createValidNetworkingFile(1)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	reserves := generateDhcpReservations(5)
	for _, res := range reserves {
		err = nFile.AddDhcpReservation(8, res.Mac, res.Address)
		if err != nil {
			t.Errorf("Failed to add dhcp reservation: %s", err)
		}
	}
	addr, err := nFile.LookupDhcpReservation(8, reserves[3].Mac)
	if err != nil {
		t.Errorf("Failed to lookup reservation: %s", err)
		return
	}
	if reserves[3].Address != addr {
		t.Errorf("Invalid address returned for dhcp reservation lookup %s != %s",
			reserves[3].Address, addr)
	}
}

func TestCasingLookupDhcpReservation(t *testing.T) {
	path := createValidNetworkingFile(1)
	defer os.Remove(path)
	nFile, err := LoadNetworkingFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load networking file: %s", err))
	}
	reserves := generateDhcpReservations(5)
	for _, res := range reserves {
		err = nFile.AddDhcpReservation(8, res.Mac, res.Address)
		if err != nil {
			t.Errorf("Failed to add dhcp reservation: %s", err)
		}
	}
	addr, err := nFile.LookupDhcpReservation(8, strings.ToUpper(reserves[3].Mac))
	if err != nil {
		t.Errorf("Failed to lookup reservation: %s", err)
		return
	}
	if reserves[3].Address != addr {
		t.Errorf("Invalid address returned for dhcp reservation lookup %s != %s",
			reserves[3].Address, addr)
		return
	}
	addr, err = nFile.LookupDhcpReservation(8, strings.ToLower(reserves[3].Mac))
	if err != nil {
		t.Errorf("Failed to lookup reservation: %s", err)
		return
	}
	if reserves[3].Address != addr {
		t.Errorf("Invalid address returned for dhcp reservation lookup %s != %s",
			reserves[3].Address, addr)
	}
}

func createValidNetworkingFile(numAdapters int) string {
	netFile, err := ioutil.TempFile("", "networking")
	if err != nil {
		panic(fmt.Sprintf("Failed to create test networking file: %s", err))
	}
	defer netFile.Close()
	err = generateVmwareNetwork(generateAdapters(numAdapters), netFile)
	if err != nil {
		panic(fmt.Sprintf(
			"Failed to write test networking file: %s", err))
	}
	return netFile.Name()
}

func generateAdapters(numAdapters int) (adapters []*Adapter) {
	// Start `i` at 1 to simulate vmware setup (0 is predefined and not within file)
	for i := 1; i <= numAdapters; i++ {
		newAdapter := &Adapter{
			Id:          strconv.Itoa(i),
			Dhcp:        "yes",
			CfgHash:     "0000000000000",
			VirtAdapter: "yes",
		}
		if i == 0 || i%2 == 0 {
			newAdapter.HostonlyNetmask = "255.255.255.0"
			newAdapter.HostonlySubnet = fmt.Sprintf("127.0.%d.0", i)
			newAdapter.Nat = "yes"
		}
		adapters = append(adapters, newAdapter)
	}
	return adapters
}

func generateVmwareNetwork(adapters []*Adapter, out *os.File) error {
	t := template.Must(template.New("networking").Parse(VNET_ADAPTER))
	for _, adapter := range adapters {
		err := t.Execute(out, adapter)
		if err != nil {
			return err
		}
	}
	return nil
}

func generatePortFwds(numFwds int) (portfwds []*NatPortFwd) {
	for i := 0; i < numFwds; i++ {
		newPortfwd := &NatPortFwd{
			Device:    strconv.Itoa(i + 1),
			Protocol:  "tcp",
			HostPort:  strconv.Itoa(3000 + i),
			GuestPort: strconv.Itoa(4000 + i),
			GuestIp:   fmt.Sprintf("172.33.1.%d", i+4),
		}
		if i == 0 || i%2 == 0 {
			newPortfwd.Type = "add"
		} else {
			newPortfwd.Type = "remove"
		}
		portfwds = append(portfwds, newPortfwd)
	}
	return portfwds
}

func generateDhcpReservations(numRes int) (reservations []*NatDhcpReservation) {
	for i := 0; i < numRes; i++ {
		buf := make([]byte, 6)
		_, err := rand.Read(buf)
		if err != nil {
			panic(fmt.Sprintf("Random generation failed: %s", err))
		}
		newRes := &NatDhcpReservation{
			Address: fmt.Sprintf("127.0.%d.%d", numRes, i),
			Device:  "8",
			Mac: fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
				buf[0], buf[1], buf[2], buf[3], buf[4], buf[5]),
		}
		reservations = append(reservations, newRes)
	}
	return reservations
}

func addPortFwds(fwds []*NatPortFwd, path string) error {
	out, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("Failed to write NAT portfwds: %s", err))
	}
	t := template.Must(template.New("portfwd").Parse(NAT_PORTFWD))
	for _, fwd := range fwds {
		err := t.Execute(out, fwd)
		if err != nil {
			return err
		}
	}
	return nil
}
