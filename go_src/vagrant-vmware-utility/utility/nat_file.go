package utility

import (
	"bufio"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"strings"

	hclog "github.com/hashicorp/go-hclog"
)

type NatFile interface {
	Load() error
	Save() error
	GetSection(name string) NatSection
}

type VMWareNatFile struct {
	Path     string
	Sections []*NatSection
	logger   hclog.Logger
}

type NatSection struct {
	Name    string
	Entries []*NatEntry
}

type NatEntry struct {
	Key   string
	Value string
}

// Load and parse the nat.conf file
func LoadNatFile(path string, logger hclog.Logger) (*VMWareNatFile, error) {
	if logger == nil {
		logger = hclog.New(&hclog.LoggerOptions{
			Output: hclog.DefaultOutput,
			Level:  hclog.Error,
			Name:   "vagrant-vmware-nat-file"})
	} else {
		logger = logger.Named("nat-file")
	}
	nFile := &VMWareNatFile{
		Path:   path,
		logger: logger}
	err := nFile.Load()
	if err != nil {
		return nil, err
	}
	return nFile, nil
}

// Read and parse nat.conf file
func (n *VMWareNatFile) Load() error {
	n.logger.Trace("loading", "path", n.Path)
	nFile, err := os.Open(n.Path)
	if err != nil {
		n.logger.Debug("load failure", "path", n.Path, "error", err)
		return err
	}
	defer nFile.Close()
	scanner := bufio.NewScanner(nFile)
	var section *NatSection
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			n.logger.Trace("discarding line", "reason", "comment", "line", line)
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			line = strings.TrimPrefix(line, "[")
			line = strings.TrimSuffix(line, "]")
			n.logger.Trace("new section identified", "section", line)
			section = &NatSection{Name: line}
			n.Sections = append(n.Sections, section)
			continue
		}
		if section == nil {
			n.logger.Trace("discarding line", "reason", "no section set", "line", line)
			continue
		}
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			entry := &NatEntry{Key: strings.TrimSpace(parts[0]),
				Value: strings.TrimSpace(parts[1])}
			section.Entries = append(section.Entries, entry)
		} else {
			n.logger.Trace("unknown line format", "line", line)
		}
	}
	return nil
}

// Save nat.conf file
func (n *VMWareNatFile) Save() error {
	n.logger.Debug("writing new file", "path", n.Path)
	tmpFile, err := ioutil.TempFile(path.Dir(n.Path), "vagrant-vmware-nat")
	if err != nil {
		n.logger.Debug("create failure", "path", tmpFile.Name(), "error", err)
		return err
	}
	defer tmpFile.Close()
	err = tmpFile.Chmod(0644)
	if err != nil {
		n.logger.Debug("file permission failure", "path", tmpFile.Name(), "error", err)
		return err
	}
	for _, section := range n.Sections {
		_, err = tmpFile.WriteString("[" + section.Name + "]\n")
		if err != nil {
			n.logger.Debug("section write failure", "path", tmpFile.Name(), "error", err)
			return err
		}
		for _, entry := range section.Entries {
			_, err = tmpFile.WriteString(entry.Key + " = " + entry.Value + "\n")
			if err != nil {
				n.logger.Debug("entry write failure", "path", tmpFile.Name(), "error", err)
				return err
			}
		}
	}
	tmpFile.Close()
	err = os.Rename(tmpFile.Name(), n.Path)
	if err != nil {
		n.logger.Debug("file relocate failed", "src", tmpFile.Name(), "dst", n.Path, "error", err)
		return err
	}
	n.logger.Debug("write complete", "path", n.Path)
	return nil
}

// Get NAT configuration section by name
func (n *VMWareNatFile) GetSection(name string) *NatSection {
	for _, section := range n.Sections {
		if section.Name == name {
			return section
		}
	}
	return nil
}

// Remove NAT entry from NAT section
func (s *NatSection) DeleteEntry(entry *NatEntry) error {
	for idx, sEntry := range s.Entries {
		if entry == sEntry {
			return s.DeleteEntryAt(idx)
		}
	}
	return errors.New("Failed to locate requested entry for removal")
}

// Remove NAT entry from NAT section by index
func (s *NatSection) DeleteEntryAt(idx int) error {
	if idx > len(s.Entries)-1 || idx < 0 {
		return errors.New("Invalid index value for entry deletion")
	}
	s.Entries = append(s.Entries[0:idx], s.Entries[idx+1:]...)
	return nil
}

// Check if entry matches given configuration key
func (e *NatEntry) Match(key string) bool {
	return e.Key == key
}
