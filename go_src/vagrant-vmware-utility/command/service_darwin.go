package command

import (
	"errors"
	"fmt"
	"os"

	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/service"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

// Expected variables:
// * string - executable path
// * integer - listen port
// * string - configuration file path
// * integer - listen port
// * string - log file path
// * string - log file path
const LAUNCHD_JOB = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.vagrant.vagrant-vmware-utility</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>api</string>
        <string>-port=%d</string>
        <string>-config-file=%s</string>
    </array>
    <key>Sockets</key>
    <dict>
        <key>Listeners</key>
        <dict>
            <key>SockServiceName</key>
            <string>127.0.0.1:%d</string>
        </dict>
    </dict>
    <key>KeepAlive</key>
    <dict>
        <key>PathState</key>
        <dict>
            <key>/Applications/VMware Fusion.app</key>
                <true/>
        </dict>
    </dict>
    <key>RunAtLoad</key>
        <true/>
    <key>StandardErrorPath</key>
        <string>%s</string>
    <key>StandardOutPath</key>
        <string>%s</string>
    <key>AbandonProcessGroup</key>
        <true/>
</dict>
</plist>
`

const LAUNCHD_STOP_JOB = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.vagrant.vagrant-vmware-utility-stopper</string>
    <key>ProgramArguments</key>
    <array>
        <string>/bin/launchctl</string>
        <string>stop</string>
        <string>com.vagrant.vagrant-vmware-utility</string>
    </array>
    <key>KeepAlive</key>
    <dict>
        <key>PathState</key>
        <dict>
            <key>/Applications/VMware Fusion.app</key>
              <false/>
        </dict>
    </dict>
    <key>RunAtLoad</key>
        <true/>
</dict>
</plist>
`

const LAUNCHD_STOP_JOB_PATH = `/Library/LaunchDaemons/com.vagrant.vagrant-vmware-utility-stopper.plist`
const LAUNCHD_JOB_PATH = `/Library/LaunchDaemons/com.vagrant.vagrant-vmware-utility.plist`
const SERVICE_CONFIGURATION_FILE = `/Library/Application Support/vagrant-vmware-utility/config.hcl`
const SERVICE_LOG_FILE = `/Library/Application Support/vagrant-vmware-utility/service.log`

func (c *ServiceInstallCommand) install() error {
	if utility.FileExists(LAUNCHD_JOB_PATH) {
		return errors.New("service is already installed")
	}
	exePath, err := os.Executable()
	if err != nil {
		c.logger.Error("failed to determine executable path", "error", err)
		return errors.New("failed to determine executable path")
	}
	config, err := c.writeConfig("")
	if err != nil {
		c.logger.Error("failed to create service configuration", "error", err)
		return err
	}
	launchctl, err := service.NewLaunchctl(c.logger)
	if err != nil {
		c.logger.Debug("launchctl service creation failure", "error", err)
		return err
	}
	c.logger.Trace("create service file", "path", LAUNCHD_JOB_PATH)
	lfile, err := os.Create(LAUNCHD_JOB_PATH)
	if err != nil {
		c.logger.Debug("create service file failure", "path", LAUNCHD_JOB_PATH, "error", err)
		return err
	}
	defer lfile.Close()
	port := c.Config.Port
	_, err = lfile.WriteString(fmt.Sprintf(LAUNCHD_JOB, exePath, port,
		config, port, SERVICE_LOG_FILE, SERVICE_LOG_FILE))
	if err != nil {
		c.logger.Debug("service file write failure", "path", LAUNCHD_JOB_PATH, "error", err)
		return err
	}
	lfile.Close()
	sfile, err := os.Create(LAUNCHD_STOP_JOB_PATH)
	if err != nil {
		c.logger.Debug("create service stop file failure", "path", LAUNCHD_STOP_JOB_PATH, "error", err)
		return err
	}
	defer sfile.Close()
	_, err = sfile.WriteString(LAUNCHD_STOP_JOB)
	if err != nil {
		c.logger.Debug("service file write failure", "path", LAUNCHD_STOP_JOB_PATH, "error", err)
		return err
	}
	sfile.Close()
	c.logger.Trace("loading service", "path", LAUNCHD_JOB_PATH)
	err = launchctl.Load(LAUNCHD_JOB_PATH)
	if err != nil {
		c.logger.Debug("service load failure", "path", LAUNCHD_JOB_PATH, "error", err)
		return err
	}
	c.logger.Trace("loading stopper service", "path", LAUNCHD_STOP_JOB_PATH)
	err = launchctl.Load(LAUNCHD_STOP_JOB_PATH)
	if err != nil {
		c.logger.Debug("service load failure", "path", LAUNCHD_STOP_JOB_PATH, "error", err)
		return err
	}
	return nil
}

func (c *ServiceInstallCommand) print() error {
	return errors.New("Service setup printing unavailable on darwin")
}

func (c *ServiceUninstallCommand) uninstall() error {
	if !utility.FileExists(LAUNCHD_JOB_PATH) {
		c.logger.Warn("service is not currently installed")
		return nil
	}
	launchctl, err := service.NewLaunchctl(c.logger)
	if err != nil {
		c.logger.Debug("launchctl service creation failure", "error", err)
		return err
	}
	c.logger.Trace("unloading service", "path", LAUNCHD_JOB_PATH)
	err = launchctl.Unload(LAUNCHD_JOB_PATH)
	if err != nil {
		c.logger.Debug("service unload failure", "path", LAUNCHD_JOB_PATH, "error", err)
		return err
	}
	c.logger.Trace("removing service file", "path", LAUNCHD_JOB_PATH)
	err = os.Remove(LAUNCHD_JOB_PATH)
	if err != nil {
		c.logger.Debug("service file remove failure", "path", LAUNCHD_JOB_PATH, "error", err)
		return err
	}
	c.logger.Trace("unloading stopper service", "path", LAUNCHD_STOP_JOB_PATH)
	err = launchctl.Unload(LAUNCHD_STOP_JOB_PATH)
	if err != nil {
		c.logger.Debug("service stopper unload failure", "path", LAUNCHD_STOP_JOB_PATH, "error", err)
		return err
	}
	c.logger.Trace("removing service stopper file", "path", LAUNCHD_STOP_JOB_PATH)
	err = os.Remove(LAUNCHD_STOP_JOB_PATH)
	if err != nil {
		c.logger.Debug("service file remove failure", "path", LAUNCHD_STOP_JOB_PATH, "error", err)
		return err
	}
	return nil
}
