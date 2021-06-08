package utility

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strings"
	"testing"
	"text/template"
	"time"
)

const LEASE_ENTRY = `
lease {{.Address}} {
        starts 4 {{.StartTime}};
        ends 4 {{.EndTime}};
        hardware ethernet {{.Mac}};
        client-hostname "{{.Hostname}}";
}
`
const PARTIAL_LEASE_ENTRY = `
lease {{.Address}} {
        starts 4 {{.StartTime}};
        ends 4 {{.EndTime}};
        hardware ethernet {{.Mac}};
}
`

const MACOS_LEASE_ENTRY = `
{
        name={{.Hostname}}
        ip_address={{.Address}}
        hw_address=1,{{.Mac}}
        identifier=1,{{.Mac}}
        lease=0x5f7e4b58
}`

const MAC_ADDRESS_SET = "0123456789abcde"

type LeaseEntry struct {
	Address, StartTime, EndTime, Mac, Hostname string
}

func TestDhcpLoadFailure(t *testing.T) {
	_, err := LoadDhcpLeaseFile("/unknown/path/to/file", defaultUtilityLogger())
	if err == nil {
		t.Errorf("Loading of dhcp leases file expected to fail")
	}
}

func TestDhcpLoadSuccess(t *testing.T) {
	path := createLeaseFile(generateLeaseEntries(1))
	defer os.Remove(path)
	_, err := LoadDhcpLeaseFile(path, defaultUtilityLogger())
	if err != nil {
		t.Errorf("Failed to load dhcp leases file: %s", err)
	}
}

func TestDhcpLoadNonactive(t *testing.T) {
	path := createLeaseFile(generateLeaseEntries(3))
	defer os.Remove(path)
	df, err := LoadDhcpLeaseFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load dhcp leases file: %s", err))
	}
	if len(df.Entries) != 1 {
		t.Errorf("Unexpected number of lease entries 1 != %d", len(df.Entries))
	}
}

func TestDhcpLookupFailure(t *testing.T) {
	path := createLeaseFile(generateLeaseEntries(5))
	defer os.Remove(path)
	df, err := LoadDhcpLeaseFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load dhcp leases file: %s", err))
	}
	address, err := df.IpForMac("MAC:02")
	if err == nil {
		t.Errorf("Unexpected address for invalid MAC %s", *address)
	}
}

func TestDhcpLookupSuccess(t *testing.T) {
	path := createLeaseFile(generateLeaseEntries(5))
	defer os.Remove(path)
	df, err := LoadDhcpLeaseFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load dhcp leases file: %s", err))
	}
	address, err := df.IpForMac("MAC:01")
	if err != nil {
		t.Errorf("Failed to receive address for valid MAC: %s", err)
	}
	if address != nil && *address != "127.0.2.1" {
		t.Errorf("Received unexpected address 127.0.2.1 != %s", *address)
	}
}

func TestDhcpPartialLookupSuccess(t *testing.T) {
	path := createPartialLeaseFile(generateLeaseEntries(5))
	defer os.Remove(path)
	df, err := LoadDhcpLeaseFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load dhcp leases file: %s", err))
	}
	address, err := df.IpForMac("MAC:01")
	if err != nil {
		t.Errorf("Failed to receive address for valid MAC: %s", err)
	}
	if address != nil && *address != "127.0.2.1" {
		t.Errorf("Received unexpected address 127.0.2.1 != %s", *address)
	}
}

func TestMacosDhcpLookupSuccess(t *testing.T) {
	entries := generateMacosLeaseEntries(5)
	baseEntry := entries[0]
	path := createMacosLeaseFile(entries)
	defer os.Remove(path)
	df, err := newMacosDhcpLeaseFile(path)
	if err != nil {
		panic(fmt.Sprintf("Failed to load dhcp leases file: %s", err))
	}
	address, err := df.IpForMac(baseEntry.Mac)
	if err != nil {
		t.Errorf("Failed to receive address for valid MAC: %s base entry: %v", err, baseEntry)
	}
	if address != nil && *address != entries[0].Address {
		t.Errorf("Received unexpected address %s != %s base entry: %v", baseEntry.Address, *address, baseEntry)
	}
}

func TestMacosDhcpLookupFailure(t *testing.T) {
	path := createMacosLeaseFile(generateMacosLeaseEntries(5))
	defer os.Remove(path)
	df, err := newMacosDhcpLeaseFile(path)
	if err != nil {
		panic(fmt.Sprintf("Failed to load dhcp leases file: %s", err))
	}
	address, err := df.IpForMac(generateMAC())
	if err == nil {
		t.Errorf("Unexpected address for invalid MAC %s", *address)
	}
}

