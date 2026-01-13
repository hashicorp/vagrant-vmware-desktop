// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/driver"
)

func (r *RegexpHandler) handleVmnetDeviceForward(writ http.ResponseWriter, req *http.Request) {
	params := r.pathParams(req.URL.Path)
	r.logger.Trace("portforward parameters", "params", params)

	switch req.Method {
	case "GET":
		r.logger.Debug("portforward list", "slot", params["vnet_slot"])
		r.listPortFwds(writ, params["vnet_slot"])
	case "PUT":
		r.netLock.Lock()
		defer r.netLock.Unlock()
		r.logger.Debug("portforward request", "slot", params["vnet_slot"])
		r.applyPortFwd(writ, req, params["vnet_slot"])
	case "DELETE":
		r.netLock.Lock()
		defer r.netLock.Unlock()
		r.logger.Debug("portforward delete", "slot", params["vnet_slot"])
		r.deletePortFwd(writ, req, params["vnet_slot"])

	default:
		r.notFound(writ)
	}
}

func (r *RegexpHandler) handlePortForwards(writ http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		r.logger.Debug("full portforward list")
		r.listPortFwds(writ, "")
	case "DELETE":
		r.netLock.Lock()
		defer r.netLock.Unlock()
		r.logger.Debug("prune inactive portforwards")
		r.prunePortFwds(writ)
	default:
		r.notFound(writ)
	}
}

func (r *RegexpHandler) prunePortFwds(writ http.ResponseWriter) {
	err := r.api.Driver.PrunePortFwds(r.api.Driver.PortFwds, r.api.Driver.DeletePortFwd)
	if err != nil {
		r.logger.Debug("portforward prune failed", "error", err)
		r.error(writ, err.Error(), 400)
		return
	}
	r.respond(writ, nil, 204)
}

func (r *RegexpHandler) listPortFwds(writ http.ResponseWriter, slotNumber string) {
	portfwds, err := r.api.Driver.PortFwds(slotNumber)
	if err != nil {
		r.logger.Debug("portforward error", "error", err)
		r.error(writ, err.Error(), 400)
		return
	}
	r.logger.Trace("full portforward list", "fwds", portfwds)
	r.respond(writ, portfwds, 200)
}

func (r *RegexpHandler) applyPortFwd(writ http.ResponseWriter, req *http.Request, slotNumber string) {
	var portFwds []driver.PortFwd
	var buf bytes.Buffer
	tr := io.TeeReader(req.Body, &buf)
	err := json.NewDecoder(tr).Decode(&portFwds)
	if err != nil {
		r.logger.Debug("portforward parse failed", "error", err)
		r.logger.Debug("portforward re-parse attempt as non-collection")
		var pfwd driver.PortFwd
		err = json.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&pfwd)
		if err != nil {
			r.logger.Debug("portforward re-parse failed", "error", err)
			r.error(writ, err.Error(), 400)
			return
		} else {
			portFwds = []driver.PortFwd{pfwd}
		}
	}
	slotNum, err := strconv.Atoi(slotNumber)
	if err != nil {
		r.logger.Debug("portforward slot parse failed", "slot", slotNumber, "error", err)
		r.error(writ, err.Error(), 400)
		return
	}
	r.logger.Debug("apply port forwards", "port-forwards", portFwds)
	pfwds := []*driver.PortFwd{}
	for i := 0; i < len(portFwds); i++ {
		fwd := &portFwds[i]
		fwd.SlotNumber = slotNum
		pfwds = append(pfwds, fwd)
	}
	r.logger.Debug("adding port forwards", "fwds", pfwds)
	err = r.api.Driver.AddPortFwd(pfwds)
	if err != nil {
		r.logger.Debug("portforward apply failure", "error", err)
		r.error(writ, err.Error(), 400)
		return
	}
	r.respond(writ, portFwds, 200)
}

func (r *RegexpHandler) deletePortFwd(writ http.ResponseWriter, req *http.Request, slotNumber string) {
	var portFwds []driver.PortFwd
	var buf bytes.Buffer
	tr := io.TeeReader(req.Body, &buf)
	err := json.NewDecoder(tr).Decode(&portFwds)
	if err != nil {
		r.logger.Debug("portforward parse failed", "error", err)
		r.logger.Debug("portforward re-parse attempt as non-collection")
		var pfwd driver.PortFwd
		err = json.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&pfwd)
		if err != nil {
			r.logger.Debug("portforward re-parse failed", "error", err)
			r.error(writ, err.Error(), 400)
			return
		} else {
			portFwds = []driver.PortFwd{pfwd}
		}
	}
	slotNum, err := strconv.Atoi(slotNumber)
	if err != nil {
		r.logger.Debug("portforward slot parse failed", "slot", slotNumber, "error", err)
		r.error(writ, err.Error(), 400)
		return
	}
	r.logger.Debug("apply port forwards", "port-forwards", portFwds)
	pfwds := []*driver.PortFwd{}
	for i := 0; i < len(portFwds); i++ {
		fwd := &portFwds[i]
		fwd.SlotNumber = slotNum
		pfwds = append(pfwds, fwd)
	}
	err = r.api.Driver.DeletePortFwd(pfwds)
	if err != nil {
		r.logger.Debug("portforward delete failure", "error", err)
		r.error(writ, err.Error(), 400)
		return
	}
	r.respond(writ, nil, 204)
}
