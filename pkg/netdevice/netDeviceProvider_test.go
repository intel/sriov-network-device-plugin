// Copyright 2020 Intel Corp. All Rights Reserved.
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

package netdevice_test

import (
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/netdevice"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types/mocks"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/pcidb"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetDeviceProvider", func() {
	Describe("getting new instance of netDeviceProvider", func() {
		Context("with correct arguments", func() {
			rf := &mocks.ResourceFactory{}
			p := netdevice.NewNetDeviceProvider(rf)
			It("should return valid instance of the provider", func() {
				Expect(p).ToNot(BeNil())
			})
		})
	})
	Describe("getting devices", func() {
		Context("when there are none", func() {
			rf := &mocks.ResourceFactory{}
			p := netdevice.NewNetDeviceProvider(rf)
			config := &types.ResourceConfig{}
			dDevs := p.GetDiscoveredDevices()
			devs := p.GetDevices(config)
			It("should return empty slice", func() {
				Expect(dDevs).To(BeEmpty())
				Expect(devs).To(BeEmpty())
			})
		})
	})
	Describe("adding 3 target devices", func() {
		Context("when 2 are valid devices, but 1 is a PF with SRIOV configured and 1 is invalid", func() {
			fs := &utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.1",
					"sys/bus/pci/devices/0000:00:00.2",
					"sys/bus/pci/drivers/fake",
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:00:00.1/driver": "../../../../bus/pci/drivers/fake",
					"sys/bus/pci/devices/0000:00:00.2/driver": "../../../../bus/pci/drivers/fake",
				},
				Files: map[string][]byte{
					"sys/bus/pci/devices/0000:00:00.1/sriov_numvfs":   []byte("32"),
					"sys/bus/pci/devices/0000:00:00.1/sriov_totalvfs": []byte("64"),
					"sys/bus/pci/devices/0000:00:00.2/sriov_numvfs":   []byte("0"),
				},
			}

			defer fs.Use()()

			rf := factory.NewResourceFactory("fake", "fake", true)
			p := netdevice.NewNetDeviceProvider(rf)
			config := &types.ResourceConfig{
				DeviceType: types.NetDeviceType,
				SelectorObj: types.NetDeviceSelectors{
					DeviceSelectors: types.DeviceSelectors{},
				},
			}
			dev1 := &ghw.PCIDevice{
				Address: "0000:00:00.1",
				Class:   &pcidb.Class{ID: "1024"},
				Vendor:  &pcidb.Vendor{Name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbb"},
				Product: &pcidb.Product{Name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbbb"},
			}

			dev2 := &ghw.PCIDevice{
				Address: "0000:00:00.2",
				Class:   &pcidb.Class{ID: "1024"},
				Vendor:  &pcidb.Vendor{Name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbb"},
				Product: &pcidb.Product{Name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbbb"},
			}

			devInvalid := &ghw.PCIDevice{
				Class: &pcidb.Class{ID: "completely unparsable"},
			}

			devsToAdd := []*ghw.PCIDevice{dev1, dev2, devInvalid}

			err := p.AddTargetDevices(devsToAdd, 0x1024)
			dDevs := p.GetDiscoveredDevices()
			devs := p.GetDevices(config)
			It("shouldn't return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return only 1 device on GetDiscoveredDevices()", func() {
				Expect(dDevs).To(HaveLen(1))
			})
			It("should return only 1 device on GetDevices()", func() {
				Expect(devs).To(HaveLen(1))
			})
		})
	})
	Describe("getting Filtered devices", func() {
		Context("using selectors", func() {
			It("should correctly filter devices", func() {
				rf := factory.NewResourceFactory("fake", "fake", false)
				p := netdevice.NewNetDeviceProvider(rf)
				all := make([]types.PciDevice, 5)
				mocked := make([]mocks.PciNetDevice, 5)

				ve := []string{"8086", "8086", "1111", "2222", "3333"}
				de := []string{"abcd", "123a", "abcd", "2222", "1024"}
				md := []string{"igb_uio", "igb_uio", "igb_uio", "iavf", "vfio-pci"}
				pa := []string{"0000:03:02.0", "0000:03:02.1", "0000:03:02.2", "0000:03:02.3", "0000:03:02.4"}
				pf := []string{"eth0", "eth0", "eth1", "net0", "net0"}
				ro := []string{"0000:86:00.0", "0000:86:00.1", "0000:86:00.2", "0000:86:00.3", "0000:86:00.4"}
				lt := []string{"ether", "infiniband", "ether", "ether", "fake"}
				dd := []string{"E710 PPPoE and PPPoL2TPv2", "fake", "fake", "gtp", "profile"}
				rd := []bool{false, true, false, false, true}
				vd := []string{"vhost", "vhost", "", "", "virtio"}

				rdmaYes := &mocks.RdmaSpec{}
				rdmaYes.On("IsRdma").Return(true)
				rdmaNo := &mocks.RdmaSpec{}
				rdmaNo.On("IsRdma").Return(false)

				vdpaVhost := &mocks.VdpaDevice{}
				vdpaVhost.On("GetDriver").Return("vhost_vdpa").
					On("GetType").Return(types.VdpaVhostType)
				vdpaVirtio := &mocks.VdpaDevice{}
				vdpaVirtio.On("GetDriver").Return("virtio_vdpa").
					On("GetType").Return(types.VdpaVirtioType)

				for i, _ := range mocked {
					mocked[i].
						On("GetVendor").Return(ve[i]).
						On("GetDeviceCode").Return(de[i]).
						On("GetDriver").Return(md[i]).
						On("GetPciAddr").Return(pa[i]).
						On("GetPFName").Return(pf[i]).
						On("GetPfPciAddr").Return(ro[i]).
						On("GetLinkType").Return(lt[i]).
						On("GetDDPProfiles").Return(dd[i])

					if rd[i] {
						mocked[i].On("GetRdmaSpec").Return(rdmaYes)
					} else {
						mocked[i].On("GetRdmaSpec").Return(rdmaNo)
					}
					switch vd[i] {
					case "vhost":
						mocked[i].On("GetVdpaDevice").Return(vdpaVhost)
					case "virtio":
						mocked[i].On("GetVdpaDevice").Return(vdpaVirtio)
					default:
						mocked[i].On("GetVdpaDevice").Return(nil)
					}

					all[i] = &mocked[i]
				}

				testCases := []struct {
					name     string
					sel      *types.NetDeviceSelectors
					expected []types.PciDevice
				}{
					{"vendors", &types.NetDeviceSelectors{DeviceSelectors: types.DeviceSelectors{Vendors: []string{"8086"}}}, []types.PciDevice{all[0], all[1]}},
					{"devices", &types.NetDeviceSelectors{DeviceSelectors: types.DeviceSelectors{Devices: []string{"abcd"}}}, []types.PciDevice{all[0], all[2]}},
					{"drivers", &types.NetDeviceSelectors{DeviceSelectors: types.DeviceSelectors{Drivers: []string{"igb_uio"}}}, []types.PciDevice{all[0], all[1], all[2]}},
					{"pciAddresses", &types.NetDeviceSelectors{DeviceSelectors: types.DeviceSelectors{PciAddresses: []string{"0000:03:02.0", "0000:03:02.3"}}}, []types.PciDevice{all[0], all[3]}},
					{"pfNames", &types.NetDeviceSelectors{PfNames: []string{"net0", "eth1"}}, []types.PciDevice{all[2], all[3], all[4]}},
					{"rootDevices", &types.NetDeviceSelectors{RootDevices: []string{"0000:86:00.0", "0000:86:00.4"}}, []types.PciDevice{all[0], all[4]}},
					{"linkTypes", &types.NetDeviceSelectors{LinkTypes: []string{"infiniband"}}, []types.PciDevice{all[1]}},
					{"linkTypes multi", &types.NetDeviceSelectors{LinkTypes: []string{"infiniband", "fake"}}, []types.PciDevice{all[1], all[4]}},
					{"ddpProfiles", &types.NetDeviceSelectors{DDPProfiles: []string{"E710 PPPoE and PPPoL2TPv2"}}, []types.PciDevice{all[0]}},
					{"rdma", &types.NetDeviceSelectors{IsRdma: true}, []types.PciDevice{all[1], all[4]}},
					{"vdpa-vhost", &types.NetDeviceSelectors{VdpaType: "vhost"}, []types.PciDevice{all[0], all[1]}},
					{"vdpa-virtio", &types.NetDeviceSelectors{VdpaType: "virtio"}, []types.PciDevice{all[4]}},
				}

				for _, tc := range testCases {
					By(tc.name)
					config := &types.ResourceConfig{SelectorObj: tc.sel}
					actual, err := p.GetFilteredDevices(all, config)
					Expect(err).NotTo(HaveOccurred())
					Expect(actual).To(HaveLen(len(tc.expected)))
					Expect(actual).To(ConsistOf(tc.expected))
				}
			})
		})
	})
})
