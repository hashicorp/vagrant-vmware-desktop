// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"flag"
	"path/filepath"
	"runtime"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

const RUNIT_DIR = "/etc/sv"

type ServiceInstallCommand struct {
	Command
	Config *ServiceInstallConfig
}

type ServiceInstallConfig struct {
	Driver          string
	LicenseOverride string
	Init            string // used on linux (style)
	Port            int64
	RunitDir        string // used on linux
	Print           bool   // used for init printing
	ExePath         string // used for init printing
	ConfigPath      string // used for init printing
	ConfigWrite     string // used for init printing

	Pdriver          *string `hcl:"driver"`
	PlicenseOverride *string `hcl:"license_override"`
	Pinit            *string `hcl:"init"`      // used on linux (style)
	PrunitDir        *string `hcl:"runit_dir"` // used on linux
	Pport            *int64  `hcl:"port"`
}

func (s *ServiceInstallConfig) Prepare() {
	s.Pdriver = &s.Driver
	s.PlicenseOverride = &s.LicenseOverride
	s.Pinit = &s.Init
	s.PrunitDir = &s.RunitDir
	s.Pport = &s.Port
}

// This is used for when we want to write
// configuration out to a config file
type ServiceInstallConfigFile struct {
	Service *ServiceInstallConfig `hcl:"service"`
}

func (s *ServiceInstallConfigFile) Prepare() {
	if s.Service != nil {
		s.Service.Prepare()
	}
}

func BuildServiceInstallCommand(name string, ui cli.Ui) cli.CommandFactory {
	return func() (cli.Command, error) {
		flags := flag.NewFlagSet("service install", flag.ContinueOnError)
		data := make(map[string]interface{})
		setDefaultFlags(flags, data)

		if runtime.GOOS != "windows" {
			data["runit_sv"] = flags.String("runit-sv", RUNIT_DIR, "Path to runit sv directory")
			data["init"] = flags.String("init-style", "", "Init in use (systemd, runit, sysv)")
		}
		data["port"] = flags.Int64("port", DEFAULT_RESTAPI_PORT, "Port for API to listen")
		data["driver"] = flags.String("driver", "", "Driver to use (simple, advanced, or vmrest)")
		data["license_override"] = flags.String("license-override", "", "Override VMware license detection (standard or professional)")
		data["print"] = flags.Bool("print", false, "Print init file to STDOUT")
		data["exe_path"] = flags.String("exe-path", "", "Path used for executable (used for print only)")
		data["config_path"] = flags.String("config-path", "", "Path for configuration file (used for print only)")
		data["config_write"] = flags.String("config-write", "./service.hcl", "Path to write configuration file (used for print only)")

		return &ServiceInstallCommand{
			Command: Command{
				DefaultConfig: &Config{},
				Name:          name,
				Flags:         flags,
				HelpText:      name + " service install",
				SynopsisText:  "Install service script",
				UI:            ui,
				flagdata:      data},
			Config: &ServiceInstallConfig{}}, nil
	}
}

func (c *ServiceInstallCommand) Run(args []string) int {
	exitCode := 1
	err := c.setup(args)
	if err != nil {
		c.UI.Error("Failed to initialize: " + err.Error())
		return exitCode
	}
	if c.Config.Print {
		err = c.print()
		if err != nil {
			c.UI.Error("Failed to print service: " + err.Error())
			return exitCode
		}
	} else {
		err = c.install()
		if err != nil {
			c.UI.Error("Failed to install service: " + err.Error())
			return exitCode
		}
		c.UI.Info("Service has been installed!")
	}

	return 0
}

func (c *ServiceInstallCommand) setup(args []string) (err error) {
	err = c.defaultSetup(args)
	if err != nil {
		return
	}

	var sc ServiceInstallConfig

	if c.DefaultConfig.configFile != nil && c.DefaultConfig.configFile.ServiceInstallConfig != nil {
		sc = *c.DefaultConfig.configFile.ServiceInstallConfig
	}

	if runtime.GOOS != "windows" {
		c.Config.Init = c.getConfigValue("init", sc.Pinit)
		c.Config.RunitDir = c.getConfigValue("runit_sv", sc.PrunitDir)
	}
	c.Config.Port = c.getConfigInt64("port", sc.Pport)
	c.Config.Driver = c.getConfigValue("driver", sc.Pdriver)
	c.Config.LicenseOverride = c.getConfigValue("license_override", sc.PlicenseOverride)
	c.Config.Print = c.getConfigBool("print", nil)
	c.Config.ExePath = c.getConfigValue("exe_path", nil)
	c.Config.ConfigPath = c.getConfigValue("config_path", nil)
	c.Config.ConfigWrite = c.getConfigValue("config_write", nil)

	return
}

func (c *ServiceInstallCommand) writeConfig(fpath string) (cpath string, err error) {
	if fpath != "" {
		cpath = fpath
	} else {
		cpath = filepath.Join(utility.DirectoryFor("config"), "service.hcl")
	}
	config := ConfigFile{Config: &Config{}}
	if c.DefaultConfig.Debug {
		config.Config.Pdebug = &c.DefaultConfig.Debug
	}
	if c.DefaultConfig.Level == "" {
		c.DefaultConfig.Level = "info"
	}
	config.Config.Plevel = &c.DefaultConfig.Level
	if c.DefaultConfig.LogFile != "" {
		config.Config.PlogFile = &c.DefaultConfig.LogFile
	}
	if c.DefaultConfig.LogAppend {
		config.Config.PlogAppend = &c.DefaultConfig.LogAppend
	}
	config.RestApiConfig = &RestApiConfig{
		Pport: &c.Config.Port}
	if c.Config.Driver != "" {
		config.RestApiConfig.Pdriver = &c.Config.Driver
	}
	if c.Config.LicenseOverride != "" {
		config.RestApiConfig.PlicenseOverride = &c.Config.LicenseOverride
	}
	err = utility.WriteConfigFile(cpath, config)
	if err != nil {
		c.logger.Debug("failed to create configuration file", "path", cpath, "error", err)
		return
	}
	return
}
