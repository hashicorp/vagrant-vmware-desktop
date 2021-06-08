package utility

import (
	"os"
	"path/filepath"
	"runtime"
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
	access = registry.QUERY_VALUE
	if runtime.GOARCH == "amd64" {
		access = access | registry.WOW64_32KEY
	}
	regKey, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\VMware, Inc.\VMware Workstation`, access)
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
	v.VmnetCli = filepath.Join(v.InstallDir, "vmnetcli.exe")
	v.Vnetlib = filepath.Join(v.InstallDir, "vnetlib.exe")
	v.Vmrun = filepath.Join(v.InstallDir, "vmrun.exe")
	v.Vmrest = filepath.Join(v.InstallDir, "vmrest.exe")
	v.Vmx = filepath.Join(v.InstallDir, "x64", "vmware-vmx.exe")
	v.Vdiskmanager = filepath.Join(v.InstallDir, "vmware-vdiskmanager.exe")

	return nil
}
