// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package driver

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	hclog "github.com/hashicorp/go-hclog"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/util"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
	"golang.org/x/crypto/bcrypt"
)

var Shutdown sync.Cond

type Client interface {
	Do(req *http.Request) (r *http.Response, err error)
}

type VmrestDriver struct {
	BaseDriver
	client      Client
	ctx         context.Context
	isBigSurMin bool
	fallback    Driver
	vmrest      *vmrest
	logger      hclog.Logger
}

type vmrest struct {
	access      sync.Mutex
	activity    chan struct{}
	command     *exec.Cmd
	config_path string
	ctx         context.Context
	home        string
	logger      hclog.Logger
	path        string
	password    string
	port        int
	username    string
}

const lowers = "abcdefghijklmnopqrstuvwxyz"
const uppers = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
const numbers = "0123456789"
const symbols = "!#$%&'()*+,-./:;<=>?@[]^_`{|}~"

const HOME_DIR_ENV = "HOME"
const VMREST_VERSION_CONSTRAINT = ">= 1.2.0"
const VMREST_URL = "http://localhost:%d/api"
const VMREST_CONFIG = ".vmrestCfg"
const WINDOWS_VMREST_CONFIG = "vmrest.cfg"
const VMREST_CONTENT_TYPE = "application/vnd.vmware.vmw.rest-v1+json"
const VMREST_VAGRANT_DESC = "vagrant: managed port"
const VMREST_KEEPALIVE_SECONDS = 300

const VMWARE_NETDEV_PREFIX = "vmnet"
const VAGRANT_NETDEV_PREFIX = "vgtnet"
const APIPA_CIDR = "169.254.0.0/16"

func (v *vmrest) Init() (err error) {
	err = v.validate()
	if err != nil {
		return
	}

	if v.isWindows() {
		// On Windows, home directory will be based on the user
		// running the command. When running as a service under
		// the SYSTEM user, the path returned by os.UserHomeDir()
		// will be incorrect due to us being a 64bit executable
		// and vmrest.exe being a 32 bit executable. The home
		// directory ends up being different for 64 and 32 bit
		// executables for the SYSTEM user, so if the SYSTEM
		// user is detected as running we must use a customized
		// path to drop the configuration file in the right place
		// On Windows, home must be %USERPROFILE%
		v.home, err = os.UserHomeDir()
		if err != nil {
			v.logger.Trace("failed to determine user home directory", "error", err)
			return
		}
		u, err := user.Current()
		if err != nil {
			v.logger.Trace("failed to determine current user", "error", err)
			return err
		}
		if strings.ToLower(u.Name) == "system" {
			_h := v.home
			v.home = strings.Replace(v.home, "system32", "SysWOW64", 1)
			v.logger.Info("modified user home directory for SYSTEM", "user", u.Name, "original", _h, "updated", v.home)
		}
		v.config_path = path.Join(v.home, WINDOWS_VMREST_CONFIG)
	} else {
		v.home, err = ioutil.TempDir("", "util")
		if err != nil {
			v.logger.Trace("failed to create configuration directory", "error", err)
			return
		}
		v.config_path = path.Join(v.home, VMREST_CONFIG)
	}
	v.username, err = v.stringgen(false, 0)
	if err != nil {
		return
	}
	v.password, err = v.stringgen(true, 0)
	if err != nil {
		return
	}
	v.port, err = v.portgen()
	if err != nil {
		return
	}
	err = v.configure()
	if err != nil {
		return
	}
	v.logger.Trace("process configuration", "home", v.home, "username", v.username,
		"password", v.password, "port", v.port)
	util.RegisterShutdownTask(v.Cleanup)
	go v.Runner()
	return
}

func (v *vmrest) Cleanup() {
	if v.isWindows() {
		v.logger.Debug("vmrest configuration not removed on Windows platform")
	} else if v.home != "" {
		v.logger.Trace("removing generated home directory", "path", v.home)
		err := os.RemoveAll(v.home)
		if err != nil {
			v.logger.Error("failed to remove generated home directory path", "path", v.home,
				"error", err)
		}
	}
}

