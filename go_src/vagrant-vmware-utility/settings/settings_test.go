// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package settings

import (
	hclog "github.com/hashicorp/go-hclog"
)

func defaultSettingsLogger() hclog.Logger {
	return hclog.New(
		&hclog.LoggerOptions{
			Output: hclog.DefaultOutput,
			Level:  hclog.Trace,
			Name:   "vagrant-vmware-settings-test"})
}
