package driver

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	hclog "github.com/hashicorp/go-hclog"

	intsvc "github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/internal/service"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/service"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/settings"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

const PORTFWD_PREFIX = "vagrant: "
const VMWARE_VERSION_PATTERN = `(?i)VMware\s+(?P<product>[A-Za-z0-9-]+)\s+(?P<version>[\d.]+|e.x.p)\s*(?P<build>\S+)?\s*(?P<type>[A-Za-z0-9-]+)?`

type BaseDriver struct {
	Natfile          func(string) (*utility.VMWareNatFile, error)
	Networkingfile   func() (utility.NetworkingFile, error)
	Vmrun            service.Vmrun
	VmwareServices   service.VmwareServices
	vmwarePaths      *utility.VmwarePaths
	logger           hclog.Logger
	path             *string
	settings         *settings.Settings
	validated        bool
	validationReason string
	vmnet            service.VmnetCli
	vmwareInfo       *VmwareInfo
	pfwdsvc          *intsvc.PortForwarding
}

func NewBaseDriver(vmxPath *string, licenseOverride string, logger hclog.Logger) (*BaseDriver, error) {
	logger.Info("created", "vmx", vmxPath)
	paths, err := utility.LoadVmwarePaths(logger)
	if err != nil {
		logger.Error("path loading failure", "error", err)
		return nil, err
	}
	vmrun, err := service.NewVmrun(paths.Vmrun, logger)
	if err != nil {
		logger.Error("vmrun setup failure", "error", err)
		return nil, err
	}
	vmsrv, err := service.NewVmwareServices(paths.Services, logger)
	if err != nil {
		logger.Error("vmware services setup failure", "error", err)
		return nil, err
	}
	s, err := settings.BuildSettings(logger)
	if err != nil {
		logger.Error("settings setup failure", "error", err)
		return nil, err
	}
	drv := &BaseDriver{
		Vmrun:          vmrun,
		VmwareServices: vmsrv,
		vmwarePaths:    paths,
		settings:       s,
		logger:         logger,
		path:           vmxPath,
		validated:      false}
	drv.Natfile = drv.LoadNatFile
	drv.Networkingfile = drv.LoadNetworkingFile
	vmnet, err := service.NewVmnetCli(
		drv.vmwarePaths.VmnetCli,
		drv.VmwareServices,
		logger)
	if err != nil {
		logger.Error("vmnet-cli creation failure", "error", err)
		return nil, err
	}
	drv.vmnet = vmnet

	logger.Debug("loading vmware information")
	i, err := drv.VmwareInfo()
	if err != nil {
		logger.Error("failed to generate VMware installation information", "error", err)
		return nil, err
	}

	// DHCP Lease Path can vary based on version for some platforms
	err = drv.vmwarePaths.UpdateVmwareDhcpLeasePath(i.Version)
	if err != nil {
		logger.Error("dhcp path loading failure", "error", err)
		return nil, err
	}
	logger.Debug("dhcp lease file", "filepath", drv.vmwarePaths.DhcpLease)

	logger.Debug("initial vmware information loaded", "license", i.License)
	if licenseOverride != "" {
		logger.Debug("applying user defined license override", "original", i.License,
			"override", licenseOverride)
		i.License = licenseOverride
	}
	i.Normalize()
	logger.Debug("normalized vmware information", "license", i.License)
	i, err = drv.VmwareInfo()
	return drv, nil
}

func (b *BaseDriver) Settings() *settings.Settings {
	return b.settings
}

func (b *BaseDriver) Validated() bool {
	return b.validated
}

func (b *BaseDriver) ValidationReason() string {
	return b.validationReason
}

func (b *BaseDriver) VmwarePaths() *utility.VmwarePaths {
	return b.vmwarePaths
}

func (b *BaseDriver) Path() (*string, error) {
	if b.path == nil {
		return nil, errors.New("No VMX path defined for this driver")
	}
	return b.path, nil
}

func (b *BaseDriver) LoadNatFile(device string) (*utility.VMWareNatFile, error) {
	return utility.LoadNatFile(b.vmwarePaths.NatConfFile(device), b.logger)
}

func (b *BaseDriver) LoadNetworkingFile() (utility.NetworkingFile, error) {
	netF, err := utility.LoadNetworkingFile(
		b.vmwarePaths.Networking, b.logger)
	if err != nil {
		b.logger.Debug(
			"network file load failure", "path",
			b.vmwarePaths.Networking, "error", err)
		return nil, err
	}
	if err := netF.MergeFwds(b.settings.NAT.PortFwds()); err != nil {
		b.logger.Debug(
			"failed to merge port forward entries from settings",
			"error", err)
		return nil, err
	}
	return netF, nil
}

func (b *BaseDriver) EnableInternalPortForwarding() (err error) {
	defer func() {
		if err != nil {
			b.logger.Error("failed to enable internal port forwarding service", "error", err)
		}
	}()
	pfwd, err := intsvc.NewPortForwarding(b.Settings(), b.logger)
	if err != nil {
		return
	}

	b.logger.Debug("starting internal port forwarding service")
	if err = pfwd.Load(); err != nil {
		return
	}
	if err = pfwd.Start(); err != nil {
		return
	}

	b.logger.Debug("internal port forwarding service running")
	b.pfwdsvc = pfwd
	return
}