func (v *vmrest) Active() (url string) {
	v.activity <- struct{}{}
	return v.buildURL()
}

func (v *vmrest) buildURL() string {
	return fmt.Sprintf(VMREST_URL, v.port)
}

func (v *vmrest) Username() string {
	return v.username
}

func (v *vmrest) Password() string {
	return v.password
}

func (v *vmrest) Runner() {
	for {
		select {
		case <-v.activity:
			v.logger.Trace("activity request detected")
			if v.command == nil {
				v.logger.Debug("starting the process")
				v.command = exec.Command(v.path)

				// Grab output from the process and send it to the logger.
				// Useful for debugging if something goes wrong so we can
				// see what the process is actually doing.
				stderr, err := v.command.StderrPipe()
				if err != nil {
					v.logger.Error("failed to get stderr pipe", "error", err)
					continue
				}
				stdout, err := v.command.StdoutPipe()
				if err != nil {
					v.logger.Error("failed to get stdout pipe", "error", err)
					continue
				}
				go func() {
					r := bufio.NewReader(stdout)
					for {
						l, _, err := r.ReadLine()
						if err != nil {
							v.logger.Warn("stdout pipe error", "error", err)
							return
						}
						v.logger.Info("vmrest stdout", "output", string(l))
					}
				}()
				go func() {
					r := bufio.NewReader(stderr)
					for {
						l, _, err := r.ReadLine()
						if err != nil {
							v.logger.Warn("stderr pipe error", "error", err)
							return
						}
						v.logger.Info("vmrest stderr", "output", string(l))
					}
				}()

				err = v.homedStart(v.command)
				if err != nil {
					v.logger.Error("failed to start", "error", err)
					continue
				}
				_, err = os.FindProcess(v.command.Process.Pid)
				if err != nil {
					v.logger.Error("failed to locate started vmrest process", "error", err)
					continue
				}

				// Start a cleanup function to prevent any unnoticed zombies from
				// hanging around
				go func() {
					v.command.Wait()
					v.command = nil
					v.logger.Debug("process has been completed and reaped")
				}()

				v.logger.Debug("process has been started")
			}
		case <-time.After(VMREST_KEEPALIVE_SECONDS * time.Second):
			if v.command != nil {
				v.logger.Debug("halting running process")
				v.command.Process.Kill()
			}
		case <-v.ctx.Done():
			v.logger.Warn("halting due to context done")
			if v.command != nil {
				v.command.Process.Kill()
			}
			break
		}
	}
}

func (v *vmrest) isWindows() bool {
	return strings.HasSuffix(v.path, ".exe")
}

func (v *vmrest) homedStart(cmd *exec.Cmd) error {
	v.access.Lock()
	defer v.access.Unlock()
	if !v.isWindows() {
		// Ensure our home directory is set to properly pickup config
		curHome := os.Getenv(HOME_DIR_ENV)
		err := os.Setenv(HOME_DIR_ENV, v.home)
		if err != nil {
			v.logger.Error("failed to set HOME environment variable, cannot start", "error", err)
			return err
		}
		defer os.Setenv(HOME_DIR_ENV, curHome)
	}

	return cmd.Start()
}

func (v *vmrest) configure() (err error) {
	f, err := os.OpenFile(v.config_path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		v.logger.Error("failed to create config file", "error", err)
		return errors.New("failed to configure process")
	}
	defer f.Close()
	salt, err := v.stringgen(true, 16)
	if err != nil {
		v.logger.Error("failed to create salt config", "error", err)
		return errors.New("failed to generate config information")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(salt+v.password), bcrypt.DefaultCost)
	if err != nil {
		v.logger.Error("failed to hash password", "error", err)
		return errors.New("failed to generate config hash")
	}
	_, err = f.Write([]byte(fmt.Sprintf("port=%d\r\nusername=%s\r\npassword=%s\r\nsalt=%s\r\n",
		v.port, v.username, hash, salt)))
	if err != nil {
		v.logger.Error("failed to write config file", "error", err)
		return errors.New("failed to store config")
	}
	return
}

