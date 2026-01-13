// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"github.com/hashicorp/cli"
)

func platformSpecificCommands(name string, ui cli.Ui, cmds map[string]cli.CommandFactory) {
	cmds["service run"] = BuildServiceRunCommand(name, ui)
}
