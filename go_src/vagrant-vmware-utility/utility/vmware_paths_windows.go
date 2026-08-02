// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package utility

import (
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
	var access uint32 = registry.QUERY_VALUE
	progDataPath := ""

	// OPTIMIZATION 1: Prioritize native 64-bit registry (Bypasses phantom 32-bit upgrade keys)
	regKey, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\VMware, Inc.\VMware Workstation`, access|registry.WOW64_64KEY)

	// Fallback for older VMware versions using legacy 32-bit registry
	if err != nil {
		regKey, err = registry.OpenKey(registry.LOCAL_MACHINE,
			`SOFTWARE\VMware, Inc.\VMware Workstation`, access)
	}

	if err != nil {
		v.logger.Trace("failed to open registry", "error", err)
		return err
	}
	defer regKey.Close()

	regVal, _, err := regKey.GetStringValue("InstallPath")
	if err != nil {
		v.logger.Trace("failed to locate registry key", "key", "InstallPath", "error", err)
		return err
	}
	v.InstallDir = regVal

	productVersion, _, err := regKey.GetStringValue("ProductVersion")
	if err != nil {
		v.logger.Trace("failed to locate registry key", "key", "ProductVersion", "error", err)
	} else {
		v.logger.Trace("found product version", "version", productVersion)
	}

	pRegKey, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList`, registry.QUERY_VALUE)
	if err == nil {
		pRegVal, _, err := pRegKey.GetStringValue("ProgramData")
		if err == nil {
			progDataPath = pRegVal
		}
		pRegKey.Close()
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

	// OPTIMIZATION 2: Graceful binary resolution
	checkPath := func(filename string, optional bool) string {
		fullPath := filepath.Join(v.InstallDir, filename)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			if optional {
				v.logger.Trace("optional binary not found, skipping", "path", fullPath)
				return ""
			}
		}
		return fullPath
	}

	// Mandatory core binaries (Will still allow hard-failure if missing)
	v.Vmrun = checkPath("vmrun.exe", false)
	v.Vmx = checkPath(filepath.Join("x64", "vmware-vmx.exe"), false)
	v.Vnetlib = checkPath("vnetlib.exe", false)
	v.Vdiskmanager = checkPath("vmware-vdiskmanager.exe", false)

	// Optional binaries (Will gracefully degrade if Broadcom drops them)
	v.VmnetCli = checkPath("vmnetcli.exe", true)
	v.Vmrest = checkPath("vmrest.exe", true)

	return nil
}

func (v *VmwarePaths) UpdateVmwareDhcpLeasePath(version string) error {
	return nil
}
