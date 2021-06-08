package service

import (
	hclog "github.com/hashicorp/go-hclog"
)

type VmwareServices interface {
	WrapOpenServices(func())
}

func NewVmwareServices(path string, logger hclog.Logger) (VmwareServices, error) {
	return buildVmwareServices(path, logger)
}
