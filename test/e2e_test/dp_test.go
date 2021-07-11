package e2e_test

import (
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/test/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SR-IOV device plugin testing", func() {
	var resourceName string

	dpSelectors := map[string]string{
		"name": "sriov-device-plugin",
		"app":  "sriovdp",
	}

	Context("PCI device discovery", func() {
		AfterEach(func() {
			err := util.DeleteCm(cs, testCmName, testNs)
			Expect(err).NotTo(BeNil())
		})

		BeforeEach(func() {
			_ = util.DeleteCm(cs, testCmName, testNs)
		})

		// =================================================================
		// PF tests
		// =================================================================

		It("Discover PF - vendor, device, driver - netdev", func() {
			resourceName = "test_pf_netdev"
			cmData := make(map[string]string)
			cmData["config.json"] = `
			{
				"resourceList": [{
					"resourceName": "` + resourceName + `",
					"selectors": {
						"vendors": ["8086"],
						"devices": ["1572"],
						"drivers": ["i40e"]
					}
				}]
			}`

			isCmDeployed, _ := util.DeployCm(cs, testCmName, testNs, &cmData, timeout)
			Expect(isCmDeployed).To(Equal(true))

			isUpdated, err := util.WaitForDeviceListUpdate(cs, testNs, dpSelectors, timeout, interval)
			Expect(err).To(BeNil())
			Expect(isUpdated).To(BeTrue())

			quantity, err := util.GetNodeAllocatableResourceQuantinty(cs, *testNodeName, resourceName)
			Expect(err).To(BeNil())
			Expect(quantity).To(Equal(*numOfPfNetdev))
		})

		It("Discover PF - vendor, device, driver - vfio", func() {
			resourceName = "test_pf_vfio"
			cmData := make(map[string]string)
			cmData["config.json"] = `
			{
				"resourceList": [{
					"resourceName": "` + resourceName + `",
					"selectors": {
						"vendors": ["8086"],
						"devices": ["1572"],
						"drivers": ["vfio-pci"]
					}
				}]
			}`

			isCmDeployed, _ := util.DeployCm(cs, testCmName, testNs, &cmData, timeout)
			Expect(isCmDeployed).To(Equal(true))

			isUpdated, err := util.WaitForDeviceListUpdate(cs, testNs, dpSelectors, timeout, interval)
			Expect(err).To(BeNil())
			Expect(isUpdated).To(BeTrue())

			quantity, err := util.GetNodeAllocatableResourceQuantinty(cs, *testNodeName, resourceName)
			Expect(err).To(BeNil())
			Expect(quantity).To(Equal(*numOfPfVfio))
		})

		It("Discover PF - vendor, device", func() {
			resourceName = "test_pf_all"
			cmData := make(map[string]string)
			cmData["config.json"] = `
			{
				"resourceList": [{
					"resourceName": "` + resourceName + `",
					"selectors": {
						"vendors": ["8086"],
						"devices": ["1572"]
					}
				}]
			}`

			isCmDeployed, _ := util.DeployCm(cs, testCmName, testNs, &cmData, timeout)
			Expect(isCmDeployed).To(Equal(true))

			isUpdated, err := util.WaitForDeviceListUpdate(cs, testNs, dpSelectors, timeout, interval)
			Expect(err).To(BeNil())
			Expect(isUpdated).To(BeTrue())

			quantity, err := util.GetNodeAllocatableResourceQuantinty(cs, *testNodeName, resourceName)
			Expect(err).To(BeNil())
			Expect(quantity).To(Equal(*numOfPfNetdev + *numOfPfVfio))
		})

		// =================================================================
		// VF tests
		// =================================================================

		It("Discover VF - vendor, device, driver - netdev", func() {
			resourceName = "test_vf_netdev"
			cmData := make(map[string]string)
			cmData["config.json"] = `
			{
				"resourceList": [{
					"resourceName": "` + resourceName + `",
					"selectors": {
						"vendors": ["8086"],
						"devices": ["154c"],
						"drivers": ["iavf"]
					}
				}]
			}`

			isCmDeployed, _ := util.DeployCm(cs, testCmName, testNs, &cmData, timeout)
			Expect(isCmDeployed).To(Equal(true))

			isUpdated, err := util.WaitForDeviceListUpdate(cs, testNs, dpSelectors, timeout, interval)
			Expect(err).To(BeNil())
			Expect(isUpdated).To(BeTrue())

			quantity, err := util.GetNodeAllocatableResourceQuantinty(cs, *testNodeName, resourceName)
			Expect(err).To(BeNil())
			Expect(quantity).To(Equal(*numOfVfNetdev))
		})

		It("Discover VF - vendor, device, driver - vfio", func() {
			resourceName = "test_vf_vfio"
			cmData := make(map[string]string)
			cmData["config.json"] = `
			{
				"resourceList": [{
					"resourceName": "` + resourceName + `",
					"selectors": {
						"vendors": ["8086"],
						"devices": ["154c"],
						"drivers": ["vfio-pci"]
					}
				}]
			}`

			isCmDeployed, _ := util.DeployCm(cs, testCmName, testNs, &cmData, timeout)
			Expect(isCmDeployed).To(Equal(true))

			isUpdated, err := util.WaitForDeviceListUpdate(cs, testNs, dpSelectors, timeout, interval)
			Expect(err).To(BeNil())
			Expect(isUpdated).To(BeTrue())

			quantity, err := util.GetNodeAllocatableResourceQuantinty(cs, *testNodeName, resourceName)
			Expect(err).To(BeNil())
			Expect(quantity).To(Equal(*numOfVfVfio))
		})

		It("Discover VF - vendor, device", func() {
			resourceName = "test_vf_all"
			cmData := make(map[string]string)
			cmData["config.json"] = `
			{
				"resourceList": [{
					"resourceName": "` + resourceName + `",
					"selectors": {
						"vendors": ["8086"],
						"devices": ["154c"]
					}
				}]
			}`

			isCmDeployed, _ := util.DeployCm(cs, testCmName, testNs, &cmData, timeout)
			Expect(isCmDeployed).To(Equal(true))

			isUpdated, err := util.WaitForDeviceListUpdate(cs, testNs, dpSelectors, timeout, interval)
			Expect(err).To(BeNil())
			Expect(isUpdated).To(BeTrue())

			quantity, err := util.GetNodeAllocatableResourceQuantinty(cs, *testNodeName, resourceName)
			Expect(err).To(BeNil())
			Expect(quantity).To(Equal(*numOfVfNetdev + *numOfVfVfio))
		})

		It("Discover VF - vendor, device, driver, ", func() {
			resourceName = "test_vf_pf_vfio"
			cmData := make(map[string]string)
			cmData["config.json"] = `
			{
				"resourceList": [{
					"resourceName": "` + resourceName + `",
					"selectors": {
						"vendors": ["8086"],
						"devices": ["154c"],
						"drivers": ["vfio-pci"],
						"pfNames": ["` + *pfNameForTest + `"]
					}
				}]
			}`

			isCmDeployed, _ := util.DeployCm(cs, testCmName, testNs, &cmData, timeout)
			Expect(isCmDeployed).To(Equal(true))

			isUpdated, err := util.WaitForDeviceListUpdate(cs, testNs, dpSelectors, timeout, interval)
			Expect(err).To(BeNil())
			Expect(isUpdated).To(BeTrue())

			quantity, err := util.GetNodeAllocatableResourceQuantinty(cs, *testNodeName, resourceName)
			Expect(err).To(BeNil())
			Expect(quantity).To(Equal(*numOfVfioVfForSelectedPf))
		})

		// =================================================================
		// VF + PF test
		// =================================================================

		It("Discover PF and VF - vendor, device, driver", func() {
			resourceName = "test_pf_vf_all"
			cmData := make(map[string]string)
			cmData["config.json"] = `
			{
				"resourceList": [{
					"resourceName": "` + resourceName + `",
					"selectors": {
						"vendors": ["8086"],
						"devices": ["1572", "154c"],
						"drivers": ["i40e", "iavf", "vfio-pci"]
					}
				}]
			}`

			isCmDeployed, _ := util.DeployCm(cs, testCmName, testNs, &cmData, timeout)
			Expect(isCmDeployed).To(Equal(true))

			isUpdated, err := util.WaitForDeviceListUpdate(cs, testNs, dpSelectors, timeout, interval)
			Expect(err).To(BeNil())
			Expect(isUpdated).To(BeTrue())

			quantity, err := util.GetNodeAllocatableResourceQuantinty(cs, *testNodeName, resourceName)
			Expect(err).To(BeNil())
			Expect(quantity).To(Equal(*numOfPfNetdev + *numOfPfVfio + *numOfVfNetdev + *numOfVfVfio))
		})
	})
})