func (v *vmrest) portgen() (int, error) {
	// Let the system generate a free port for us
	a, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		v.logger.Trace("failed to setup free port detection", "error", err)
		return 0, err
	}
	l, err := net.ListenTCP("tcp", a)
	if err != nil {
		v.logger.Trace("failed to locate free port", "error", err)
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func (v *vmrest) stringgen(syms bool, l int) (string, error) {
	var collections int
	g := strings.Builder{}
	bl, err := rand.Int(rand.Reader, big.NewInt(3))
	if l == 0 {
		l = int(bl.Int64()) + 8
	}
	g.Grow(l)
	if err != nil {
		v.logger.Trace("failed to produce random value", "error", err)
		return "", err
	}
	if syms {
		collections = 4
	} else {
		collections = 3
	}
	for i := 0; i < l; i++ {
		set := i % collections
		switch set {
		case 3:
			idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(symbols))))
			if err != nil {
				v.logger.Trace("failed to produce random index", "error", err)
				return "", err
			}
			g.WriteByte(symbols[idx.Int64()])
		case 2:
			idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(uppers))))
			if err != nil {
				v.logger.Trace("failed to produce random index", "error", err)
				return "", err
			}
			g.WriteByte(uppers[idx.Int64()])
		case 1:
			idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(numbers))))
			if err != nil {
				v.logger.Trace("failed to produce random index", "error", err)
				return "", err
			}
			g.WriteByte(numbers[idx.Int64()])
		default:
			idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(lowers))))
			if err != nil {
				v.logger.Trace("failed to produce random index", "error", err)
				return "", err
			}
			g.WriteByte(lowers[idx.Int64()])
		}
	}
	return g.String(), nil
}

func (v *vmrest) validate() error {
	if !utility.FileExists(v.path) {
		v.logger.Trace("missing vmrest executable", "path", v.path)
		return errors.New("Failed to locate the vmrest executable")
	}

	cmd := exec.Command(v.path, "-v")
	_, o := utility.ExecuteWithOutput(cmd)
	m, err := utility.MatchPattern(`vmrest (?P<version>[\d+.]+) `, o)
	if err != nil {
		v.logger.Trace("failed to determine vmrest version information", "output", o)
		return errors.New("failed to determine vmrest version")
	}
	v.logger.Trace("detected vmrest version", "version", m["version"])
	constraint, err := version.NewConstraint(VMREST_VERSION_CONSTRAINT)
	if err != nil {
		v.logger.Warn("failed to parse vmrest constraint", "constraint", VMREST_VERSION_CONSTRAINT, "error", err)
		return errors.New("failed to setup vmrest constraint for version check")
	}
	checkV, err := version.NewVersion(m["version"])
	if err != nil {
		v.logger.Warn("failed to parse vmrest version for check", "version", m["version"], "error", err)
		return errors.New("failed to parse vmrest version for validation check")
	}

	v.logger.Trace("validating vmrest version", "constraint", constraint, "version", checkV)

	if !constraint.Check(checkV) {
		v.logger.Warn("installed vmrest does not meet constraint requirements", "constraint", constraint, "version", checkV)
		return errors.New("vmrest version is incompatible")
	}

	return nil
}

func NewVmrest(ctx context.Context, vmrestPath string, logger hclog.Logger) (v *vmrest, err error) {
	logger = logger.Named("process")
	v = &vmrest{
		activity: make(chan struct{}),
		ctx:      ctx,
		logger:   logger,
		path:     vmrestPath}
	err = v.Init()
	return
}

