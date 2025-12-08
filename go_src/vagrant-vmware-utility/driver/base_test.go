// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package driver

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/service"
)

func TestMatchVmPathExact(t *testing.T) {
	dir, err := createFiles([]string{"test.vmx"})
	if err != nil {
		t.Errorf("Failed to create test files: %s", err)
		return
	}
	defer os.RemoveAll(dir)
	expected := path.Join(dir, "test.vmx")
	bt := &BaseDriver{
		Vmrun: &service.VmrunMock{
			Responses: []*service.VmrunResponse{
				&service.VmrunResponse{
					Vms: []*service.Vm{
						&service.Vm{Path: expected},
					},
				},
			},
		},
		logger: logger("base-driver"),
	}
	result, err := bt.matchVmPath(expected)
	if err != nil {
		t.Errorf("Unexpected error during VMX match - %s", err)
	}
	if result != expected {
		t.Errorf("Failed to match VMX path '%s' != '%s'", expected, result)
	}
}

func createFiles(names []string) (string, error) {
	dir, err := ioutil.TempDir("", "vagrant-vmware-utility")
	if err != nil {
		return dir, err
	}
	for _, name := range names {
		fpath := path.Join(dir, name)
		fhandle, err := os.Create(fpath)
		if err != nil {
			return dir, err
		}
		fhandle.Close()
	}
	return dir, err
}

func logger(name string) hclog.Logger {
	level := hclog.Error
	if os.Getenv("DEBUG") != "" {
		level = hclog.Debug
	} else if os.Getenv("DEBUG") == "trace" {
		level = hclog.Trace
	}
	return hclog.New(
		&hclog.LoggerOptions{
			Output: hclog.DefaultOutput,
			Level:  level,
			Name:   name + "-test"})
}
