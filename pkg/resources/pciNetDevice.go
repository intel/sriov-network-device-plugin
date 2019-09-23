package resources

import (
	"strconv"

	"github.com/golang/glog"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"github.com/intel/sriov-network-device-plugin/pkg/utils"
	"github.com/jaypipes/ghw"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

type pciNetDevice struct {
	pciDevice   *ghw.PCIDevice
	ifName      string
	pfName      string
	pfAddr      string
	driver      string
	vendor      string
	product     string
	vfID        int
	linkSpeed   string
	env         string
	numa        string
	apiDevice   *pluginapi.Device
	deviceSpecs []*pluginapi.DeviceSpec
	mounts      []*pluginapi.Mount
	rdmaSpec    types.RdmaSpec
	linkType    string
}

// Convert NUMA node number to string.
// A node of -1 represents "unknown" and is converted to the empty string.
func nodeToStr(nodeNum int) string {
	if nodeNum >= 0 {
		return strconv.Itoa(nodeNum)
	}
	return ""
}

// NewPciNetDevice returns an instance of PciNetDevice interface
func NewPciNetDevice(pciDevice *ghw.PCIDevice, rFactory types.ResourceFactory) (types.PciNetDevice, error) {
	// populate all fields in pciNetDevice here

	// 			1. get PF details, add PF info in its pciNetDevice struct
	// 			2. Get driver info
	var ifName string
	pciAddr := pciDevice.Address
	driverName, err := utils.GetDriverName(pciAddr)
	if err != nil {
		return nil, err
	}
	netDevs, _ := utils.GetNetNames(pciAddr)
	if len(netDevs) == 0 {
		ifName = ""
	} else {
		ifName = netDevs[0]
	}
	pfName, err := utils.GetPfName(pciAddr)
	if err != nil {
		glog.Warningf("unable to get PF name %q", err.Error())
	}
	vfID, err := utils.GetVFID(pciAddr)
	if err != nil {
		return nil, err
	}

	// 			3. Get Device file info (e.g., uio, vfio specific)
	// Get DeviceInfoProvider using device driver
	infoProvider := rFactory.GetInfoProvider(driverName)
	dSpecs := infoProvider.GetDeviceSpecs(pciAddr)
	mnt := infoProvider.GetMounts(pciAddr)
	env := infoProvider.GetEnvVal(pciAddr)
	rdmaSpec := rFactory.GetRdmaSpec(pciDevice.Address)
	nodeNum := utils.GetDevNode(pciAddr)
	apiDevice := &pluginapi.Device{
		ID:     pciAddr,
		Health: pluginapi.Healthy,
	}
	if nodeNum >= 0 {
		numaInfo := &pluginapi.NUMANode{
			ID: int64(nodeNum),
		}
		apiDevice.Topology = &pluginapi.TopologyInfo{
			Nodes: []*pluginapi.NUMANode{numaInfo},
		}
	}

	linkType := ""
	if len(ifName) > 0 {
		la, err := utils.GetLinkAttrs(ifName)
		if err != nil {
			return nil, err
		}
		linkType = la.EncapType
	}

	// 			4. Create pciNetDevice object with all relevent info
	return &pciNetDevice{
		pciDevice:   pciDevice,
		ifName:      ifName,
		pfName:      pfName,
		driver:      driverName,
		vfID:        vfID,
		linkSpeed:   "", // TO-DO: Get this using utils pkg
		apiDevice:   apiDevice,
		deviceSpecs: dSpecs,
		mounts:      mnt,
		env:         env,
		numa:        nodeToStr(nodeNum),
		rdmaSpec:    rdmaSpec,
		linkType:    linkType,
	}, nil
}

func (nd *pciNetDevice) GetPFName() string {
	return nd.pfName
}

func (nd *pciNetDevice) GetNetName() string {
	return nd.ifName
}

func (nd *pciNetDevice) GetPfPciAddr() string {
	return nd.pfAddr
}

func (nd *pciNetDevice) GetVendor() string {
	return nd.pciDevice.Vendor.ID
}

func (nd *pciNetDevice) GetDeviceCode() string {
	return nd.pciDevice.Product.ID
}

func (nd *pciNetDevice) GetPciAddr() string {
	return nd.pciDevice.Address
}

func (nd *pciNetDevice) GetDriver() string {
	return nd.driver
}

func (nd *pciNetDevice) IsSriovPF() bool {
	return false
}

func (nd *pciNetDevice) GetLinkSpeed() string {
	return nd.linkSpeed
}

func (nd *pciNetDevice) GetSubClass() string {
	return nd.pciDevice.Subclass.ID
}

func (nd *pciNetDevice) GetDeviceSpecs() []*pluginapi.DeviceSpec {
	return nd.deviceSpecs
}

func (nd *pciNetDevice) GetEnvVal() string {
	return nd.env
}

func (nd *pciNetDevice) GetMounts() []*pluginapi.Mount {
	return nd.mounts
}

func (nd *pciNetDevice) GetAPIDevice() *pluginapi.Device {
	return nd.apiDevice
}

func (nd *pciNetDevice) GetRdmaSpec() types.RdmaSpec {
	return nd.rdmaSpec
}

func getPFInfos(pciAddr string) (pfAddr, pfName string, err error) {
	return "", "", nil
}

func (nd *pciNetDevice) GetLinkType() string {
	return nd.linkType
}

func (nd *pciNetDevice) GetVFID() int {
	return nd.vfID
}
