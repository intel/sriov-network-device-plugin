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

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/golang/glog"
	"github.com/jaypipes/ghw"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

const (
	socketSuffix = "sock"
)

type cliParams struct {
	configFile     string
	resourcePrefix string
	featureGates   string
}

type resourceManager struct {
	cliParams
	pluginWatchMode bool
	socketSuffix    string
	rFactory        types.ResourceFactory
	configList      []*types.ResourceConfig
	resourceServers []types.ResourceServer
	deviceProviders map[types.DeviceType]types.DeviceProvider
}

func newResourceManager(cp *cliParams) *resourceManager {
	pluginWatchMode := utils.DetectPluginWatchMode(types.SockDir)
	if pluginWatchMode {
		glog.Infof("Using Kubelet Plugin Registry Mode")
	} else {
		glog.Infof("Using Deprecated Device Plugin Registry Path")
	}

	rf := factory.NewResourceFactory(cp.resourcePrefix, socketSuffix, pluginWatchMode)
	dp := make(map[types.DeviceType]types.DeviceProvider)
	for k := range types.SupportedDevices {
		dp[k] = rf.GetDeviceProvider(k)
	}

	return &resourceManager{
		cliParams:       *cp,
		pluginWatchMode: pluginWatchMode,
		rFactory:        rf,
		deviceProviders: dp,
	}
}

//readConfig reads and validate configurations from Config file
func (rm *resourceManager) readConfig() error {

	resources := &types.ResourceConfList{}
	rawBytes, err := ioutil.ReadFile(rm.configFile)

	if err != nil {
		return fmt.Errorf("error reading file %s, %v", rm.configFile, err)

	}

	glog.Infof("raw ResourceList: %s", rawBytes)
	if err = json.Unmarshal(rawBytes, resources); err != nil {
		return fmt.Errorf("error unmarshalling raw bytes %v please make sure the config is in json format", err)
	}

	for i := range resources.ResourceList {
		conf := &resources.ResourceList[i]
		// Validate deviceType
		if conf.DeviceType == "" {
			conf.DeviceType = types.NetDeviceType // Default to NetDeviceType
		} else {
			// Check if the DeviceType is supported
			if _, ok := types.SupportedDevices[conf.DeviceType]; !ok {
				return fmt.Errorf("unsupported deviceType:  \"%s\"", conf.DeviceType)
			}
		}
		if conf.SelectorObj, err = rm.rFactory.GetDeviceFilter(conf); err == nil {
			rm.configList = append(rm.configList, &resources.ResourceList[i])
		} else {
			glog.Warningf("unable to get SelectorObj from selectors list:'%s' for deviceType: %s error: %s",
				*conf.Selectors, conf.DeviceType, err)
		}

	}
	glog.Infof("unmarshalled ResourceList: %+v", resources.ResourceList)
	return nil
}

func (rm *resourceManager) initServers() error {
	rf := rm.rFactory
	glog.Infof("number of config: %d\n", len(rm.configList))
	for _, rc := range rm.configList {
		// Create new ResourcePool
		glog.Infof("")
		glog.Infof("Creating new ResourcePool: %s", rc.ResourceName)
		glog.Infof("DeviceType: %+v", rc.DeviceType)
		dp, ok := rm.deviceProviders[rc.DeviceType]
		if !ok {
			glog.Infof("Unable to get device provider from deviceType: %s", rc.DeviceType)
			return fmt.Errorf("error getting device provider")
		}

		devices := dp.GetDevices(rc)
		filteredDevices, err := dp.GetFilteredDevices(devices, rc)
		if err != nil {
			glog.Errorf("initServers(): error getting filtered devices for config %+v: %q", rc, err)
		}
		if len(filteredDevices) < 1 {
			glog.Infof("no devices in device pool, skipping creating resource server for %s", rc.ResourceName)
			continue
		}
		rPool, err := rm.rFactory.GetResourcePool(rc, filteredDevices)
		if err != nil {
			glog.Errorf("initServers(): error creating ResourcePool with config %+v: %q", rc, err)
			return err
		}

		// Create ResourceServer with this ResourcePool
		s, err := rf.GetResourceServer(rPool)
		if err != nil {
			glog.Errorf("initServers(): error creating ResourceServer: %v", err)
			return err
		}
		glog.Infof("New resource server is created for %s ResourcePool", rc.ResourceName)
		rm.resourceServers = append(rm.resourceServers, s)
	}
	return nil
}

func (rm *resourceManager) startAllServers() error {
	for _, rs := range rm.resourceServers {
		if err := rs.Start(); err != nil {
			return err
		}

		// start watcher
		if !rm.pluginWatchMode {
			go rs.Watch()
		}
	}
	return nil
}

func (rm *resourceManager) stopAllServers() error {
	for _, rs := range rm.resourceServers {
		if err := rs.Stop(); err != nil {
			return err
		}
	}
	return nil
}

// Validate configurations
func (rm *resourceManager) validConfigs() bool {
	resourceNames := make(map[string]string) // resource names placeholder

	for _, conf := range rm.configList {
		// check if name contains acceptable characters
		if !utils.ValidResourceName(conf.ResourceName) {
			glog.Errorf("resource name \"%s\" contains invalid characters", conf.ResourceName)
			return false
		}

		// resourcePrefix might be overriden for a given resource pool
		resourcePrefix := rm.cliParams.resourcePrefix
		if conf.ResourcePrefix != "" {
			resourcePrefix = conf.ResourcePrefix
		}

		resourceName := resourcePrefix + "/" + conf.ResourceName

		glog.Infof("validating resource name \"%s\"", resourceName)

		// ensure that resource name is unique
		if _, exists := resourceNames[resourceName]; exists {
			// resource name already exist
			glog.Errorf("resource name \"%s\" already exists", resourceName)
			return false
		}

		// Check if the DeviceType is valid
		if _, ok := types.SupportedDevices[conf.DeviceType]; !ok {
			glog.Errorf("unsupported deviceType:  \"%s\" already exists", conf.DeviceType)
			return false
		}

		resourceNames[resourceName] = resourceName
	}

	return true
}

func (rm *resourceManager) discoverHostDevices() error {

	pci, err := ghw.PCI()
	if err != nil {
		return fmt.Errorf("discoverDevices(): error getting PCI info: %v", err)
	}

	devices := pci.ListDevices()
	if len(devices) == 0 {
		glog.Warningf("discoverDevices(): no PCI network device found")
	}

	for k, v := range types.SupportedDevices {
		if dp, ok := rm.deviceProviders[k]; ok {
			dp.AddTargetDevices(devices, v)
		}
	}
	return nil
}
