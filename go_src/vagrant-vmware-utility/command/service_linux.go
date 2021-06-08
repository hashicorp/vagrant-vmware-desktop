package command

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

// Template for init.d script
//
// string - executable path
// string - configuration path
const SYSV_TEMPLATE = `#!/bin/bash
### BEGIN INIT INFO
# Provides:          vagrant-vmware-utility
# Required-Start:    $remote_fs $syslog
# Required-Stop:     $remote_fs $syslog
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: Vagrant VMware Utility Service
# Description:       Provides the Vagrant VMware REST API
### END INIT INFO

name=$(basename $0)
pid_file="/var/run/$name.pid"
log="/var/log/$name.log"
cmd="%s"
cmdargs="api -config-file="%s" -log-file=$log"

if [ -f /usr/lsb/init-functions ]; then
    lsb=1
    . /usr/lsb/init-functions
else
    lsb=0
    . /etc/init.d/functions
    function log_success_msg() {
        echo $1
    }
    function log_warning_msg() {
        echo $1
    }
    function log_failure_msg() {
        echo $1
    }
fi

is_running() {
    pidofproc -p $pid_file $cmd
}

case "$1" in
    start)
        if is_running; then
            log_warning_msg "Service already running"
        else
            if [ $lsb -eq 1 ]; then
                start_daemon -p $pid_file $cmd $cmdargs
            else
                daemon --pidfile=$pid_file $cmd $cmdargs
            fi
            if ! is_running; then
                log_failure_msg "Failed to start $name"
                exit 1
            fi
            log_success_msg "Started service $name"
        fi
        ;;
    stop)
        if ! is_running; then
            log_warning_msg "Service not running"
        else
            killproc -p $pid_file $cmd
            if is_running; then
                log_failure_msg "Failed to stop $name"
                exit 1
            fi
            log_success_msg "Stopped service $name"
        fi
        ;;
    restart)
        $0 stop
        if is_running; then
            log_failure_msg "Failed to restart $name"
            exit 1
        fi
        $0 start
        if ! is_running; then
            log_failure_msg "Failed to restart $name"
            exit 1
        fi
        log_success_msg "Restarted service $name"
        ;;
    force-reload)
        if ! is_running; then
            log_failure_msg "Service not running"
            exit 1
        fi
        $0 restart
        if !is_running; then
            log_failure_msg "Failed to reload $name"
            exit 1
        fi
        log_success_msg "Reloaded service $name"
        ;;
    status)
        if ! is_running; then
            exit 3
        fi
        ;;
esac

exit 0
`

// Temlate for systemd
//
// string - exectuable path
// string - configuration path
const SYSTEMD_TEMPLATE = `[Unit]
Description=Vagrant VMware Utility
After=network.target

[Service]
Type=simple
ExecStart=%s api -config-file=%s
Restart=on-abort

[Install]
WantedBy=multi-user.target
`

// Temlate for runit
//
// string - exectuable path
// string - configuration path
const RUNIT_TEMPLATE = `#!/bin/sh
exec %s api -config-file="%s"
`

const SYSV_PATH = "/etc/init.d/vagrant-vmware-utility"

func (c *Command) systemdServicePath(exePath string) string {
	return path.Join(path.Dir(exePath), c.Name+".service")
}

// Attached to generic command so both install and uninstall can access
func (c *Command) detectInit() string {
	// Get the command name for init
	exitCode, out := utility.ExecuteWithOutput(
		exec.Command("ps", "-o", "comm=", "1"))
	c.logger.Trace("init process check", "exitcode", exitCode, "output", out)
	if exitCode == 0 {
		out = strings.TrimSpace(out)
		if strings.Contains(out, "systemd") {
			return "systemd"
		} else if strings.Contains(out, "runit") {
			return "runit"
		}
	}
	// Check if sys-v directory exists
	if utility.FileExists(path.Dir(SYSV_PATH)) {
		c.logger.Trace("sysv init check", "path", path.Dir(SYSV_PATH))
		return "sysv"
	}
	return "unknown"
}

func (c *ServiceInstallCommand) print() (err error) {
	initStyle := c.Config.Init
	if initStyle == "" {
		initStyle = c.detectInit()
	}
	exePath := c.Config.ExePath
	if exePath == "" {
		exePath, err = os.Executable()
		if err != nil {
			c.logger.Error("executable path detection failure", "error", err)
			return
		}
	}
	config, err := c.writeConfig(c.Config.ConfigWrite)
	if c.Config.ConfigPath != "" {
		config = c.Config.ConfigPath
	}
	switch initStyle {
	case "sysv":
		fmt.Printf(SYSV_TEMPLATE, exePath, config)
	case "systemd":
		fmt.Printf(SYSTEMD_TEMPLATE, exePath, config)
	case "runit":
		fmt.Printf(RUNIT_TEMPLATE, exePath, config)
	default:
		return errors.New("Unknown init for installation: " + initStyle)
	}
	if err != nil {
		c.logger.Debug("service install failure", "error", err)
		return err
	}
	return nil
}

