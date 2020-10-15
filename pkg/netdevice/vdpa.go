/*
Copyright 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package netdevice

import (
	"fmt"
	"github.com/golang/glog"

	vdpa "github.com/redhat-virtio-net/govdpa/pkg/kvdpa"

	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

/*vdpaTypeToDriver translates vdpaTypes (as specified in the netDevice selectors)
  to vdpa bus drivers*/
var supportedVdpaTypes = map[types.VdpaType]string{
	types.VdpaVirtioType: vdpa.VirtioVdpaDriver,
	types.VdpaVhostType:  vdpa.VhostVdpaDriver,
}

type vdpaDevice struct {
	vdpa.VdpaDevice
}

// GetType returns the VdpaType associated with the VdpaDevice
func (v *vdpaDevice) GetType() types.VdpaType {
	for vtype, driver := range supportedVdpaTypes {
		if driver == v.GetDriver() {
			return vtype
		}
	}
	return types.VdpaInvalidType
}

// GetVdpaDevice returns a VdpaDevice from a given VF PCI address
func GetVdpaDevice(pciAddr string) types.VdpaDevice {
	detailVdpaDev, err := vdpa.GetVdpaDeviceByPci(pciAddr)
	if err != nil {
		glog.Infof("%s - No vDPA device found: %s", pciAddr, err)
		return nil
	}
	return &vdpaDevice{
		detailVdpaDev,
	}
}

// vdpaInfoProvider is a DeviceInfoProvider that handles the API information of vdpa-capable devices
type vdpaInfoProvider struct {
	dev      types.VdpaDevice
	vdpaType types.VdpaType
}

// NewVdpaInfoProvider returns a new InfoProvider associated with the given VDPAInfo
func NewVdpaInfoProvider(vdpaType types.VdpaType, vdpaDev types.VdpaDevice) types.DeviceInfoProvider {
	vdpaInfoProvider := &vdpaInfoProvider{
		dev:      vdpaDev,
		vdpaType: vdpaType,
	}
	// Still in Device construction time, so it's still not fatal if there are discrepancies
	// between the device information and the ResourceConfig. They should be removed by
	// selector filtering
	if healthy, err := vdpaInfoProvider.isHealthy(); !healthy {
		glog.Warning(err)
	}
	return vdpaInfoProvider
}

// GetDeviceSpecs returns the DeviceSpec slice
func (vip *vdpaInfoProvider) GetDeviceSpecs() []*pluginapi.DeviceSpec {
	if healthy, err := vip.isHealthy(); !healthy {
		glog.Errorf("GetDeviceSpecs(): vDPA is required in the configuration but device does not have a healthy vdpa device: %s",
			err)
		return nil
	}
	devSpecs := make([]*pluginapi.DeviceSpec, 0)

	// DeviceSpecs only required for vhost vdpa type as the
	if vip.vdpaType == types.VdpaVhostType {
		devSpecs = append(devSpecs, &pluginapi.DeviceSpec{
			HostPath:      vip.dev.GetPath(),
			ContainerPath: vip.dev.GetPath(),
			Permissions:   "mrw",
		})
	}
	return devSpecs
}

// GetEnvVal returns the environment variable value
func (vip *vdpaInfoProvider) GetEnvVal() string {
	return ""
}

// GetMounts returns the mount points (none for this InfoProvider)
func (vip *vdpaInfoProvider) GetMounts() []*pluginapi.Mount {
	mounts := make([]*pluginapi.Mount, 0)
	return mounts
}

// isHealthy returns whether the device's vDPA information is healthy
func (vip *vdpaInfoProvider) isHealthy() (bool, error) {
	if vip.dev == nil {
		return false, fmt.Errorf("No vDPA device found")
	}

	if _, ok := supportedVdpaTypes[vip.vdpaType]; !ok {
		return false, fmt.Errorf("VdpaType not supported %s", vip.vdpaType)
	}
	vType := vip.dev.GetType()
	if vType == types.VdpaInvalidType {
		return false, fmt.Errorf("Device does not have a valid vdpa types")
	}
	if vType != vip.vdpaType {
		return false, fmt.Errorf("Wrong vdpa type. Config expects %s but device is %s",
			vip.vdpaType, vType)
	}
	return true, nil
}
