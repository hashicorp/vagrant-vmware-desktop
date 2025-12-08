// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package service

type VnetlibMock struct {
	CreateDeviceResponses          []*CreateDeviceResponse
	CreateDeviceRequests           []string
	DeleteDeviceResponses          []error
	DeleteDeviceRequests           []string
	SetSubnetAddressResponses      []error
	SetSubnetAddressRequests       []*SetSubnetAddressRequest
	SetSubnetMaskResponses         []error
	SetSubnetMaskRequests          []*SetSubnetMaskRequest
	SetNATResponses                []error
	SetNATRequests                 []*SetNATRequest
	UpdateDeviceNATResponses       []error
	UpdateDeviceNATRequests        []string
	StatusNATResponses             []bool
	StatusNATRequests              []string
	StartNATResponses              []error
	StartNATRequests               []string
	StopNATResponses               []error
	StopNATRequests                []string
	SetDHCPResponses               []error
	SetDHCPRequests                []*SetDHCPRequest
	StatusDHCPResponses            []bool
	StatusDHCPRequests             []string
	StartDHCPResponses             []error
	StartDHCPRequests              []string
	StopDHCPResponses              []error
	StopDHCPRequests               []string
	LookupReservedAddressResponses []*LookupReservedAddressResponse
	LookupReservedAddressRequests  []*LookupReservedAddressRequest
	ReserveAddressResponses        []error
	ReserveAddressRequests         []*ReserveAddressRequest
	EnableDeviceResponses          []error
	EnableDeviceRequests           []string
	DisableDeviceResponses         []error
	DisableDeviceRequests          []string
	UpdateDeviceResponses          []error
	UpdateDeviceRequests           []string
	DeletePortFwdResponses         []error
	DeletePortFwdRequests          []*DeletePortFwdRequest
	GetUnusedDeviceResponses       []*GetUnusedDeviceResponse
}

type CreateDeviceResponse struct {
	DevName string
	Error   error
}

type SetSubnetAddressRequest struct {
	DevName string
	Address string
}

type SetSubnetMaskRequest struct {
	DevName string
	Mask    string
}

type SetNATRequest struct {
	DevName string
	Enable  bool
}

type SetDHCPRequest struct {
	DevName string
	Enable  bool
}

type LookupReservedAddressResponse struct {
	Address string
	Error   error
}

type LookupReservedAddressRequest struct {
	DevName string
	MAC     string
}

type ReserveAddressRequest struct {
	DevName string
	MAC     string
	Address string
}

type DeletePortFwdRequest struct {
	DevName  string
	Protocol string
	HostPort string
}

type GetUnusedDeviceResponse struct {
	DevName string
	Error   error
}

func (v *VnetlibMock) CreateDevice(newName string) (devName string, err error) {
	if len(v.CreateDeviceResponses) > 0 {
		r := v.CreateDeviceResponses[0]
		v.CreateDeviceResponses = v.CreateDeviceResponses[1:]
		devName = r.DevName
		err = r.Error
	}
	v.CreateDeviceRequests = append(v.CreateDeviceRequests, newName)

	return
}

func (v *VnetlibMock) DeleteDevice(devName string) (err error) {
	if len(v.DeleteDeviceResponses) > 0 {
		err = v.DeleteDeviceResponses[0]
		v.DeleteDeviceResponses = v.DeleteDeviceResponses[1:]
	}
	v.DeleteDeviceRequests = append(v.DeleteDeviceRequests, devName)

	return
}

func (v *VnetlibMock) SetSubnetAddress(devName string, addr string) (err error) {
	if len(v.SetSubnetAddressResponses) > 0 {
		err = v.SetSubnetAddressResponses[0]
		v.SetSubnetAddressResponses = v.SetSubnetAddressResponses[1:]
	}
	v.SetSubnetAddressRequests = append(v.SetSubnetAddressRequests, &SetSubnetAddressRequest{DevName: devName, Address: addr})

	return
}

func (v *VnetlibMock) SetSubnetMask(devName string, mask string) (err error) {
	if len(v.SetSubnetMaskResponses) > 0 {
		err = v.SetSubnetMaskResponses[0]
		v.SetSubnetMaskResponses = v.SetSubnetMaskResponses[1:]
	}
	v.SetSubnetMaskRequests = append(v.SetSubnetMaskRequests, &SetSubnetMaskRequest{DevName: devName, Mask: mask})

	return
}

func (v *VnetlibMock) SetNAT(devName string, enable bool) (err error) {
	if len(v.SetNATResponses) > 0 {
		err = v.SetNATResponses[0]
		v.SetNATResponses = v.SetNATResponses[1:]
	}
	v.SetNATRequests = append(v.SetNATRequests, &SetNATRequest{DevName: devName, Enable: enable})

	return
}

func (v *VnetlibMock) UpdateDeviceNAT(devName string) (err error) {
	if len(v.UpdateDeviceNATResponses) > 0 {
		err = v.UpdateDeviceNATResponses[0]
		v.UpdateDeviceNATResponses = v.UpdateDeviceNATResponses[1:]
	}
	v.UpdateDeviceNATRequests = append(v.UpdateDeviceNATRequests, devName)

	return
}