func NewVmrestDriver(ctx context.Context, f Driver, logger hclog.Logger) (d Driver, err error) {
	logger = logger.Named("vmrest")
	i, err := f.VmwareInfo()
	if err != nil {
		logger.Warn("failed to get vmware info", "error", err)
		logger.Info("using fallback driver")
		return f, nil
	}
	if i.IsStandard() {
		logger.Warn("standard vmware license detected, using fallback")
		return f, nil
	}
	logger.Debug("attempting to setup vmrest")
	v, err := NewVmrest(ctx, f.VmwarePaths().Vmrest, logger)
	if err != nil {
		logger.Warn("failed to create vmrest driver", "error", err)
		logger.Info("using fallback driver")
		return f, nil
	}
	var b BaseDriver
	if s, ok := f.(*SimpleDriver); ok {
		b = s.BaseDriver
	} else {
		if a, ok := f.(*AdvancedDriver); ok {
			b = a.BaseDriver
		} else {
			return nil, errors.New("failed to convert to known driver type")
		}
	}

	d = &VmrestDriver{
		BaseDriver:  b,
		client:      retryablehttp.NewClient().StandardClient(),
		ctx:         ctx,
		fallback:    f,
		vmrest:      v,
		isBigSurMin: utility.IsBigSurMin(),
		logger:      logger}
	// License detection is not always correct so we need to validate
	// that networking functionality is available via the vmrest process
	logger.Debug("validating that vmrest service provides networking functionality")
	_, err = d.Vmnets()
	if err != nil {
		logger.Error("vmrest driver failed to access networking functions, using fallback",
			"status", "invalid", "error", err)
		return f, nil
	}
	logger.Debug("validation of vmrest service is complete", "status", "valid")
	return
}

func (v *VmrestDriver) Vmnets() (vmns *Vmnets, err error) {
	v.logger.Trace("requesting list of current vmnets")
	r, err := v.Do("get", "vmnet", nil)
	if err != nil {
		v.logger.Error("vmnets list request failed", "error", err)
		return
	}
	vmns = &Vmnets{}
	err = json.Unmarshal(r, vmns)
	v.logger.Trace("current vmnets request list", "vmnets", vmns, "error", err)
	return
}

func (v *VmrestDriver) AddVmnet(vnet *Vmnet) (err error) {
	v.logger.Trace("adding vmnet device", "vmnet", vnet)

	// Big Sur and beyond require using vmrest for vmnet management
	if v.isBigSurMin {
		// Check if a specific address is attempting to be set. If so,
		// we need to force an error since the subnet/mask is not available
		// for modification via the vmnet framework
		if vnet.Type != "bridged" && (vnet.Mask != "" || vnet.Subnet != "") {
			return errors.New("Networks with custom subnet/mask values are not supported on this platform")
		}
		// we need a name, so if one is not set provide one
		if vnet.Name == "" {
			if err = v.setVmnetName(vnet); err != nil {
				return
			}
		}
		var f []byte
		f, err = json.Marshal(vnet)
		if err != nil {
			v.logger.Error("failed to encode vmnet", "vmnet", vnet, "error", err)
			return
		}
		_, err = v.Do("post", "vmnets", bytes.NewBuffer(f))
		if err != nil {
			v.logger.Error("failed to create new network", "vmnet", vnet, "error", err)
		}
		return
	}
	return v.fallback.AddVmnet(vnet)
}

func (v *VmrestDriver) UpdateVmnet(vnet *Vmnet) (err error) {
	v.logger.Trace("updating vmnet device (proxy to create request)", "vmnet", vnet)
	// Big Sur and beyond require using vmrest for vmnet management and
	// vmrest does not support updating existing vmnet devices
	if v.isBigSurMin {
		return errors.New("VMware does not support updating vmnet device")
	}
	return v.fallback.UpdateVmnet(vnet)
}

func (v *VmrestDriver) DeleteVmnet(vnet *Vmnet) (err error) {
	// The vmrest interface does not provide any method for removing
	// interfaces, only creating them. We can use the fallback driver
	// here, but it may have no affect on platforms like Big Sur where
	// the VMware vmnet implementation isn't actually being used
	v.logger.Trace("deleting vmnet device", "vmnet", vnet)
	if v.isBigSurMin {
		return errors.New("VMware does not support deleting vmnet device")
	}
	return v.fallback.DeleteVmnet(vnet)
}

