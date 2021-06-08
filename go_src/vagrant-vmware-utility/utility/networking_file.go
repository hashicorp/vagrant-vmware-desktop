package utility

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	hclog "github.com/hashicorp/go-hclog"
)

const ADAPTER_PATTERN = `(?i)^answer\s+VNET_(?P<slot_number>\d+)_(?P<key>[a-z0-9_]+)\s+(?P<value>.+?)$`
const DHCP_RES_PATTERN = `(?i)^add_dhcp_mac_to_ip (?P<device>\d+)\s+(?P<mac>[^\s]+)\s+(?P<address>.+?)$`
const PORTFWD_PATTERN = `(?i)^(?P<type>add|remove)_nat_portfwd\s+(?P<device>\d+)\s+(?P<proto>udp|tcp)\s+(?P<host_port>\d+)\s+(?P<guest_ip>\S+)\s+(?P<guest_port>\d+)(?P<description>.*)$`

type NetworkingFile interface {
	AddDhcpReservation(device int, mac, address string) error
	AddPortFwd(fwd *PortFwd) error
	CreateDevice(params ...string) *Device
	GetDeviceByName(devName string) (foundDevice *Device)
	GetDeviceBySlot(slotNumber int) (foundDevice *Device)
	GetPath() string
	GetPortFwds() []*PortFwd
	GetDevices() []*Device
	HostPortFwd(port int, protocol string) *PortFwd
	Load() error
	LookupDhcpReservation(device int, mac string) (addr string, err error)
	Merge(netF NetworkingFile) error
	MergeFwds(fwds []*PortFwd) error
	PortfwdExists(fwd *PortFwd) bool
	RemovePortFwd(fwd *PortFwd) error
	RemoveDeviceByName(devName string) error
	RemoveDeviceBySlot(slotNumber int) error
	Save() (string, error)
}

type VMWareNetworkingFile struct {
	Path             string
	Devices          []*Device
	DhcpReservations []*DhcpReservation
	PortFwds         []*PortFwd
	logger           hclog.Logger
}

type PortFwd struct {
	Enable      bool   `json:"enable"`
	Device      string `json:"device"`
	Protocol    string `json:"protocol"`
	HostPort    int    `json:"hostport"`
	GuestIp     string `json:"guestip"`
	GuestPort   int    `json:"guestport"`
	Description string `json:"description"`
}

type DhcpReservation struct {
	Mac     string
	Address string
	Device  int
}

type Device struct {
	Name            string
	Number          int
	Dhcp            bool
	Nat             bool
	HostonlyNetmask string
	HostonlySubnet  string
	VirtualAdapter  bool
}

// Loads the networking file at the given path and parses
// the network adapters defined within the file
func LoadNetworkingFile(path string, logger hclog.Logger) (*VMWareNetworkingFile, error) {
	if logger == nil {
		logger = hclog.New(&hclog.LoggerOptions{
			Output: hclog.DefaultOutput,
			Level:  hclog.Error,
			Name:   "vagrant-vmware-networking-file"})
	} else {
		logger = logger.Named("networking-file")
	}
	nFile := &VMWareNetworkingFile{
		Path:   path,
		logger: logger,
	}
	err := nFile.Load()
	if err != nil {
		return nil, err
	}
	return nFile, nil
}

func (n *VMWareNetworkingFile) GetPortFwds() []*PortFwd {
	return n.PortFwds
}

func (n *VMWareNetworkingFile) GetPath() string {
	return n.Path
}

func (n *VMWareNetworkingFile) GetDevices() []*Device {
	return n.Devices
}

