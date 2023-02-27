// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utility

import (
	"path/filepath"
)

func installDirectory() string {
	return ExpandPath(filepath.Join("%systemdrive%", "ProgramData",
		"HashiCorp", "vagrant-vmware-desktop"))
}
