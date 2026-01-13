// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package service

import (
	"errors"
	"os/exec"
	"syscall"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

type VmwareFusionServices struct {
	ExePath string
	logger  hclog.Logger
}

func (v *VmwareFusionServices) WrapOpenServices(callback func()) {
	cmd := exec.Command(v.ExePath)
	v.logger.Trace("starting vmware fusion services")
	if err := cmd.Start(); err != nil {
		v.logger.Warn("failure during vmware fusion services startup", "error", err)
	}
	defer cmd.Wait()
	defer v.logger.Trace("finished vmware fusion services")
	time.Sleep(2 * time.Second)
	callback()
	if cmd.Process.Signal(syscall.Signal(0)) != nil {
		return
	}
	if cmd.Process.Signal(syscall.Signal(15)) == nil {
		v.logger.Trace("killed vmware fusion services with TERM")
		return
	}
	if cmd.Process.Signal(syscall.Signal(9)) == nil {
		v.logger.Trace("killed vmware fusion services with KILL")
		return
	}
	if cmd.Process.Signal(syscall.Signal(0)) == nil {
		v.logger.Error("failed to halt the vmware fusion services process")
	}
	return
}

func buildVmwareServices(path string, logger hclog.Logger) (VmwareServices, error) {
	if path != "" && !utility.RootOwned(path, true) {
		return nil, errors.New("Failed to locate valid vmware services executable")
	}
	logger = logger.Named("vmware-fusion-services")
	return &VmwareFusionServices{
		ExePath: path,
		logger:  logger}, nil
}
