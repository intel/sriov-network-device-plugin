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
	"sort"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// DeviceSet is to hold and manipulate a set of PciDevice
type DeviceSet map[string]types.PciDevice

// PackedAllocator implements the Allocator interface
type PackedAllocator struct {
}

// Allocator implements the Allocator interface
type Allocator struct {
}

// NewPackedAllocator create instance of PackedAllocator
func NewPackedAllocator() types.Allocator {
	return &PackedAllocator{}
}

// NewAllocator create instance of Allocator
func NewAllocator() types.Allocator {
	return &Allocator{}
}

// NewDeviceSet is to create an empty DeviceSet
func NewDeviceSet() DeviceSet {
	set := make(DeviceSet)
	return set
}

// Insert is to add a PciDevice in DeviceSet
func (ds DeviceSet) Insert(pciAddr string, device types.PciDevice) {
	ds[pciAddr] = device
}

// Delete is to delete a PciDevice in DeviceSet
func (ds DeviceSet) Delete(pciAddr string) {
	delete(ds, pciAddr)
}

// Sort is to sort the DeviceSet and return the sorted keys
func (ds DeviceSet) Sort() []string {
	keys := make([]string, 0, len(ds))
	for k := range ds {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Allocate return the preferred allocation
func (pa *PackedAllocator) Allocate(rqt *pluginapi.ContainerPreferredAllocationRequest, rp types.ResourcePool) []string {
	size := rqt.AllocationSize
	preferredDevices := make([]string, 0)
	if size <= 0 || len(rqt.AvailableDeviceIDs) < int(size) || len(rqt.MustIncludeDeviceIDs) > int(size) {
		return preferredDevices
	}

	availableSet := NewDeviceSet()
	for _, available := range rqt.AvailableDeviceIDs {
		dev, ok := rp.GetDevicePool()[available]
		if ok {
			availableSet.Insert(available, dev)
		} else {
			return preferredDevices
		}
	}
	for _, required := range rqt.MustIncludeDeviceIDs {
		_, ok := rp.GetDevicePool()[required]
		if ok {
			availableSet.Delete(required)
		} else {
			return preferredDevices
		}
	}
	sortedAvailableSet := availableSet.Sort()

	preferredDevices = append(preferredDevices, rqt.MustIncludeDeviceIDs...)
	if len(preferredDevices) < int(size) {
		preferredDevices = append(preferredDevices, sortedAvailableSet[:int(size)-len(preferredDevices)]...)
	}
	return preferredDevices
}

// Allocate return the preferred allocation
func (a *Allocator) Allocate(rqt *pluginapi.ContainerPreferredAllocationRequest, rp types.ResourcePool) []string {
	return make([]string, 0)
}
