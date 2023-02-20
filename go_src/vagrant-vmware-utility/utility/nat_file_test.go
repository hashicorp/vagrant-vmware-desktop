// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utility

import (
	"fmt"
	"io/ioutil"
	"os"
	//	"strings"
	"testing"
)

const NAT_CONF_CONTENT = `
[section1]
key1 = value1
key2 = multi value2

# a comment
[section2]
s2k1 = section2 value
# comment
# another comment
s2k2 = value with = embedded
`

func TestNatLoadFailure(t *testing.T) {
	_, err := LoadNatFile("/unknown/path/to/file", defaultUtilityLogger())
	if err == nil {
		t.Errorf("NAT loading of unknown file did not fail")
	}
}

func TestNatLoadSuccess(t *testing.T) {
	path := createNatFile()
	defer os.Remove(path)
	_, err := LoadNatFile(path, defaultUtilityLogger())
	if err != nil {
		t.Errorf("Failed to load NAT file: %s", err)
	}
}

func TestNatSectionsCount(t *testing.T) {
	path := createNatFile()
	defer os.Remove(path)
	nfile, err := LoadNatFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat.conf file: %s", err))
	}
	if len(nfile.Sections) != 2 {
		t.Errorf("Invalid number of sections. Expected 2 but found %d", len(nfile.Sections))
	}
}

func TestNatSectionNames(t *testing.T) {
	path := createNatFile()
	defer os.Remove(path)
	nfile, err := LoadNatFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat.conf file: %s", err))
	}
	found1 := false
	found2 := false
	for _, section := range nfile.Sections {
		if section.Name == "section1" {
			found1 = true
		}
		if section.Name == "section2" {
			found2 = true
		}
	}
	if !found1 {
		t.Errorf("Failed to locate expected section `section1`")
	}
	if !found2 {
		t.Errorf("Failed to locate expected section `section2`")
	}
}

func TestNatSave(t *testing.T) {
	path := createNatFile()
	defer os.Remove(path)
	nfile, err := LoadNatFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat.conf file: %s", err))
	}
	err = nfile.Save()
	if err != nil {
		t.Errorf("Failed to save nat.conf file: %s", err)
	}
}

func TestNatSaveModified(t *testing.T) {
	path := createNatFile()
	defer os.Remove(path)
	nfile, err := LoadNatFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat.conf file: %s", err))
	}
	nfile.Sections = []*NatSection{}
	err = nfile.Save()
	if err != nil {
		t.Errorf("Failed to save nat.conf file: %s", err)
	}
	err = nfile.Load()
	if err != nil {
		t.Errorf("Failed to reload nat.conf file: %s", err)
	}
	if len(nfile.Sections) > 0 {
		t.Errorf("Expected no sections but found %d sections", len(nfile.Sections))
	}
}

func TestNatGetSection(t *testing.T) {
	path := createNatFile()
	defer os.Remove(path)
	nfile, err := LoadNatFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat.conf file: %s", err))
	}
	s := nfile.GetSection("section1")
	if s == nil {
		t.Errorf("Failed to get section1")
		return
	}
	if s.Name != "section1" {
		t.Errorf("Expecting section named `section1` but got `%s`", s.Name)
	}
}

func TestNatGetSectionUnknown(t *testing.T) {
	path := createNatFile()
	defer os.Remove(path)
	nfile, err := LoadNatFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat.conf file: %s", err))
	}
	s := nfile.GetSection("unknown")
	if s != nil {
		t.Errorf("Received unexpected section `%s`", s.Name)
	}
}

func TestNatSectionEntries(t *testing.T) {
	path := createNatFile()
	defer os.Remove(path)
	nfile, err := LoadNatFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat.conf file: %s", err))
	}
	s := nfile.GetSection("section1")
	if len(s.Entries) != 2 {
		t.Errorf("Expected entries in section1: 2 Actual: %d", len(s.Entries))
		return
	}
	s = nfile.GetSection("section2")
	if len(s.Entries) != 2 {
		t.Errorf("Expected entries in section2: 2 Actual: %d", len(s.Entries))
	}
}

func TestNatSection1Entries(t *testing.T) {
	path := createNatFile()
	defer os.Remove(path)
	nfile, err := LoadNatFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat.conf file: %s", err))
	}
	s := nfile.GetSection("section1")
	matches := map[string]string{
		"key1": "value1",
		"key2": "multi value2"}
	for key, value := range matches {
		found := false
		for _, entry := range s.Entries {
			if entry.Key == key {
				if entry.Value != value {
					t.Errorf("Expected `%s` to have value `%s` but was `%s`", key, value, entry.Value)
					return
				}
				found = true
			}
		}
		if !found {
			t.Errorf("Failed to locate expected entry key `%s`", key)
			return
		}
	}
}

