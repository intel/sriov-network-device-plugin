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
	"github.com/golang/glog"

	vdpa "github.com/redhat-virtio-net/govdpa/pkg/kvdpa"

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
