# Using vDPA devices in Kubernetes
## Introduction to vDPA
vDPA (Virtio DataPath Acceleration) is a technology that enables the acceleration of virtIO devices while allowing the implementations of such devices (e.g: NIC vendors) to use their own control plane.
The consumers of the virtIO devices (VMs or containers) interact with the devices using the standard virtIO datapath and virtio-compatible control paths (virtIO, vhost). While the data-plane is mapped directly to the accelerator device, the contol-plane gets translated the vDPA kernel framework.

The vDPA kernel framework is composed of a vdpa bus (/sys/bus/vdpa), vdpa devices (/sys/bus/vdpa/devices) and vdpa drivers (/sys/bus/vdpa/drivers). Currently, two vdpa drivers are implemented:
*  virtio_vdpa: Exposes the device as a virtio-net netdev
*  vhost_vdpa: Exposes the device as a vhost-vdpa device. This device uses an extension of the vhost-net protocol to allow userspace applications access the rings directly

For more information about the vDPA framework, read the article on [LWN.net](https://lwn.net/Articles/816063/) or the blog series written by one of the main authors ([part 1](https://www.redhat.com/en/blog/vdpa-kernel-framework-part-1-vdpa-bus-abstracting-hardware), [part 2](https://www.redhat.com/en/blog/vdpa-kernel-framework-part-2-vdpa-bus-drivers-kernel-subsystem-interactions), [part3](https://www.redhat.com/en/blog/vdpa-kernel-framework-part-3-usage-vms-and-containers))

## vDPA Management
Currently, the management of vDPA devices is performed using the sysfs interface exposed by the vDPA Framework. However, in order to decouple the management of vdpa devices from the SR-IOV Device Plugin functionality, this management is low-level management is done in an external library called [go-vdpa](https://github.com/redhat-virtio-net/govdpa).

At the time of this writing (Jan 2021), there is work being done to provide a [unified management tool for vDPA devices] (https://lists.linuxfoundation.org/pipermail/virtualization/2020-November/050623.html). This tool will provide many *additional* features such as support for SubFunctions and 1:N mapping between VFs and vDPA devices.

In the context of the SR-IOV Device Plugin and the SR-IOV CNI, the current plan is to support only 1:1 mappings between SR-IOV VFs and vDPA devices. The adoption of the unified management interface might be considered while keeping this limitation.

## Tested NICs:
* Mellanox ConnectX®-6 DX *

\* NVIDIA Mellanox official support for vDPA devices [is limited to SwitchDev mode](https://docs.mellanox.com/pages/viewpage.action?pageId=39285091#OVSOffloadUsingASAP%C2%B2Direct-hwvdpaVirtIOAccelerationthroughHardwarevDPA), which is out of the scope of the SR-IOV Network Device Plugin

## Tested Kernel versions:
* 5.10.0

## vDPA device creation
Currently, each NIC might requires different steps to create vDPA devices on top of the VFs. The unified management tool mentioned above will help unify this. The creation of vDPA devices in the vDPA bus is out of the scope of this project.

## Bind the desired vdpa driver
The vdpa bus works similar to the pci bus. To unbind a driver from a device, run:

    echo ${DEV_NAME} > /sys/bus/vdpa/devices/${DEV_NAME}/driver/unbind

To bind a driver to a device, run:

    echo ${DEV_NAME} > /sys/bus/vdpa/drivers/${DRIVER_NAME}/bind

## Priviledges
IPC_LOCK capability privilege is required for "vhost" mode to be used in a Kubernetes Pod.
