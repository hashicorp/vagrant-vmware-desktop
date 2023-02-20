// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package service

import (
	"errors"
	"os/exec"
	"strings"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

type Vmrun interface {
	RunningVms() ([]*Vm, error)
}

type VmrunExe struct {
	exePath string
	logger  hclog.Logger
}

type Vm struct {
	Path  string
	vmrun Vmrun
}

func NewVmrun(path string, logger hclog.Logger) (Vmrun, error) {
	if !utility.RootOwned(path, true) {
		return nil, errors.New("Failed to locate valid vmrun executable")
	}
	logger = logger.Named("vmrun")
	return &VmrunExe{
		exePath: path,
		logger:  logger}, nil
}

func (v *VmrunExe) RunningVms() ([]*Vm, error) {
	result := []*Vm{}
	cmd := exec.Command(v.exePath, "list")
	exitCode, out := utility.ExecuteWithOutput(cmd)
	if exitCode != 0 {
		v.logger.Debug("vmrun list failed", "exitcode", exitCode)
		v.logger.Trace("vmrun list failed", "output", out)
		return result, errors.New("Failed to list running VMs")
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		v.logger.Trace("vmrun path check", "path", line)
		if utility.FileExists(line) {
			v.logger.Trace("vmrun path valid", "path", line)
			result = append(result, &Vm{Path: line, vmrun: v})
		}
	}
	return result, nil
}
