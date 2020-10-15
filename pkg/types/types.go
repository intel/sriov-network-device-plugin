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
	"encoding/json"

	"github.com/jaypipes/ghw"
	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	vdpa "github.com/redhat-virtio-net/govdpa/pkg/kvdpa"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

var (
	// SockDir is the default Kubelet device plugin socket directory
	SockDir = "/var/lib/kubelet/plugins_registry"
	// DeprecatedSockDir is the deprecated Kubelet device plugin socket directory
	DeprecatedSockDir = "/var/lib/kubelet/device-plugins"
)

const (
	// KubeEndPoint is kubelet socket name
	KubeEndPoint = "kubelet.sock"
)

// DeviceType is custom type to define supported device types
type DeviceType string

// VdpaType is a type to define the supported vdpa device types
type VdpaType string

const (
	// NetDeviceType is DeviceType for network class devices
	NetDeviceType DeviceType = "netDevice"
	// AcceleratorType is DeviceType for accelerator class devices
	AcceleratorType DeviceType = "accelerator"

	// VdpaVirtioType is VdpaType for virtio-net devices
	VdpaVirtioType VdpaType = "virtio"
	// VdpaVhostType is VdpaType for vhost-vdpa devices
	VdpaVhostType VdpaType = "vhost"
	// VdpaInvalidType is VdpaType to represent an invalid or unsupported type
	VdpaInvalidType VdpaType = "invalid"
)

// SupportedDevices is map of 'device identifier as string' to 'device class hexcode as int'
/*
Supported PCI Device Classes. ref: https://pci-ids.ucw.cz/read/PD
02	Network controller
12	Processing accelerators

Network controller subclasses. ref: https://pci-ids.ucw.cz/read/PD/02
00	Ethernet controller
01	Token ring network controller
02	FDDI network controller
03	ATM network controller
04	ISDN controller
05	WorldFip controller
06	PICMG controller
07	Infiniband controller
08	Fabric controller
80	Network controller

Processing accelerators subclasses. ref: https://pci-ids.ucw.cz/read/PD/12
00	Processing accelerators
01	AI Inference Accelerator
*/
var SupportedDevices = map[DeviceType]int{
	NetDeviceType:   0x02,
	AcceleratorType: 0x12,
}

// ResourceConfig contains configuration paremeters for a resource pool
type ResourceConfig struct {
	ResourcePrefix string           `json:"resourcePrefix,omitempty"` // optional resource prefix that will ovewrite global prefix specified in cli params
	ResourceName   string           `json:"resourceName"`             // the resource name will be added with resource prefix in K8s api
	DeviceType     DeviceType       `json:"deviceType,omitempty"`
	Selectors      *json.RawMessage `json:"selectors,omitempty"`
	SelectorObj    interface{}
}

// DeviceSelectors contains common device selectors fields
type DeviceSelectors struct {
	Vendors      []string `json:"vendors,omitempty"`
	Devices      []string `json:"devices,omitempty"`
	Drivers      []string `json:"drivers,omitempty"`
	PciAddresses []string `json:"pciAddresses,omitempty"`
}

// NetDeviceSelectors contains network device related selectors fields
type NetDeviceSelectors struct {
	DeviceSelectors
	PfNames      []string `json:"pfNames,omitempty"`
	RootDevices  []string `json:"rootDevices,omitempty"`
	LinkTypes    []string `json:"linkTypes,omitempty"`
	DDPProfiles  []string `json:"ddpProfiles,omitempty"`
	IsRdma       bool     // the resource support rdma
	NeedVhostNet bool     // share vhost-net along the selected ressource
	VdpaType     VdpaType `json:"vdpaType,omitempty"`
}

// AccelDeviceSelectors contains accelerator(FPGA etc.) related selectors fields
type AccelDeviceSelectors struct {
	DeviceSelectors
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
	GetDefaultInfoProvider(string, string) DeviceInfoProvider
	GetSelector(string, []string) (DeviceSelector, error)
	GetResourcePool(rc *ResourceConfig, deviceList []PciDevice) (ResourcePool, error)
	GetRdmaSpec(string) RdmaSpec
	GetVdpaDevice(string) VdpaDevice
	GetDeviceProvider(DeviceType) DeviceProvider
	GetDeviceFilter(*ResourceConfig) (interface{}, error)
	GetNadUtils() NadUtils
}

// ResourcePool represents a generic resource entity
type ResourcePool interface {
	// extended API for internal use
	GetResourceName() string
	GetResourcePrefix() string
	GetDevices() map[string]*pluginapi.Device // for ListAndWatch
	Probe() bool
	GetDeviceSpecs(deviceIDs []string) []*pluginapi.DeviceSpec
	GetEnvs(deviceIDs []string) []string
	GetMounts(deviceIDs []string) []*pluginapi.Mount
	StoreDeviceInfoFile(resourceNamePrefix string) error
	CleanDeviceInfoFile(resourceNamePrefix string) error
}

// DeviceProvider provides interface for device discovery
type DeviceProvider interface {
	// AddTargetDevices adds a list of devices in a DeviceProvider that matches the 'device class hexcode as int'
	AddTargetDevices([]*ghw.PCIDevice, int) error
	GetDiscoveredDevices() []*ghw.PCIDevice

	// Get Devices runs through the Discovered Devices and returns a list of fully populated PciDevices accoring to the given ResourceConfig
	GetDevices(*ResourceConfig) []PciDevice

	GetFilteredDevices([]PciDevice, *ResourceConfig) ([]PciDevice, error)
}

// PciDevice provides an interface to get generic device specific information
type PciDevice interface {
	GetVendor() string
	GetDriver() string
	GetDeviceCode() string
	GetPciAddr() string
	GetPfPciAddr() string
	IsSriovPF() bool
	GetSubClass() string
	GetDeviceSpecs() []*pluginapi.DeviceSpec
	GetEnvVal() string
	GetMounts() []*pluginapi.Mount
	GetAPIDevice() *pluginapi.Device
	GetVFID() int
	GetNumaInfo() string
}

// PciNetDevice extends PciDevice interface
type PciNetDevice interface {
	PciDevice
	GetPFName() string
	GetNetName() string
	GetLinkSpeed() string
	GetLinkType() string
	GetRdmaSpec() RdmaSpec
	GetDDPProfiles() string
	GetVdpaDevice() VdpaDevice
}

// AccelDevice extends PciDevice interface
type AccelDevice interface {
	PciDevice
}

// DeviceInfoProvider is an interface to get Device Plugin API specific device information
type DeviceInfoProvider interface {
	GetDeviceSpecs() []*pluginapi.DeviceSpec
	GetEnvVal() string
	GetMounts() []*pluginapi.Mount
}

// DeviceSelector provides an interface for filtering a list of devices
type DeviceSelector interface {
	Filter([]PciDevice) []PciDevice
}

// LinkWatcher in interface to watch Network link status
type LinkWatcher interface { // This is not fully defined yet!!
	Subscribe()
}

// RdmaSpec rdma device data
type RdmaSpec interface {
	IsRdma() bool
	GetRdmaDeviceSpec() []*pluginapi.DeviceSpec
}

// NadUtils is an interface for Network-Attachment-Definition utilities
type NadUtils interface {
	SaveDeviceInfoFile(resourceName string, deviceID string, devInfo *nettypes.DeviceInfo) error
	CleanDeviceInfoFile(resourceName string, deviceID string) error
}

// VdpaDevice is an interface to access vDPA device information
type VdpaDevice interface {
	vdpa.VdpaDevice
	GetType() VdpaType
}
