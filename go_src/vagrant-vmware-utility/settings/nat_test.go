// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package settings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

func TestNatLoadMissing(t *testing.T) {
	td := mkdir()
	defer os.RemoveAll(td)
	nfile := path.Join(td, "nat.json")
	_, err := LoadNATSettings(nfile, defaultSettingsLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat settings - %s", err))
	}
	_, err = os.Stat(nfile)
	if err != nil {
		t.Errorf("NAT file should have been created - %s", nfile)
	}
}

func TestNatLoadExsiting(t *testing.T) {
	natf := stubNat(&NatInfo{
		Fwds: []*utility.PortFwd{
			&utility.PortFwd{
				HostPort: 22}}})
	defer os.RemoveAll(path.Dir(natf))
	nat, err := LoadNATSettings(natf, defaultSettingsLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat settings - %s", err))
	}
	if len(nat.PortFwds()) != 1 {
		t.Errorf("Expecting 1 port forward but found %d", len(nat.PortFwds()))
	}
}

func TestNatAddEntry(t *testing.T) {
	td := mkdir()
	defer os.RemoveAll(td)
	nfile := path.Join(td, "nat.json")
	nat, err := LoadNATSettings(nfile, defaultSettingsLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat settings - %s", err))
	}
	fwd := &utility.PortFwd{
		HostPort:    22,
		GuestPort:   22,
		Protocol:    "tcp",
		GuestIp:     "127.0.0.3",
		Description: "vagrant: a-path"}
	e := nat.Add(fwd)
	if e != nil {
		t.Errorf("Failed to add port forward entry - %s", e)
		return
	}
	if len(nat.PortFwds()) == 0 {
		t.Errorf("Failed to add port forward entry")
		return
	}
	nfwd := nat.PortFwds()[0]
	if nfwd != fwd {
		t.Errorf("Failed to detect added port forward - %v != %v",
			nfwd, fwd)
	}
}

func TestNatAddEntryConflictAllowed(t *testing.T) {
	td := mkdir()
	defer os.RemoveAll(td)
	nfile := path.Join(td, "nat.json")
	nat, err := LoadNATSettings(nfile, defaultSettingsLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat settings - %s", err))
	}
	fwd := &utility.PortFwd{
		HostPort:    22,
		GuestPort:   22,
		Protocol:    "tcp",
		GuestIp:     "127.0.0.3",
		Description: "vagrant: a-path"}
	e := nat.Add(fwd)
	if e != nil {
		panic("Failed to add port forward entry")
	}
	if len(nat.PortFwds()) == 0 {
		panic("Failed to add port forward entry")
	}
	e = nat.Add(fwd)
	if e != nil {
		t.Errorf("Port forward should be allowed to be added")
		return
	}
	if len(nat.PortFwds()) != 1 {
		t.Errorf("Port forward list should only contain single entry - actual: %d", len(nat.PortFwds()))
	}
}

func TestNatAddEntryConflict(t *testing.T) {
	td := mkdir()
	defer os.RemoveAll(td)
	nfile := path.Join(td, "nat.json")
	nat, err := LoadNATSettings(nfile, defaultSettingsLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat settings - %s", err))
	}
	fwd := &utility.PortFwd{
		HostPort:    22,
		GuestPort:   22,
		Protocol:    "tcp",
		GuestIp:     "127.0.0.3",
		Description: "vagrant: a-path"}
	e := nat.Add(fwd)
	if e != nil {
		panic("Failed to add port forward entry")
	}
	if len(nat.PortFwds()) == 0 {
		panic("Failed to add port forward entry")
	}
	fwd = &utility.PortFwd{
		HostPort:    22,
		GuestPort:   22,
		Protocol:    "tcp",
		GuestIp:     "127.0.0.2",
		Description: "vagrant: b-path"}
	e = nat.Add(fwd)
	if e != nil {
		t.Errorf("Unexpected error when adding conflict forward: %s", e)
	}
}

func TestNatRemoveEntry(t *testing.T) {
	td := mkdir()
	defer os.RemoveAll(td)
	nfile := path.Join(td, "nat.json")
	nat, err := LoadNATSettings(nfile, defaultSettingsLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat settings - %s", err))
	}
	fwd := &utility.PortFwd{
		HostPort:    22,
		GuestPort:   22,
		Protocol:    "tcp",
		GuestIp:     "127.0.0.3",
		Description: "vagrant: a-path"}
	e := nat.Add(fwd)
	if e != nil {
		panic("Failed to add port forward entry")
	}
	if len(nat.PortFwds()) == 0 {
		panic("Failed to add port forward entry")
	}
	e = nat.Remove(fwd)
	if len(nat.PortFwds()) != 0 {
		t.Errorf("Failed to remove port forward with fuzzy match")
	}
}

func TestNatClear(t *testing.T) {
	td := mkdir()
	defer os.RemoveAll(td)
	nfile := path.Join(td, "nat.json")
	nat, err := LoadNATSettings(nfile, defaultSettingsLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat settings - %s", err))
	}
	fwd := &utility.PortFwd{
		HostPort:    22,
		GuestPort:   22,
		Protocol:    "tcp",
		GuestIp:     "127.0.0.3",
		Description: "vagrant: a-path"}
	e := nat.Add(fwd)
	if e != nil {
		panic("Failed to add port forward entry")
	}
	if len(nat.PortFwds()) == 0 {
		panic("Failed to add port forward entry")
	}
	nat.Clear()
	if len(nat.PortFwds()) != 0 {
		t.Errorf("Failed to clear port forwards")
	}
}

