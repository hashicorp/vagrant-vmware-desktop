package service

type VmwareServicesMock struct {
	Requests []func()
}

func (v *VmwareServicesMock) WrapOpenServices(f func()) {
	v.Requests = append(v.Requests, f)
}

func NewVmwareServicesMock() VmwareServices {
	return &VmwareServicesMock{
		Requests: []func(){}}
}