func (b *BaseDriver) InternalPortForwarding() bool {
	return b.pfwdsvc != nil
}

func (b *BaseDriver) InternalPortFwds() (fwds []*PortFwd, err error) {
	if b.pfwdsvc == nil {
		return nil, errors.New("internal port forwarding service is not enabled")
	}
	sFwds := b.pfwdsvc.Fwds()
	for i := 0; i < len(sFwds); i++ {
		pFwd := b.makePortFwd(sFwds[i].Fwd)
		fwds = append(fwds, pFwd)
	}
	return
}

func (b *BaseDriver) AddInternalPortForward(fwd *PortFwd) (err error) {
	if b.pfwdsvc == nil {
		return errors.New("internal port forwarding service is not enabled")
	}
	return b.pfwdsvc.Add(b.makeSettingsFwd(fwd))
}

func (b *BaseDriver) DeleteInternalPortForward(fwd *PortFwd) (err error) {
	if b.pfwdsvc == nil {
		return errors.New("internal port forwarding service is not enabled")
	}
	return b.pfwdsvc.Remove(b.makeSettingsFwd(fwd))
}

// Converts a settings.Forward struct into local PortFwd
func (b *BaseDriver) makePortFwd(fwd *settings.Forward) *PortFwd {
	return &PortFwd{
		Description: fwd.Description,
		Port:        fwd.Host.Port,
		Protocol:    fwd.Host.Type,
		Guest: &PortFwdGuest{
			Ip:   fwd.Guest.Host,
			Port: fwd.Guest.Port,
		},
	}
}

// Converts a local PortFwd into a settings.Forward struct
func (b *BaseDriver) makeSettingsFwd(fwd *PortFwd) *settings.Forward {
	return &settings.Forward{
		Host: &settings.Address{
			Host: "0.0.0.0", // NOTE: vmware binds forwards to all devices so mimic here
			Port: fwd.Port,
			Type: fwd.Protocol,
		},
		Guest: &settings.Address{
			Host: fwd.Guest.Ip,
			Port: fwd.Guest.Port,
			Type: fwd.Protocol,
		},
		Description: fwd.Description,
	}
}

func (b *BaseDriver) detectNAT(d Driver) (vnet *Vmnet, err error) {
	devices, err := d.Vmnets()
	if err != nil {
		b.logger.Warn("failed to fetch vmnet list for nat detection", "error", err)
		return
	}
	for i := 0; i < len(devices.Vmnets); i++ {
		n := devices.Vmnets[i]
		b.logger.Trace("inspecting device for nat support", "vmnet", n)
		if n.Type == "nat" {
			b.logger.Debug("located nat device", "vmnet", n)
			vnet = n
			break
		}
	}
	if vnet == nil {
		err = errors.New("failed to locate NAT vmnet device")
	}
	return
}

// Prune any port
func (b *BaseDriver) PrunePortFwds(pfwds func(string) (*PortFwds, error), deleter func([]*PortFwd) error) error {
	fwds, err := pfwds("")
	if err != nil {
		return err
	}
	if err != nil {
		b.logger.Debug("list running vms failure", "error", err)
		return err
	}
	delfwds := []*PortFwd{}
	for i := 0; i < len(fwds.PortForwards); i++ {
		fwd := fwds.PortForwards[i]
		if !strings.Contains(fwd.Description, "vagrant: ") {
			b.logger.Warn("prune check description no match", "wanted", "vagrant: ", "description", fwd.Description)
			continue
		}
		checkPath, chkErr := b.matchVmPath(strings.Replace(fwd.Description, "vagrant: ", "", -1))
		if chkErr == nil && b.vmAlive(checkPath) {
			continue
		}
		b.logger.Trace("prune forward - not in use", "path", checkPath, "fwd", fwd)
		delfwds = append(delfwds, fwd)
	}
	if err := deleter(delfwds); err != nil {
		b.logger.Trace("prune forward failed", "error", err)
		return err
	}
	return nil
}

// Verify the VMware networking services are up and healthy
func (b *BaseDriver) VerifyVmnet() (err error) {
	if b.vmnet.Status() {
		b.logger.Trace("vmnet services reporting as healthy")
		return nil
	}
	b.logger.Debug("ensuring vmnet service is stopped")
	_ = b.vmnet.Stop()
	b.logger.Debug("attempting to start the vmnet services")
	err = b.vmnet.Start()
	if err == nil {
		b.logger.Trace("vmnet services started")
		return nil
	}
	b.logger.Debug("running vmnet configure after failed vmnet start")
	_ = b.vmnet.Stop()
	_ = b.vmnet.Configure("")
	b.logger.Debug("attempting to start vmnet services again")
	err = b.vmnet.Start()
	if err == nil {
		b.logger.Trace("vmnet services started")
		return nil
	}
	b.logger.Debug("attempting final vmnet services start")
	_ = b.vmnet.Stop()
	err = b.vmnet.Start()
	if err != nil {
		b.logger.Debug("failed to start vmnet services")
		return err
	}
	return nil
}

