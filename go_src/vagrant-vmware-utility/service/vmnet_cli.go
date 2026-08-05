// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package service

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

type VmnetCli interface {
	Start() (err error)
	Stop() (err error)
	Status() bool
	Restart() (err error)
	Configure(path string) (err error)
}

type VmnetCliExe struct {
	ExePath  string
	Services VmwareServices
	logger   hclog.Logger
}

func NewVmnetCli(path string, services VmwareServices, logger hclog.Logger) (VmnetCli, error) {
	if !utility.RootOwned(path, true) {
		return nil, errors.New("Failed to locate valid vmnet executable")
	}
	logger = logger.Named("vmnetcli")
	return &VmnetCliExe{
		ExePath:  path,
		Services: services,
		logger:   logger}, nil
}

func (v *VmnetCliExe) Start() (err error) {
	if v.Status() {
		v.logger.Debug("start ignored - service running")
		return nil
	}
	v.Services.WrapOpenServices(func() {
		err = v.start()
	})
	return err
}

func (v *VmnetCliExe) Stop() (err error) {
	v.Services.WrapOpenServices(func() {
		err = v.stop()
	})
	return err
}

func (v *VmnetCliExe) Status() bool {
	cmd := exec.Command(v.ExePath, "--status")
	if utility.Execute(cmd) == 0 {
		v.logger.Debug("service status", "state", "running")
		return true
	}
	v.logger.Debug("service status", "state", "stopped")
	return false
}

func (v *VmnetCliExe) Restart() (err error) {
	v.Services.WrapOpenServices(func() {
		err = v.stop()
		if err != nil {
			return
		}
		err = v.start()
	})
	return err
}

func (v *VmnetCliExe) Configure(path string) (err error) {
	cmd := exec.Command(v.ExePath)
	cmd.Args = []string{v.ExePath, "--configure"}
	if runtime.GOOS == "linux" {
		if path == "" {
			v.logger.Debug("received empty path for configure, ignoring")
			return
		}
		cpy, err := v.copyFile(path)
		if err != nil {
			return err
		}
		defer os.Remove(cpy)
		v.logger.Debug("configure via migrate settings", "path", cpy)
		cmd = exec.Command(v.ExePath, "--migrate-network-settings", cpy)
	}
	v.Services.WrapOpenServices(func() {
		_ = v.stop()
		v.logger.Debug("configuring service")
		exitCode, out := utility.ExecuteWithOutput(cmd)
		if exitCode != 0 {
			v.logger.Debug("service configure failed", "exitcode", exitCode)
			v.logger.Trace("service failure", "output", out)
			err = errors.New("Failed to configure vmnet service")
		}
	})
	return err
}

func (v *VmnetCliExe) stop() (err error) {
	v.logger.Debug("stopping service")
	cmd := exec.Command(v.ExePath, "--stop")
	exitCode, out := utility.ExecuteWithOutput(cmd)
	if exitCode != 0 {
		v.logger.Debug("service stop failed", "exitcode", exitCode)
		v.logger.Trace("service failure", "output", out)
		err = errors.New("Failed to stop vmnet service")
	}
	// Ensure things are dead
	cmd = exec.Command("/usr/bin/pkill", "vmnet-natd", "vmnet-bridge", "vmnet-dhcpd")
	exitCode, _ = utility.ExecuteWithOutput(cmd)
	v.logger.Trace("service orphan cleanup", "exitcode", exitCode)
	return err
}

func (v *VmnetCliExe) start() (err error) {
	v.logger.Debug("starting service")
	cmd := exec.Command(v.ExePath, "--start")
	exitCode, out := utility.ExecuteWithOutput(cmd)
	if exitCode != 0 {
		v.logger.Debug("service start failed", "exitcode", exitCode)
		v.logger.Trace("service failure", "output", out)
		err = errors.New("Failed to start vmnet service")
	}
	return err
}

func (v *VmnetCliExe) copyFile(fpath string) (tpath string, err error) {
	dst, err := ioutil.TempFile(path.Dir(fpath), "vagrant-vmnet-temp")
	if err != nil {
		v.logger.Error("failed to create temporary file", "error", err)
		return tpath, err
	}
	defer dst.Close()
	err = dst.Chmod(0644)
	if err != nil {
		v.logger.Error("failed to modify temporary file permissions", "error", err,
			"path", dst.Name())
		return tpath, err
	}
	src, err := os.Open(fpath)
	if err != nil {
		v.logger.Error("failed to open source file for copy", "path", fpath, "error", err)
		return tpath, err
	}
	defer src.Close()
	_, err = io.Copy(dst, src)
	if err != nil {
		v.logger.Error("failed to copy source file", "path", fpath, "error", err)
		return tpath, err
	}
	return dst.Name(), err
}