func TestNatSection2Entries(t *testing.T) {
	path := createNatFile()
	defer os.Remove(path)
	nfile, err := LoadNatFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat.conf file: %s", err))
	}
	s := nfile.GetSection("section2")
	matches := map[string]string{
		"s2k1": "section2 value",
		"s2k2": "value with = embedded"}
	for key, value := range matches {
		found := false
		for _, entry := range s.Entries {
			if entry.Key == key {
				if entry.Value != value {
					t.Errorf("Expected `%s` to have value `%s` but was `%s`", key, value, entry.Value)
					return
				}
				found = true
			}
		}
		if !found {
			t.Errorf("Failed to locate expected entry key `%s`", key)
			return
		}
	}
}

func TestNatSectionEntryDelete(t *testing.T) {
	path := createNatFile()
	defer os.Remove(path)
	nfile, err := LoadNatFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat.conf file: %s", err))
	}
	s := nfile.GetSection("section1")
	entry := s.Entries[0]
	err = s.DeleteEntry(entry)
	if err != nil {
		t.Errorf("Failed to delete entry: %s", err)
		return
	}
	for _, e := range s.Entries {
		if e == entry {
			t.Errorf("Found deleted entry within section entry list")
			return
		}
	}
}

func TestNatSectionEntryIndexDelete(t *testing.T) {
	path := createNatFile()
	defer os.Remove(path)
	nfile, err := LoadNatFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat.conf file: %s", err))
	}
	s := nfile.GetSection("section1")
	entry := s.Entries[0]
	err = s.DeleteEntryAt(0)
	if err != nil {
		t.Errorf("Failed to delete entry: %s", err)
		return
	}
	for _, e := range s.Entries {
		if e == entry {
			t.Errorf("Found deleted entry within section entry list")
			return
		}
	}
}

func TestNatSectionEntryDeleteNotFound(t *testing.T) {
	path := createNatFile()
	defer os.Remove(path)
	nfile, err := LoadNatFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat.conf file: %s", err))
	}
	s := nfile.GetSection("section1")
	entry := s.Entries[0]
	err = s.DeleteEntry(entry)
	if err != nil {
		t.Errorf("Failed to delete entry: %s", err)
		return
	}
	err = s.DeleteEntry(entry)
	if err == nil {
		t.Errorf("Delete of missing entry did not generate error")
		return
	}
}

func TestNatSectionEntryDeleteIndexUnderBounds(t *testing.T) {
	path := createNatFile()
	defer os.Remove(path)
	nfile, err := LoadNatFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat.conf file: %s", err))
	}
	s := nfile.GetSection("section1")
	err = s.DeleteEntryAt(-1)
	if err == nil {
		t.Errorf("Entry delete with invalid index did not fail")
		return
	}
}

func TestNatSectionEntryDeleteIndexOverBounds(t *testing.T) {
	path := createNatFile()
	defer os.Remove(path)
	nfile, err := LoadNatFile(path, defaultUtilityLogger())
	if err != nil {
		panic(fmt.Sprintf("Failed to load nat.conf file: %s", err))
	}
	s := nfile.GetSection("section1")
	err = s.DeleteEntryAt(5)
	if err == nil {
		t.Errorf("Entry delete with invalid index did not fail")
		return
	}
}

func TestNatSectionEntryMatchValid(t *testing.T) {
	entry := &NatEntry{Key: "key1", Value: "value1"}
	if !entry.Match("key1") {
		t.Errorf("Failed to match entry with key `key1`")
		return
	}
}

func TestNatSectionEntryMatchInvalid(t *testing.T) {
	entry := &NatEntry{Key: "key1", Value: "value1"}
	if entry.Match("key2") {
		t.Errorf("Matched entry with invalid key `key2`")
		return
	}
}

func createNatFile() string {
	nfile, err := ioutil.TempFile("", "nat")
	if err != nil {
		panic(fmt.Sprintf("Failed to create test nat.conf file: %s", err))
	}
	defer nfile.Close()
	_, err = nfile.WriteString(NAT_CONF_CONTENT)
	if err != nil {
		panic(fmt.Sprintf("Failed to write test nat.conf file: %s", err))
	}
	return nfile.Name()
}
