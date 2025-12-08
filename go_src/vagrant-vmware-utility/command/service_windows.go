// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/service"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const WINDOWS_SERVICE_NAME = "VagrantVMware"

func (c *ServiceInstallCommand) install() error {
	exePath, err := os.Executable()
	if err != nil {
		c.logger.Debug("path detection failure", "error", err)
		return err
	}
	config, err := c.writeConfig("")
	if err != nil {
		c.logger.Debug("failed to write configuration file", "path", config, "error", err)
		return err
	}
	log := path.Join(utility.DirectoryFor("logs"), "utility.log")
	m, err := service.ManagerConnect(c.logger)
	if err != nil {
		c.logger.Debug("service manager connect failure", "error", err)
		return err
	}
	defer m.Disconnect()
	cmd := fmt.Sprintf(`%s service run -config-file="%s" -log-file="%s"`,
		exePath, config, log)
	s, err := m.Get(WINDOWS_SERVICE_NAME)
	if err == nil {
		defer s.Close()
	} else {
		c.logger.Trace("service not found, creating...")
		s, err = m.Manager.CreateService(WINDOWS_SERVICE_NAME, exePath,
			mgr.Config{
				StartType:      mgr.StartAutomatic,
				DisplayName:    c.Name,
				Description:    "Vagrant VMware Utility REST API",
				BinaryPathName: exePath})
		if err != nil {
			c.logger.Debug("service create failure", "name", WINDOWS_SERVICE_NAME,
				"error", err)
			return errors.New("failed to create service")
		}
		err = eventlog.InstallAsEventCreate(WINDOWS_SERVICE_NAME,
			eventlog.Error|eventlog.Warning|eventlog.Info)
		if err != nil {
			c.logger.Trace("eventlog setup failure, deleting service", "error", err)
			eventlog.Remove(WINDOWS_SERVICE_NAME)
			s.Delete()
			return errors.New("failed to configure service")
		}
	}
	if err = m.StopSvc(s); err != nil {
		return errors.New("failed to stop service for update")
	}
	scon, err := s.Config()
	if err != nil {
		c.logger.Debug("service config fetch failure", "name", WINDOWS_SERVICE_NAME,
			"error", err)
		return errors.New("failed to fetch service configuration for update")
	}
	scon.BinaryPathName = cmd
	if err = s.UpdateConfig(scon); err != nil {
		c.logger.Debug("service update failure", "name", WINDOWS_SERVICE_NAME,
			"error", err)
		return errors.New("failed to update service")
	}
	c.logger.Trace("existing service has been updated")
	err = m.StartSvc(s)
	if err != nil {
		c.logger.Debug("service start failure", "error", err)
		return errors.New("failed to start the service")
	}
	c.logger.Trace("service installed", "name", WINDOWS_SERVICE_NAME)
	return nil
}

func (c *ServiceInstallCommand) print() error {
	return errors.New("Service setup printing unavailable on Windows")
}

func (c *ServiceUninstallCommand) uninstall() error {
	m, err := service.ManagerConnect(c.logger)
	if err != nil {
		c.logger.Debug("service manager connect failure", "error", err)
		return errors.New("failed to connect to service manager")
	}
	defer m.Disconnect()
	s, err := m.Get(WINDOWS_SERVICE_NAME)
	if err != nil {
		c.logger.Warn("failed to locate utility service", "error", err)
		return nil
	}
	defer s.Close()
	if err := m.StopSvc(s); err != nil {
		c.logger.Trace("service stop failure", "error", err)
		return err
	}
	if err := s.Delete(); err != nil {
		c.logger.Debug("service delete failure", "name", WINDOWS_SERVICE_NAME, "error", err)
		return errors.New("failed to delete service")
	}
	if err := eventlog.Remove(WINDOWS_SERVICE_NAME); err != nil {
		c.logger.Warn("service eventlog delete failure", "name", WINDOWS_SERVICE_NAME,
			"error", err)
	}
	c.logger.Trace("service uninstalled", "name", WINDOWS_SERVICE_NAME)
	return nil
}