func (v *VmrestDriver) PortFwds(slot string) (*PortFwds, error) {
	f := &PortFwds{}
	if v.InternalPortForwarding() {
		var err error
		f.PortForwards, err = v.InternalPortFwds()
		return f, err
	}

	fwds := []*PortFwd{}
	if v.InternalPortForwarding() {
		iFwds, err := v.InternalPortFwds()
		if err != nil {
			return nil, err
		}
		fwds = append(fwds, iFwds...)
	} else {
		device := "vmnet" + slot
		if slot == "" {
			var nat *Vmnet
			nat, err := v.detectNAT(v)
			if err != nil {
				return nil, err
			}
			device = nat.Name
			slot = string(device[len(device)-1])
		}
		slotNum, err := strconv.Atoi(string(device[len(device)-1]))
		if err != nil {
			v.logger.Error("failed to parse slot number from device", "device", device, "error", err)
			return nil, errors.New("error parsing vmnet device name for slot")
		}
		v.logger.Trace("requesting list of port forwards", "device", device)
		r, err := v.Do("get", "vmnet/"+device+"/portforward", nil)
		if err != nil {
			v.logger.Error("port forwards list request failed", "error", err)
			return nil, err
		}
		tmp := map[string]interface{}{}
		err = json.Unmarshal(r, &tmp)
		if err != nil {
			v.logger.Warn("failed initial port forward parsing", "error", err)
			return nil, err
		}
		ifwds, ok := tmp["port_forwardings"].([]interface{})
		if !ok {
			v.logger.Warn("failed to convert port forwardings", "forwards", tmp["port_forwardings"])
			return nil, errors.New("failed to parse port forwards")
		}

		for _, i := range ifwds {
			fwd := i.(map[string]interface{})
			g := fwd["guest"].(map[string]interface{})
			pfwd := &PortFwd{
				Port:        int(fwd["port"].(float64)),
				Protocol:    fwd["protocol"].(string),
				Description: fwd["desc"].(string),
				SlotNumber:  slotNum,
				Guest: &PortFwdGuest{
					Ip:   g["ip"].(string),
					Port: int(g["port"].(float64))}}
			fwds = append(fwds, pfwd)
		}
	}

	for _, pfwd := range fwds {
		for _, natFwd := range v.settings.NAT.PortFwds() {
			nfwd := v.utilityToDriverFwd(natFwd)
			if pfwd.Matches(nfwd) {
				v.logger.Trace("updating port forward description", "portforward", pfwd, "description", nfwd.Description)
				pfwd.Description = nfwd.Description
			}
		}
		f.PortForwards = append(f.PortForwards, pfwd)
	}

	v.logger.Trace("current port forwards list", "portforwards", f)
	return f, nil
}

