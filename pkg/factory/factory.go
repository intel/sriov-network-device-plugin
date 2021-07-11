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

package factory

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/accelerator"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/netdevice"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/resources"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

type resourceFactory struct {
	endPointPrefix string
	endPointSuffix string
	pluginWatch    bool
}

var instance *resourceFactory

// NewResourceFactory returns an instance of Resource Server factory
func NewResourceFactory(prefix, suffix string, pluginWatch bool) types.ResourceFactory {

	if instance == nil {
		return &resourceFactory{
			endPointPrefix: prefix,
			endPointSuffix: suffix,
			pluginWatch:    pluginWatch,
		}
	}
	return instance
}

// GetResourceServer returns an instance of ResourceServer for a ResourcePool
func (rf *resourceFactory) GetResourceServer(rp types.ResourcePool) (types.ResourceServer, error) {
	if rp != nil {
		prefix := rf.endPointPrefix
		if prefixOverride := rp.GetResourcePrefix(); prefixOverride != "" {
			prefix = prefixOverride
		}
		policy := rp.GetAllocatePolicy()
		return resources.NewResourceServer(prefix, rf.endPointSuffix, rf.pluginWatch, rp, rf.GetAllocator(policy)), nil
	}
	return nil, fmt.Errorf("factory: unable to get resource pool object")
}

// GetAllocator returns an instance of Allocator using preferredAllocationPolicy
func (rf *resourceFactory) GetAllocator(policy string) types.Allocator {
	switch policy {
	case "packed":
		return resources.NewPackedAllocator()
	default:
		return nil
	}
}

// GetDefaultInfoProvider returns an instance of DeviceInfoProvider using name as string
func (rf *resourceFactory) GetDefaultInfoProvider(pciAddr string, name string) types.DeviceInfoProvider {
	switch name {
	case "vfio-pci":
		return resources.NewVfioInfoProvider(pciAddr)
	case "uio", "igb_uio":
		return resources.NewUioInfoProvider(pciAddr)
	default:
		return resources.NewGenericInfoProvider(pciAddr)
	}
}

// GetSelector returns an instance of DeviceSelector using selector attribute string and its associated values
func (rf *resourceFactory) GetSelector(attr string, values []string) (types.DeviceSelector, error) {
	switch attr {
	case "vendors":
		return resources.NewVendorSelector(values), nil
	case "devices":
		return resources.NewDeviceSelector(values), nil
	case "drivers":
		return resources.NewDriverSelector(values), nil
	case "pciAddresses":
		return resources.NewPciAddressSelector(values), nil
	case "pfNames":
		return resources.NewPfNameSelector(values), nil
	case "rootDevices":
		return resources.NewRootDeviceSelector(values), nil
	case "linkTypes":
		return resources.NewLinkTypeSelector(values), nil
	case "ddpProfiles":
		return resources.NewDdpSelector(values), nil
	default:
		return nil, fmt.Errorf("GetSelector(): invalid attribute %s", attr)
	}
}

// GetResourcePool returns an instance of resourcePool
func (rf *resourceFactory) GetResourcePool(rc *types.ResourceConfig, filteredDevice []types.PciDevice) (types.ResourcePool, error) {

	devicePool := make(map[string]types.PciDevice, 0)
	apiDevices := make(map[string]*pluginapi.Device)
	for _, dev := range filteredDevice {
		pciAddr := dev.GetPciAddr()
		devicePool[pciAddr] = dev
		apiDevices[pciAddr] = dev.GetAPIDevice()
		glog.Infof("device added: [pciAddr: %s, vendor: %s, device: %s, driver: %s]",
			dev.GetPciAddr(),
			dev.GetVendor(),
			dev.GetDeviceCode(),
			dev.GetDriver())
	}

	var rPool types.ResourcePool
	var err error
	switch rc.DeviceType {
	case types.NetDeviceType:
		if len(filteredDevice) > 0 {
			if _, ok := filteredDevice[0].(types.PciNetDevice); ok {
				nadUtils := rf.GetNadUtils()
				rPool = netdevice.NewNetResourcePool(nadUtils, rc, apiDevices, devicePool)
			} else {
				err = fmt.Errorf("invalid device list for NetDeviceType")
			}
		}
	case types.AcceleratorType:
		if len(filteredDevice) > 0 {
			if _, ok := filteredDevice[0].(types.AccelDevice); ok {
				rPool = accelerator.NewAccelResourcePool(rc, apiDevices, devicePool)
			} else {
				err = fmt.Errorf("invalid device list for AcceleratorType")
			}
		}
	default:
		err = fmt.Errorf("cannot create resourcePool: invalid device type %s", rc.DeviceType)
	}
	return rPool, err
}

func (rf *resourceFactory) GetRdmaSpec(pciAddrs string) types.RdmaSpec {
	return netdevice.NewRdmaSpec(pciAddrs)
}

// GetDeviceProvider returns an instance of DeviceProvider based on DeviceType
func (rf *resourceFactory) GetDeviceProvider(dt types.DeviceType) types.DeviceProvider {
	switch dt {
	case types.NetDeviceType:
		return netdevice.NewNetDeviceProvider(rf)
	case types.AcceleratorType:
		return accelerator.NewAccelDeviceProvider(rf)
	default:
		return nil
	}
}

// GetDeviceFilter unmarshal the "selector" values from ResourceConfig and returns an instance of DeviceSelector based on
// DeviceType in the ResourceConfig
func (rf *resourceFactory) GetDeviceFilter(rc *types.ResourceConfig) (interface{}, error) {
	switch rc.DeviceType {
	case types.NetDeviceType:
		netDeviceSelector := &types.NetDeviceSelectors{}

		if err := json.Unmarshal(*rc.Selectors, netDeviceSelector); err != nil {
			return nil, fmt.Errorf("error unmarshalling NetDevice selector bytes %v", err)
		}

		glog.Infof("net device selector for resource %s is %+v", rc.ResourceName, netDeviceSelector)
		return netDeviceSelector, nil
	case types.AcceleratorType:
		accelDeviceSelector := &types.AccelDeviceSelectors{}

		if err := json.Unmarshal(*rc.Selectors, accelDeviceSelector); err != nil {
			return nil, fmt.Errorf("error unmarshalling Accelerator selector bytes %v", err)
		}

		glog.Infof("accelerator device selector for resource %s is %+v", rc.ResourceName, accelDeviceSelector)
		return accelDeviceSelector, nil
	default:
		return nil, fmt.Errorf("unable to get deviceFilter, invalid deviceType %s", rc.DeviceType)
	}
}

// GetNadUtils returns an instance of NadUtils
func (rf *resourceFactory) GetNadUtils() types.NadUtils {
	return netdevice.NewNadUtils()
}
