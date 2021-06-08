package driver

import (
	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/utility"
)

type MockDriver struct{}

func (t *MockDriver) Path() (p *string, err error) {
	return
}

func (t *MockDriver) Vmnets() (v *Vmnets, err error) {
	return
}

func (t *MockDriver) AddVmnet(v *Vmnet) (err error) {
	return
}

func (t *MockDriver) UpdateVmnet(v *Vmnet) (err error) {
	return
}

func (t *MockDriver) DeleteVmnet(v *Vmnet) (err error) {
	return
}

func (t *MockDriver) PortFwds(device string) (fwds *PortFwds, err error) {
	return
}

func (t *MockDriver) AddPortFwd(fwds []*PortFwd) (err error) {
	return
}

func (t *MockDriver) DeletePortFwd(fwds []*PortFwd) (err error) {
	return
}

func (t *MockDriver) PrunePortFwds(fwds func(string) (*PortFwds, error), deleter func([]*PortFwd) error) (err error) {
	return
}

func (t *MockDriver) LookupDhcpAddress(device, mac string) (ip string, err error) {
	return
}

func (t *MockDriver) ReserveDhcpAddress(slot int, mac, ip string) (err error) {
	return
}

func (t *MockDriver) VmwareInfo() (info *VmwareInfo, err error) {
	return
}

func (t *MockDriver) VmwarePaths() (p *utility.VmwarePaths) {
	return &utility.VmwarePaths{
		Vmrest: "/bin/true"}
}

func (t *MockDriver) LoadNetworkingFile() (f utility.NetworkingFile, err error) {
	return
}

func (t *MockDriver) Validated() bool {
	return true
}

func (t *MockDriver) Validate() bool {
	return true
}

func (t *MockDriver) ValidationReason() (r string) {
	return
}

func (t *MockDriver) VerifyVmnet() (err error) {
	return
}
