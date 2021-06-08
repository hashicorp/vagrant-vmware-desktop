package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/driver"
)

// VMware host adapter
func (r *RegexpHandler) handleVmnet(writ http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		r.logger.Debug("vmnet list request")
		r.listVmnetDevices(writ)

	case "POST":
		r.netLock.Lock()
		defer r.netLock.Unlock()
		r.logger.Debug("vmnet create request")
		r.createVmnetDevice(writ, req)

	default:
		r.notFound(writ)
	}
}

func (r *RegexpHandler) handleVmnetVerify(writ http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "POST":
		r.netLock.Lock()
		defer r.netLock.Unlock()
		r.logger.Debug("vmnet verification request")
		r.verifyVmnet(writ)

	default:
		r.notFound(writ)
	}
}

func (r *RegexpHandler) handleVmnetDevice(writ http.ResponseWriter, req *http.Request) {
	params := r.pathParams(req.URL.Path)
	r.logger.Trace("vmnet device parameters", "params", params)

	switch req.Method {
	case "GET":
		r.logger.Debug("vmnet device", "name", params["vnet_name"])
		r.getVmnetDevice(writ, params["vnet_name"])

	case "PUT":
		r.netLock.Lock()
		defer r.netLock.Unlock()
		r.logger.Debug("vmnet update request", "name", params["vnet_name"])
		r.updateVmnetDevice(writ, req, params["vnet_name"])

	case "DELETE":
		r.netLock.Lock()
		defer r.netLock.Unlock()
		r.logger.Debug("vmnet delete request", "name", params["vnet_name"])
		r.deleteVmnetDevice(writ, params["vnet_name"])

	default:
		r.notFound(writ)
	}
}

func (r *RegexpHandler) handleVmnetDhcpLease(writ http.ResponseWriter, req *http.Request) {
	params := r.pathParams(req.URL.Path)
	r.logger.Trace("vmnet dhcp lease parameters", "params", params)

	switch req.Method {
	case "GET":
		r.logger.Debug("vmnet dhcp lease request", "device", params["vnet_name"], "mac", params["mac"])
		r.getVmnetDhcpLease(writ, params["vnet_name"], params["mac"])
	default:
		r.notFound(writ)
	}
}

func (r *RegexpHandler) handleVmnetDhcpReserve(writ http.ResponseWriter, req *http.Request) {
	params := r.pathParams(req.URL.Path)
	r.logger.Trace("vmnet dhcp reserve parameters", "params", params)

	switch req.Method {
	case "PUT":
		r.logger.Debug("vmnet dhcp reserve request", "device", params["vnet_slog"], "mac", params["mac"],
			"address", params["ip"])
		r.reserveVmnetDhcpAddress(writ, params["vnet_slot"], params["mac"], params["ip"])
	default:
		r.notFound(writ)
	}
}

func (r *RegexpHandler) getVmnetDhcpLease(writ http.ResponseWriter, device string, mac string) {
	ip, err := r.api.Driver.LookupDhcpAddress(device, mac)
	if err != nil {
		r.logger.Debug("vmnet dhcp lease lookup error", "error", err)
		r.error(writ, err.Error(), 400)
		return
	}
	result := map[string]string{"ip": ip}
	r.respond(writ, result, 200)
}

func (r *RegexpHandler) listVmnetDevices(writ http.ResponseWriter) {
	devices, err := r.api.Driver.Vmnets()
	if err != nil {
		r.logger.Debug("vmnet list error", "error", err.Error())
		r.error(writ, err.Error(), 400)
		return
	}
	r.respond(writ, devices, 200)
}

func (r *RegexpHandler) createVmnetDevice(writ http.ResponseWriter, req *http.Request) {
	var newDevice driver.Vmnet
	err := json.NewDecoder(req.Body).Decode(&newDevice)
	if err != nil {
		r.logger.Debug("vmnet parse failed", "error", err)
		r.error(writ, err.Error(), 400)
		return
	}
	r.logger.Debug("creating new device")
	err = r.api.Driver.AddVmnet(&newDevice)
	if err != nil {
		r.logger.Debug("vmnet create failure", "error", err)
		r.error(writ, err.Error(), 400)
		return
	}
	r.respond(writ, newDevice, 200)
}

func (r *RegexpHandler) getVmnetDevice(writ http.ResponseWriter, deviceName string) {
	devices, err := r.api.Driver.Vmnets()
	if err != nil {
		r.logger.Debug("vmnet get error", "device", deviceName, "error", err.Error())
		r.error(writ, err.Error(), 400)
		return
	}
	for _, device := range devices.Vmnets {
		if device.Name == deviceName {
			r.respond(writ, device, 200)
			return
		}
	}
	r.error(writ, "device not found", 400)
}

func (r *RegexpHandler) reserveVmnetDhcpAddress(writ http.ResponseWriter, slotNumber, mac, ip string) {
	slotNum, _ := strconv.Atoi(slotNumber)
	err := r.api.Driver.ReserveDhcpAddress(slotNum, mac, ip)
	if err != nil {
		r.logger.Debug("dhcp address reservation failed", "device", "vmnet"+slotNumber, "mac", mac,
			"address", ip, "error", err)
		r.error(writ, err.Error(), 400)
		return
	}
	r.respond(writ, nil, 204)
}

func (r *RegexpHandler) updateVmnetDevice(writ http.ResponseWriter, req *http.Request, deviceName string) {
	var upDevice driver.Vmnet
	err := json.NewDecoder(req.Body).Decode(&upDevice)
	if err != nil {
		r.logger.Debug("vmnet parse failed", "error", err)
		r.error(writ, err.Error(), 400)
		return
	}
	upDevice.Name = deviceName
	r.logger.Debug("updating device", "name", upDevice.Name)
	err = r.api.Driver.UpdateVmnet(&upDevice)
	if err != nil {
		r.logger.Debug("vmnet update failure", "error", err)
		r.error(writ, err.Error(), 400)
		return
	}
	r.respond(writ, upDevice, 200)
}

func (r *RegexpHandler) deleteVmnetDevice(writ http.ResponseWriter, deviceName string) {
	err := r.api.Driver.DeleteVmnet(&driver.Vmnet{Name: deviceName})
	if err != nil {
		r.logger.Debug("vmnet delete failure", "error", err)
		r.error(writ, err.Error(), 400)
		return
	}
	r.logger.Debug("vmnet device removed", "name", deviceName)
	r.respond(writ, nil, 204)
}

func (r *RegexpHandler) verifyVmnet(writ http.ResponseWriter) {
	err := r.api.Driver.VerifyVmnet()
	if err != nil {
		r.logger.Debug("vmnet verify failure", "error", err)
		r.error(writ, err.Error(), 400)
	}
	r.respond(writ, nil, 204)
}
