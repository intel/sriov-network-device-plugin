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

package factory_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/resources"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types/mocks"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func TestFactory(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Factory Suite")
}

var _ = Describe("Factory", func() {
	Describe("getting factory instance", func() {
		Context("always", func() {
			It("should return the same instance", func() {
				f0 := factory.NewResourceFactory("fake", "fake", true)
				Expect(f0).NotTo(BeNil())
				f1 := factory.NewResourceFactory("fake", "fake", true)
				Expect(f1).To(Equal(f0))
			})
		})
	})
	DescribeTable("getting allocator",
		func(policy string, expected reflect.Type) {
			f := factory.NewResourceFactory("fake", "fake", true)
			a := f.GetAllocator(policy)
			if expected != nil {
				Expect(reflect.TypeOf(a)).To(Equal(expected))
			} else {
				Expect(a).To(BeNil())
			}
		},
		Entry("packed", "packed", reflect.TypeOf(resources.NewPackedAllocator())),
		Entry("empty", "", nil),
		Entry("any other value", "random policy", nil),
	)
	DescribeTable("getting info provider",
		func(name string, expected reflect.Type) {
			f := factory.NewResourceFactory("fake", "fake", true)
			p := f.GetDefaultInfoProvider("fakePCIAddr", name)
			Expect(reflect.TypeOf(p)).To(Equal(expected))
		},
		Entry("vfio-pci", "vfio-pci", reflect.TypeOf(resources.NewVfioInfoProvider("fakePCIAddr"))),
		Entry("uio", "uio", reflect.TypeOf(resources.NewUioInfoProvider("fakePCIAddr"))),
		Entry("igb_uio", "igb_uio", reflect.TypeOf(resources.NewUioInfoProvider("fakePCIAddr"))),
		Entry("any other value", "netdevice", reflect.TypeOf(resources.NewGenericInfoProvider("fakePCIAddr"))),
	)
	DescribeTable("getting selector",
		func(selector string, shouldSucceed bool, expected reflect.Type) {
			f := factory.NewResourceFactory("fake", "fake", true)
			v := []string{"val1", "val2", "val3"}
			s, e := f.GetSelector(selector, v)

			if shouldSucceed {
				Expect(reflect.TypeOf(s)).To(Equal(expected))
				Expect(e).NotTo(HaveOccurred())
			} else {
				Expect(s).To(BeNil())
				Expect(e).To(HaveOccurred())
			}

			// if statement below because gomega refuses to do "nil == nil" assertions
			if expected != nil {
				Expect(reflect.TypeOf(s)).To(Equal(expected))
			} else {
				Expect(reflect.TypeOf(s)).To(BeNil())
			}
		},
		Entry("vendors", "vendors", true, reflect.TypeOf(resources.NewVendorSelector([]string{}))),
		Entry("devices", "devices", true, reflect.TypeOf(resources.NewDeviceSelector([]string{}))),
		Entry("drivers", "drivers", true, reflect.TypeOf(resources.NewDriverSelector([]string{}))),
		Entry("pciAddresses", "pciAddresses", true, reflect.TypeOf(resources.NewPciAddressSelector([]string{}))),
		Entry("pfNames", "pfNames", true, reflect.TypeOf(resources.NewPfNameSelector([]string{}))),
		Entry("rootDevices", "rootDevices", true, reflect.TypeOf(resources.NewRootDeviceSelector([]string{}))),
		Entry("linkTypes", "linkTypes", true, reflect.TypeOf(resources.NewLinkTypeSelector([]string{}))),
		Entry("ddpProfiles", "ddpProfiles", true, reflect.TypeOf(resources.NewDdpSelector([]string{}))),
		Entry("invalid", "fakeAndInvalid", false, reflect.TypeOf(nil)),
	)
	Describe("getting resource pool for netdevice", func() {
		Context("with all types of selectors used and matching devices found", func() {
			defer utils.UseFakeLinks()()
			var (
				rp   types.ResourcePool
				err  error
				devs []types.PciDevice
			)
			BeforeEach(func() {
				f := factory.NewResourceFactory("fake", "fake", true)

				devs = make([]types.PciDevice, 4)
				vendors := []string{"8086", "8086", "8086", "1234"}
				codes := []string{"1111", "1111", "1234", "4321"}
				drivers := []string{"vfio-pci", "i40evf", "igb_uio", "igb_uio"}
				pciAddr := []string{"0000:03:02.0", "0000:03:02.1", "0000:03:02.2", "0000:03:02.3"}
				pfNames := []string{"enp2s0f2", "ens0", "eth0", "net2"}
				rootDevices := []string{"0000:86:00.0", "0000:86:00.1", "0000:86:00.2", "0000:86:00.3"}
				linkTypes := []string{"ether", "infiniband", "other", "other2"}
				ddpProfiles := []string{"GTP", "PPPoE", "GTP", "PPPoE"}
				for i := range devs {
					d := &mocks.PciNetDevice{}
					d.On("GetVendor").Return(vendors[i]).
						On("GetDeviceCode").Return(codes[i]).
						On("GetDriver").Return(drivers[i]).
						On("GetPciAddr").Return(pciAddr[i]).
						On("GetPFName").Return(pfNames[i]).
						On("GetPfPciAddr").Return(rootDevices[i]).
						On("GetAPIDevice").Return(&pluginapi.Device{}).
						On("GetLinkType").Return(linkTypes[i]).
						On("GetDDPProfiles").Return(ddpProfiles[i])
					devs[i] = d
				}

				var selectors json.RawMessage
				err := selectors.UnmarshalJSON([]byte(`
					[
						{
							"vendors": ["8086"],
							"devices": ["1111"],
							"drivers": ["vfio-pci"],
							"pciAddresses": ["0000:03:02.0"],
							"pfNames": ["enp2s0f2"],
							"rootDevices": ["0000:86:00.0"],
							"linkTypes": ["ether"],
							"ddpProfiles": ["GTP"]
						}
					]`),
				)

				Expect(err).NotTo(HaveOccurred())

				c := types.ResourceConfig{
					ResourceName: "fake",
					Selectors:    &selectors,
					DeviceType:   types.NetDeviceType,
				}

				rp, err = f.GetResourcePool(&c, devs)

			})
			It("should return valid resource pool", func() {
				Expect(rp).NotTo(BeNil())
				Expect(rp.GetDevices()).To(HaveLen(4))
				Expect(rp.GetDevices()).To(HaveKey("0000:03:02.0"))
			})
			It("should not fail", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
	Describe("getting resource pool for accelerator", func() {
		Context("with all types of selectors used and matching devices found", func() {
			defer utils.UseFakeLinks()()
			var (
				rp   types.ResourcePool
				err  error
				devs []types.PciDevice
			)
			BeforeEach(func() {
				f := factory.NewResourceFactory("fake", "fake", true)

				devs = make([]types.PciDevice, 1)
				vendors := []string{"8086"}
				codes := []string{"1024"}
				drivers := []string{"uio_pci_generic"}
				pciAddr := []string{"0000:04:00.0"}
				for i := range devs {
					d := &mocks.AccelDevice{}
					d.On("GetVendor").Return(vendors[i]).
						On("GetDeviceCode").Return(codes[i]).
						On("GetDriver").Return(drivers[i]).
						On("GetPciAddr").Return(pciAddr[i]).
						On("GetAPIDevice").Return(&pluginapi.Device{})
					devs[i] = d
				}

				var selectors json.RawMessage
				err := selectors.UnmarshalJSON([]byte(`
					[
						{
							"vendors": ["8086"],
							"devices": ["1024"],
							"drivers": ["uio_pci_generic"],
						}
					]`),
				)

				Expect(err).NotTo(HaveOccurred())

				c := types.ResourceConfig{
					ResourceName: "fake",
					Selectors:    &selectors,
					DeviceType:   types.AcceleratorType,
				}

				rp, err = f.GetResourcePool(&c, devs)
			})
			It("should return valid resource pool", func() {
				Expect(rp).NotTo(BeNil())
				Expect(rp.GetDevices()).To(HaveLen(1))
				Expect(rp.GetDevices()).To(HaveKey("0000:04:00.0"))
			})
			It("should not fail", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
	DescribeTable("getting device provider",
		func(dt types.DeviceType, shouldSucceed bool) {
			f := factory.NewResourceFactory("fake", "fake", true)
			p := f.GetDeviceProvider(dt)
			if shouldSucceed {
				Expect(p).NotTo(BeNil())
			} else {
				Expect(p).To(BeNil())
			}
		},
		Entry("of a netdevice shouldn't return nil", types.NetDeviceType, true),
		Entry("of an accelerator shouldn't return nil", types.AcceleratorType, true),
		Entry("of unsupported device type should return nil", nil, false),
	)
	DescribeTable("getting device filter",
		func(dt types.DeviceType, sel string, expected interface{}, shouldSucceed bool) {
			// prepare json rawmessage selector
			s := json.RawMessage{}
			err := s.UnmarshalJSON([]byte(sel))
			Expect(err).NotTo(HaveOccurred())

			rc := &types.ResourceConfig{
				DeviceType: dt,
				Selectors:  &s,
			}

			f := factory.NewResourceFactory("fake", "fake", true)

			_, e := f.GetDeviceFilter(rc)
			if shouldSucceed {
				Expect(e).NotTo(HaveOccurred())
			} else {
				Expect(e).To(HaveOccurred())
			}
		},
		Entry("successful netdevice", types.NetDeviceType, `{"PfNames":["eth0"]}`, nil, true),
		Entry("failed netdevice", types.NetDeviceType, `invalid selectors!`, nil, false),
		Entry("successful accelerator", types.AcceleratorType, `{"Vendors": ["8086"]}`, nil, true),
		Entry("failed accelerator", types.AcceleratorType, `invalid selectors!`, nil, false),
		Entry("unsupported type", nil, ``, nil, false),
	)
	Describe("getting rdma spec", func() {
		Context("check c rdma spec", func() {
			f := factory.NewResourceFactory("fake", "fake", true)
			rs := f.GetRdmaSpec("0000:00:00.1")
			isRdma := rs.IsRdma()
			deviceSpec := rs.GetRdmaDeviceSpec()
			It("shoud return valid rdma spec", func() {
				Expect(isRdma).ToNot(BeTrue())
				Expect(deviceSpec).To(HaveLen(0))
			})
		})
	})
	Describe("getting resource server", func() {
		Context("when resource pool is nil", func() {
			f := factory.NewResourceFactory("fake", "fake", true)
			rs, e := f.GetResourceServer(nil)
			It("should fail", func() {
				Expect(e).To(HaveOccurred())
				Expect(rs).To(BeNil())
			})
		})
		Context("when resouce pool uses overriden prefix", func() {
			f := factory.NewResourceFactory("fake", "fake", true)
			rp := mocks.ResourcePool{}
			rp.On("GetResourcePrefix").Return("overriden").
				On("GetResourceName").Return("fake").
				On("GetAllocatePolicy").Return("")
			rs, e := f.GetResourceServer(&rp)
			It("should not fail", func() {
				Expect(e).NotTo(HaveOccurred())
				Expect(rs).NotTo(BeNil())
			})
		})
	})
})
