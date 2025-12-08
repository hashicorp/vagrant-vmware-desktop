// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package utility

import (
	"strings"

	hclog "github.com/hashicorp/go-hclog"
)

type VmwarePaths struct {
	BridgePid    string       `json:"bridge_pid"`
	DhcpLease    string       `json:"dhcp_lease"`
	InstallDir   string       `json:"install_dir"`
	NatConf      string       `json:"nat_conf"`
	Networking   string       `json:"networking"`
	Services     string       `json:"services"`
	VmnetCli     string       `json:"vmnet_cli"`
	Vnetlib      string       `json:"vnetlib"`
	Vmx          string       `json:"vmx"`
	Vmrun        string       `json:"vmrun"`
	Vmrest       string       `json:"vmrest"`
	Vdiskmanager string       `json:"vdiskmanager"`
	logger       hclog.Logger `json:-`
}

func LoadVmwarePaths(logger hclog.Logger) (*VmwarePaths, error) {
	logger = logger.Named("vmware-paths")
	paths := &VmwarePaths{
		logger: logger}
	err := paths.Load()
	if err != nil {
		return nil, err
	}
	return paths, nil
}

func (v *VmwarePaths) DhcpLeaseFile(device string) string {
	return strings.Replace(v.DhcpLease, "{{device}}", device, -1)
}

func (v *VmwarePaths) NatConfFile(device string) string {
	return strings.Replace(v.NatConf, "{{device}}", device, -1)
}
