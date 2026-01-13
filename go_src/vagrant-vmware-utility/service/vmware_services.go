// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

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
