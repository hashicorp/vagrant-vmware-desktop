// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utility

import (
	hclog "github.com/hashicorp/go-hclog"
)

func defaultUtilityLogger() hclog.Logger {
	return hclog.New(
		&hclog.LoggerOptions{
			Output: hclog.DefaultOutput,
			Level:  hclog.Debug,
			Name:   "vagrant-vmware-utility-test"})
}