func (c *ServiceInstallCommand) install() (err error) {
	initStyle := c.Config.Init
	if initStyle == "" {
		initStyle = c.detectInit()
	}
	exePath, err := os.Executable()
	if err != nil {
		c.logger.Error("executable path detection failure", "error", err)
		return
	}
	config, err := c.writeConfig("")
	if err != nil {
		c.logger.Error("failed to create configuration file", "path", config, "error", err)
		return err
	}
	c.logger.Trace("installing service", "init", initStyle, "exe", exePath, "port", c.Config.Port)
	switch initStyle {
	case "sysv":
		err = c.installSysv(exePath, config)
	case "systemd":
		err = c.installSystemd(exePath, config)
	case "runit":
		err = c.installRunit(exePath, config)
	default:
		return errors.New("Unknown init for installation: " + initStyle)
	}
	if err != nil {
		c.logger.Debug("service install failure", "error", err)
		return err
	}
	return nil
}

func (c *ServiceInstallCommand) installSysv(exePath, configPath string) error {
	if utility.FileExists(SYSV_PATH) {
		return errors.New("service is already installed")
	}
	ifile, err := os.OpenFile(SYSV_PATH, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		c.logger.Debug("create service file failure", "path", SYSV_PATH, "error", err)
		return err
	}
	defer ifile.Close()
	_, err = ifile.WriteString(fmt.Sprintf(SYSV_TEMPLATE, exePath, configPath))
	if err != nil {
		c.logger.Debug("service file write failure", "path", SYSV_PATH, "error", err)
		return err
	}
	ifile.Close()
	exitCode, out := utility.ExecuteWithOutput(exec.Command(SYSV_PATH, "start"))
	if exitCode != 0 {
		c.logger.Debug("service start failure", "path", SYSV_PATH, "exitcode", exitCode,
			"output", out)
		return errors.New("Failed to start service")
	}
	return nil
}

func (c *ServiceInstallCommand) installSystemd(exePath, configPath string) error {
	servicePath := c.systemdServicePath(exePath)
	if utility.FileExists(servicePath) {
		return errors.New("service is already installed")
	}
	ifile, err := os.OpenFile(servicePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		c.logger.Debug("create service file failure", "path", servicePath, "error", err)
		return err
	}
	defer ifile.Close()
	_, err = ifile.WriteString(fmt.Sprintf(SYSTEMD_TEMPLATE, exePath, configPath))
	if err != nil {
		c.logger.Debug("service file write failure", "path", servicePath, "error", err)
		return err
	}
	ifile.Close()
	exitCode, out := utility.ExecuteWithOutput(exec.Command("systemctl", "enable", servicePath))
	if exitCode != 0 {
		c.logger.Debug("service enable failure", "path", servicePath, "exitcode", exitCode,
			"output", out)
		return errors.New("Failed to enable service")
	}
	exitCode, out = utility.ExecuteWithOutput(exec.Command("systemctl", "start", path.Base(servicePath)))
	if exitCode != 0 {
		c.logger.Debug("service start failure", "name", path.Base(servicePath), "exitcode", exitCode,
			"output", out)
		return errors.New("Failed to start service")
	}
	return nil
}

func (c *ServiceInstallCommand) installRunit(exePath, configPath string) error {
	svcDir := c.Config.RunitDir
	c.logger.Trace("runit service directory", "path", svcDir)
	runPath := path.Join(path.Dir(exePath), "runit", "run")
	c.logger.Trace("runit install path", "path", runPath)
	svcPath := path.Join(svcDir, c.Name)
	c.logger.Trace("runit service path", "path", svcPath)
	if utility.FileExists(svcPath) {
		return errors.New("service is already installed")
	}
	err := os.MkdirAll(svcDir, 0755)
	if err != nil {
		c.logger.Debug("service directory create failure", "path", svcDir, "error", err)
		return err
	}
	err = os.MkdirAll(path.Dir(runPath), 0755)
	if err != nil {
		c.logger.Debug("serviec run directory create failure", "path", path.Dir(runPath), "error", err)
		return err
	}
	ifile, err := os.OpenFile(runPath, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		c.logger.Debug("create service file failure", "path", runPath, "error", err)
		return err
	}
	defer ifile.Close()
	_, err = ifile.WriteString(fmt.Sprintf(RUNIT_TEMPLATE, exePath, configPath))
	if err != nil {
		c.logger.Debug("service file write failure", "path", runPath, "error", err)
		return err
	}
	ifile.Close()
	err = os.Symlink(path.Dir(runPath), svcPath)
	if err != nil {
		c.logger.Debug("service enable failure", "path", svcPath, "error", err)
		return err
	}
	return nil
}

