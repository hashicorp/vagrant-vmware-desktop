// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package utility

type NatFileMock struct {
	Path     string
	Sections []*NatSection
}

func LoadNatFileMock(path string, sections []*NatSection) *NatFileMock {
	if path == "" {
		path = "/some/test/path/nat"
	}
	if sections == nil {
		sections = []*NatSection{}
	}

	return &NatFileMock{
		Path:     path,
		Sections: sections,
	}
}

func (n *NatFileMock) Load() error {
	return nil
}

func (n *NatFileMock) Save() error {
	return nil
}

func (n *NatFileMock) GetSection(name string) *NatSection {
	for _, section := range n.Sections {
		if section.Name == name {
			return section
		}
	}
	return nil
}
