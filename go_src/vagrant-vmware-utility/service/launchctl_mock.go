// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package service

type LaunchctlMock struct {
	LoadResponses   []error
	LoadRequests    []string
	UnloadResponses []error
	UnloadRequests  []string
}

func (l *LaunchctlMock) Load(p string) (err error) {
	if len(l.LoadResponses) > 0 {
		err = l.LoadResponses[0]
		l.LoadResponses = l.LoadResponses[1:]
	}
	l.LoadRequests = append(l.LoadRequests, p)
	return
}

func (l *LaunchctlMock) Unload(p string) (err error) {
	if len(l.UnloadResponses) > 0 {
		err = l.UnloadResponses[0]
		l.UnloadResponses = l.UnloadResponses[1:]
	}
	l.UnloadRequests = append(l.UnloadRequests, p)
	return
}
