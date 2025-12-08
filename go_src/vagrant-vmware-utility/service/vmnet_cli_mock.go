// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package service

type VmnetCliMock struct {
	StartResponses     []error
	StopResponses      []error
	RestartResponses   []error
	ConfigureResponses []error
	StatusResponses    []bool

	ConfigureRequests []string
}

func (v *VmnetCliMock) Start() (err error) {
	if len(v.StartResponses) > 0 {
		err = v.StartResponses[0]
		v.StartResponses = v.StartResponses[1:]
	}
	return
}

func (v *VmnetCliMock) Stop() (err error) {
	if len(v.StopResponses) > 0 {
		err = v.StopResponses[0]
		v.StopResponses = v.StopResponses[1:]
	}
	return
}

func (v *VmnetCliMock) Status() (s bool) {
	s = true
	if len(v.StatusResponses) > 0 {
		s = v.StatusResponses[0]
		v.StatusResponses = v.StatusResponses[1:]
	}
	return s
}

func (v *VmnetCliMock) Restart() (err error) {
	if len(v.RestartResponses) > 0 {
		err = v.RestartResponses[0]
		v.RestartResponses = v.RestartResponses[1:]
	}
	return
}

func (v *VmnetCliMock) Configure(path string) (err error) {
	if len(v.ConfigureResponses) > 0 {
		err = v.ConfigureResponses[0]
		v.ConfigureResponses = v.ConfigureResponses[1:]
	}
	v.ConfigureRequests = append(v.ConfigureRequests, path)
	return
}
