package resources

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

// NewVendorSelector returns a DeviceSelector interface for vendor list
func NewVendorSelector(vendors []string) types.DeviceSelector {
	return &vendorSelector{vendors: vendors}
}

type vendorSelector struct {
	vendors []string
}

func (s *vendorSelector) Filter(inDevices []types.PciDevice) []types.PciDevice {
	filteredList := make([]types.PciDevice, 0)
	for _, dev := range inDevices {
		devVendor := dev.GetVendor()
		if contains(s.vendors, devVendor) {
			filteredList = append(filteredList, dev)
		}
	}
	return filteredList
}

// NewDeviceSelector returns a DeviceSelector interface for device list
func NewDeviceSelector(devices []string) types.DeviceSelector {
	return &deviceSelector{devices: devices}
}

type deviceSelector struct {
	devices []string
}

func (s *deviceSelector) Filter(inDevices []types.PciDevice) []types.PciDevice {
	filteredList := make([]types.PciDevice, 0)
	for _, dev := range inDevices {
		devCode := dev.GetDeviceCode()
		if contains(s.devices, devCode) {
			filteredList = append(filteredList, dev)
		}
	}
	return filteredList
}

// NewDriverSelector returns a DeviceSelector interface for driver list
func NewDriverSelector(drivers []string) types.DeviceSelector {
	return &driverSelector{drivers: drivers}
}

type driverSelector struct {
	drivers []string
}

func (s *driverSelector) Filter(inDevices []types.PciDevice) []types.PciDevice {
	filteredList := make([]types.PciDevice, 0)
	for _, dev := range inDevices {
		devDriver := dev.GetDriver()
		if contains(s.drivers, devDriver) {
			filteredList = append(filteredList, dev)
		}
	}
	return filteredList
}

// NewPciAddressSelector returns a NetDevSelector interface for netDev list
func NewPciAddressSelector(pciAddresses []string) types.DeviceSelector {
	return &pciAddressSelector{pciAddresses: pciAddresses}
}

type pciAddressSelector struct {
	pciAddresses []string
}

func (s *pciAddressSelector) Filter(inDevices []types.PciDevice) []types.PciDevice {
	filteredList := make([]types.PciDevice, 0)
	for _, dev := range inDevices {
		if contains(s.pciAddresses, dev.GetPciAddr()) {
			filteredList = append(filteredList, dev)
		}
	}
	return filteredList
}

// NewPfNameSelector returns a NetDevSelector interface for netDev list
func NewPfNameSelector(pfNames []string) types.DeviceSelector {
	return &pfNameSelector{pfNames: pfNames}
}

type pfNameSelector struct {
	pfNames []string
}

func (s *pfNameSelector) Filter(inDevices []types.PciDevice) []types.PciDevice {
	filteredList := make([]types.PciDevice, 0)
	for _, dev := range inDevices {
		pfName := dev.(types.PciNetDevice).GetPFName()
		if pfName == "" {
			// Exclude devices that doesn't have a PF name
			continue
		}
		selector := getItem(s.pfNames, pfName, "pfname")
		if selector != "" {
			if isSelected(dev, selector) {
				filteredList = append(filteredList, dev)
			}
		}
	}

	return filteredList
}

// NewRootDeviceSelector returns a NetDevSelector interface for netDev list
func NewRootDeviceSelector(rootDevices []string) types.DeviceSelector {
	return &rootDeviceSelector{rootDevices: rootDevices}
}

type rootDeviceSelector struct {
	rootDevices []string
}

func (s *rootDeviceSelector) Filter(inDevices []types.PciDevice) []types.PciDevice {
	filteredList := make([]types.PciDevice, 0)
	for _, dev := range inDevices {
		rootDevice := dev.(types.PciNetDevice).GetPfPciAddr()
		if rootDevice == "" {
			// Exclude devices that doesn't have a root PCI device
			continue
		}
		selector := getItem(s.rootDevices, rootDevice, "rootDevice")
		if selector != "" {
			if isSelected(dev, selector) {
				filteredList = append(filteredList, dev)
			}
		}
	}
	return filteredList
}

// NewLinkTypeSelector returns a interface for netDev list
func NewLinkTypeSelector(linkTypes []string) types.DeviceSelector {
	return &linkTypeSelector{linkTypes: linkTypes}
}

type linkTypeSelector struct {
	linkTypes []string
}

func (s *linkTypeSelector) Filter(inDevices []types.PciDevice) []types.PciDevice {
	filteredList := make([]types.PciDevice, 0)
	for _, dev := range inDevices {
		linkType := dev.(types.PciNetDevice).GetLinkType()
		if contains(s.linkTypes, linkType) {
			filteredList = append(filteredList, dev)
		}
	}
	return filteredList
}

func contains(hay []string, needle string) bool {
	for _, s := range hay {
		if s == needle {
			return true
		}
	}
	return false
}

func getItem(hay []string, needle string, what string) string {
	for _, item := range hay {
		// May be an exact match or a regex, try both.
		if strings.EqualFold(strings.Split(item, "#")[0], needle) {
			return item
		}
		re, err := regexp.Compile(strings.Split(item, "#")[0])
		if err != nil {
			glog.Infof("getItem(): error compiling %s, may however not be a regex: %v", what, err)
		} else {
			if re.Match([]byte(needle)) {
				return item
			}
		}
	}
	return ""
}

func isSelected(dev types.PciDevice, selector string) bool {
	if strings.Contains(selector, "#") {
		// Selector does contain VF index in next format:
		// <PFName>#<VFIndexStart>-<VFIndexEnd> or
		// <PFAddr>#<VFIndexStart>-<VFIndexEnd>
		// In this case both <VFIndexStart> and <VFIndexEnd>
		// are included in range, for example: "netpf0#3-5"
		// The VFs 3,4 and 5 of the PF 'netpf0' will be included
		// in selector pool
		fields := strings.Split(selector, "#")
		if len(fields) != 2 {
			fmt.Printf("Failed to parse %s PF (name|address) selector, probably incorrect separator character usage\n", selector)
			return false
		}
		entries := strings.Split(fields[1], ",")
		for i := 0; i < len(entries); i++ {
			if strings.Contains(entries[i], "-") {
				rng := strings.Split(entries[i], "-")
				if len(rng) != 2 {
					fmt.Printf("Failed to parse %s PF (name|address) selector, probably incorrect range character usage\n", selector)
					return false
				}
				rngSt, err := strconv.Atoi(rng[0])
				if err != nil {
					fmt.Printf("Failed to parse %s PF (name|address) selector, start range is incorrect\n", selector)
					return false
				}
				rngEnd, err := strconv.Atoi(rng[1])
				if err != nil {
					fmt.Printf("Failed to parse %s PF (name|address) selector, end range is incorrect\n", selector)
					return false
				}
				vfID := dev.GetVFID()
				if vfID >= rngSt && vfID <= rngEnd {
					return true
				}
			} else {
				vfid, err := strconv.Atoi(entries[i])
				if err != nil {
					fmt.Printf("Failed to parse %s PF (name|address) selector, index is incorrect\n", selector)
					return false
				}
				vfID := dev.GetVFID()
				if vfID == vfid {
					return true
				}
			}
		}
	} else {
		return true
	}
	return false
}