// This merges a networking file into the current networking file. Useful
// in places like Linux with Workstation where we lose all metadata within
// the actual networking file when settings are updated.
// NOTE: This only merges in port forward directives
func (n *VMWareNetworkingFile) Merge(netF NetworkingFile) error {
	n.logger.Trace("merging networking file", "origin path", n.Path, "remote path", netF.GetPath())
	for _, sFwd := range n.PortFwds {
		if sFwd.Enable {
			// Find a match
			for _, rFwd := range netF.GetPortFwds() {
				if sFwd.HostPort == rFwd.HostPort && sFwd.Protocol == rFwd.Protocol {
					n.logger.Trace("merging port forward", "source", sFwd, "merge", rFwd)
					sFwd.Description = rFwd.Description
					sFwd.Device = rFwd.Device
					sFwd.Protocol = rFwd.Protocol
					sFwd.GuestIp = rFwd.GuestIp
					sFwd.GuestPort = rFwd.GuestPort
				}
			}
		}
	}
	return nil
}

func (n *VMWareNetworkingFile) MergeFwds(fwds []*PortFwd) error {
	for _, nFwd := range fwds {
		for _, eFwd := range n.PortFwds {
			if nFwd.HostPort == eFwd.HostPort &&
				nFwd.Protocol == eFwd.Protocol &&
				nFwd.Device == eFwd.Device &&
				nFwd.GuestIp == eFwd.GuestIp &&
				nFwd.GuestPort == eFwd.GuestPort &&
				nFwd.Enable {
				n.logger.Trace("direct port forward merge", "source", eFwd, "merge", nFwd)
				eFwd.Description = nFwd.Description
			}
		}
	}
	return nil
}

// Saves the current configuration to the networking file at defined path
func (n *VMWareNetworkingFile) Save() (string, error) {
	n.logger.Debug("writing new file", "path", n.Path)
	tmpFile, err := ioutil.TempFile(path.Dir(n.Path), "vagrant-vmware-networking")
	if err != nil {
		n.logger.Debug("write failure", "path", n.Path, "error", err)
		return n.Path, err
	}
	defer tmpFile.Close()
	err = tmpFile.Chmod(0644)
	if err != nil {
		n.logger.Debug("file permission failure", "path", tmpFile.Name(), "error", err)
		return n.Path, err
	}
	_, err = tmpFile.WriteString("VERSION=1,0\n")
	if err != nil {
		n.logger.Debug("write failure", "path", n.Path, "error", err)
		return n.Path, err
	}
	// Write adapters
	for _, adapter := range n.Devices {
		aInfo := make(map[string]string)
		if adapter.Dhcp {
			aInfo["DHCP"] = "yes"
		} else {
			aInfo["DHCP"] = "no"
		}
		if adapter.Nat {
			aInfo["NAT"] = "yes"
		} else {
			aInfo["NAT"] = "no"
		}
		if adapter.HostonlyNetmask != "" {
			aInfo["HOSTONLY_NETMASK"] = adapter.HostonlyNetmask
		}
		if adapter.HostonlySubnet != "" {
			aInfo["HOSTONLY_SUBNET"] = adapter.HostonlySubnet
		}
		if adapter.VirtualAdapter {
			aInfo["VIRTUAL_ADAPTER"] = "yes"
		} else {
			aInfo["VIRTUAL_ADAPTER"] = "no"
		}
		for key, val := range aInfo {
			_, err := tmpFile.WriteString(fmt.Sprintf(
				"answer VNET_%d_%s %s\n", adapter.Number, key, val))
			if err != nil {
				tmpFile.Close()
				n.logger.Debug("write failure", "path", n.Path, "error", err)
				return n.Path, err
			}
		}
	}
	// Write portfwds
	for _, portfwd := range n.PortFwds {
		action := "remove"
		if portfwd.Enable {
			action = "add"
		}
		_, err := tmpFile.WriteString(fmt.Sprintf(
			"%s_nat_portfwd %s %s %d %s %d %s\n",
			action, portfwd.Device, portfwd.Protocol,
			portfwd.HostPort, portfwd.GuestIp, portfwd.GuestPort,
			portfwd.Description))
		if err != nil {
			n.logger.Debug("write failure", "path", n.Path, "error", err)
			return n.Path, err
		}
	}
	// Write DHCP reservations
	for _, res := range n.DhcpReservations {
		_, err := tmpFile.WriteString(fmt.Sprintf(
			"add_dhcp_mac_to_ip %d %s %s",
			res.Device, res.Mac, res.Address))
		if err != nil {
			n.logger.Debug("write failure", "path", n.Path, "error", err)
			return n.Path, err
		}
	}
	tmpFile.Close()

	err = os.Rename(tmpFile.Name(), n.Path)
	if err != nil {
		n.logger.Debug("write failure", "path", n.Path, "error", err)
		return n.Path, err
	}
	n.logger.Debug("write complete", "path", n.Path)
	return n.Path, nil
}

