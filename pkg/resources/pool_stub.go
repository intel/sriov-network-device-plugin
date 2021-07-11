// Copyright 2018 Intel Corp. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resources

import (
	"github.com/golang/glog"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// ResourcePoolImpl implements stub ResourcePool interface
type ResourcePoolImpl struct {
	config     *types.ResourceConfig
	devices    map[string]*pluginapi.Device
	devicePool map[string]types.PciDevice
}

var _ types.ResourcePool = &ResourcePoolImpl{}

// NewResourcePool returns an instance of resourcePool
func NewResourcePool(rc *types.ResourceConfig, apiDevices map[string]*pluginapi.Device, devicePool map[string]types.PciDevice) *ResourcePoolImpl {
	return &ResourcePoolImpl{
		config:     rc,
		devices:    apiDevices,
		devicePool: devicePool,
	}
}

// GetConfig returns ResourceConfig for this resourcePool
func (rp *ResourcePoolImpl) GetConfig() *types.ResourceConfig {
	return rp.config
}

// InitDevice - not implemented
func (rp *ResourcePoolImpl) InitDevice() error {
	// Not implemented
	return nil
}

// GetAllocatePolicy returns the allocate policy as string
func (rp *ResourcePoolImpl) GetAllocatePolicy() string {
	return rp.config.AllocatePolicy
}

// GetResourceName returns the resource name as string
func (rp *ResourcePoolImpl) GetResourceName() string {
	return rp.config.ResourceName
}

// GetResourcePrefix returns the resource name prefix as string
func (rp *ResourcePoolImpl) GetResourcePrefix() string {
	return rp.config.ResourcePrefix
}

// GetDevices returns a map of Kubelet API devices
func (rp *ResourcePoolImpl) GetDevices() map[string]*pluginapi.Device {
	// returns all devices from devices[]
	return rp.devices
}

// Probe - does device healthcheck. Not implemented
func (rp *ResourcePoolImpl) Probe() bool {
	// TO-DO: Implement this
	return false
}

// GetDeviceSpecs returns list of plugin API device specs for a list of device IDs
func (rp *ResourcePoolImpl) GetDeviceSpecs(deviceIDs []string) []*pluginapi.DeviceSpec {
	glog.Infof("GetDeviceSpecs(): for devices: %v", deviceIDs)
	devSpecs := make([]*pluginapi.DeviceSpec, 0)

	// Add vfio group specific devices
	for _, id := range deviceIDs {
		if dev, ok := rp.devicePool[id]; ok {
			newSpecs := dev.GetDeviceSpecs()
			for _, ds := range newSpecs {
				if !rp.DeviceSpecExist(devSpecs, ds) {
					devSpecs = append(devSpecs, ds)
				}

			}

		}
	}
	return devSpecs
}

// GetEnvs returns a list of device specific Env values for device IDs
func (rp *ResourcePoolImpl) GetEnvs(deviceIDs []string) []string {
	glog.Infof("GetEnvs(): for devices: %v", deviceIDs)
	devEnvs := make([]string, 0)

	// Consolidates all Envs
	for _, id := range deviceIDs {
		if dev, ok := rp.devicePool[id]; ok {
			env := dev.GetEnvVal()
			devEnvs = append(devEnvs, env)
		}
	}

	return devEnvs
}

// GetMounts returns a list of Mount for device IDs
func (rp *ResourcePoolImpl) GetMounts(deviceIDs []string) []*pluginapi.Mount {
	glog.Infof("GetMounts(): for devices: %v", deviceIDs)
	devMounts := make([]*pluginapi.Mount, 0)

	for _, id := range deviceIDs {
		if dev, ok := rp.devicePool[id]; ok {
			mnt := dev.GetMounts()
			devMounts = append(devMounts, mnt...)
		}
	}
	return devMounts
}

// DeviceSpecExist checks if a DeviceSpec already exist in a DeviceSpec list
func (rp *ResourcePoolImpl) DeviceSpecExist(specs []*pluginapi.DeviceSpec, newSpec *pluginapi.DeviceSpec) bool {
	for _, sp := range specs {
		if sp.HostPath == newSpec.HostPath {
			return true
		}
	}
	return false
}

// GetDevicePool returns PciDevices pool as a map
func (rp *ResourcePoolImpl) GetDevicePool() map[string]types.PciDevice {
	return rp.devicePool
}

// StoreDeviceInfoFile does nothing. DeviceType-specific ResourcePools might
// store information according to the k8snetworkplumbingwg/device-info-spec
func (rp *ResourcePoolImpl) StoreDeviceInfoFile(resourceNamePrefix string) error {
	return nil
}

// CleanDeviceInfoFile does nothing. DeviceType-specific ResourcePools might
// clean the Device Info file
func (rp *ResourcePoolImpl) CleanDeviceInfoFile(resourceNamePrefix string) error {
	return nil
}
