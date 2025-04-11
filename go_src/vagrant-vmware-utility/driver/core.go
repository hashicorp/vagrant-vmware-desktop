// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package driver

import (
	"runtime"
	"strconv"
	"strings"

	hclog "github.com/hashicorp/go-hclog"

	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/settings"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

type VmwareInfo struct {
	Product string `json:"product"`
	Version string `json:"version"`
	Build   string `json:"build"`
	Type    string `json:"type"`
	License string `json:"license"`
}

func (v *VmwareInfo) IsStandard() bool {
	return !v.IsProfessional()
}

func (v *VmwareInfo) IsProfessional() bool {
	// First, lets check if the license has been explicitly
	// set to standard or professional
	if v.License == "professional" {
		return true
	}
	if v.License == "standard" {
		return false
	}

	// Empty value may be produced when using the license
	// generated for free fusion/workstation. If the license
	// content is empty, just assume it's professional
	if v.License == "" {
		return true
	}

	// Now we need to check if we are using a product that
	// is a professional version. These include Workstation
	// and Fusion (but not player)
	if !strings.Contains(v.License, "pro") &&
		!strings.Contains(v.License, "workstation") &&
		!strings.Contains(v.License, "ws") {
		return false
	}

	return true
}

func (v *VmwareInfo) Normalize() {
	if v.IsProfessional() {
		v.License = "professional"
	} else {
		v.License = "standard"
	}
}

type Vmnet struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Dhcp   string `json:"dhcp"`
	Subnet string `json:"subnet"`
	Mask   string `json:"mask"`
}

type Vmnets struct {
	Num    int      `json:"num"`
	Vmnets []*Vmnet `json:"vmnets"`
}

type PortFwdGuest struct {
	Ip   string `json:"ip"`
	Port int    `json:"port"`
}

type PortFwd struct {
	Port        int           `json:"port"`
	Protocol    string        `json:"protocol"`
	Description string        `json:"description"`
	Guest       *PortFwdGuest `json:"guest"`
	SlotNumber  int           `json:"-"`
}

func (p *PortFwd) Matches(fwd *PortFwd) bool {
	return p.Port == fwd.Port &&
		p.Protocol == fwd.Protocol &&
		p.Guest.Ip == fwd.Guest.Ip &&
		p.Guest.Port == fwd.Guest.Port
}

type PortFwds struct {
	Num          int        `json:"num"`
	PortForwards []*PortFwd `json:"port_forwards"`
}

type MacToIp struct {
	Vmnet string `json:"vmnet"`
	Mac   string `json:"mac"`
	Ip    string `json:"ip"`
}

type MacToIps struct {
	Num      int        `json:"num"`
	MacToIps []*MacToIp `json:"mactoips"`
}

const FUSION_ADVANCED_MAJOR_MIN = 10

type Driver interface {
	AddInternalPortForward(fwd *PortFwd) error
	AddPortFwd(fwds []*PortFwd) error
	AddVmnet(v *Vmnet) error
	DeleteInternalPortForward(fwd *PortFwd) error
	DeletePortFwd(fwds []*PortFwd) error
	DeleteVmnet(v *Vmnet) error
	EnableInternalPortForwarding() error
	InternalPortFwds() (fwds []*PortFwd, err error)
	LoadNetworkingFile() (f utility.NetworkingFile, err error)
	LookupDhcpAddress(device, mac string) (addr string, err error)
	Path() (path *string, err error)
	PortFwds(device string) (fwds *PortFwds, err error)
	PrunePortFwds(fwds func(string) (*PortFwds, error), deleter func([]*PortFwd) error) error
	ReserveDhcpAddress(slot int, mac, ip string) error
	Settings() *settings.Settings
	UpdateVmnet(v *Vmnet) error
	Validated() bool
	Validate() bool
	ValidationReason() string
	VerifyVmnet() error
	Vmnets() (v *Vmnets, err error)
	VmwareInfo() (info *VmwareInfo, err error)
	VmwarePaths() *utility.VmwarePaths
}

func CreateDriver(vmxPath *string, b *BaseDriver, logger hclog.Logger) (Driver, error) {
	var d Driver
	var err error
	switch runtime.GOOS {
	case "windows":
		logger.Debug("creating new advanced driver")
		d, err = NewAdvancedDriver(vmxPath, b, logger)
		if err != nil {
			return d, err
		}
	case "darwin":
		// create a simple driver initially so we can test the version. if under
		// fusion 10 we need to use the simple driver due to issues with networking
		// file rewrite behavior. otherwise we can use the advanced driver.
		d, err = NewSimpleDriver(vmxPath, b, logger)
		if err != nil {
			return d, err
		}
		info, err := d.VmwareInfo()
		if err != nil {
			logger.Error("failed to get VMware information", "error", err)
			return d, err
		}

		// Assume advanced driver for experimental builds
		if info.Version == "e.x.p" {
			logger.Debug("creating new advanced driver")
			d, err = NewAdvancedDriver(vmxPath, b, logger)
			if err != nil {
				return d, err
			}
			return d, nil
		}

		verParts := strings.Split(info.Version, ".")
		if len(verParts) < 1 {
			logger.Warn("failed to determine major version, using simple driver",
				"version", info.Version)
			return d, err
		}
		major, err := strconv.Atoi(verParts[0])
		if err != nil {
			logger.Warn("failed to convert major version for comparison, using simple driver",
				"major", verParts[0], "error", err)
			err = nil
			return d, err
		}
		if major < FUSION_ADVANCED_MAJOR_MIN {
			logger.Debug("using simple driver due to fusion version", "major", major,
				"required-minimum", FUSION_ADVANCED_MAJOR_MIN)
			return d, err
		}
		logger.Debug("creating new advanced driver")
		d, err = NewAdvancedDriver(vmxPath, b, logger)
		if err != nil {
			return d, err
		}
	default:
		logger.Debug("creating new simple driver")
		d, err = NewSimpleDriver(vmxPath, b, logger)
		if err != nil {
			return d, err
		}
	}
	return d, nil
}