// Add a new DHCP reservation
func (n *VMWareNetworkingFile) AddDhcpReservation(device int, mac, address string) error {
	idx := -1
	for i, res := range n.DhcpReservations {
		if strings.ToLower(res.Mac) == strings.ToLower(mac) && res.Device == device {
			idx = i
			break
		}
	}
	newRes := &DhcpReservation{
		Address: address,
		Device:  device,
		Mac:     mac}
	if idx < 0 {
		n.DhcpReservations = append(n.DhcpReservations, newRes)
	} else {
		n.DhcpReservations[idx] = newRes
	}
	return nil
}

// Lookup a DHCP reservation
func (n *VMWareNetworkingFile) LookupDhcpReservation(device int, mac string) (addr string, err error) {
	seekMac := strings.ToLower(mac)
	for _, res := range n.DhcpReservations {
		if strings.ToLower(res.Mac) == seekMac && res.Device == device {
			return res.Address, err
		}
	}
	return addr, errors.New(fmt.Sprintf("No entry found for MAC %s", mac))
}

// Add a new port forward
func (n *VMWareNetworkingFile) AddPortFwd(fwd *PortFwd) error {
	if n.PortfwdExists(fwd) {
		return errors.New("Given port forward has already been added")
	}
	existingFwd := n.HostPortFwd(fwd.HostPort, fwd.Protocol)
	if existingFwd != nil {
		n.logger.Trace("update existing port forward entry", "fwd", existingFwd)
		existingFwd.GuestIp = fwd.GuestIp
		existingFwd.GuestPort = fwd.GuestPort
		existingFwd.Description = fwd.Description
		existingFwd.Device = fwd.Device
		fwd = existingFwd
	}
	n.logger.Debug("add port forward", "host.port", fwd.HostPort,
		"guest.ip", fwd.GuestIp, "guest.port", fwd.GuestPort)
	fwd.Enable = true
	n.PortFwds = append(n.PortFwds, fwd)
	return nil
}

// Remove a port forward
func (n *VMWareNetworkingFile) RemovePortFwd(fwd *PortFwd) error {
	if n.PortfwdExists(fwd) {
		return errors.New("Given port forward has already been added")
	}
	existingFwd := n.HostPortFwd(fwd.HostPort, fwd.Protocol)
	if existingFwd != nil {
		n.logger.Trace("update existing port forward entry", "fwd", existingFwd)
		fwd.Enable = false
		existingFwd.Enable = false
	} else {
		fwd.Enable = false
		n.PortFwds = append(n.PortFwds, fwd)
	}
	n.logger.Debug("remove port forward", "host.port", fwd.HostPort,
		"guest.ip", fwd.GuestIp, "guest.port", fwd.GuestPort)
	return nil
}

// Remove a defined adapter with given name
func (n *VMWareNetworkingFile) RemoveDeviceByName(devName string) error {
	n.logger.Debug("remove device", "name", devName)
	devIdx := -1
	for idx, adapter := range n.Devices {
		if adapter.Name == devName {
			devIdx = idx
			break
		}
	}
	if devIdx < 0 {
		return errors.New(fmt.Sprintf(
			"Failed to locate device with name %s", devName))
	}
	n.Devices = append(n.Devices[0:devIdx], n.Devices[devIdx+1:]...)
	return nil
}

