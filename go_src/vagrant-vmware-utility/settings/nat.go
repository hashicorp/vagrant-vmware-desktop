// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"sync"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

type NatInfo struct {
	Fwds []*utility.PortFwd `json:"fwds"`
}

type NAT struct {
	Path   string
	info   NatInfo
	logger hclog.Logger
	access sync.Mutex
}

func LoadNATSettings(path string, logger hclog.Logger) (nat *NAT, err error) {
	logger = logger.Named("nat")
	nat = &NAT{
		Path: path,
		info: NatInfo{
			Fwds: []*utility.PortFwd{}},
		logger: logger}
	err = nat.Init()
	return nat, err
}

func (n *NAT) PortFwds() []*utility.PortFwd {
	return n.info.Fwds
}

func (n *NAT) Init() error {
	if !n.exists() {
		n.logger.Debug("nat configuration file does not exist - creating", "path", n.Path)
		return n.Save()
	}
	return n.Reload()
}

func (n *NAT) Clear() {
	n.access.Lock()
	defer n.access.Unlock()
	n.info.Fwds = []*utility.PortFwd{}
}

func (n *NAT) MultiAdd(fwds []*utility.PortFwd) error {
	for _, fwd := range fwds {
		if err := n.Add(fwd); err != nil {
			return err
		}
	}
	return nil
}

func (n *NAT) Add(fwd *utility.PortFwd) error {
	n.access.Lock()
	defer n.access.Unlock()
	cfwd := n.conflict(fwd)
	if cfwd != nil {
		n.logger.Warn("port forward addition conflict", "add", fwd, "existing", cfwd)
		cidx, err := n.index(cfwd)
		if err != nil {
			return fmt.Errorf("Failed to persist NAT metadata: %s", err)
		}
		n.logger.Trace("port forward removal", "remove", cfwd)
		n.info.Fwds = append(n.info.Fwds[0:cidx], n.info.Fwds[cidx+1:]...)
	}
	n.info.Fwds = append(n.info.Fwds, fwd)
	n.logger.Trace("port forward addition", "add", fwd)
	return nil
}

func (n *NAT) Remove(fwd *utility.PortFwd) error {
	n.access.Lock()
	defer n.access.Unlock()
	cfwd := n.conflict(fwd)
	if cfwd == nil {
		n.logger.Trace("port forward removal not found - noop", "remove", fwd)
		return nil
	}
	cidx, err := n.index(cfwd)
	if err != nil {
		return err
	}
	n.logger.Trace("port forward removal", "remove", cfwd)
	n.info.Fwds = append(n.info.Fwds[0:cidx], n.info.Fwds[cidx+1:]...)
	return nil
}

func (n *NAT) Reload() error {
	n.access.Lock()
	defer n.access.Unlock()
	if !n.exists() {
		n.logger.Debug("no nat file to reload - clearing")
		n.info = NatInfo{
			Fwds: []*utility.PortFwd{}}
		return nil
	}
	var info NatInfo
	data, err := os.ReadFile(n.Path)
	if err != nil {
		n.logger.Error("failed to read nat file", "error", err, "path", n.Path)
		return err
	}
	if err := json.Unmarshal(data, &info); err != nil {
		n.logger.Error("failed to load nat data", "error", err)
		if renameErr := os.Rename(n.Path, n.Path+".invalid"); renameErr != nil {
			n.logger.Error("failed to move invalid nat settings file", "error", renameErr)
			return err
		}
		n.logger.Warn("moved invalid nat settings file, clearing", "invalid-path", n.Path+".invalid")

		return nil
	}
	n.info = info
	return nil
}

func (n *NAT) Save() error {
	n.access.Lock()
	defer n.access.Unlock()
	if !n.exists() {
		if err := os.MkdirAll(path.Dir(n.Path), 0755); err != nil {
			n.logger.Error("failed to create parent directory", "error", err, "path", n.Path)
			return err
		}
	}
	data, err := json.MarshalIndent(n.info, "", "  ")
	if err != nil {
		n.logger.Error("failed to dump nat data", "error", err)
		return err
	}
	f, err := os.CreateTemp(path.Dir(n.Path), "nat")
	if err != nil {
		n.logger.Error("failed to create file", "error", err, "path", f.Name())
		return err
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		n.logger.Error("failed to write file", "error", err, "path", f.Name())
		return err
	}
	if err := f.Close(); err != nil {
		n.logger.Error("failed closing new file", "error", err, "path", f.Name())
		return err
	}
	if err := os.Rename(f.Name(), n.Path); err != nil {
		n.logger.Error("failed to save file", "error", err, "path", n.Path)
		return err
	}
	n.logger.Debug("NAT settings have been saved")
	return nil
}

func (n *NAT) exists() bool {
	if _, err := os.Stat(n.Path); err == nil {
		return true
	}
	return false
}

func (n *NAT) index(fwd *utility.PortFwd) (int, error) {
	for idx, item := range n.info.Fwds {
		if fwd == item {
			return idx, nil
		}
	}
	return -1, errors.New("failed to locate port forward")
}

func (n *NAT) conflict(fwd *utility.PortFwd) *utility.PortFwd {
	for _, item := range n.info.Fwds {
		if item.HostPort == fwd.HostPort && item.Protocol == fwd.Protocol {
			return item
		}
	}
	return nil
}