// Validate the portforward description format and VMX path
func (b *BaseDriver) validatePortFwdDescription(description string) (string, error) {
	if strings.HasPrefix(description, PORTFWD_PREFIX) {
		match, err := b.matchVmPath(strings.Replace(description, PORTFWD_PREFIX, "", -1))
		if err != nil {
			return "", err
		}
		return PORTFWD_PREFIX + match, nil
	}
	b.logger.Debug("port forward description prefix invalid", "description", description)
	return "", errors.New("Invalid port forward description format")
}

// Check given path and determine if file system is case-(in)sensitive. If it
// is, return back the downcased version of the path. Otherwise, return the
// given path as valid if it exists.
func (b *BaseDriver) matchVmPath(checkPath string) (string, error) {
	lowerPath := strings.ToLower(checkPath)
	checkStat, checkErr := os.Stat(checkPath)
	lowerStat, lowerErr := os.Stat(lowerPath)
	if checkErr == nil && lowerErr != nil {
		b.logger.Trace("exact vmx path match", "path", checkPath)
		return checkPath, nil
	} else if checkErr != nil && lowerErr == nil {
		b.logger.Trace("lower vmx path match only", "path", checkPath, "lower", lowerPath)
		return "", errors.New("Failed to validate VMX path")
	} else if checkErr != nil && lowerErr != nil {
		return "", errors.New("Failed to detect VMX path")
	} else if os.SameFile(checkStat, lowerStat) {
		b.logger.Trace("exact and lower valid match (case insensitive)", "path", checkPath,
			"lower", lowerPath, "used", lowerPath)
		return lowerPath, nil
	}
	b.logger.Trace("no vmx path match found", "path", checkPath)
	return "", errors.New("VMX path provided invalid")
}

// Check if the VM at a given VMX path is alive. Since Windows filters based
// on the user running the vmrun command we use a process ID lookup instead.
func (b *BaseDriver) vmAlive(vmxPath string) bool {
	if runtime.GOOS == "windows" {
		return b.vmPidAlive(vmxPath)
	}
	return b.vmrunAlive(vmxPath)
}

func (b *BaseDriver) vmrunAlive(vmxPath string) bool {
	runningVms, err := b.Vmrun.RunningVms()
	if err != nil {
		b.logger.Error("failed to list running vms", "error", err)
		return true
	}
	for _, vm := range runningVms {
		if vmxPath == vm.Path {
			return true
		}
		if strings.ToLower(vmxPath) == strings.ToLower(vm.Path) {
			vmxStat, vmxErr := os.Stat(vmxPath)
			vmStat, vmErr := os.Stat(vm.Path)
			if vmxErr != nil || vmErr != nil {
				continue
			}
			if os.SameFile(vmxStat, vmStat) {
				return true
			}
		}
	}
	return false
}

func (b *BaseDriver) vmPidAlive(vmxPath string) bool {
	lckPaths, err := filepath.Glob(filepath.Join(filepath.Dir(vmxPath), "*.lck", "*.lck"))
	if err != nil {
		b.logger.Error("lock path detection failed during vm status check", "error", err)
		return true
	}
	if len(lckPaths) == 0 {
		b.logger.Trace("no lock path found for vmx", "path", vmxPath)
		return false
	}
	if len(lckPaths) > 1 {
		b.logger.Error("lock path detection returned multiple paths - unexpected state")
		return true
	}
	lck, err := ioutil.ReadFile(lckPaths[0])
	if err != nil {
		b.logger.Error("failed to read lock file", "error", err)
		return true
	}
	match, err := utility.MatchPattern(`\s(?P<pid>\d+)-`, string(lck))
	if err != nil {
		b.logger.Error("pid match failed for vm status check", "error", err)
		return true
	}
	pid, err := strconv.Atoi(match["pid"])
	if err != nil {
		b.logger.Error("pid type conversion failed", "error", err)
		return true
	}
	_, err = os.FindProcess(pid)
	if err == nil {
		b.logger.Trace("process was found running", "pid", pid)
		return true
	}
	b.logger.Trace("process was not found running", "pid", pid)
	return false
}

func (b *BaseDriver) utilityToDriverFwd(f *utility.PortFwd) *PortFwd {
	slot, err := strconv.Atoi(string(f.Device[len(f.Device)-1]))
	if err != nil {
		slot = -1
	}
	return &PortFwd{
		Port:        f.HostPort,
		Protocol:    f.Protocol,
		Description: f.Description,
		Guest: &PortFwdGuest{
			Ip:   f.GuestIp,
			Port: f.GuestPort},
		SlotNumber: slot}
}

func (b *BaseDriver) driverToUtilityFwd(f *PortFwd) *utility.PortFwd {
	return &utility.PortFwd{
		HostPort:    f.Port,
		Protocol:    f.Protocol,
		Description: f.Description,
		GuestIp:     f.Guest.Ip,
		GuestPort:   f.Guest.Port,
		Device:      fmt.Sprintf("vmnet%d", f.SlotNumber)}
}