// Remove a defined adapter with given slot number
func (n *VMWareNetworkingFile) RemoveDeviceBySlot(slotNumber int) error {
	n.logger.Debug("remove device", "slot", slotNumber)
	devIdx := -1
	for idx, adapter := range n.Devices {
		if adapter.Number == slotNumber {
			devIdx = idx
			break
		}
	}
	if devIdx < 0 {
		return errors.New(fmt.Sprintf(
			"Failed to locate device with slot number %d", slotNumber))
	}
	n.Devices = append(n.Devices[0:devIdx], n.Devices[devIdx+1:]...)
	return nil
}

// Find defined adapter with given name
func (n *VMWareNetworkingFile) GetDeviceByName(devName string) (foundDevice *Device) {
	for _, adapter := range n.Devices {
		if adapter.Name == devName {
			foundDevice = adapter
			break
		}
	}
	return foundDevice
}

// Find defined adapter with given slot number
func (n *VMWareNetworkingFile) GetDeviceBySlot(slotNumber int) (foundDevice *Device) {
	for _, adapter := range n.Devices {
		if adapter.Number == slotNumber {
			foundDevice = adapter
			break
		}
	}
	return foundDevice
}

// Creates a new virtual network adapter. If the adapter
// should be host only two parameters can be provided to
// configure:
// HostonlyNetmask - Ex: "255.255.255.0"
// HostonlySubnet  - Ex: "172.32.3.0"
func (n *VMWareNetworkingFile) CreateDevice(params ...string) *Device {
	var netmask *string
	var subnetIp *string
	if len(params) > 1 {
		netmask = &params[0]
		subnetIp = &params[1]
	}
	slot := 1
	usedSlots := n.usedAdapterSlots()
	for _, uSlot := range usedSlots {
		if uSlot > slot {
			break
		}
		slot++
	}
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
	n.logger.Debug("create device", "name", adapter.Name, "slot", adapter.Number)
	return adapter
}

// Parses the defined networking configuration file. This
// can also be used to "reload" if the file may have changed.
func (n *VMWareNetworkingFile) Load() error {
	n.logger.Trace("loading", "path", n.Path)
	nFile, err := os.Open(n.Path)
	if err != nil {
		n.logger.Warn("load failure - stubbing file contents", "path", n.Path, "error", err)
		n.Devices = []*Device{}
		n.DhcpReservations = []*DhcpReservation{}
		n.PortFwds = []*PortFwd{}

		return nil
	}
	defer nFile.Close()
	scanner := bufio.NewScanner(nFile)
	adapterPattern, err := regexp.Compile(ADAPTER_PATTERN)
	if err != nil {
		n.logger.Debug("load failure", "path", n.Path, "error", err)
		return err
	}
	adapterNames := adapterPattern.SubexpNames()
	dhcpresPattern, err := regexp.Compile(DHCP_RES_PATTERN)
	if err != nil {
		n.logger.Debug("load failure", "path", n.Path, "error", err)
		return err
	}
	dhcpresNames := dhcpresPattern.SubexpNames()
	portfwdPattern, err := regexp.Compile(PORTFWD_PATTERN)
	if err != nil {
		n.logger.Debug("load failure", "path", n.Path, "error", err)
		return err
	}
	portfwdNames := portfwdPattern.SubexpNames()

	adapters := [][]string{}
	portfwds := [][]string{}
	dhcpres := [][]string{}

	for scanner.Scan() {
		line := scanner.Text()
		match := adapterPattern.FindStringSubmatch(line)
		if match != nil {
			adapters = append(adapters, match)
			continue
		}
		match = portfwdPattern.FindStringSubmatch(line)
		if match != nil {
			portfwds = append(portfwds, match)
			continue
		}
		match = dhcpresPattern.FindStringSubmatch(line)
		if match != nil {
			dhcpres = append(dhcpres, match)
			continue
		}
	}
	n.loadAdapters(adapters, adapterNames)
	n.loadPortFwds(portfwds, portfwdNames)
	n.loadDhcpRes(dhcpres, dhcpresNames)

	n.logger.Debug("loaded", "num.adapters", len(n.Devices),
		"num.dhcpres", len(n.DhcpReservations),
		"num.portfwds", len(n.PortFwds), "path", n.Path)

	return nil
}

