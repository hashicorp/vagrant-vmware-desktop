// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package settings

import (
	"path"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

type Settings struct {
	NAT            *NAT
	PortForwarding *PortForwarding
}

func BuildSettings(logger hclog.Logger) (*Settings, error) {
	logger = logger.Named("settings")
	npath := path.Join(utility.DirectoryFor("settings"), "nat.json")
	logger.Trace("building nat settings", "path", npath)
	nat, err := LoadNATSettings(npath, logger)
	if err != nil {
		return nil, err
	}
	ppath := path.Join(utility.DirectoryFor("settings"), "portforwarding.json")
	logger.Trace("building port forwarding settings", "path", ppath)
	pfwds, err := LoadPortForwardingSettings(ppath, logger)
	if err != nil {
		return nil, err
	}

	return &Settings{
		NAT:            nat,
		PortForwarding: pfwds}, nil
}

// Save all settings. Currently this is only
// the nat settings as that's all we support.
func (s *Settings) Save() error {
	err := s.PortForwarding.Save(nil)
	if err != nil {
		return err
	}
	return s.NAT.Save()
}
