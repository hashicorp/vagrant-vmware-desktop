// +build !windows

package utility

import (
	"os"
	"syscall"
)

// Check if a given file is owned by the root user, with write access restricted
// to the root user, and optionally if the given path is executable by root.
func RootOwned(checkPath string, andOperated bool) bool {
	pathInfo, err := os.Stat(checkPath)
	if err != nil {
		return false
	}
	pathStat, ok := pathInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	if pathStat.Uid != 0 {
		return false
	}
	filePerm := pathInfo.Mode().Perm()
	// Check for allowed write access
	if (filePerm & os.FileMode(0022)) != 0 {
		return false
	}

	// Check for execute permission if requested
	if andOperated && ((filePerm & os.FileMode(0100)) == 0) {
		return false
	}
	return true
}

// Check if running as root user.
func IsRoot() bool {
	return os.Geteuid() == 0
}
