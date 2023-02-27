// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package service

import (
	"os"
	"path/filepath"
	"syscall"

	hclog "github.com/hashicorp/go-hclog"
)

type VmwareWorkstationServices struct {
	ExePath    string
	logger     hclog.Logger
	devConfigs map[string]*VmwareDeviceConfiguration
}

type VmwareDeviceConfiguration struct {
	Mode os.FileMode
	Uid  int
	Gid  int
}

func (v *VmwareWorkstationServices) WrapOpenServices(callback func()) {
	v.cacheDevices()
	callback()
	v.remodeDevices()
}

func buildVmwareServices(path string, logger hclog.Logger) (VmwareServices, error) {
	logger = logger.Named("vmware-workstation-services")
	return &VmwareWorkstationServices{
		ExePath:    path,
		logger:     logger,
		devConfigs: map[string]*VmwareDeviceConfiguration{}}, nil
}

func (v *VmwareWorkstationServices) cacheDevices() {
	devices, err := filepath.Glob("/dev/vmnet*")
	if err != nil {
		v.logger.Warn("failed to glob vmnet devices", "error", err)
		return
	}
	for _, dpath := range devices {
		dinfo, derr := os.Stat(dpath)
		if derr != nil {
			v.logger.Warn("failed to get vmnet device mode. skipping", "path",
				dpath, "error", derr)
		}
		dconf := &VmwareDeviceConfiguration{Mode: dinfo.Mode()}
		dstat, ok := dinfo.Sys().(*syscall.Stat_t)
		if ok {
			dconf.Gid = int(dstat.Gid)
			dconf.Uid = int(dstat.Uid)
		}
		v.devConfigs[dpath] = dconf
	}
}

func (v *VmwareWorkstationServices) remodeDevices() {
	defer v.cacheDevices()
	for dpath, dconf := range v.devConfigs {
		if _, err := os.Stat(dpath); err != nil {
			v.logger.Debug("vmnet device no longer exists. skipping.",
				"path", dpath, "error", err)
			continue
		}
		err := os.Chmod(dpath, dconf.Mode)
		if err != nil {
			v.logger.Warn("failed to reset vmnet device mode", "path",
				dpath, "error", err)
		} else {
			v.logger.Trace("restored vmnet device mode", "path", dpath)
		}
		err = os.Chown(dpath, dconf.Uid, dconf.Gid)
		if err != nil {
			v.logger.Warn("failed to reset vmnet device ownership", "path",
				dpath, "error", err)
		} else {
			v.logger.Trace("restored vmnet device ownership", "path", dpath)
		}
	}
}
