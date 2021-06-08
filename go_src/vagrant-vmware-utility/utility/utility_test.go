package utility

import (
	hclog "github.com/hashicorp/go-hclog"
)

func defaultUtilityLogger() hclog.Logger {
	return hclog.New(
		&hclog.LoggerOptions{
			Output: hclog.DefaultOutput,
			Level:  hclog.Error,
			Name:   "vagrant-vmware-utility-test"})
}
