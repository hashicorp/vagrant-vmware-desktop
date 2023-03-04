// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// +build !windows

package utility

import (
	"os"
	"path/filepath"
	"strings"
)

func installDirectory() string {
	idir := "/opt/vagrant-vmware-desktop"
	exePath, err := os.Executable()
	if err == nil && !strings.HasPrefix(exePath, idir) {
		idir = filepath.Dir(exePath)
	}
	return idir
}
