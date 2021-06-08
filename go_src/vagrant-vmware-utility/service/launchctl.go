package service

import (
	"errors"
	"fmt"
	"os/exec"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

const LAUNCHCTL_PATH = `/bin/launchctl`

type Launchctl interface {
	Load(servicePath string) (err error)
	Unload(servicePath string) (err error)
}

type LaunchctlExe struct {
	logger hclog.Logger
}

func NewLaunchctl(logger hclog.Logger) (Launchctl, error) {
	if !utility.RootOwned(LAUNCHCTL_PATH, true) {
		return nil, errors.New("Failed to locate valid launchctl executable")
	}
	logger = logger.Named("launchctl")
	return &LaunchctlExe{logger: logger}, nil
}

func (l *LaunchctlExe) Load(servicePath string) error {
	l.logger.Debug("loading service", "path", servicePath)
	lcmd, err := l.launchctl("load", servicePath)
	if err != nil {
		l.logger.Debug("command generation failure", "error", err)
		return err
	}
	exitCode, out := utility.ExecuteWithOutput(lcmd)
	if exitCode != 0 {
		l.logger.Debug("service load failure", "exitcode", exitCode, "output", out)
		return errors.New(fmt.Sprintf(
			"Failed to load service: %s", servicePath))
	}
	l.logger.Debug("service loaded", "path", servicePath)
	return nil
}

func (l *LaunchctlExe) Unload(servicePath string) error {
	l.logger.Debug("unloading service", "path", servicePath)
	lcmd, err := l.launchctl("unload", servicePath)
	if err != nil {
		l.logger.Debug("command generation failure", "error", err)
		return err
	}
	exitCode, out := utility.ExecuteWithOutput(lcmd)
	if exitCode != 0 {
		l.logger.Debug("service unload failure", "exitcode", exitCode, "output", out)
		return errors.New(fmt.Sprintf(
			"Failed to unload service: %s", servicePath))
	}
	l.logger.Debug("service unloaded", "path", servicePath)
	return nil
}

func (l *LaunchctlExe) launchctl(args ...string) (*exec.Cmd, error) {
	if !utility.RootOwned(LAUNCHCTL_PATH, true) {
		return nil, errors.New("Failed to locate valid launchctl executable")
	}
	return exec.Command(LAUNCHCTL_PATH, args...), nil
}