func TestNatRemoveEntryFuzzy(t *testing.T) {
	td := mkdir()
	defer os.RemoveAll(td)
	nfile := path.Join(td, "nat.json")
	nat, err := LoadNATSettings(nfile, defaultSettingsLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat settings - %s", err))
	}
	fwd := &utility.PortFwd{
		HostPort:    22,
		GuestPort:   22,
		Protocol:    "tcp",
		GuestIp:     "127.0.0.3",
		Description: "vagrant: a-path"}
	e := nat.Add(fwd)
	if e != nil {
		panic("Failed to add port forward entry")
	}
	if len(nat.PortFwds()) == 0 {
		panic("Failed to add port forward entry")
	}
	fwd = &utility.PortFwd{
		HostPort:    22,
		Protocol:    "tcp",
		Description: "vagrant: a-path"}
	e = nat.Remove(fwd)
	if len(nat.PortFwds()) != 0 {
		t.Errorf("Failed to remove port forward with fuzzy match")
	}
}

func TestNatAddEntrySave(t *testing.T) {
	td := mkdir()
	defer os.RemoveAll(td)
	nfile := path.Join(td, "nat.json")
	nat, err := LoadNATSettings(nfile, defaultSettingsLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat settings - %s", err))
	}
	fwd := &utility.PortFwd{
		HostPort:    22,
		GuestPort:   22,
		Protocol:    "tcp",
		GuestIp:     "127.0.0.3",
		Description: "vagrant: a-path"}
	e := nat.Add(fwd)
	if e != nil {
		panic(fmt.Sprintf("Failed to add port forward entry - %s", e))
	}
	if len(nat.PortFwds()) == 0 {
		panic("Failed to add port forward entry")
	}
	e = nat.Save()
	if e != nil {
		panic(fmt.Sprintf("Failed to save nat settings - %s", e))
	}
	b, e := ioutil.ReadFile(nfile)
	if e != nil {
		panic(fmt.Sprintf("Failed to read nat settings - %s", e))
	}
	if !strings.Contains(string(b), "vagrant: a-path") {
		t.Errorf("Stored nat.json file does not contain expected content.")
	}
}

func TestNatReload(t *testing.T) {
	natf := stubNat(&NatInfo{
		Fwds: []*utility.PortFwd{
			&utility.PortFwd{
				HostPort: 22}}})
	defer os.RemoveAll(path.Dir(natf))
	nat, err := LoadNATSettings(natf, defaultSettingsLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat settings - %s", err))
	}
	if len(nat.PortFwds()) != 1 {
		panic("Invalid port forwards count")
	}
	fwd := &utility.PortFwd{
		HostPort:    2222,
		GuestPort:   2222,
		Protocol:    "tcp",
		GuestIp:     "127.0.0.3",
		Description: "vagrant: a-path"}
	e := nat.Add(fwd)
	if e != nil {
		panic(fmt.Sprintf("Failed to add port forward entry - %s", e))
	}
	if len(nat.PortFwds()) != 2 {
		panic("Failed to add port forward entry")
	}
	e = nat.Reload()
	if e != nil {
		t.Errorf("Failed to reload nat settings file - %s", e)
		return
	}
	if len(nat.PortFwds()) != 1 {
		t.Errorf("Failed to reload nat settings. Invalid number of port forwards.")
	}
}

func TestNatMultiAddEntry(t *testing.T) {
	td := mkdir()
	defer os.RemoveAll(td)
	nfile := path.Join(td, "nat.json")
	nat, err := LoadNATSettings(nfile, defaultSettingsLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat settings - %s", err))
	}
	fwds := []*utility.PortFwd{
		&utility.PortFwd{
			HostPort:    22,
			GuestPort:   22,
			Protocol:    "tcp",
			GuestIp:     "127.0.0.3",
			Description: "vagrant: a-path"},
		&utility.PortFwd{
			HostPort:    2222,
			GuestPort:   22,
			Protocol:    "tcp",
			GuestIp:     "127.0.0.2",
			Description: "vagrant: b-path"}}
	e := nat.MultiAdd(fwds)
	if e != nil {
		t.Errorf("Failed to add port forwards - %s", e)
		return
	}
	if len(nat.PortFwds()) != 2 {
		t.Errorf("Expected 2 port forward entries but found %d", len(nat.PortFwds()))
	}
}

func TestNatMultiAddEntryConflictAllowed(t *testing.T) {
	td := mkdir()
	defer os.RemoveAll(td)
	nfile := path.Join(td, "nat.json")
	nat, err := LoadNATSettings(nfile, defaultSettingsLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat settings - %s", err))
	}
	fwds := []*utility.PortFwd{
		&utility.PortFwd{
			HostPort:    22,
			GuestPort:   22,
			Protocol:    "tcp",
			GuestIp:     "127.0.0.3",
			Description: "vagrant: a-path"},
		&utility.PortFwd{
			HostPort:    22,
			GuestPort:   22,
			Protocol:    "tcp",
			GuestIp:     "127.0.0.2",
			Description: "vagrant: b-path"}}
	e := nat.MultiAdd(fwds)
	if e != nil {
		t.Errorf("Unexpected error when adding conflict: %s", e)
	}
}

func stubNat(info *NatInfo) string {
	f := path.Join(mkdir(), "nat.json")
	data, err := json.Marshal(info)
	if err != nil {
		panic(fmt.Sprintf("Failed to dump nat.json info: %s", err))
	}
	if e := ioutil.WriteFile(f, data, 0644); e != nil {
		panic(fmt.Sprintf("Failed to create nat.json: %s", e))
	}
	return f
}

func mkdir() string {
	d, e := ioutil.TempDir("", "nat-test")
	if e != nil {
		panic(fmt.Sprintf("Failed to create temporary directory: %s", e))
	}
	return d
}
