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

	"github.com/intel/sriov-network-device-plugin/pkg/resources"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"github.com/intel/sriov-network-device-plugin/pkg/utils"
)

const (
	socketSuffix = "sock"
)

type cliParams struct {
	configFile     string
	resourcePrefix string
}

type resourceManager struct {
	cliParams
	socketSuffix    string
	rFactory        types.ResourceFactory
	configList      []*types.ResourceConfig // resourceName -> resourcePool
	resourceServers []types.ResourceServer
}

func newResourceManager(cp *cliParams) *resourceManager {
	return &resourceManager{
		cliParams: *cp,
		rFactory:  resources.NewResourceFactory(cp.resourcePrefix, socketSuffix),
	}
}

// Read and validate configurations from Config file
func (rm *resourceManager) readConfig() error {

	resources := &types.ResourceConfList{}
	rawBytes, err := ioutil.ReadFile(rm.configFile)

	if err != nil {
		return fmt.Errorf("error reading file %s, %v", rm.configFile, err)

	}

	if err = json.Unmarshal(rawBytes, resources); err != nil {
		return fmt.Errorf("error unmarshalling raw bytes %v", err)
	}

	for i := range resources.ResourceList {
		rm.configList = append(rm.configList, &resources.ResourceList[i])
	}

	return nil
}

func (rm *resourceManager) initServers() error {
	rf := rm.rFactory
	fmt.Printf("number of config: %d\n", len(rm.configList))
	for _, rc := range rm.configList {
		fmt.Printf("Resource name: %+v\n", rc)
		s, err := rf.GetResourceServer(rf.GetResourcePool(rc))
		if err != nil {
			return err
		}

		// Only add a server to the managed server list if it can be initialized without any error
		if err := s.Init(); err != nil {
			glog.Errorf("error initializing server: %v", err)
			glog.Errorf("skipping...")
			continue
		}
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
		go rs.Watch()
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
	resourceName := make(map[string]string) // resource name placeholder

	for _, conf := range rm.configList {
		// check pci addresses
		pciAddrs := conf.RootDevices
		for i, addr := range pciAddrs {
			parsedAddr, err := utils.ValidPciAddr(addr)
			if err != nil {
				glog.Errorf("error validating pci addr: %v", err)
				return false
			}
			pciAddrs[i] = parsedAddr
		}

		// check if name contains acceptable characters
		if !utils.ValidResourceName(conf.ResourceName) {
			glog.Errorf("resource name \"%s\" contains invalid characters", conf.ResourceName)
			return false
		}
		// check resource names are unique
		_, ok := resourceName[conf.ResourceName]
		if ok {
			// resource name already exist
			glog.Errorf("resource name \"%s\" already exists", conf.ResourceName)
			return false
		}

		// validate sriovMode
		if conf.SriovMode {
			// Check that VF are created for all root devices
			for _, addr := range pciAddrs {
				if configured := utils.SriovConfigured(addr); !configured {
					glog.Errorf("no SRIOV VF configured for root device %s", addr)
					return false
				}
			}
		}

		// validate MDEV
		if conf.MdevMode {
			// Check that MDEV devices are created for all root devices
			for _, addr := range pciAddrs {
				if configured := utils.MdevConfigured(addr); !configured {
					glog.Errorf("no Mdev configured for root device %s", addr)
					return false
				}
			}
		}

		// [To-Do]: Validate deviceType

		resourceName[conf.ResourceName] = conf.ResourceName
	}

	return true
}
