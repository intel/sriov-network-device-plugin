// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

import v1beta1 "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

// DeviceInfoProvider is an autogenerated mock type for the DeviceInfoProvider type
type DeviceInfoProvider struct {
	mock.Mock
}

// GetDeviceSpecs provides a mock function with given fields: pciAddr
func (_m *DeviceInfoProvider) GetDeviceSpecs(pciAddr string) []*v1beta1.DeviceSpec {
	ret := _m.Called(pciAddr)

	var r0 []*v1beta1.DeviceSpec
	if rf, ok := ret.Get(0).(func(string) []*v1beta1.DeviceSpec); ok {
		r0 = rf(pciAddr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*v1beta1.DeviceSpec)
		}
	}

	return r0
}

// GetEnvVal provides a mock function with given fields: pciAddr
func (_m *DeviceInfoProvider) GetEnvVal(pciAddr string) string {
	ret := _m.Called(pciAddr)

	var r0 string
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(pciAddr)
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetMounts provides a mock function with given fields: pciAddr
func (_m *DeviceInfoProvider) GetMounts(pciAddr string) []*v1beta1.Mount {
	ret := _m.Called(pciAddr)

	var r0 []*v1beta1.Mount
	if rf, ok := ret.Get(0).(func(string) []*v1beta1.Mount); ok {
		r0 = rf(pciAddr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*v1beta1.Mount)
		}
	}

	return r0
}
