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

package utils

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	sysBusPci = "/sys/bus/pci/devices"
)

const (
	totalVfFile        = "sriov_totalvfs"
	configuredVfFile   = "sriov_numvfs"
	configuredMdevFile = "mdev_supported_types"
)

// GetVFconfigured returns number of VF configured for a PF
func GetVFconfigured(pf string) int {
	configuredVfPath := filepath.Join(sysBusPci, pf, configuredVfFile)
	vfs, err := ioutil.ReadFile(configuredVfPath)
	if err != nil {
		return 0
	}
	configuredVFs := bytes.TrimSpace(vfs)
	numConfiguredVFs, err := strconv.Atoi(string(configuredVFs))
	if err != nil {
		return 0
	}
	return numConfiguredVFs
}

// GetMDEVconfigured returns whether MDEV supported in this PF
func GetMDEVconfigured(pf string) bool {
	configuredMDEVPath := filepath.Join(sysBusPci, pf, configuredMdevFile)
	_, err := os.Lstat(configuredMDEVPath)
	if err != nil {
		return false
	}
	return true
}

// GetVFList returns a List containing PCI addr for all VF discovered in a given PF
func GetVFList(pf string) (vfList []string, err error) {
	vfList = make([]string, 0)
	pfDir := filepath.Join(sysBusPci, pf)
	_, err = os.Lstat(pfDir)
	if err != nil {
		err = fmt.Errorf("Error. Could not get PF directory information for device: %s, Err: %v", pf, err)
		return
	}

	vfDirs, err := filepath.Glob(filepath.Join(pfDir, "virtfn*"))

	if err != nil {
		err = fmt.Errorf("error reading VF directories %v", err)
		return
	}

	//Read all VF directory and get add VF PCI addr to the vfList
	for _, dir := range vfDirs {
		dirInfo, err := os.Lstat(dir)
		if err == nil && (dirInfo.Mode()&os.ModeSymlink != 0) {
			linkName, err := filepath.EvalSymlinks(dir)
			if err == nil {
				vfLink := filepath.Base(linkName)
				vfList = append(vfList, vfLink)
			}
		}
	}
	return
}

// GetMDEVList returns a List containing PCI addr for all MDEV discovered in a given PF
func GetMDEVList(pf string) (mdevList []string, err error) {
	mdevList = make([]string, 0)
	pfDir := filepath.Join(sysBusPci, pf)
	_, err = os.Lstat(pfDir)
	if err != nil {
		err = fmt.Errorf("Error. Could not get PF directory information for device: %s, Err: %v", pf, err)
		return
	}

	mdevDirs, err := filepath.Glob(filepath.Join(pfDir, "*-*-*-*-*"))

	if err != nil {
		err = fmt.Errorf("error reading MDEV directories %v", err)
		return
	}

	//Read all MDEV directory and get add MDEV PCI addr to the mdevList
	for _, dir := range mdevDirs {
		_, err := os.Lstat(dir)
		if err == nil {
			mdevUUID := filepath.Base(dir)
			mdevList = append(mdevList, pf+"/"+mdevUUID)

		}
	}
	return
}

// GetPciAddrFromVFID returns PCI address for VF ID
func GetPciAddrFromVFID(pf string, vf int) (pciAddr string, err error) {
	vfDir := fmt.Sprintf("%s/%s/virtfn%d", sysBusPci, pf, vf)
	dirInfo, err := os.Lstat(vfDir)
	if err != nil {
		err = fmt.Errorf("Error. Could not get directory information for device: %s, VF: %v. Err: %v", pf, vf, err)
		return "", err
	}

	if (dirInfo.Mode() & os.ModeSymlink) == 0 {
		err = fmt.Errorf("Error. No symbolic link between virtual function and PCI - Device: %s, VF: %v", pf, vf)
		return
	}

	pciInfo, err := os.Readlink(vfDir)
	if err != nil {
		err = fmt.Errorf("Error. Cannot read symbolic link between virtual function and PCI - Device: %s, VF: %v. Err: %v", pf, vf, err)
		return
	}

	pciAddr = pciInfo[len("../"):]
	return
}

// GetSriovVFcapacity returns SRIOV VF capacity
func GetSriovVFcapacity(pf string) int {
	totalVfFilePath := filepath.Join(sysBusPci, pf, totalVfFile)
	vfs, err := ioutil.ReadFile(totalVfFilePath)
	if err != nil {
		return 0
	}
	totalvfs := bytes.TrimSpace(vfs)
	numvfs, err := strconv.Atoi(string(totalvfs))
	if err != nil {
		return 0
	}
	return numvfs
}

