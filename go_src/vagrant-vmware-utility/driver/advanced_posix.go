// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// +build !windows

package driver

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

// When adding a port forward we use the networking file
// utility to write the new rules directly into the file.
// The vnetlib CLI does not support custom descriptions
// used for forward rules, so we just do it manually.
func (a *AdvancedDriver) AddPortFwd(pfwds []*PortFwd) error {
	netF, err := utility.LoadNetworkingFile(
		a.vmwarePaths.Networking, a.logger)
	if err != nil {
		return err
	}
	rdev := []string{}
	for _, pfwd := range pfwds {
		if a.InternalPortForwarding() {
			if err := a.AddInternalPortForward(pfwd); err != nil {
				return err
			}
		} else {
			description, err := a.validatePortFwdDescription(pfwd.Description)
			if err != nil {
				return err
			}
			pfwd.Description = description
			newPf := &utility.PortFwd{
				Device:      strconv.Itoa(pfwd.SlotNumber),
				Protocol:    pfwd.Protocol,
				Description: pfwd.Description,
				HostPort:    pfwd.Port,
				GuestIp:     pfwd.Guest.Ip,
				GuestPort:   pfwd.Guest.Port,
			}
			err = netF.AddPortFwd(newPf)
			if err != nil {
				a.logger.Debug("port forwarding failure", "error", err)
				return err
			}
			if err := a.savePortFwd(pfwd); err != nil {
				a.logger.Debug("port forward settings failure", "error", err)
				return err
			}
			device := fmt.Sprintf("vmnet%d", pfwd.SlotNumber)
			found := false
			for _, d := range rdev {
				if d == device {
					found = true
				}
			}
			if !found {
				rdev = append(rdev, device)
			}
		}
	}
	for _, device := range rdev {
		if err := a.saveAndRestartNAT(device, netF); err != nil {
			return err
		}
	}
	return nil
}

func (a *AdvancedDriver) saveAndRestartNAT(device string, netF utility.NetworkingFile) error {
	if err := a.settings.NAT.Save(); err != nil {
		a.logger.Debug("nat settings file save failure", "error", err)
		return err
	}
	if _, err := netF.Save(); err != nil {
		a.logger.Debug("network file save failure", "error", err)
		return err
	}
	return a.restartNAT(device)
}
