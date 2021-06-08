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