// IsNetlinkStatusUp returns 'false' if 'operstate' is not "up" for a Linux network device.
// This function will only return 'false' if the 'operstate' file of the device is readable
// and holds value anything other than "up". Or else we assume link is up.
func IsNetlinkStatusUp(dev string) bool {

	if opsFiles, err := filepath.Glob(filepath.Join(sysBusPci, dev, "net", "*", "operstate")); err == nil {
		for _, f := range opsFiles {
			bytes, err := ioutil.ReadFile(f)
			if err != nil || strings.TrimSpace(string(bytes)) != "up" {
				return false
			}
		}
	}
	return true
}

// ValidPciAddr validates PciAddr given as string with host system
func ValidPciAddr(addr string) (string, error) {
	//Check system pci address

	// sysbus pci address regex
	var validLongID = regexp.MustCompile(`^0{4}:[0-9a-f]{2}:[0-9a-f]{2}.[0-7]{1}$`)
	var validShortID = regexp.MustCompile(`^[0-9a-f]{2}:[0-9a-f]{2}.[0-7]{1}$`)

	if validLongID.MatchString(addr) {
		return addr, deviceExist(addr)
	} else if validShortID.MatchString(addr) {
		addr = "0000:" + addr // make short form to sysfs represtation
		return addr, deviceExist(addr)
	}

	return "", fmt.Errorf("invalid pci address %s", addr)
}

func deviceExist(addr string) error {
	devPath := filepath.Join(sysBusPci, addr)
	_, err := os.Lstat(devPath)
	if err != nil {
		return fmt.Errorf("error: unable to read device directory %s", devPath)
	}
	return nil
}

// SriovConfigured returns true if sriov_numvfs reads > 0 else false
func SriovConfigured(addr string) bool {
	if GetVFconfigured(addr) > 0 {
		return true
	}
	return false
}

// MdevConfigured returns true if mdev_supported_types existed else false
func MdevConfigured(addr string) bool {
	if GetMDEVconfigured(addr) {
		return true
	}
	return false
}

// ValidResourceName returns true if it contains permitted characters otherwise false
func ValidResourceName(name string) bool {
	// name regex
	var validString = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	return validString.MatchString(name)
}

// GetVFIODeviceFile returns a vfio device files for vfio-pci bound PCI device's PCI address
func GetVFIODeviceFile(dev string) (devFile string, err error) {
	// Get iommu group for this device
	devPath := filepath.Join(sysBusPci, dev)
	_, err = os.Lstat(devPath)
	if err != nil {
		err = fmt.Errorf("GetVFIODeviceFile(): Could not get directory information for device: %s, Err: %v", dev, err)
		return
	}

	iommuDir := filepath.Join(devPath, "iommu_group")
	if err != nil {
		err = fmt.Errorf("GetVFIODeviceFile(): error reading iommuDir %v", err)
		return
	}

	dirInfo, err := os.Lstat(iommuDir)
	if err != nil {
		err = fmt.Errorf("GetVFIODeviceFile(): unable to find iommu_group %v", err)
		return
	}

	if dirInfo.Mode()&os.ModeSymlink == 0 {
		err = fmt.Errorf("GetVFIODeviceFile(): invalid symlink to iommu_group %v", err)
		return
	}

	linkName, err := filepath.EvalSymlinks(iommuDir)
	if err != nil {
		err = fmt.Errorf("GetVFIODeviceFile(): error reading symlink to iommu_group %v", err)
		return
	}

	devFile = filepath.Join("/dev/vfio", filepath.Base(linkName))

	return
}

// GetUIODeviceFile returns a vfio device files for vfio-pci bound PCI device's PCI address
func GetUIODeviceFile(dev string) (devFile string, err error) {

	vfDir := filepath.Join(sysBusPci, dev, "uio")

	_, err = os.Lstat(vfDir)
	if err != nil {
		return "", fmt.Errorf("GetUIODeviceFile(): could not get directory information for device: %s Err: %v", vfDir, err)
	}

	files, err := ioutil.ReadDir(vfDir)

	if err != nil {
		return
	}

	// uio directory should only contain one directory e.g uio1
	// assuption is there's a corresponding device file in /dev e.g. /dev/uio1
	devFile = filepath.Join("/dev", files[0].Name())

	return
}
