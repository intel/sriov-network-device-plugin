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

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/netdevice"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types/mocks"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

func TestSriovdp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sriovdp Suite")
}

func assertShouldFail(err error, shouldFail bool) {
	if shouldFail {
		Expect(err).To(HaveOccurred())
	} else {
		Expect(err).NotTo(HaveOccurred())
	}
}

var _ = Describe("Resource manager", func() {
	var (
		cp *cliParams
		rm *resourceManager
	)
	Describe("reading config", func() {
		BeforeEach(func() {
			cp = &cliParams{
				configFile:     "/tmp/sriovdp/test_config",
				resourcePrefix: "test_",
			}
			rm = newResourceManager(cp)
		})
		Context("when there's an error reading file", func() {
			BeforeEach(func() {
				os.RemoveAll("/tmp/sriovdp")
			})
			It("should fail", func() {
				err := rm.readConfig()
				Expect(err).To(HaveOccurred())
			})
		})
		Context("when there's an error unmarshalling config", func() {
			BeforeEach(func() {
				err := os.MkdirAll("/tmp/sriovdp", 0755)
				if err != nil {
					panic(err)
				}
				ioutil.WriteFile("/tmp/sriovdp/test_config", []byte("junk"), 0644)
			})
			AfterEach(func() {
				err := os.RemoveAll("/tmp/sriovdp")
				if err != nil {
					panic(err)
				}
				rm = nil
				cp = nil
			})
			It("should fail", func() {
				err := rm.readConfig()
				Expect(err).To(HaveOccurred())
			})
		})
		Context("when config reading is successful", func() {
			var err error
			BeforeEach(func() {
				// add err handling
				testErr := os.MkdirAll("/tmp/sriovdp", 0755)
				if testErr != nil {
					panic(testErr)
				}
				testErr = ioutil.WriteFile("/tmp/sriovdp/test_config", []byte(`{
						"resourceList": [{
								"resourceName": "intel_sriov_netdevice",
								"isRdma": false,
								"selectors": {
									"vendors": ["8086"],
									"devices": ["154c", "10ed"],
									"drivers": ["i40evf", "ixgbevf"]
								}
							},
							{
								"resourceName": "intel_sriov_dpdk",
								"selectors": {
									"vendors": ["8086"],
									"devices": ["154c", "10ed"],
									"drivers": ["vfio-pci"]
								}
							}
						],
						"featureGates": {
							"testFeatureOne": true,
							"testFeatureTwo": false
						}
					}`), 0644)
				if testErr != nil {
					panic(testErr)
				}
				err = rm.readConfig()
			})
			AfterEach(func() {
				testErr := os.RemoveAll("/tmp/sriovdp")
				if testErr != nil {
					panic(testErr)
				}
				rm = nil
				cp = nil
			})
			It("shouldn't fail", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("should load resources list", func() {
				Expect(len(rm.configList)).To(Equal(2))
			})
		})
	})
	Describe("validating configuration", func() {
		var fs *utils.FakeFilesystem
		BeforeEach(func() {
			cp = &cliParams{
				configFile:     "/tmp/sriovdp/test_config",
				resourcePrefix: "test_",
			}
			rm = newResourceManager(cp)
			fs = &utils.FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:02:00.0", "sys/bus/pci/devices/0000:03:00.0"},
				Files: map[string][]byte{
					"sys/bus/pci/devices/0000:02:00.0/sriov_numvfs": []byte("32"),
					"sys/bus/pci/devices/0000:03:00.0/sriov_numvfs": []byte("0"),
				},
			}
		})
		AfterEach(func() {
			err := os.RemoveAll("/tmp/sriovdp")
			if err != nil {
				panic(err)
			}
			rm = nil
			cp = nil
		})
		Context("when resource name is invalid", func() {
			BeforeEach(func() {
				err := os.MkdirAll("/tmp/sriovdp", 0755)
				if err != nil {
					panic(err)
				}
				err = ioutil.WriteFile("/tmp/sriovdp/test_config", []byte(`{
					"resourceList":	[{
						"resourceName": "invalid-name",
						"isRdma": false,
						"selectors": {
							"vendors": ["8086"],
							"devices": ["154c", "10ed"],
							"drivers": ["i40evf", "ixgbevf"]
						}
					}]
				}`), 0644)
				if err != nil {
					panic(err)
				}
				rm.readConfig()
			})
			It("should return false", func() {
				defer fs.Use()()
				Expect(rm.validConfigs()).To(Equal(false))
			})
		})
		Context("when resource name is duplicated", func() {
			BeforeEach(func() {
				err := os.MkdirAll("/tmp/sriovdp", 0755)
				if err != nil {
					panic(err)
				}
				err = ioutil.WriteFile("/tmp/sriovdp/test_config", []byte(`{
					"resourceList":	[{
						"resourceName": "duplicate",
						"isRdma": true,
						"selectors": {
							"vendors": ["8086"],
							"devices": ["154c", "10ed"],
							"drivers": ["i40evf", "ixgbevf"]
						}
					},{
						"resourceName": "duplicate",
						"selectors": {
							"vendors": ["8086"],
							"devices": ["154c", "10ed"],
							"drivers": ["vfio-pci"]
						}
					}]
				}`), 0644)
				if err != nil {
					panic(err)
				}
				rm.readConfig()
			})
			It("should return false", func() {
				defer fs.Use()()
				Expect(rm.validConfigs()).To(Equal(false))
			})
		})
		Describe("managing resources servers", func() {
			Describe("initializing servers", func() {
				Context("when initializing server fails", func() {
					var (
						mockedServer *mocks.ResourceServer
					)
					BeforeEach(func() {
						mockedServer = &mocks.ResourceServer{}
						mockedServer.On("Init").Return(fmt.Errorf("fake error"))

						mockedRf := &mocks.ResourceFactory{}
						mockedRf.On("GetResourceServer", &mocks.ResourcePool{}).
							Return(mockedServer, nil)
					})
					It("should not return an error", func() {
						Expect(rm.initServers()).NotTo(HaveOccurred())
					})
					It("should finish with empty list of servers", func() {
						Expect(len(rm.resourceServers)).To(Equal(0))
					})
				})
				Context("when server is properly initialized", func() {
					mockedServer := &mocks.ResourceServer{}
					mockedServer.On("Init").Return(nil)

					dev := &mocks.PciDevice{}
					devs := []types.PciDevice{dev}

					rc := &types.ResourceConfig{
						ResourceName: "fake",
						DeviceType:   types.NetDeviceType,
						SelectorObj:  types.NetDeviceSelectors{},
					}
					dp := &mocks.DeviceProvider{}
					dp.On("GetFilteredDevices", devs, rc).Return(devs, nil)

					rp := &mocks.ResourcePool{}

					mockedRf := &mocks.ResourceFactory{}
					mockedRf.On("GetResourcePool", rc, devs).Return(rp, nil).
						On("GetResourceServer", rp).Return(mockedServer, nil)

					dp.On("GetDevices", rc).Return(devs)
					rm := &resourceManager{
						rFactory:   mockedRf,
						configList: []*types.ResourceConfig{rc},
						deviceProviders: map[types.DeviceType]types.DeviceProvider{
							types.NetDeviceType: dp,
						},
						resourceServers: []types.ResourceServer{},
					}

					err := rm.initServers()

					It("should not return an error", func() {
						Expect(err).NotTo(HaveOccurred())
					})
					It("should call Init() method on the server without getting errors", func() {
						Expect(mockedServer.MethodCalled("Init")).To(Equal(mock.Arguments{nil}))
					})
					It("should end up with one element in the list of servers", func() {
						Expect(rm.resourceServers).To(HaveLen(1))
					})
				})
			})
		})
	})
	DescribeTable("discovering devices",
		func(fs *utils.FakeFilesystem) {
			defer fs.Use()()
			os.Setenv("GHW_CHROOT", fs.RootDir)
			defer os.Unsetenv("GHW_CHROOT")

			rf := factory.NewResourceFactory("fake", "fake", true)

			rm := &resourceManager{
				rFactory: rf,
				configList: []*types.ResourceConfig{
					&types.ResourceConfig{
						ResourceName: "fake",
						DeviceType:   types.NetDeviceType,
					},
				},
				deviceProviders: map[types.DeviceType]types.DeviceProvider{
					types.NetDeviceType: netdevice.NewNetDeviceProvider(rf),
				},
				resourceServers: []types.ResourceServer{},
			}

			err := rm.discoverHostDevices()
			Expect(err).NotTo(HaveOccurred())
		},
		Entry("no devices",
			&utils.FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices"},
			},
		),
		Entry("unparsable modalias",
			&utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.0",
					"sys/bus/pci/drivers/i40e",
				},
				Files: map[string][]byte{
					"sys/bus/pci/devices/0000:00:00.0/modalias":       []byte("pci:junk"),
					"sys/bus/pci/devices/0000:00:00.0/sriov_totalvfs": []byte("0"),
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:00:00.0/driver": "../../../../bus/pci/drivers/i40e",
				},
			},
		),
		Entry("PF device with no VFs configured",
			&utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.0",
					"sys/bus/pci/drivers/i40e",
				},
				Files: map[string][]byte{
					"sys/bus/pci/devices/0000:00:00.0/modalias": []byte(
						"pci:v00008086d00001572sv00008086sd00000004bc02sc00i00",
					),
					"sys/bus/pci/devices/0000:00:00.0/sriov_totalvfs": []byte("0"),
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:00:00.0/driver": "../../../../bus/pci/drivers/i40e",
				},
			},
		),
		Entry("PF device with no VFs configured and not bound to any driver",
			&utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.0",
				},
				Files: map[string][]byte{
					"sys/bus/pci/devices/0000:00:00.0/modalias": []byte(
						"pci:v00008086d00001572sv00008086sd00000004bc02sc00i00",
					),
					"sys/bus/pci/devices/0000:00:00.0/sriov_totalvfs": []byte("0"),
				},
			},
		),
		Entry("PF device with VF configured",
			&utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.0",
					"sys/bus/pci/devices/0000:00:00.1",
					"sys/bus/pci/drivers/i40e",
					"sys/bus/pci/drivers/i40evf",
				},
				Files: map[string][]byte{
					"sys/bus/pci/devices/0000:00:00.0/modalias": []byte(
						"pci:v00008086d00001572sv00008086sd00000004bc02sc00i00",
					),
					"sys/bus/pci/devices/0000:00:00.1/modalias": []byte(
						"pci:v00008086d0000154Csv00008086sd00000000bc02sc00i00",
					),
					"sys/bus/pci/devices/0000:00:00.0/sriov_totalvfs": []byte("1"),
					"sys/bus/pci/devices/0000:00:00.0/sriov_numvfs":   []byte("1"),
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:00:00.0/driver":  "../../../../bus/pci/drivers/i40e",
					"sys/bus/pci/devices/0000:00:00.1/driver":  "../../../../bus/pci/drivers/i40evf",
					"sys/bus/pci/devices/0000:00:00.0/virtfn0": "../0000:00:00.1",
				},
			},
		),
		Entry("VF device without PF and bound to dpdk driver",
			&utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:03:02.0",
					"sys/bus/pci/devices/0000:03:02.1",
				},
				Files: map[string][]byte{
					"sys/bus/pci/devices/0000:03:02.0/modalias": []byte(
						"pci:v00008086d0000154Csv00008086sd00000000bc02sc00i00",
					),
					"sys/bus/pci/devices/0000:03:02.1/modalias": []byte(
						"pci:v00008086d0000154Csv00008086sd00000000bc02sc00i00",
					),
					"sys/bus/pci/devices/0000:03:02.0/max_vfs": []byte("0"),
					"sys/bus/pci/devices/0000:03:02.1/max_vfs": []byte("0"),
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:03:02.0/driver": "../../../../bus/pci/drivers/igb_uio",
					"sys/bus/pci/devices/0000:03:02.1/driver": "../../../../bus/pci/drivers/igb_uio",
				},
			},
		),
	)
	Describe("starting all server", func() {
		Context("when resource servers are starting fine", func() {
			rs := &mocks.ResourceServer{}
			rs.On("Start").Return(nil).On("Watch").Return()

			rm := resourceManager{
				resourceServers: []types.ResourceServer{rs, rs, rs},
				pluginWatchMode: false,
			}

			err := rm.startAllServers()
			It("shouldn't fail", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("when resource server start fails", func() {
			rs := &mocks.ResourceServer{}
			rs.On("Start").Return(fmt.Errorf("failed"))

			rm := resourceManager{
				resourceServers: []types.ResourceServer{rs},
				pluginWatchMode: false,
			}

			err := rm.startAllServers()
			It("should fail", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})
	Describe("stopping all server", func() {
		Context("when resource servers are stopping fine", func() {
			rs := &mocks.ResourceServer{}
			rs.On("Stop").Return(nil)

			rm := resourceManager{
				resourceServers: []types.ResourceServer{rs, rs, rs},
			}

			err := rm.stopAllServers()
			It("shouldn't fail", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("when resource server stop fails", func() {
			rs := &mocks.ResourceServer{}
			rs.On("Stop").Return(fmt.Errorf("failed"))

			rm := resourceManager{
				resourceServers: []types.ResourceServer{rs},
			}

			err := rm.stopAllServers()
			It("should fail", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
