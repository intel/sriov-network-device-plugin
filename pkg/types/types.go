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

package types

import (
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

const (
	// SockDir is the default Kubelet device plugin socket directory
	SockDir = "/var/lib/kubelet/device-plugins"
	// KubeEndPoint is kubelet socket name
	KubeEndPoint = "kubelet.sock"
)

// ResourceConfig contains cofiguration paremeters for a resource pool
type ResourceConfig struct {
	ResourceName string   `json:"resourceName"`         // the resource name will be added with resource prefix in K8s api
	RootDevices  []string `json:"rootDevices"`          // list of PCI address of root devices e.g. "0000:05:00.0",
	DeviceType   string   `json:"deviceType,omitempty"` // Device driver type of the device
	SriovMode    bool     `json:"sriovMode,omitempty"`  // Whether devices have SRIOV virtual function capabilities or not
	MdevMode     bool     `json:"mdevMode,omitempty"`   // Whether devices have Mediate device capabilities or not
}

// ResourceConfList is list of ResourceConfig
type ResourceConfList struct {
	ResourceList []ResourceConfig `json:"resourceList"` // config file: "resourceList" :[{<ResourceConfig configs>},{},{},...]
}

// ResourceServer is gRPC server implements K8s device plugin api
type ResourceServer interface {
	// Device manager API
	pluginapi.DevicePluginServer
	// grpc server related
	Start() error
	Stop() error
	// Init initializes resourcePool
	Init() error
	// Watch watches for socket file deleteion and restart server if needed
	Watch()
}

// ResourceFactory is an interface to get instances of ResourcePool and ResouceServer
type ResourceFactory interface {
	GetResourceServer(ResourcePool) (ResourceServer, error)
	GetResourcePool(*ResourceConfig) ResourcePool
}

// ResourcePool represents a generic resource entity
type ResourcePool interface {
	// extended API for internal use
	InitDevice() error
	DiscoverDevices() error
	GetResourceName() string
	GetConfig() *ResourceConfig

	GetDevices() map[string]*pluginapi.Device // for ListAndWatch
	GetDeviceFiles() map[string]string
	IBaseResource
}

// IBaseResource represents a specific resource pool
type IBaseResource interface {
	GetDeviceFile(dev string) (devFile string, err error)
	GetDeviceSpecs(deviceFiles map[string]string, deviceIDs []string) []*pluginapi.DeviceSpec
	GetEnvs(deviceIDs []string) map[string]string
	GetMounts() []*pluginapi.Mount
	// Probe does device health-check and update devices and returns 'true' if any of device in resource pool changed
	Probe(*ResourceConfig, map[string]*pluginapi.Device) bool
}
