// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build !windows
// +build !windows

package command

import (
	"github.com/hashicorp/cli"
)

func platformSpecificCommands(name string, ui cli.Ui, cmds map[string]cli.CommandFactory) {}
