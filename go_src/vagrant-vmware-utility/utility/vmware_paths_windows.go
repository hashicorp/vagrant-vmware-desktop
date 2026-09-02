// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0
package utility

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// Expands path replacing readonly environment variables
func ExpandPath(ePath string) string {
	expandedPath := strings.ToLower(ePath)
	systemDrive := os.Getenv("SystemRoot")[0:2]
	expandedPath = strings.Replace(expandedPath, "%homedrive%", os.Getenv("HOMEDRIVE"), -1)
	expandedPath = strings.Replace(expandedPath, "%systemroot%", os.Getenv("SystemRoot"), -1)
	expandedPath = strings.Replace(expandedPath, "%systemdrive%", systemDrive, -1)
	return expandedPath
}

func (v *VmwarePaths) Load() error {
	var access uint32
	progDataPath := ""
	var regKey registry.Key
	var err error

	pathsToCheck := []struct {
		path string
		flag uint32
	}{
		{`SOFTWARE\VMware, Inc.\VMware Workstation`, registry.WOW64_64KEY},
		{`SOFTWARE\VMware, Inc.\VMware Workstation`, registry.WOW64_32KEY},
		{`SOFTWARE\VMware, Inc.\VMware Player`, registry.WOW64_64KEY},
		{`SOFTWARE\VMware, Inc.\VMware Player`, registry.WOW64_32KEY},
	}

	for _, p := range pathsToCheck {
		access = registry.QUERY_VALUE | p.flag
		regKey, err = registry.OpenKey(registry.LOCAL_MACHINE, p.path, access)
		if err == nil {
			break
		}
	}

	if err != nil {
		v.logger.Trace("failed to open registry", "error", err)
		return fmt.Errorf("failed to locate VMware Workstation or Player in registry: %w", err)
	}

	defer regKey.Close()
	regVal, _, err := regKey.GetStringValue("InstallPath")
	if err != nil {
		v.logger.Trace("failed to locate registry key", "key", "InstallPath", "error", err)
		return fmt.Errorf("failed to read InstallPath string from registry: %w", err)
	}
	v.InstallDir = regVal
	pRegKey, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList`, registry.QUERY_VALUE)
	if err == nil {
		pRegVal, _, err := pRegKey.GetStringValue("ProgramData")
		if err == nil {
			progDataPath = pRegVal
		}
	}
	if progDataPath == "" {
		progDataPath = os.Getenv("ProgramData")
		if progDataPath == "" {
			progDataPath = ExpandPath(filepath.Join("%systemdrive%", "ProgramData"))
		}
	}
	progDataPath = ExpandPath(progDataPath)
	v.NatConf = filepath.Join(progDataPath, "VMware", "vmnetnat.conf")
	v.Networking = filepath.Join(progDataPath, "VMware", "netmap.conf")
	v.DhcpLease = filepath.Join(progDataPath, "VMware", "vmnetdhcp.leases")

	checkPath := func(filename string, optional bool) string {
		fullPath := filepath.Join(v.InstallDir, filename)
		if _, err := os.Stat(fullPath); err != nil {
			if optional {
				v.logger.Trace("optional binary not found, skipping", "path", fullPath)
				return ""
			}
		}
		return fullPath
	}

	// Mandatory core binaries
	v.Vmrun = checkPath("vmrun.exe", false)
	v.Vmx = checkPath(filepath.Join("x64", "vmware-vmx.exe"), false)
	v.Vnetlib = checkPath("vnetlib.exe", false)
	v.Vdiskmanager = checkPath("vmware-vdiskmanager.exe", false)

	// Optional binaries (Graceful fallback)
	v.VmnetCli = checkPath("vmnetcli.exe", true)
	v.Vmrest = checkPath("vmrest.exe", true)

	return nil
}

func (v *VmwarePaths) UpdateVmwareDhcpLeasePath(version string) error {
	return nil
}
