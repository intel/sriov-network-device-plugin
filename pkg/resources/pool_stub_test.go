package resources_test

import (
	"reflect"

	"github.com/jaypipes/ghw"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/netdevice"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/resources"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PoolStub", func() {
	var (
		fs     *utils.FakeFilesystem
		f      types.ResourceFactory
		devs   []string
		d1, d2 types.PciNetDevice
		rp     types.ResourcePool
		rc     *types.ResourceConfig
	)
	BeforeEach(func() {
		fs = &utils.FakeFilesystem{
			Dirs: []string{
				"sys/bus/pci/devices/0000:00:00.1/net/enp2s0f0v0",
				"sys/bus/pci/devices/0000:01:00.0/net/enp2s0f0",
				"sys/bus/pci/devices/0000:00:00.2/net/enp2s0f1v0",
				"sys/kernel/iommu_groups/0",
				"sys/kernel/iommu_groups/1",
				"sys/bus/pci/drivers/vfio-pci",
			},
			Symlinks: map[string]string{
				"sys/bus/pci/devices/0000:00:00.1/iommu_group": "../../../../kernel/iommu_groups/0",
				"sys/bus/pci/devices/0000:00:00.2/iommu_group": "../../../../kernel/iommu_groups/1",
				"sys/bus/pci/devices/0000:00:00.1/driver":      "../../../../bus/pci/drivers/vfio-pci",
				"sys/bus/pci/devices/0000:00:00.2/driver":      "../../../../bus/pci/drivers/vfio-pci",
				"sys/bus/pci/devices/0000:00:00.1/physfn":      "../0000:01:00.0",
			},
		}
		f = factory.NewResourceFactory("fake", "fake", true)
		rc = &types.ResourceConfig{SelectorObj: types.NetDeviceSelectors{}}
		devs = []string{"0000:00:00.1", "0000:00:00.2"}
	})
	Describe("getting device specs", func() {
		Context("for valid devices", func() {
			It("should return valid DeviceSpec array", func() {
				defer fs.Use()()
				defer utils.UseFakeLinks()()

				d1, _ = netdevice.NewPciNetDevice(&ghw.PCIDevice{Address: "0000:00:00.1"}, f, rc)
				d2, _ = netdevice.NewPciNetDevice(&ghw.PCIDevice{Address: "0000:00:00.2"}, f, rc)
				rp = resources.NewResourcePool(rc,
					map[string]*pluginapi.Device{},
					map[string]types.PciDevice{
						"0000:00:00.1": d1,
						"0000:00:00.2": d2,
					},
				)
				specs := rp.GetDeviceSpecs(devs)

				expected := []*pluginapi.DeviceSpec{
					{ContainerPath: "/dev/vfio/vfio", HostPath: "/dev/vfio/vfio", Permissions: "mrw"},
					{ContainerPath: "/dev/vfio/0", HostPath: "/dev/vfio/0", Permissions: "mrw"},
					{ContainerPath: "/dev/vfio/1", HostPath: "/dev/vfio/1", Permissions: "mrw"},
				}
				Expect(specs).To(HaveLen(3))
				Expect(specs).To(ConsistOf(expected))
			})
		})
	})
	Describe("getting envs", func() {
		Context("for valid devices", func() {
			It("should return valid envs array", func() {
				defer fs.Use()()
				defer utils.UseFakeLinks()()

				d1, _ = netdevice.NewPciNetDevice(&ghw.PCIDevice{Address: "0000:00:00.1"}, f, rc)
				d2, _ = netdevice.NewPciNetDevice(&ghw.PCIDevice{Address: "0000:00:00.2"}, f, rc)
				rp = resources.NewResourcePool(rc,
					map[string]*pluginapi.Device{},
					map[string]types.PciDevice{
						"0000:00:00.1": d1,
						"0000:00:00.2": d2,
					},
				)
				envs := rp.GetEnvs(devs)

				expected := []string{"0000:00:00.1", "0000:00:00.2"}
				Expect(envs).To(HaveLen(2))
				Expect(envs).To(ConsistOf(expected))
			})
		})
	})
	Describe("getting mounts", func() {
		Context("for valid devices", func() {
			It("should return valid mounts array", func() {
				defer fs.Use()()
				defer utils.UseFakeLinks()()

				d1, _ = netdevice.NewPciNetDevice(&ghw.PCIDevice{Address: "0000:00:00.1"}, f, rc)
				d2, _ = netdevice.NewPciNetDevice(&ghw.PCIDevice{Address: "0000:00:00.2"}, f, rc)
				rp = resources.NewResourcePool(rc,
					map[string]*pluginapi.Device{},
					map[string]types.PciDevice{
						"0000:00:00.1": d1,
						"0000:00:00.2": d2,
					},
				)
				mounts := rp.GetMounts(devs)

				// in current implementation vfio and others return empty arrays
				expected := []*pluginapi.Mount{}
				Expect(mounts).To(HaveLen(0))
				Expect(mounts).To(ConsistOf(expected))
			})

		})
	})
	Describe("getting device pool", func() {
		Context("for valid devices", func() {
			It("should return valid device pool", func() {
				defer fs.Use()()
				defer utils.UseFakeLinks()()

				d1, _ = netdevice.NewPciNetDevice(&ghw.PCIDevice{Address: "0000:00:00.1"}, f, rc)
				d2, _ = netdevice.NewPciNetDevice(&ghw.PCIDevice{Address: "0000:00:00.2"}, f, rc)
				rp = resources.NewResourcePool(rc,
					map[string]*pluginapi.Device{},
					map[string]types.PciDevice{
						"0000:00:00.1": d1,
						"0000:00:00.2": d2,
					},
				)
				pool := rp.GetDevicePool()

				expected := map[string]types.PciDevice{
					"0000:00:00.1": d1,
					"0000:00:00.2": d2,
				}
				Expect(pool).To(HaveLen(2))
				Expect(reflect.DeepEqual(pool, expected)).To(Equal(true))
			})

		})
	})
})
