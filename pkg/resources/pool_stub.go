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
	"fmt"

	"github.com/golang/glog"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"github.com/intel/sriov-network-device-plugin/pkg/utils"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

type resourcePool struct {
	config      *types.ResourceConfig
	devices     map[string]*pluginapi.Device
	bindScript  string
	deviceFiles map[string]string // DeviceID -> Device file; e.g. 00:05.01 -> /dev/uio1
	types.IBaseResource
}

var _ types.ResourcePool = &resourcePool{}

func newResourcePool(rc *types.ResourceConfig, bScript string) types.ResourcePool {
	return &resourcePool{
		config:      rc,
		devices:     make(map[string]*pluginapi.Device),
		bindScript:  bScript,
		deviceFiles: make(map[string]string),
	}
}

func (rp *resourcePool) GetConfig() *types.ResourceConfig {
	return rp.config
}

func (rp *resourcePool) InitDevice() error {
	// Not implemented
	return nil
}
func (rp *resourcePool) DiscoverDevices() error {
	// discover SRIOV VFs
	// populate devices[] for each device discovered
	glog.Infof("Discovering devices with config: %+v", rp.config)
	deviceList := make([]string, 0)
	SriovMode := rp.config.SriovMode
	MdevMode := rp.config.MdevMode

	if SriovMode {

		// Discover VFs
		for _, pf := range rp.config.RootDevices {
			numvfs := utils.GetSriovVFcapacity(pf)
			glog.Infof("Total number of VFs for device %v is %v", pf, numvfs)
			if numvfs > 0 {
				glog.Infof("SRIOV capable device discovered: %v", pf)

				numConfiguredVFs := utils.GetVFconfigured(pf)
				glog.Infof("Number of Configured VFs for device %v is %d", pf, numConfiguredVFs)

				if numConfiguredVFs < 1 {
					glog.Errorf("Error. Virtual Functions are not configured for the device: %s", pf)
					return fmt.Errorf("Virtual Functions are not configured for the device: %s", pf)
				}

				if newList, err := utils.GetVFList(pf); err == nil {
					deviceList = append(deviceList, newList...)
				}
			}
		}
	} else if MdevMode {

		// Discover VFs
		for _, pf := range rp.config.RootDevices {

			supportConfiguredMDEVs := utils.GetMDEVconfigured(pf)
			glog.Infof("Number of Configured VFs for device %v is %d", pf, supportConfiguredMDEVs)

			if !supportConfiguredMDEVs {
				glog.Errorf("Error. Mediated Device are not configured for the device: %s", pf)
				return fmt.Errorf("Mediated Device are not configured for the device: %s", pf)
			}

			if newList, err := utils.GetMDEVList(pf); err == nil {
				fmt.Println("Found new MDEV device:", newList)
				deviceList = append(deviceList, newList...)
			}
		}
	} else { // PFs only
		deviceList = rp.config.RootDevices
	}

	// Create plugin.Device instaces for all devices
	for _, dev := range deviceList {
		devFile, err := rp.GetDeviceFile(dev)
		if err == nil {
			rp.devices[dev] = &pluginapi.Device{
				ID:     dev,
				Health: pluginapi.Healthy}
			rp.deviceFiles[dev] = devFile
		}
	}
	glog.Infof("Discovered Devices: %v\n", rp.devices)

	return nil
}

func (rp *resourcePool) GetResourceName() string {
	return rp.config.ResourceName // dummy return
}

func (rp *resourcePool) GetDevices() map[string]*pluginapi.Device {
	// returns all devices from devices[]
	return rp.devices
}

func (rp *resourcePool) GetDeviceFiles() map[string]string {
	return rp.deviceFiles
}
