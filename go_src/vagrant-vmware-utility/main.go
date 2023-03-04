// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/command"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/util"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/version"
	"github.com/mitchellh/cli"
)

// Available commands
var Commands map[string]cli.CommandFactory

// Pretend to do stuff
func main() {
	defer cleanPanic()
	s := make(chan os.Signal)
	signal.Notify(s, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-s
		util.RunShutdownTasks()
	}()

	os.Exit(realMain())
}

// Actually do stuff
func realMain() int {
	baseUi := &cli.BasicUi{Writer: os.Stdout, ErrorWriter: os.Stderr}
	ui := &cli.ColoredUi{
		ErrorColor:  cli.UiColorRed,
		InfoColor:   cli.UiColorGreen,
		OutputColor: cli.UiColorNone,
		WarnColor:   cli.UiColorYellow,
		Ui:          baseUi,
	}

	exitCode := 1

	commands := command.Commands(version.APP_NAME, ui)

	c := &cli.CLI{
		Args:     os.Args[1:],
		Commands: commands,
		Name:     version.APP_NAME,
		Version:  version.VERSION,
	}

	exitCode, err := c.Run()
	if err != nil {
		ui.Error(fmt.Sprintf("Error executing CLI: %s", err))
	}

	return exitCode
}

func cleanPanic() {
	if err := recover(); err != nil {
		if fe, ok := err.(command.ForceExit); ok {
			os.Exit(fe.ExitCode)
		}
		panic(err)
	}
}