func (n *VMWareNetworkingFile) loadAdapters(adapters [][]string, mnames []string) {
	a := map[string]map[string]string{}
	for _, match := range adapters {
		info := NamePatternResults(match, mnames)
		sn := info["slot_number"]
		if _, ok := a[sn]; !ok {
			a[sn] = map[string]string{}
		}
		a[sn][info["key"]] = info["value"]
	}
	n.Devices = []*Device{}
	for slotNumStr, adapter := range a {
		if adapter["virtual_adapter"] == "no" {
			continue
		}
		slotNum, err := strconv.Atoi(slotNumStr)
		if err != nil {
			n.logger.Warn("failed to convert device slot number", "slot",
				slotNumStr, "error", err)
			continue
		}
		n.logger.Trace("loaded device", "adapter", adapter)
		dName := adapter["NAME"]
		if dName == "" {
			dName = fmt.Sprintf("vmnet%d", slotNum)
		}
		n.Devices = append(n.Devices, &Device{
			Name:            dName,
			Number:          slotNum,
			HostonlyNetmask: adapter["HOSTONLY_NETMASK"],
			HostonlySubnet:  adapter["HOSTONLY_SUBNET"],
			Dhcp:            adapter["DHCP"] == "yes",
			Nat:             adapter["NAT"] == "yes",
			VirtualAdapter:  adapter["VIRTUAL_ADAPTER"] == "yes",
		})
	}
}

func (n *VMWareNetworkingFile) loadPortFwds(portfwds [][]string, mnames []string) {
	n.PortFwds = []*PortFwd{}
	for _, match := range portfwds {
		portfwd := NamePatternResults(match, mnames)
		hostPort, err := strconv.Atoi(portfwd["host_port"])
		if err != nil {
			n.logger.Debug("failed to convert host port", "port",
				portfwd["host_port"], "error", err)
			continue
		}
		guestPort, err := strconv.Atoi(portfwd["guest_port"])
		if err != nil {
			n.logger.Debug("failed to convert guest port", "port",
				portfwd["guest_port"], "error", err)
			continue
		}

		n.PortFwds = append(n.PortFwds, &PortFwd{
			Enable:      portfwd["type"] == "add",
			Device:      portfwd["device"],
			Protocol:    portfwd["proto"],
			HostPort:    hostPort,
			GuestIp:     portfwd["guest_ip"],
			GuestPort:   guestPort,
			Description: strings.TrimSpace(portfwd["description"]),
		})
	}
}

func (n *VMWareNetworkingFile) loadDhcpRes(dhcpres [][]string, mnames []string) {
	n.DhcpReservations = []*DhcpReservation{}
	for _, match := range dhcpres {
		res := NamePatternResults(match, mnames)
		device, _ := strconv.Atoi(res["device"])

		n.DhcpReservations = append(n.DhcpReservations, &DhcpReservation{
			Address: res["address"],
			Device:  device,
			Mac:     res["mac"],
		})
	}
}

// Check if a port forward has already been added
func (n *VMWareNetworkingFile) PortfwdExists(fwd *PortFwd) bool {
	for _, eFwd := range n.PortFwds {
		if eFwd == fwd {
			return true
		}
	}
	return false
}

func (n *VMWareNetworkingFile) HostPortFwd(port int, protocol string) *PortFwd {
	for _, fwd := range n.PortFwds {
		if fwd.HostPort == port && fwd.Protocol == protocol {
			return fwd
		}
	}
	return nil
}

// Finds all currently used slot numbers. Returns
// in ascending sorted order.
func (n *VMWareNetworkingFile) usedAdapterSlots() []int {
	slots := []int{}
	for _, adapter := range n.Devices {
		slots = append(slots, adapter.Number)
	}
	sort.Ints(slots)
	n.logger.Debug("used adapter slots", "slots", slots)
	return slots
}