func (c *ServiceUninstallCommand) uninstall() error {
	initStyle := c.Config.Init
	if initStyle == "" {
		initStyle = c.detectInit()
	}
	exePath, err := os.Executable()
	if err != nil {
		c.logger.Debug("path detection failure", "error", err)
		return err
	}
	c.logger.Trace("uninstalling service", "init", initStyle, "exe", exePath)
	switch initStyle {
	case "sysv":
		err = c.uninstallSysv(exePath)
	case "systemd":
		err = c.uninstallSystemd(exePath)
	case "runit":
		err = c.uninstallRunit(exePath)
	default:
		return errors.New("Unknown init for installation: " + initStyle)
	}
	if err != nil {
		c.logger.Debug("service install failure", "error", err)
		return err
	}
	return nil
}

func (c *ServiceUninstallCommand) uninstallSysv(exePath string) error {
	if !utility.FileExists(SYSV_PATH) {
		c.logger.Warn("service is not installed", "path", SYSV_PATH)
		return nil
	}
	exitCode, out := utility.ExecuteWithOutput(exec.Command(SYSV_PATH, "stop"))
	if exitCode != 0 {
		c.logger.Debug("service stop failure", "path", SYSV_PATH, "exitcode", exitCode, "output", out)
		return errors.New("Failed to stop service")
	}
	err := os.Remove(SYSV_PATH)
	if err != nil {
		c.logger.Warn("service file remove failure", "path", SYSV_PATH, "error", err)
		return err
	}
	return nil
}

func (c *ServiceUninstallCommand) uninstallSystemd(exePath string) error {
	servicePath := c.systemdServicePath(exePath)
	serviceName := path.Base(servicePath)
	// Check if service is enabled
	exitCode, out := utility.ExecuteWithOutput(exec.Command(
		"systemctl", "is-enabled", serviceName))
	c.logger.Trace("service enable check", "name", serviceName, "exitcode", exitCode, "output", out)
	if exitCode == 0 {
		exitCode, out = utility.ExecuteWithOutput(exec.Command(
			"systemctl", "disable", serviceName))
		c.logger.Trace("service disable", "name", serviceName, "exitcode", exitCode, "output", out)
		if exitCode != 0 {
			return errors.New("Failed to disable service")
		}
		// clean up systemd unit list
		utility.Execute(exec.Command("systemctl", "reset-failed"))
	}
	if utility.FileExists(servicePath) {
		err := os.Remove(servicePath)
		if err != nil {
			c.logger.Warn("service file remove failure", "path", servicePath, "error", err)
			return errors.New("failed to remove systemd unit file")
		}
	}
	return nil
}

func (c *ServiceUninstallCommand) uninstallRunit(exePath string) (err error) {
	svcDir := c.Config.RunitDir
	c.logger.Trace("runit service directory", "path", svcDir)
	runDir := path.Join(path.Dir(exePath), "runit")
	c.logger.Trace("runit install directory", "path", runDir)
	svcPath := path.Join(svcDir, c.Name)
	c.logger.Trace("runit service path", "path", svcPath)
	if !utility.FileExists(svcPath) {
		c.logger.Warn("service is not installed", "path", svcPath)
		return nil
	}
	exitCode, out := utility.ExecuteWithOutput(exec.Command("sv", "status", c.Name))
	c.logger.Trace("service status check", "name", c.Name, "exitcode", exitCode, "output", out)
	if exitCode == 0 {
		exitCode, out := utility.ExecuteWithOutput(exec.Command("sv", "shutdown", c.Name))
		if exitCode != 0 {
			c.logger.Debug("service stop failure", "name", c.Name, "exitcode", exitCode, "output", out)
			return errors.New("Failed to stop service")
		}
	}
	if utility.FileExists(runDir) {
		err = os.RemoveAll(runDir)
		if err != nil {
			c.logger.Warn("service run path remove failure", "path", runDir, "error", err)
			return errors.New("failed to remove runit run directory for service")
		}
	}
	if utility.FileExists(svcPath) {
		err := os.Remove(svcPath)
		if err != nil {
			c.logger.Warn("service path remove failure", "path", svcPath, "error", err)
			return errors.New("failed to remove runit service file")
		}
	}
	return nil
}
