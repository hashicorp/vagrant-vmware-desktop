// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utility

import (
	"path/filepath"
)

func InstallDirectory() string {
	return installDirectory()
}

func DirectoryFor(thing string) string {
	return filepath.Join(installDirectory(), thing)
}
