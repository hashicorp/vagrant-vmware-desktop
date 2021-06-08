// +build !windows

package driver

import (
	"errors"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"

	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

// Generate current list of vmnets
func (b *BaseDriver) Vmnets() (*Vmnets, error) {
	b.logger.Info("collecting vmnets")
	netF, err := utility.LoadNetworkingFile(
		b.vmwarePaths.Networking, b.logger)
	if err != nil {
		b.logger.Debug(
			"network file load failure", "path",
			b.vmwarePaths.Networking, "error", err)
		return nil, err
	}
	vmnets := &Vmnets{Num: len(netF.Devices)}

	for _, device := range netF.Devices {
		vn := &Vmnet{Name: device.Name}
		if device.Dhcp {
			vn.Dhcp = "yes"
		} else {
			vn.Dhcp = "no"
		}
		if device.Nat {
			vn.Type = "nat"
		}
		if device.HostonlySubnet != "" {
			if vn.Type == "" {
				vn.Type = "hostOnly"
			}
			vn.Subnet = device.HostonlySubnet
			vn.Mask = device.HostonlyNetmask
		}
		if vn.Type == "" {
			vn.Type = "bridged"
		}
		vmnets.Vmnets = append(vmnets.Vmnets, vn)
	}
	return vmnets, nil
}

// Generate current list of port forwards for given device
func (b *BaseDriver) PortFwds(device string) (pfwds *PortFwds, err error) {
	pfwds = &PortFwds{}
	if b.InternalPortForwarding() {
		pfwds.PortForwards, err = b.InternalPortFwds()
		return
	}

	b.logger.Trace("loading networking file using dynamic loader", "loader", b.Networkingfile)
	netF, err := b.Networkingfile()
	if err != nil {
		return nil, err
	}
	fwdList := []*PortFwd{}
	for _, fwd := range netF.GetPortFwds() {
		if !fwd.Enable {
			b.logger.Trace("portfoward discard - not enabled", "port", fwd.HostPort)
			continue
		}
		if device != "" && fwd.Device != device {
			b.logger.Trace("portforward discard", "device", fwd.Device, "wanted-device", device)
			continue
		}
		slot, err := strconv.Atoi(strings.Replace(fwd.Device, "vmnet", "", -1))
		if err != nil {
			return nil, err
		}
		prtfwd := &PortFwd{
			SlotNumber:  slot,
			Port:        fwd.HostPort,
			Protocol:    fwd.Protocol,
			Description: fwd.Description,
			Guest: &PortFwdGuest{
				Ip:   fwd.GuestIp,
				Port: fwd.GuestPort}}
		fwdList = append(fwdList, prtfwd)
	}
	pfwds.PortForwards = fwdList
	pfwds.Num = len(fwdList)
	return pfwds, nil
}

// Find installed VMware product information
func (b *BaseDriver) VmwareInfo() (*VmwareInfo, error) {
	if b.vmwareInfo != nil {
		b.logger.Trace("returning cached vmware information")
		return b.vmwareInfo, nil
	}
	b.logger.Trace("vmware version check", "vmx-path", b.vmwarePaths.Vmx)
	cmd := exec.Command(b.vmwarePaths.Vmx, "-v")
	exitCode, out := utility.ExecuteWithOutput(cmd)
	if exitCode != 0 {
		b.logger.Trace("vmware version check failed", "output", out)
		return nil, errors.New("Failed attempting to check VMware version")
	}
	matches, err := utility.MatchPattern(VMWARE_VERSION_PATTERN, out)
	if err != nil {
		b.logger.Trace("vmware version match failed", "output", out, "pattern", VMWARE_VERSION_PATTERN, "error", err)
		return nil, errors.New("Failed to extract VMware version information")
	}
	v := &VmwareInfo{
		Product: matches["product"],
		Version: matches["version"],
		Build:   matches["build"],
		Type:    matches["type"]}
	cmd = exec.Command(b.vmwarePaths.Vmx, "--query-license", "LicenseEdition")
	exitCode, out = utility.ExecuteWithOutput(cmd)
	if exitCode != 0 {
		b.logger.Warn("failed to determine license edition", "output", out)
		v.License = "unknown"
	} else {
		v.License = strings.TrimSpace(out)
	}
	b.vmwareInfo = v
	return v, nil
}

// Validate the VMware installation
func (b *BaseDriver) Validate() bool {
	if runtime.GOOS == "darwin" {
		b.validateFusionApp()
	}
	// Stub the reason as it is the same for all failures
	b.validationReason = "Invalid ownership/permissions detected for VMware installation.\n" +
		"Please re-install VMware and restart the vagrant-vmware-utility\nservice."

	// Check permissions of install directory
	if !utility.RootOwned(b.VmwarePaths().InstallDir, true) {
		b.logger.Error("VMware validation failure", "cause", "invalid installation directory ownership/permissions")
		b.logger.Trace("validation failure", "path", b.VmwarePaths().InstallDir)
		b.validated = false
		return false
	}

	// Require executables and their parent directory to be owned by root
	// and not writable by group or others
	requireValidation := []string{
		b.VmwarePaths().VmnetCli,
		b.VmwarePaths().Vnetlib,
		b.VmwarePaths().Vmrun,
	}
	for _, checkPath := range requireValidation {
		if checkPath == "" {
			continue
		}
		// For darwin we can validate the signature of the executable
		if runtime.GOOS == "darwin" {
			cmd := exec.Command("/usr/bin/codesign", "--verify", "--verbose", checkPath)
			exitCode, out := utility.ExecuteWithOutput(cmd)
			if exitCode != 0 {
				b.logger.Error("VMware validation failure", "cause", out)
				b.validated = false
				return false
			}
		}
		if !utility.RootOwned(checkPath, true) {
			b.logger.Error("VMware validation failure", "cause", "invalid file ownership/permissions")
			b.logger.Trace("validation failure", "path", checkPath)
			b.validated = false
			return false
		}
		if !utility.RootOwned(path.Dir(checkPath), true) {
			b.logger.Error("VMware validation failure", "cause", "invalid file parent directory ownership/permissions")
			b.logger.Trace("validation failure", "path", path.Dir(checkPath))
			b.validated = false
			return false
		}
	}
	b.validationReason = "VMware validation successful"
	b.validated = true
	return true
}

// Validate the fusion app bundle
// NOTE: This is not actually used for validation. Instead it will log if there
//       are parts of the bundle that are invalid. In-place upgrades of Fusion will
//       break the bundle verification since new iso files will be downloaded into
//       the bundle. Use of this verification is for informational purposes only.
func (b *BaseDriver) validateFusionApp() bool {
	b.logger.Trace("running VMware Fusion app bundle validation")
	cmd := exec.Command("/usr/bin/codesign", "--verify", "--verbose", b.VmwarePaths().InstallDir)
	exitCode, out := utility.ExecuteWithOutput(cmd)
	if exitCode != 0 {
		b.logger.Warn("failed to validate VMware Fusion app bundle", "cause", out)
		return false
	}
	return true
}