func newMacosDhcpLeaseFile(path string) (l *DhcpLeaseFile, err error) {
	l = &DhcpLeaseFile{
		Path:      path,
		logger:    defaultUtilityLogger(),
		timeF:     MACOS_TIME_FORMAT,
		leaseP:    MACOS_LEASE_PATTERN,
		startP:    MACOS_START_PATTERN,
		endP:      MACOS_END_PATTERN,
		macP:      MACOS_MAC_PATTERN,
		hostnameP: MACOS_HOSTNAME_PATTERN}
	err = l.Load()
	return
}

func generateLeaseEntries(num int) (entries []*LeaseEntry) {
	var startTime time.Time
	var endTime time.Time
	location, _ := time.LoadLocation("UTC")
	now := time.Now().In(location)
	for i := 1; i <= num; i++ {
		switch i {
		case 2:
			startTime = now.Add(time.Duration(i) * time.Hour)
			endTime = now.Add(time.Duration(i+1) * time.Hour)
		case 3:
			startTime = now.Add(time.Duration(-(i + 1)) * time.Hour)
			endTime = now.Add(time.Duration(-i) * time.Hour)
		default:
			startTime = now.Add(time.Duration(-i) * time.Hour)
			endTime = now.Add(time.Duration(i) * time.Hour)
		}
		entry := &LeaseEntry{
			Address:   fmt.Sprintf("127.0.2.%d", i),
			Mac:       fmt.Sprintf("MAC:%d", i),
			Hostname:  "vagrant",
			StartTime: startTime.Format(VMWARE_TIME_FORMAT),
			EndTime:   endTime.Format(VMWARE_TIME_FORMAT),
		}
		entries = append(entries, entry)
	}
	return entries
}

func generateMAC() string {
	m := strings.Builder{}
	for i := 0; i < 6; i++ {
		if i != 0 {
			m.WriteByte(':')
		}
		for j := 0; j < 2; j++ {
			idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(MAC_ADDRESS_SET))))
			if err != nil {
				panic("failed to get random mac character" + err.Error())
			}
			m.WriteByte(MAC_ADDRESS_SET[idx.Int64()])
		}
	}
	return m.String()
}

func generateMacosLeaseEntries(num int) (entries []*LeaseEntry) {
	for i := 1; i <= num; i++ {
		entry := &LeaseEntry{
			Address:  fmt.Sprintf("127.0.2.%d", i),
			Mac:      generateMAC(),
			Hostname: "vagrant",
		}
		entries = append(entries, entry)
	}
	return entries
}

func createLeaseFile(leases []*LeaseEntry) string {
	leaseFile, err := ioutil.TempFile("", "leases")
	if err != nil {
		panic(fmt.Sprintf(
			"Failed to create test dhcpd leases file: %s", err))
	}
	defer leaseFile.Close()
	t := template.Must(template.New("leases").Parse(LEASE_ENTRY))
	for _, lease := range leases {
		err := t.Execute(leaseFile, lease)
		if err != nil {
			panic(fmt.Sprintf(
				"Failed to write dhcpd lease to file: %s", err))
		}
	}
	return leaseFile.Name()
}

func createMacosLeaseFile(leases []*LeaseEntry) string {
	leaseFile, err := ioutil.TempFile("", "leases")
	if err != nil {
		panic(fmt.Sprintf(
			"Failed to create test dhcpd leases file: %s", err))
	}
	defer leaseFile.Close()
	t := template.Must(template.New("leases").Parse(MACOS_LEASE_ENTRY))
	for _, lease := range leases {
		err := t.Execute(leaseFile, lease)
		if err != nil {
			panic(fmt.Sprintf(
				"Failed to write dhcpd lease to file: %s", err))
		}
	}
	return leaseFile.Name()
}

func createPartialLeaseFile(leases []*LeaseEntry) string {
	leaseFile, err := ioutil.TempFile("", "leases")
	if err != nil {
		panic(fmt.Sprintf(
			"Failed to create test dhcpd leases file: %s", err))
	}
	defer leaseFile.Close()
	t := template.Must(template.New("leases").Parse(PARTIAL_LEASE_ENTRY))
	for _, lease := range leases {
		err := t.Execute(leaseFile, lease)
		if err != nil {
			panic(fmt.Sprintf(
				"Failed to write dhcpd lease to file: %s", err))
		}
	}
	return leaseFile.Name()
}
