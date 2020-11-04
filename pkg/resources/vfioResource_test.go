package resources_test

import (
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/resources"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"

	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("VfioPool", func() {
	Describe("creating new VFIO resource", func() {
		var vfioPool types.DeviceInfoProvider
		BeforeEach(func() {
			vfioPool = resources.NewVfioResource()
		})
		It("should return valid vfioResource object", func() {
			Expect(vfioPool).NotTo(Equal(nil))
			// FIXME: Expect(reflect.TypeOf(vfioPool)).To(Equal(reflect.TypeOf(&vfioResource{})))
		})
	})
	DescribeTable("GetDeviceSpecs",
		func(fs *utils.FakeFilesystem, pciAddr string, expected []*pluginapi.DeviceSpec) {
			defer fs.Use()()

			pool := resources.NewVfioResource()
			specs := pool.GetDeviceSpecs(pciAddr)
			Expect(specs).To(ConsistOf(expected))
		},
		Entry("empty and returning default common vfio device file only",
			&utils.FakeFilesystem{},
			"",
			[]*pluginapi.DeviceSpec{
				{HostPath: "/dev/vfio/vfio", ContainerPath: "/dev/vfio/vfio", Permissions: "mrw"},
			},
		),
		Entry("PCI address passed, returns DeviceSpec with paths to its VFIO devices and additional default VFIO path",
			&utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:02:00.0", "sys/kernel/iommu_groups/0",
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:02:00.0/iommu_group": "../../../../kernel/iommu_groups/0",
				},
			},
			"0000:02:00.0",
			[]*pluginapi.DeviceSpec{
				{HostPath: "/dev/vfio/0", ContainerPath: "/dev/vfio/0", Permissions: "mrw"},
				{HostPath: "/dev/vfio/vfio", ContainerPath: "/dev/vfio/vfio", Permissions: "mrw"},
			},
		),
	)
	Describe("getting mounts", func() {
		It("should always return empty array of mounts", func() {
			pool := resources.NewVfioResource()
			result := pool.GetMounts("fakeAddr")
			Expect(result).To(BeEmpty())
		})
	})
	Describe("getting env val", func() {
		It("should always return passed PCI address", func() {
			in := "00:02.0"
			pool := resources.NewVfioResource()
			out := pool.GetEnvVal(in)
			Expect(out).To(Equal(in))
		})
	})
})
