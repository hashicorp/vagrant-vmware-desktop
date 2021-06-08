package utility

import (
	"path/filepath"
)

func installDirectory() string {
	return ExpandPath(filepath.Join("%systemdrive%", "ProgramData",
		"HashiCorp", "vagrant-vmware-desktop"))
}
