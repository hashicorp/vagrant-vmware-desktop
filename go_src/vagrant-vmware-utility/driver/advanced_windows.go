// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package driver

import (
	"fmt"
	"golang.org/x/sys/windows/registry"
	"strconv"
	"strings"
)

const VMNETCONFIG_REGISTRY_PATH = `SOFTWARE\VMware, Inc.\VMnetLib\VMnetConfig`

func (a *AdvancedDriver) AddPortFwd(pfwds []*PortFwd) error {
	rdev := []string{}
	for _, pfwd := range pfwds {
		if a.InternalPortForwarding() {
			if err := a.AddInternalPortForward(pfwd); err != nil {
				return err
			}
		} else {
			device := fmt.Sprintf("vmnet%d", pfwd.SlotNumber)
			fwdPath := VMNETCONFIG_REGISTRY_PATH + `\` + device + `\NAT\`
			if pfwd.Protocol == "udp" {
				fwdPath = fwdPath + `UDPForward`
			} else {
				fwdPath = fwdPath + `TCPForward`
			}
			description, err := a.validatePortFwdDescription(pfwd.Description)
			if err != nil {
				return err
			}
			a.logger.Trace("adding port forward", "device", device, "port", pfwd.Port,
				"registry-path", fwdPath)
			access := a.registryAccess(registry.ALL_ACCESS)
			regKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE, fwdPath, access)
			if err != nil {
				a.logger.Trace("failed to open registry", "path", fwdPath, "error", err)
				if a.registryTakeOwnership(registry.LOCAL_MACHINE, strings.Replace(fwdPath, "SOFTWARE", `SOFTWARE\WOW6432Node`, 1)) {
					access := a.registryAccess(registry.ALL_ACCESS)
					regKey, _, err = registry.CreateKey(registry.LOCAL_MACHINE, fwdPath, access)
					if err != nil {
						return err
					}
				} else {
					return err
				}
			}
			hostPort := strconv.Itoa(pfwd.Port)
			guestPort := strconv.Itoa(pfwd.Guest.Port)
			err = regKey.SetStringValue(hostPort, pfwd.Guest.Ip+":"+guestPort)
			if err != nil {
				a.logger.Trace("failed to set port forward", "path", fwdPath, "error", err)
				return err
			}
			err = regKey.SetStringValue(hostPort+"Description", description)
			if err != nil {
				a.logger.Trace("failed to set port forward description", "path", fwdPath, "error", err)
				return err
			}
			if err := a.savePortFwd(pfwd); err != nil {
				a.logger.Debug("port forward settings failure", "error", err)
				return err
			}
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
		if err := a.restartNAT(device); err != nil {
			return err
		}
	}
	return nil
}
