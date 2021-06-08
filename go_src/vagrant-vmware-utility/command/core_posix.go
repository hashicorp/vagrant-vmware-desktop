// +build !windows

package command

import (
	"github.com/mitchellh/cli"
)

func platformSpecificCommands(name string, ui cli.Ui, cmds map[string]cli.CommandFactory) {}