func (v *VnetlibMock) StatusNAT(devName string) (s bool) {
	s = true
	if len(v.StatusNATResponses) > 0 {
		s = v.StatusNATResponses[0]
		v.StatusNATResponses = v.StatusNATResponses[1:]
	}
	v.StatusNATRequests = append(v.StatusNATRequests, devName)

	return
}

func (v *VnetlibMock) StartNAT(devName string) (err error) {
	if len(v.StartNATResponses) > 0 {
		err = v.StartNATResponses[0]
		v.StartNATResponses = v.StartNATResponses[1:]
	}
	v.StartNATRequests = append(v.StartNATRequests, devName)

	return
}

func (v *VnetlibMock) StopNAT(devName string) (err error) {
	if len(v.StopNATResponses) > 0 {
		err = v.StopNATResponses[0]
		v.StopNATResponses = v.StopNATResponses[1:]
	}
	v.StopNATRequests = append(v.StopNATRequests, devName)

	return
}

func (v *VnetlibMock) SetDHCP(devName string, enable bool) (err error) {
	if len(v.SetDHCPResponses) > 0 {
		err = v.SetDHCPResponses[0]
		v.SetDHCPResponses = v.SetDHCPResponses[1:]
	}
	v.SetDHCPRequests = append(v.SetDHCPRequests, &SetDHCPRequest{DevName: devName, Enable: enable})

	return
}

func (v *VnetlibMock) StatusDHCP(devName string) (s bool) {
	s = true
	if len(v.StatusDHCPResponses) > 0 {
		s = v.StatusDHCPResponses[0]
		v.StatusDHCPResponses = v.StatusDHCPResponses[1:]
	}
	v.StatusDHCPRequests = append(v.StatusDHCPRequests, devName)

	return
}

func (v *VnetlibMock) StartDHCP(devName string) (err error) {
	if len(v.StartDHCPResponses) > 0 {
		err = v.StartDHCPResponses[0]
		v.StartDHCPResponses = v.StartDHCPResponses[1:]
	}
	v.StartDHCPRequests = append(v.StartDHCPRequests, devName)

	return
}

func (v *VnetlibMock) StopDHCP(devName string) (err error) {
	if len(v.StopDHCPResponses) > 0 {
		err = v.StopDHCPResponses[0]
		v.StopDHCPResponses = v.StopDHCPResponses[1:]
	}
	v.StopDHCPRequests = append(v.StopDHCPRequests, devName)

	return
}

func (v *VnetlibMock) LookupReservedAddress(device, mac string) (addr string, err error) {
	if len(v.LookupReservedAddressResponses) > 0 {
		r := v.LookupReservedAddressResponses[0]
		v.LookupReservedAddressResponses = v.LookupReservedAddressResponses[1:]
		addr = r.Address
		err = r.Error
	}
	v.LookupReservedAddressRequests = append(v.LookupReservedAddressRequests, &LookupReservedAddressRequest{DevName: device, MAC: mac})

	return
}

func (v *VnetlibMock) ReserveAddress(device, mac, ip string) (err error) {
	if len(v.ReserveAddressResponses) > 0 {
		err = v.ReserveAddressResponses[0]
		v.ReserveAddressResponses = v.ReserveAddressResponses[1:]
	}
	v.ReserveAddressRequests = append(v.ReserveAddressRequests, &ReserveAddressRequest{DevName: device, MAC: mac, Address: ip})

	return
}

func (v *VnetlibMock) EnableDevice(devName string) (err error) {
	if len(v.EnableDeviceResponses) > 0 {
		err = v.EnableDeviceResponses[0]
		v.EnableDeviceResponses = v.EnableDeviceResponses[1:]
	}
	v.EnableDeviceRequests = append(v.EnableDeviceRequests, devName)

	return
}
func (v *VnetlibMock) DisableDevice(devName string) (err error) {
	if len(v.DisableDeviceResponses) > 0 {
		err = v.DisableDeviceResponses[0]
		v.DisableDeviceResponses = v.DisableDeviceResponses[1:]
	}
	v.DisableDeviceRequests = append(v.DisableDeviceRequests, devName)

	return
}

func (v *VnetlibMock) UpdateDevice(devName string) (err error) {
	if len(v.UpdateDeviceResponses) > 0 {
		err = v.UpdateDeviceResponses[0]
		v.UpdateDeviceResponses = v.UpdateDeviceResponses[1:]
	}
	v.UpdateDeviceRequests = append(v.UpdateDeviceRequests, devName)

	return
}

func (v *VnetlibMock) DeletePortFwd(device, protocol, hostPort string) (err error) {
	if len(v.DeletePortFwdResponses) > 0 {
		err = v.DeletePortFwdResponses[0]
		v.DeletePortFwdResponses = v.DeletePortFwdResponses[1:]
	}
	v.DeletePortFwdRequests = append(v.DeletePortFwdRequests, &DeletePortFwdRequest{DevName: device, Protocol: protocol, HostPort: hostPort})

	return
}

func (v *VnetlibMock) GetUnusedDevice() (devName string, err error) {
	if len(v.GetUnusedDeviceResponses) > 0 {
		r := v.GetUnusedDeviceResponses[0]
		v.GetUnusedDeviceResponses = v.GetUnusedDeviceResponses[1:]
		devName = r.DevName
		err = r.Error
	}

	return
}
