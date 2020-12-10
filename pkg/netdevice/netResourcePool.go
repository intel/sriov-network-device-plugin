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

package netdevice

import (
	"fmt"
	"github.com/golang/glog"
	"strings"

	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/resources"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

type netResourcePool struct {
	*resources.ResourcePoolImpl
	selectors *types.NetDeviceSelectors
	nadutils  types.NadUtils
}

var _ types.ResourcePool = &netResourcePool{}

// NewNetResourcePool returns an instance of resourcePool
func NewNetResourcePool(nadutils types.NadUtils, rc *types.ResourceConfig, apiDevices map[string]*pluginapi.Device, devicePool map[string]types.PciDevice) types.ResourcePool {
	rp := resources.NewResourcePool(rc, apiDevices, devicePool)
	s, _ := rc.SelectorObj.(*types.NetDeviceSelectors)
	return &netResourcePool{
		ResourcePoolImpl: rp,
		selectors:        s,
		nadutils:         nadutils,
	}
}

// Overrides GetDeviceSpecs
func (rp *netResourcePool) GetDeviceSpecs(deviceIDs []string) []*pluginapi.DeviceSpec {
	glog.Infof("GetDeviceSpecs(): for devices: %v", deviceIDs)
	devSpecs := make([]*pluginapi.DeviceSpec, 0)

	devicePool := rp.GetDevicePool()

	// Add device driver specific and rdma specific devices
	for _, id := range deviceIDs {
		if dev, ok := devicePool[id]; ok {
			netDev := dev.(types.PciNetDevice) // convert generic PciDevice to PciNetDevice
			newSpecs := netDev.GetDeviceSpecs()
			for _, ds := range newSpecs {
				if !rp.DeviceSpecExist(devSpecs, ds) {
					devSpecs = append(devSpecs, ds)
				}

			}

		}
	}
	return devSpecs
}

// StoreDeviceInfoFile stores the Device Info files according to the
//  k8snetworkplumbingwg/device-info-spec
func (rp *netResourcePool) StoreDeviceInfoFile(resourceNamePrefix string) error {
	var devInfo nettypes.DeviceInfo
	for id, dev := range rp.GetDevicePool() {
		netDev, ok := dev.(types.PciNetDevice)
		if !ok {
			return fmt.Errorf("StoreDeviceInfoFile: Only pciNetDevices are supported")
		}

		vdpaDev := netDev.GetVdpaDevice()
		if vdpaDev != nil {
			vdpaType := vdpaDev.GetType()
			devInfo = nettypes.DeviceInfo{
				Type:    nettypes.DeviceInfoTypeVDPA,
				Version: nettypes.DeviceInfoVersion,
				Vdpa: &nettypes.VdpaDevice{
					ParentDevice: vdpaDev.GetParent(),
					Driver:       string(vdpaType),
					Path:         vdpaDev.GetPath(),
					PciAddress:   netDev.GetPciAddr(),
				},
			}

		} else {
			devInfo = nettypes.DeviceInfo{
				Type:    nettypes.DeviceInfoTypePCI,
				Version: nettypes.DeviceInfoVersion,
				Pci: &nettypes.PciDevice{
					PciAddress: netDev.GetPciAddr(),
				},
			}
		}
		resource := fmt.Sprintf("%s/%s", resourceNamePrefix, rp.GetConfig().ResourceName)
		if err := rp.nadutils.SaveDeviceInfoFile(resource, id, &devInfo); err != nil {
			return err
		}
	}
	return nil
}

// CleanDeviceInfoFile cleans the Device Info files
func (rp *netResourcePool) CleanDeviceInfoFile(resourceNamePrefix string) error {
	errors := make([]string, 0)
	for id := range rp.GetDevicePool() {
		resource := fmt.Sprintf("%s/%s", resourceNamePrefix, rp.GetConfig().ResourceName)
		if err := rp.nadutils.CleanDeviceInfoFile(resource, id); err != nil {
			// Continue trying to clean.
			errors = append(errors, err.Error())
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, ","))
	}
	return nil
}
