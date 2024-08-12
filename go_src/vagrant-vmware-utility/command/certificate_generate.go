// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"flag"
	"path/filepath"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

type CertificateGenerateCommand struct {
	Command
}

func BuildCertificateGenerateCommand(name string, ui cli.Ui) cli.CommandFactory {
	return func() (cli.Command, error) {
		flags := flag.NewFlagSet("certificate generate", flag.ContinueOnError)
		data := make(map[string]interface{})
		setDefaultFlags(flags, data)

		return &CertificateGenerateCommand{
			Command: Command{
				DefaultConfig: &Config{},
				Name:          name,
				Flags:         flags,
				HelpText:      name + " certificate generate",
				SynopsisText:  "Generate required certificates",
				UI:            ui,
				flagdata:      data}}, nil
	}
}

func (c *CertificateGenerateCommand) Run(args []string) int {
	exitCode := 1
	err := c.setup(args)
	if err != nil {
		c.UI.Error("Failed to initialize: " + err.Error())
		return exitCode
	}

	paths, err := utility.GetCertificatePaths()
	if err != nil {
		c.UI.Error("Certificate generation setup failed: " + err.Error())
		return exitCode
	}

	certDir := filepath.Dir(paths.Certificate)

	if err := utility.GenerateCertificate(); err != nil {
		c.UI.Error("Certificate generation failed: " + err.Error())
		return exitCode
	}

	c.UI.Info("Certificate generation complete!")
	c.UI.Output(" -> " + certDir)
	return 0
}

func (c *CertificateGenerateCommand) setup(args []string) (err error) {
	return c.defaultSetup(args)
}
