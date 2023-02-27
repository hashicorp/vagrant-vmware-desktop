// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"sync"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/version"
)

const API_CONTENT_TYPE = "application/vnd.hashicorp.vagrant.vmware.rest-v1+json"

type route struct {
	handler http.Handler
	path    *regexp.Regexp
}

type RegexpHandler struct {
	routes     []*route
	logger     hclog.Logger
	api        *Api
	leaseCache []*utility.DhcpEntry
	netLock    sync.Mutex
}

func NewRegexpHandler(api *Api, logger hclog.Logger) *RegexpHandler {
	logger = logger.Named("handler")
	return &RegexpHandler{
		api:    api,
		logger: logger}
}

func (r *RegexpHandler) HandleFunc(pattern *regexp.Regexp,
	handler func(http.ResponseWriter, *http.Request)) {
	r.routes = append(r.routes, &route{
		handler: http.HandlerFunc(handler),
		path:    pattern,
	})
}

func (r *RegexpHandler) ServeHTTP(writ http.ResponseWriter, req *http.Request) {
	r.logger.Info("request start", "method", req.Method, "path", req.URL.Path, "request-id", fmt.Sprintf("%p", writ))
	writ.Header().Set("Content-Type", API_CONTENT_TYPE)
	if r.invalidRequester(writ, req) {
		return
	}
	for _, route := range r.routes {
		if route.path.MatchString(req.URL.Path) {
			if r.api.Driver.Validated() {
				route.handler.ServeHTTP(writ, req)
				return
			} else {
				r.invalidDriver(writ)
				return
			}
		}
	}
	r.notFound(writ)
}

type StandardResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (r *RegexpHandler) respond(writ http.ResponseWriter, body interface{}, code int) {
	writ.WriteHeader(code)
	if body != nil {
		if err := json.NewEncoder(writ).Encode(body); err != nil {
			r.logger.Error("error encoding response body", "error", err)
			http.Error(writ, http.StatusText(500), 500)
		}
	}
	r.logger.Info("request complete", "code", code, "request-id", fmt.Sprintf("%p", writ))
}

func (r *RegexpHandler) notFound(writ http.ResponseWriter) {
	r.error(writ, "not found", 404)
}

func (r *RegexpHandler) invalidDriver(writ http.ResponseWriter) {
	r.error(writ, "Validation failure: "+r.api.Driver.ValidationReason(), 500)
}

func (r *RegexpHandler) invalidRequester(writ http.ResponseWriter, req *http.Request) bool {
	invalid := false
	validOrigin := fmt.Sprintf("https://%s:%d", r.api.Address, r.api.Port)
	if len(req.Header["X-Requested-With"]) != 1 || req.Header["X-Requested-With"][0] != "Vagrant" {
		invalid = true
	}
	if len(req.Header["Origin"]) != 1 || req.Header["Origin"][0] != validOrigin {
		invalid = true
	}
	if invalid {
		r.error(writ, "invalid client requester", 403)
	}
	return invalid
}

func (r *RegexpHandler) error(writ http.ResponseWriter, msg string, code int) {
	r.logger.Debug("request error", "code", code, "message", msg)
	response := StandardResponse{
		Code:    code,
		Message: msg}
	r.respond(writ, response, code)
}

// VMware VM Network Adapter handler
func (r *RegexpHandler) handleVmNicAdapter(writ http.ResponseWriter, req *http.Request) {
	r.error(writ, "not implemented", 501)
}

// VMware VM Network handler
func (r *RegexpHandler) handleVmNic(writ http.ResponseWriter, req *http.Request) {
	r.error(writ, "not implemented", 501)
	return
}

// VMware VM IP handler
func (r *RegexpHandler) handleVmIp(writ http.ResponseWriter, req *http.Request) {
	r.error(writ, "not implemented", 501)
	return
}

// VMware root handler
func (r *RegexpHandler) handleRoot(writ http.ResponseWriter, req *http.Request) {
	r.error(writ, "not implemented", 501)
	return
}

// Custom handlers
func (r *RegexpHandler) handleStatus(writ http.ResponseWriter, req *http.Request) {
	response := map[string]string{
		"status":   "running",
		"inflight": strconv.Itoa(r.api.Inflight()),
	}
	r.respond(writ, response, 200)
}

func (r *RegexpHandler) handleVersion(writ http.ResponseWriter, req *http.Request) {
	response := map[string]string{"version": version.VERSION}
	r.respond(writ, response, 200)
}

func (r *RegexpHandler) handleHealthRoot(writ http.ResponseWriter, req *http.Request) {
	r.notFound(writ)
}

func (r *RegexpHandler) pathParams(path string) map[string]string {
	params := map[string]string{}
	for _, route := range r.routes {
		match := route.path.FindStringSubmatch(path)
		if match != nil {
			for i, name := range route.path.SubexpNames() {
				if i == 0 {
					continue
				}
				params[name] = match[i]
			}
			return params
		}
	}
	return params
}
