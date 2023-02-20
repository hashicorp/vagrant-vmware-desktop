// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package settings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"

	hclog "github.com/hashicorp/go-hclog"
)

type Address struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	Type string `json:"type"`
}

func (a *Address) Network() string {
	return a.Type
}

func (a *Address) String() string {
	return fmt.Sprintf("%s:%d", a.Host, a.Port)
}

func (a *Address) Equal(a1 *Address) bool {
	return a.Host == a1.Host &&
		a.Port == a1.Port &&
		a.Type == a1.Type
}

type Forward struct {
	Host        *Address `json:"host"`
	Guest       *Address `json:"guest"`
	Description string   `json:"description"`
}

func (f *Forward) Equal(f1 *Forward) bool {
	return f.Host.Equal(f1.Host) &&
		f.Guest.Equal(f1.Guest) &&
		f.Description == f1.Description
}

type PortForwarding struct {
	Forwards []*Forward `json:"forwards"`
	Path     string     `json:"-"`

	access sync.Mutex
	logger hclog.Logger
}

func LoadPortForwardingSettings(path string, logger hclog.Logger) (pf *PortForwarding, err error) {
	logger = logger.Named("portfwding")

	pf = &PortForwarding{
		Path:     path,
		Forwards: []*Forward{},
		logger:   logger,
	}
	err = pf.Init()

	return
}

func (p *PortForwarding) Init() error {
	if !p.exists() {
		p.logger.Debug("settings file does not exist - creating", "path", p.Path)
		return p.Save(nil)
	}
	return p.Reload()
}

func (p *PortForwarding) Add(fwd *Forward) (err error) {
	return p.Save(func() error {
		for _, f := range p.Forwards {
			if f.Equal(fwd) {
				p.logger.Warn("port forward already exists", "fwd", fwd)
				return nil
			}
		}
		p.Forwards = append(p.Forwards, fwd)
		return nil
	})
}

func (p *PortForwarding) Delete(fwd *Forward) (err error) {
	return p.Save(func() error {
		for i, f := range p.Forwards {
			if f.Equal(fwd) {
				p.Forwards = append(p.Forwards[0:i], p.Forwards[i+1:]...)
				return nil
			}
		}
		return nil
	})
}

func (p *PortForwarding) Save(presave func() error) error {
	p.access.Lock()
	defer p.access.Unlock()

	if presave != nil {
		if err := presave(); err != nil {
			return err
		}
	}

	if !p.exists() {
		if err := os.MkdirAll(path.Dir(p.Path), 0755); err != nil {
			p.logger.Error("failed to create parent directory", "error", err, "path", p.Path)
			return err
		}
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		p.logger.Error("failed to dump forward data", "error", err)
		return err
	}

	f, err := ioutil.TempFile(path.Dir(p.Path), "pfwds")
	if err != nil {
		p.logger.Error("failed to create file", "error", err, "path", f.Name())
		return err
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		p.logger.Error("failed to write file", "error", err, "path", f.Name())
		return err
	}
	if err := f.Close(); err != nil {
		p.logger.Error("failed closing new file", "error", err, "path", f.Name())
		return err
	}
	if err := os.Rename(f.Name(), p.Path); err != nil {
		p.logger.Error("failed to save file", "error", err, "path", p.Path)
		return err
	}
	p.logger.Debug("settings have been saved")
	return nil
}

func (p *PortForwarding) Reload() error {
	p.access.Lock()
	defer p.access.Unlock()

	if !p.exists() {
		p.logger.Debug("no file to reload - clearing")
		p.Forwards = []*Forward{}
		return nil
	}
	data, err := ioutil.ReadFile(p.Path)
	if err != nil {
		p.logger.Error("failed to read settings", "error", err)
		return err
	}
	if err := json.Unmarshal(data, p); err != nil {
		p.logger.Error("failed to load settings", "error", err)
		return err
	}
	p.logger.Debug("reload complete")
	return nil
}

func (p *PortForwarding) exists() bool {
	if _, err := os.Stat(p.Path); err == nil {
		return true
	}
	return false
}
