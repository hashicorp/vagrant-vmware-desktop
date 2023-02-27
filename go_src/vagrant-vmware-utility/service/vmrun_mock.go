// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package service

type VmrunMock struct {
	Responses []*VmrunResponse
}

type VmrunResponse struct {
	Vms   []*Vm
	Error error
}

func (v *VmrunMock) AddResponse(r *VmrunResponse) {
	v.Responses = append(v.Responses, r)
}

func (v *VmrunMock) GenerateResponse() {
	r := &VmrunResponse{
		Vms: []*Vm{
			&Vm{
				Path:  "/test/path/vm.vmx",
				vmrun: v}}}
	v.Responses = append(v.Responses, r)
}

func (v *VmrunMock) RunningVms() (r []*Vm, err error) {
	if len(v.Responses) > 0 {
		result := v.Responses[0]
		v.Responses = v.Responses[1:]
		r = result.Vms
		err = result.Error
	}
	return
}