func (v *VmrestDriver) AddPortFwd(pfwds []*PortFwd) (err error) {
	v.logger.Trace("adding port forwards", "portforwards", pfwds)
	for _, fwd := range pfwds {
		fwd.Description, err = v.validatePortFwdDescription(fwd.Description)
		if err != nil {
			return err
		}
		v.logger.Trace("creating port forward", "portforward", fwd)
		// Check if we have the internal port forward service enabled, and if so
		// add the port forward there. Otherwise, call up to the vmrest service
		if v.InternalPortForwarding() {
			if err = v.AddInternalPortForward(fwd); err != nil {
				return
			}
		} else {
			f := map[string]interface{}{
				"guestIp":   fwd.Guest.Ip,
				"guestPort": fwd.Guest.Port,
				"desc":      VMREST_VAGRANT_DESC}
			body, e := json.Marshal(f)
			if e != nil {
				v.logger.Error("failed to encode portforward request", "content", fwd.
					Guest, "error", e)
				return errors.New("failed to generate port forward request")
			}
			v.logger.Trace("new port forward request", "body", string(body))
			_, err = v.Do("put", fmt.Sprintf("vmnet/vmnet%d/portforward/%s/%d",
				fwd.SlotNumber, fwd.Protocol, fwd.Port), bytes.NewBuffer(body))
			if err != nil {
				v.logger.Error("failed to create port forward", "portforward", fwd, "error", err)
				return
			}
		}
		v.logger.Info("port forward added", "portforward", fwd)
		ufwd := v.driverToUtilityFwd(fwd)
		// Ensure port forward is not already stored
		err = v.settings.NAT.Remove(ufwd)
		if err != nil {
			v.logger.Trace("failure encountered attempting to remove port forward", "portforward", ufwd, "error", err)
		}
		err = v.settings.NAT.Add(ufwd)
		if err != nil {
			v.logger.Trace("failed to store port forward in nat settings", "portforward", ufwd, "error", err)
			return errors.New("failed to persist port forward information")
		}
		err = v.settings.NAT.Save()
		if err != nil {
			v.logger.Error("failed to save port forward nat settings", "error", err)
			return errors.New("failed to store persistent port forward information")
		}
	}
	v.logger.Trace("all port forwards added", "portforwards", pfwds)
	return
}

func (v *VmrestDriver) DeletePortFwd(pfwds []*PortFwd) (err error) {
	v.logger.Trace("removing port forwards", "portforwards", pfwds)
	for _, fwd := range pfwds {
		v.logger.Trace("deleting port forward", "portforward", fwd)
		if v.InternalPortForwarding() {
			if err = v.DeleteInternalPortForward(fwd); err != nil {
				return
			}
		} else {
			_, err = v.Do("delete", fmt.Sprintf("vmnet/vmnet%d/portforward/%s/%d",
				fwd.SlotNumber, fwd.Protocol, fwd.Port), nil)
			if err != nil {
				v.logger.Error("failed to delete port forward", "portforward", fwd, "error", err)
				return
			}
		}
		v.logger.Info("port forward removed", "portforward", fwd)
		ufwd := v.driverToUtilityFwd(fwd)
		err = v.settings.NAT.Remove(ufwd)
		if err != nil {
			v.logger.Error("failed to remove port forward from nat settings", "portforward", ufwd, "error", err)
			return errors.New("failed to persist port forward removal information")
		}
		err = v.settings.NAT.Save()
		if err != nil {
			v.logger.Error("failed to save port forward nat settings", "error", err)
			return errors.New("failed to store persistent port forward information")
		}
	}
	v.logger.Trace("all port fowards removed", "portforwards", pfwds)
	return
}

func (v *VmrestDriver) LookupDhcpAddress(device string, mac string) (addr string, err error) {
	return v.fallback.LookupDhcpAddress(device, mac)
}

func (v *VmrestDriver) ReserveDhcpAddress(slot int, mac string, ip string) (err error) {
	// Big Sur does not support dhcp address reservation
	if v.isBigSurMin {
		return errors.New("DHCP reservations are not available on this platform")
	}
	v.logger.Trace("reserving dhcp address", "slot", slot, "mac", mac, "ip", ip)
	body, err := json.Marshal(map[string]string{"IP": ip})
	if err != nil {
		v.logger.Error("failed to encode dhcp reservation request", "error", err)
		return errors.New("failed to encode dhcp reservation request")
	}
	_, err = v.Do("put", fmt.Sprintf("vmnet/vmnet%d/mactoip/%s", slot, mac),
		bytes.NewBuffer(body))
	if err != nil {
		v.logger.Error("failed to create dhcp reservation", "error", err)
		return errors.New("failed to create dhcp reservation")
	}
	return
}

// All of these we pass through to the fallback driver

func (v *VmrestDriver) LoadNetworkingFile() (utility.NetworkingFile, error) {
	return v.fallback.LoadNetworkingFile()
}

