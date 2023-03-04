// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utility

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	hclog "github.com/hashicorp/go-hclog"
)

// Patterns used for VMware provided networking + dhcpd
const VMWARE_TIME_FORMAT = `2006/01/02 15:04:05`
const VMWARE_LEASE_PATTERN = `(?ims)^\s*lease\s+(?P<address>\d+.\d+.\d+.\d+)\s*\{(?P<info>.+?)\}`
const VMWARE_START_PATTERN = `(?im)^\s+starts\s+(?P<start_weekday>\d)\s+(?P<start_date>[^;]+);\s*`
const VMWARE_END_PATTERN = `(?im)^\s+ends\s+(?P<end_weekday>\d)\s+(?P<end_date>[^;]+);\s*`
const VMWARE_MAC_PATTERN = `(?im)^\s+hardware\s+ethernet\s+(?P<mac>[^;]+);\s*`
const VMWARE_HOSTNAME_PATTERN = `(?im)^\s+client-hostname\s+"(?P<hostname>[^"]+)";`

// Patterns used for macOS provided networking + dhcpd
const MACOS_TIME_FORMAT = `2006/01/02 15:04:05` // This isn't used, no time information in leases
const MACOS_LEASE_PATTERN = `(?ims)^\{(?P<info>.+?ip_address=(?P<address>\d+.\d+.\d+.\d+).+?)\}`
const MACOS_START_PATTERN = `` // No time in lease
const MACOS_END_PATTERN = ``   // No time in lease
const MACOS_MAC_PATTERN = `(?im)hw_address=.*,(?P<mac>[^\s]+)`
const MACOS_HOSTNAME_PATTERN = `(?im)name=(?P<hostname>[^\s]+)`

// Path prefix for where VMware DHCP file is located
const VMWARE_LEASE_FILE_PREFIX = "/var/db/vmware"

type DhcpLeaseFile struct {
	Path    string
	Entries []*DhcpEntry
	logger  hclog.Logger

	timeF     string
	leaseP    string
	startP    string
	endP      string
	macP      string
	hostnameP string
}

type DhcpEntry struct {
	Address  string
	Mac      string
	Hostname string
	Created  time.Time
	Expires  time.Time
}

func (d *DhcpEntry) NormalizeMac() {
	if d.Mac == "" {
		return
	}

	p := strings.Split(d.Mac, ":")
	for i := 0; i < len(p); i++ {
		if len(p[i]) < 2 {
			p[i] = "0" + p[i]
		}
	}
	d.Mac = strings.Join(p, ":")
	return
}

func LoadDhcpLeaseFile(path string, logger hclog.Logger) (leaseFile *DhcpLeaseFile, err error) {
	if logger == nil {
		logger = hclog.New(&hclog.LoggerOptions{
			Output: hclog.DefaultOutput,
			Level:  hclog.Error,
			Name:   "vagrant-vmware-dhcpd-leases"})
	} else {
		logger = logger.Named("dhcpd-leases")
	}
	if strings.HasPrefix(path, "/var") && !strings.HasPrefix(path, VMWARE_LEASE_FILE_PREFIX) {
		logger.Info("loading macOS style DHCP lease file", "path", path)
		leaseFile = &DhcpLeaseFile{
			Path:      path,
			logger:    logger,
			timeF:     MACOS_TIME_FORMAT,
			leaseP:    MACOS_LEASE_PATTERN,
			startP:    MACOS_START_PATTERN,
			endP:      MACOS_END_PATTERN,
			macP:      MACOS_MAC_PATTERN,
			hostnameP: MACOS_HOSTNAME_PATTERN}
	} else {
		logger.Info("loading VMware style DHCP lease file", "path", path)
		leaseFile = &DhcpLeaseFile{
			Path:      path,
			logger:    logger,
			timeF:     VMWARE_TIME_FORMAT,
			leaseP:    VMWARE_LEASE_PATTERN,
			startP:    VMWARE_START_PATTERN,
			endP:      VMWARE_END_PATTERN,
			macP:      VMWARE_MAC_PATTERN,
			hostnameP: VMWARE_HOSTNAME_PATTERN}
	}
	err = leaseFile.Load()
	return leaseFile, err
}

