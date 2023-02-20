// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"flag"
	"fmt"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/mitchellh/cli"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

type WindowsEventId = uint32

const (
	ServiceStart WindowsEventId = 1 << iota
	ServiceStop
	ServiceSetupFailure
	ServiceStateChangeFailure
	ServiceFailure
)

type ServiceRunCommand struct {
	RestApiCommand
}

func BuildServiceRunCommand(name string, ui cli.Ui) cli.CommandFactory {
	return func() (cli.Command, error) {
		flags := flag.NewFlagSet("service run", flag.ContinueOnError)
		data := make(map[string]interface{})
		setDefaultFlags(flags, data)

		data["port"] = flags.Int64("port", DEFAULT_RESTAPI_PORT, "Port for API to listen")
		data["driver"] = flags.String("driver", "", "Driver to use (simple or advanced)")
		data["license_override"] = flags.String("license-override", "", "Override VMware license detection (standard or professional)")

		return &ServiceRunCommand{
			RestApiCommand: RestApiCommand{
				Command: Command{
					DefaultConfig: &Config{},
					Name:          name,
					Flags:         flags,
					HelpText:      name + " service run",
					SynopsisText:  "Run the Vagrant VMware Service",
					UI:            ui,
					flagdata:      data},
				Config: &RestApiConfig{}}}, nil
	}
}

const VALID_SERVICE_COMMANDS = svc.AcceptStop | svc.AcceptShutdown

type apiservice struct {
	Command  *ServiceRunCommand
	eventlog *eventlog.Log
	logger   hclog.Logger
}

func (a *apiservice) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	// Send notification that we are starting up
	a.eventlog.Info(1, "vagrant-vmware-utility service is starting")
	changes <- svc.Status{State: svc.StartPending}
	restApi, err := a.Command.buildRestApi(a.Command.Config.Driver, a.Command.Config.Port)
	if err != nil {
		a.logger.Debug("api setup failure", "error", err)
		a.eventlog.Error(ServiceSetupFailure, fmt.Sprintf(
			"%s service setup failed: %s", a.Command.Name, err))
		return
	}
	err = restApi.Start()
	if err != nil {
		a.logger.Debug("api startup failure", "error", err)
		a.eventlog.Error(ServiceStateChangeFailure, fmt.Sprintf(
			"%s service startup failed: %s", a.Command.Name, err))
		return
	}
	a.eventlog.Info(ServiceStart, "api service is ready and accepting requests")
	changes <- svc.Status{State: svc.Running, Accepts: VALID_SERVICE_COMMANDS}
	for running := true; running; {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Stop, svc.Shutdown:
				running = false
				a.eventlog.Info(ServiceStop, "api shutdown requested")
				a.logger.Trace("api shutdown requested")
			default:
				a.logger.Trace("unexpected control request", "command", c)
			}
		}
	}
	a.eventlog.Info(ServiceStop, "api service is shutting down")
	restApi.Stop()
	<-restApi.HaltedChan
	a.eventlog.Info(ServiceStop, "api service has been halted")
	return
}

func (c *ServiceRunCommand) Run(args []string) int {
	exitCode := 1
	err := c.setup(args)
	if err != nil {
		c.UI.Error("Failed to initialize: " + err.Error())
		return exitCode
	}
	c.logger = c.logger.Named("service-run")
	interactiveSession, err := svc.IsAnInteractiveSession()
	if err != nil {
		c.logger.Debug("cannot determine if session is interactive", "error", err)
		c.UI.Error("Failed to properly setup.")
		return exitCode
	}
	if interactiveSession {
		c.UI.Error("Run command is not intended for interactive execution.")
		return exitCode
	}
	evtLog, err := eventlog.Open(WINDOWS_SERVICE_NAME)
	if err != nil {
		c.logger.Debug("service eventlog open failure", "name", WINDOWS_SERVICE_NAME,
			"error", err)
		return exitCode
	}
	defer evtLog.Close()
	evtLog.Info(ServiceStart, fmt.Sprintf("starting %s service", c.Name))
	runner := svc.Run
	err = runner(c.Name, &apiservice{
		Command:  c,
		eventlog: evtLog,
		logger:   c.logger.Named("windows")})
	if err != nil {
		evtLog.Error(ServiceFailure, fmt.Sprintf("%s service failed: %v", c.Name, err))
		return exitCode
	}
	evtLog.Info(ServiceStop, fmt.Sprintf("%s service stopped", c.Name))
	return 0
}

func (c *ServiceRunCommand) setup(args []string) (err error) {
	err = c.defaultSetup(args)
	if err != nil {
		return
	}

	var rc RestApiConfig

	if c.DefaultConfig.configFile != nil && c.DefaultConfig.configFile.RestApiConfig != nil {
		rc = *c.DefaultConfig.configFile.RestApiConfig
	}

	c.Config.Port = c.getConfigInt64("port", rc.Pport)
	c.Config.Driver = c.getConfigValue("driver", rc.Pdriver)
	c.Config.LicenseOverride = c.getConfigValue("license_override", rc.PlicenseOverride)
	c.Config.LogDisplay = c.DefaultConfig.LogFile != ""

	return
}