func (v *VmrestDriver) VerifyVmnet() error {
	return v.fallback.VerifyVmnet()
}

// Sends a request to the vmrest service
func (v *VmrestDriver) Do(method, path string, body io.Reader) (r []byte, err error) {
	v.logger.Info("starting remote request to vmware service")
	url := strings.Join(
		[]string{
			v.vmrest.Active(),
			path}, "/")
	method = strings.ToUpper(method)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return
	}
	req.SetBasicAuth(v.vmrest.Username(), v.vmrest.Password())
	req.Header.Add("Accept", VMREST_CONTENT_TYPE)
	if body != nil {
		req.Header.Add("Content-Type", VMREST_CONTENT_TYPE)
	}
	v.logger.Debug("sending request", "method", method, "url", url)
	resp, err := v.client.Do(req.WithContext(v.ctx))
	if err != nil {
		v.logger.Warn("request failed", "error", err)
		return
	}
	defer resp.Body.Close()
	r, err = ioutil.ReadAll(resp.Body)
	v.logger.Debug("received response", "code", resp.StatusCode, "status", resp.Status, "body", string(r), "error", err)
	if resp.StatusCode > 299 {
		result := map[string]interface{}{}
		err = json.Unmarshal(r, &result)
		if err != nil {
			err = errors.New("unknown error encountered with vmrest process")
			return
		}
		msg, ok := result["Message"].(string)
		if !ok {
			err = errors.New("unknown error encountered with vmrest process")
			return
		}
		err = errors.New("failure encountered: " + msg)
	}
	return
}

// Finds a free vmnet device. Currently very stupid and does not
// match on missing devices
func (v *VmrestDriver) setVmnetName(vnet *Vmnet) (err error) {
	vmns, err := v.Vmnets()
	names := []string{}
	for _, n := range vmns.Vmnets {
		names = append(names, n.Name)
	}
	slot := freeSlot(names, []string{VMWARE_NETDEV_PREFIX})
	vnet.Name = fmt.Sprintf("vmnet%d", slot)
	return
}

func (v *VmrestDriver) deviceList(prefix string) (list []*Vmnet, err error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return
	}
	list = []*Vmnet{}
	_, filter, err := net.ParseCIDR(APIPA_CIDR)
	if err != nil {
		return
	}
	for _, i := range ifs {
		if !strings.HasPrefix(i.Name, prefix) {
			v.logger.Trace("skipping device with invalid prefix", "prefix", prefix, "device", i.Name)
			continue
		}
		vn := &Vmnet{Name: i.Name}
		addrs, err := i.Addrs()
		if err != nil {
			v.logger.Warn("failed to fetch addresses for interface", "iface", i.Name, "error", err)
			continue
		}
		for _, a := range addrs {
			ip, cidr, err := net.ParseCIDR(a.String())
			if err != nil {
				v.logger.Warn("failed to parse interface address", "address", a.String(), "error", err)
				continue
			}
			if ip.To4() == nil {
				v.logger.Trace("skipping non-ipv4 address", "address", a.String())
				continue
			}
			if filter.Contains(ip) {
				v.logger.Trace("skipping filtered address", "address", a.String())
				continue
			}
			vn.Subnet = fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3])
			vn.Mask = fmt.Sprintf("%d.%d.%d.%d", cidr.Mask[0], cidr.Mask[1], cidr.Mask[2], cidr.Mask[3])
		}
		list = append(list, vn)
	}

	return
}

func freeSlot(list []string, prefixes []string) (slot int) {
	slot = -1
	slots := []int{}
	for _, n := range list {
		for _, p := range prefixes {
			n = strings.TrimPrefix(n, p)
		}
		val, err := strconv.Atoi(n)
		if err != nil {
			continue
		}
		slots = append(slots, val)
	}
	sort.Ints(slots)
	for i := 1; i <= len(slots); i++ {
		if slots[i-1] != i {
			return i
		}
	}
	return len(slots) + 2
}