func (d *DhcpLeaseFile) Load() error {
	d.logger.Debug("loading DHCP lease file", "path", d.Path)
	dFile, err := os.Open(d.Path)
	if err != nil {
		d.logger.Warn("failed to load DHCP lease data file", "path", d.Path, "error", err)
		return err
	}
	defer dFile.Close()
	reader := bufio.NewReader(dFile)
	ridx := 0
	leasePattern, err := regexp.Compile(d.leaseP)
	if err != nil {
		d.logger.Warn("failed to load DHCP lease data file", "path", d.Path, "error", err)
		return err
	}
	leaseNames := leasePattern.SubexpNames()
	matches := []map[string]string{}
	for {
		loc := leasePattern.FindReaderIndex(reader)
		if loc == nil {
			break
		}
		_, err := dFile.Seek(int64(ridx+loc[0]), 0)
		if err != nil {
			d.logger.Warn("failed to seek DHCP lease data file", "path", d.Path, "error", err)
			return err
		}
		buf := make([]byte, loc[1])
		_, err = dFile.Read(buf)
		content := string(buf)
		match := leasePattern.FindStringSubmatch(content)
		if match != nil {
			result := map[string]string{}
			for i, name := range leaseNames {
				if i == 0 {
					continue
				}
				result[name] = match[i]
			}
			matches = append(matches, result)
		}
		ridx = ridx + loc[1]
		if _, err = dFile.Seek(int64(ridx), 0); err != nil {
			return err
		}
		reader.Reset(dFile)
	}
	for _, entry := range matches {
		err = d.loadEntry(entry)
		if err != nil {
			d.logger.Trace("failed to load DHCP entry", "entry", entry, "error", err)
		}
	}
	d.logger.Debug("loaded active leases", "path", d.Path, "leases", len(d.Entries))
	return nil
}

func (d *DhcpLeaseFile) IpForMac(mac string) (*string, error) {
	for _, entry := range d.Entries {
		if entry.Mac == mac {
			return &entry.Address, nil
		}
	}
	return nil, fmt.Errorf("No entry found for MAC %s", mac)
}

func (d *DhcpLeaseFile) AddEntry(entry *DhcpEntry) error {
	d.logger.Trace("adding external entry", "mac", entry.Mac, "entry", entry)
	// check if expired
	if time.Now().After(entry.Expires) {
		d.logger.Trace("external entry is expired", "entry", entry)
		return fmt.Errorf("DHCP entry is expired")
	}
	// check if entry exists
	var existing *DhcpEntry
	for _, check := range d.Entries {
		if check.Mac == entry.Mac {
			existing = check
			break
		}
	}
	if existing != nil {
		d.logger.Trace("existing entry found", "mac", existing.Mac, "entry", existing)
		return fmt.Errorf("DHCP entry already exists for MAC")
	}
	d.Entries = append(d.Entries, entry)
	return nil
}

// Load new lease entry. This validates lease entries and only
// loads the new entry if it is currently active.
func (d *DhcpLeaseFile) loadEntry(rawEntry map[string]string) error {
	entry, err := d.extractEntry(rawEntry)
	if err != nil {
		return err
	}
	currentTime := time.Now()
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		return err
	}
	var startTime, endTime time.Time
	if entry["start_date"] != "" {
		startTime, err = time.ParseInLocation(d.timeF, entry["start_date"], loc)
		if err != nil {
			return err
		}
		endTime, err = time.ParseInLocation(d.timeF, entry["end_date"], loc)
		if err != nil {
			return err
		}
		if currentTime.After(endTime) || currentTime.Before(startTime) {
			return fmt.Errorf("Lease entry is not currently active")
		}
	}
	newEntry := &DhcpEntry{
		Address:  entry["address"],
		Mac:      entry["mac"],
		Hostname: entry["hostname"],
		Created:  startTime,
		Expires:  endTime,
	}
	newEntry.NormalizeMac()
	d.Entries = append(d.Entries, newEntry)
	return nil
}

func (d *DhcpLeaseFile) extractEntry(rawEntry map[string]string) (map[string]string, error) {
	entry := map[string]string{"address": rawEntry["address"]}
	patterns := []string{
		d.startP,
		d.endP,
		d.macP,
		d.hostnameP,
	}
	for _, pattern := range patterns {
		match, err := MatchPattern(pattern, rawEntry["info"])
		if err != nil {
			continue
		}
		for k, v := range match {
			entry[k] = v
		}
	}
	return entry, nil
}
