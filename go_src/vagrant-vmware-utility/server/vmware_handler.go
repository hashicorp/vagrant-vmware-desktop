// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"net/http"
	"strings"

	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/driver"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

type LicenseFeature struct {
	Product string `json:"product"`
	Version string `json:"version"`
}

type VagrantVmwareValidate struct {
	Features []LicenseFeature `json:"features"`
}

func (r *RegexpHandler) handleVmwareInfo(writ http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		r.logger.Debug("vmware info")
		r.getVmwareInfo(writ)
	default:
		r.notFound(writ)
	}
}

func (r *RegexpHandler) handleVmwarePaths(writ http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		r.logger.Debug("vmware paths")
		paths, err := utility.LoadVmwarePaths(r.logger)
		if err != nil {
			r.error(writ, err.Error(), 400)
			return
		}
		r.respond(writ, paths, 200)
	default:
		r.notFound(writ)
	}
}

func (r *RegexpHandler) getVmwareInfo(writ http.ResponseWriter) {
	info, err := r.api.Driver.VmwareInfo()
	if err != nil {
		r.logger.Debug("vmware info error", "error", err)
		r.error(writ, err.Error(), 400)
		return
	}
	r.logger.Trace("vmware version info", "version", info.Version, "product", info.Product,
		"type", info.Type, "build", info.Build)
	r.respond(writ, info, 200)
}

func (r *RegexpHandler) hasValidFeature(vmware *driver.VmwareInfo, vagrant *VagrantVmwareValidate) bool {
	vmwareVersionParts := strings.Split(vmware.Version, ".")
	if len(vmwareVersionParts) < 1 {
		r.logger.Trace("failed to split vmware version", "vmware-version", vmware.Version)
		return false
	}
	vmwareMajor := vmwareVersionParts[0]
	r.logger.Trace("vagrant vmware feature matching", "vmware-product", vmware.Product, "version-major", vmwareMajor)
	for _, feature := range vagrant.Features {
		r.logger.Trace("vagrant vmware feature matching", "product", feature.Product, "version", feature.Version)
		if strings.ToLower(feature.Product) == strings.ToLower(vmware.Product) && feature.Version == vmwareMajor {
			r.logger.Trace("vagrant vmware feature matched", "product", feature.Product, "version", feature.Version,
				"vmware-product", vmware.Product, "vmware-version", vmwareMajor)
			return true
		}
	}
	r.logger.Trace("vagrant vmware feature matching failed")
	return false
}
