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
	"flag"
	// "os"
	// "os/signal"
	// "syscall"

	"github.com/golang/glog"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/config"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/features"
)

const (
	defaultConfig = "/etc/pcidp/config.json"
)

// Parse Command line flags
func flagInit(cp *cliParams) {
	flag.StringVar(&cp.configFile, "config-file", defaultConfig,
		"JSON device pool config file location")
	flag.StringVar(&cp.resourcePrefix, "resource-prefix", "intel.com",
		"resource name prefix used for K8s extended resource")
	flag.StringVar(&cp.featureGates, "feature-gates", "",
		"enables or disables selected features")
}

func main() {
	cp := &cliParams{}
	flagInit(cp)
	flag.Parse()
	rm := newResourceManager(cp)

	glog.Infof("resource manager reading configs")
	if err := rm.readConfig(); err != nil {
		glog.Errorf("error getting resources from file %v", err)
		return
	}

	if len(rm.configList) < 1 {
		glog.Errorf("no resource configuration; exiting")
		return // No config found
	}

	// Validate configs
	if !rm.validConfigs() {
		glog.Fatalf("Exiting.. one or more invalid configuration(s) given")
		return
	}

	// Read global config
	if err := config.Config.ReadConfig(cp.configFile); err != nil {
		glog.Error(err)
		return
	}

	// Set FeatureGates with ConfigMap
	if err := features.FeatureGate.SetFromMap(config.Config.FeatureGates); err != nil {
		glog.Error(err)
		return
	}

	// Set FeatureGates with CLI arguments
	if err := features.FeatureGate.SetFromString(cp.featureGates); err != nil {
		glog.Error(err)
		return
	}

	glog.Infof("Discovering host devices")
	if err := rm.discoverHostDevices(); err != nil {
		glog.Errorf("error discovering host devices%v", err)
		return
	}

	// glog.Infof("Initializing resource servers")
	// if err := rm.initServers(); err != nil {
	// 	glog.Errorf("error initializing resource servers %v", err)
	// 	return
	// }

	// glog.Infof("Starting all servers...")
	// if err := rm.startAllServers(); err != nil {
	// 	glog.Errorf("error starting resource servers %v\n", err)
	// 	return
	// }
	// glog.Infof("All servers started.")
	// glog.Infof("Listening for term signals")
	// // respond to syscalls for termination
	// sigCh := make(chan os.Signal, 1)
	// signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// // Catch termination signals
	// select {
	// case sig := <-sigCh:
	// 	glog.Infof("Received signal \"%v\", shutting down.", sig)
	// 	rm.stopAllServers()
	// 	return
	// }
}
