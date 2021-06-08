package service

import (
	hclog "github.com/hashicorp/go-hclog"
)

type VmwareWorkstationServices struct {
	ExePath string
}

func (v *VmwareWorkstationServices) WrapOpenServices(callback func()) {
	callback()
}

func buildVmwareServices(path string, logger hclog.Logger) (VmwareServices, error) {
	return &VmwareWorkstationServices{ExePath: path}, nil
}
