# SRIOV Network device plugin for Kubernetes

[![Travis CI](https://travis-ci.org/intel/sriov-network-device-plugin.svg?branch=master)](https://travis-ci.org/intel/sriov-network-device-plugin/builds)

## Table of Contents

- [SRIOV Network device plugin](#sriov-network-device-plugin)
- [Features](#features)
- [Prerequisites](#prerequisites)
  - [Supported SRIOV NICs](#supported-sriov-nics)
- [Quick Start](#quick-start)
  - [Network Object CRDs](#network-object-crds)
  - [Build and configure Multus](#build-and-configure-multus)
  - [Build SRIOV CNI](#build-sriov-cni)
  - [Build and run SRIOV network device plugin](#build-and-run-sriov-network-device-plugin)
  - [Configurations](#configurations)
    - [Config parameters](#config-parameters)
    - [Command line arguments](#command-line-arguments)
    - [Assumptions](#assumptions)
    - [Workflow](#workflow)
  - [Example deployments](#example-deployments)
    - [Testing SRIOV workloads](#testing-sriov-workloads)
      - [Deploy test Pod](#deploy-test-pod)
      - [Verify Pod network interfaces](#verify-pod-network-interfaces)
      - [Verify Pod routing table](#verify-pod-routing-table)
    - [Pod device information](#pod-device-information)
- [New mdev device plugin](#new-mdev-device-plugin)
- [Issues and Contributing](#issues-and-contributing)

## SRIOV Network Device Plugin

The SRIOV network device plugin is Kubernetes device plugin for discovering and advertising SRIOV network virtual functions (VFs) in a Kubernetes host. 

## Features

- Handles SRIOV capable/not-capable devices (NICs and Accelerators alike)
- Supports devices with both Kernel and userspace(uio and VFIO) drivers
- Supports PF bound to DPDK driver to meet certain use-cases
- Allow grouping together multiple PCI devices as one aggregated resource pool
- Can represent each PF as a separately addressable resource pool to K8s
- User configurable resourceName
- Detects Kubelet restarts and auto-re-register
- Detects Link status (for Linux network devices) and updates associated VFs health accordingly
- Extensible to support new device types with minimal effort if not already supported

To deploy workloads with SRIOV VF this plugin needs to work together with the following two CNI plugins:

- Multus CNI

  - Retrieves allocated network device information of a Pod

- SRIOV CNI

  - During Pod creation, plumbs allocated SRIOV VF to a Pods network namespace using VF information given by Multus

  - On Pod deletion, reset and release the VF from the Pod

This implementation follows the design discuessed in [this proposal document](https://docs.google.com/document/d/1Ewe9Of84GkP0b2Q2PC0y9RVZNkN2WeVEagX9m99Nrzc/).


Please follow the Multus [Quick Start](#quick-start) for multi network interface support in Kubernetes.

### Supported SRIOV NICs

The following  NICs were tested with this implementation. However, other SRIOV capable NICs should work as well.
-  Intel® Ethernet Controller X710 Series 4x10G
		- PF driver : v2.4.6
		- VF driver: v3.5.6
> please refer to Intel download center for installing latest [Intel Ethernet Controller-X710-Series](https://downloadcenter.intel.com/product/82947/Intel-Ethernet-Controller-X710-Series) drivers
 - Intel® 82599ES 10 Gigabit Ethernet Controller
	- PF driver : v4.4.0-k
	- VF driver: v3.2.2-k
> please refer to Intel download center for installing latest [Intel-® 82599ES 10 Gigabit Ethernet](https://ark.intel.com/products/41282/Intel-82599ES-10-Gigabit-Ethernet-Controller) drivers

- Mellanox ConnectX�-4 Lx EN Adapter
- Mellanox ConnectX�-5 Adapter
> Network card drivers are available as a part of the various linux distributions and upstream.
To download the latest Mellanox NIC drivers, click [here](http://www.mellanox.com/page/software_overview_eth).

## Quick Start

This section explains an exmaple deployment of SRIOV Network device plugin in Kubernetes. Required YAML files can be found in [deployments/](deployments/) directory.

### Network Object CRDs

Multus uses Custom Resource Definitions(CRDs) for defining additional network attachements. These network attachment CRDs follow the standards defined by K8s Network Plumbing Working Group(NPWG). Please refer to [Multus documentation](https://github.com/intel/multus-cni/blob/master/README.md) for more information.

### Build and configure Multus

1. Compile Multus executable:
```
$ git clone https://github.com/intel/multus-cni.git
$ cd multus-cni
$ ./build
$ cp bin/multus /opt/cni/bin
```
2. Copy the multus Configuration file from the Deployments folder to the CNI Configuration diectory
```
$ cp deployments/cni-conf.json /etc/cni/net.d/
```

3. Configure Kubernetes network CRD with [Multus](https://github.com/intel/multus-cni/tree/dev/network-plumbing-working-group-crd-change#creating-network-resources-in-kubernetes)
```
$ kubectl create -f deployments/crdnetwork.yaml
```

### Build SRIOV CNI

1. Compile SRIOV-CNI (dev/k8s-deviceid-model branch):
```
$ git clone https://github.com/intel/sriov-cni.git
$ cd sriov-cni
$ make
$ cp build/sriov /opt/cni/bin
```
2. Create the SRIOV Network CRD
```
$ kubectl create -f deployments/sriov-crd.yaml
```

### Build and run SRIOV network device plugin

 1. Clone the sriov-network-device-plugin
 ```
$ git clone https://github.com/intel/sriov-network-device-plugin.git
$ cd sriov-network-device-plugin
 ```  
 2. Build executable binary using `make` 
 ``` 
$ make
```      
> On successful build the `sriovdp` executable can be found in `./build` directory. It is recommended to run the plugin in a container or K8s Pod. The follow on steps cover how to build and run the Docker image of the plugin.

 3. Build docker image
 ```
$ make image
``` 

> See following sections on how to configure and run SRIOV device plugin.

## Configurations

### Config parameters

This plugin creates device plugin endpoints based on the configurations given in file `/etc/pcidp/config.json`. This configuration file is in json format as shown below:

```json
{
    "resourceList":
    [
        {
            "resourceName": "sriov_net_A",
            "rootDevices": ["02:00.0", "02:00.2"],
            "sriovMode": true,
            "deviceType": "netdevice"
        },
        {
            "resourceName": "sriov_net_B",
            "rootDevices": ["02:00.1", "02:00.3"],
            "sriovMode": true,
            "deviceType": "vfio"
        }
    ]
}
```

`"resourceList"` should contain a list of config objects. Each config object may consist of following fields:

|     Field      | Required |                    Description                    |                       Type - Accepted values                        |         Example          |
|----------------|----------|---------------------------------------------------|---------------------------------------------------------------------|--------------------------|
| "resourceName" | Yes      | Endpoint resource name                            | `string` - must be unique and should not contain special characters | `"sriov_net_A"`          |
| "rootDevices"  | Yes      | List of PCI address for a resource pool           | A list of `string` - in sysfs pci address format                    | `["02:00.0", "02:00.2"]` |
| "sriovMode"    | No       | Whether the root devices are SRIOV capable or not | `bool` - true OR false[default]                                     | `true`                   |
| "deviceType"   | No       | Device driver type                                | `string` - "netdevice"\|"uio"\|"vfio"                               | `"netdevice"`            |

### Command line arguments

This plugin accepts the following optional run-time command line arguments:

```bash
./sriovdp --help

Usage of ./sriovdp:
  -alsologtostderr
        log to standard error as well as files
  -config-file string
        JSON device pool config file location (default "/etc/pcidp/config.json")
  -log_backtrace_at value
        when logging hits line file:N, emit a stack trace
  -log_dir string
        If non-empty, write log files in this directory
  -logtostderr
        log to standard error instead of files
  -resource-prefix string
        resource name prefix used for K8s extended resource (default "intel.com")
  -stderrthreshold value
        logs at or above this threshold go to stderr
  -v value
        log level for V logs
  -vmodule value
        comma-separated list of pattern=N settings for file-filtered logging
```

### Assumptions

This plugin does not bind or unbind any driver to any device whether it's PFs or VFs. It also doesn't create Virtual functions either. Usually, the virtual functions are created at boot time when kernel module for the device is loaded. Required device drivers could be loaded on system boot-up time by white-listing/black-listing the right modules. But plugin needs to be aware of the driver type of the resources(i.e. devices) that it is registering as K8s extended resource so that it's able to create appropriate Device Specs for the requested resource.

For exmaple, if the driver type is uio(i.e. igb_uio.ko) then there are specific device files to add in Device 
Spec. For vfio-pci, device files are different. And if it is Linux kernel network driver then there is no device file to be added.

The idea here is, user creates a resource config for each resource pool as shown in [Config parameters](#config-parameters) by specifying the resource name, a list of device PCI addresses, whether the resources are physical functions or virtual functions(`"sriovMode": true`) and its type.

If `"sriovMode": true` is given for a resource config then plugin will look for virtual functions(VFs) for all the devices listed in `"rootDevices"` and export the discovered VFs as allocatable extended resource list. Otherwise, plugin will export the root devices themsleves as the allocatable extended resources.

### Workflow

- Load device's (Physical funtion if it is SRIOV capable) kernel module and bind the driver to the PF
- Create required Virtual functions
- Bind all VF with right drivers
- Create resource config entry in `/etc/pcidp/config.json`
- Run SRIOV device plugin (as daemonset)

On successfull run, the allocatable resource list for the node should be updated with resource discovered by the plugin as shown below. Note that the resource name appended with the `-resource-prefix` i.e. `"intel.com/sriov_net_A"`.

```json
$ kubectl get node node1 -o json | jq '.status.allocatable'

{
  "cpu": "8",
  "ephemeral-storage": "169986638772",
  "hugepages-1Gi": "0",
  "hugepages-2Mi": "8Gi",
  "intel.com/sriov_net_A": "8",
  "intel.com/sriov_net_B": "8",
  "memory": "7880620Ki",
  "pods": "1k"
}

```

## Example deployments

We assume that you have working K8s cluster configured with Multus meta plugin for multi-network support. Please see [Features](#features) and [Quick Start](#quick-start) sections for more information on required CNI plugins.

The [images](./images) directory contains example Docker file, sample specs along with build scripts to deploy the SRIOV device plugin as daemonset. Please see [README.md](./images/README.md) building docker the image.

There are some example Pod specs and related network CRD yaml files can be found in [deployments](./deployments) directory for a sample deployments.


### Testing SRIOV workloads

Leave the sriov device plugin running and open a new terminal session for following steps.

#### Deploy test Pod

````
$ kubectl create -f pod-tc1.yaml
pod "testpod1" created

$ kubectl get pods
NAME                  READY     STATUS    RESTARTS   AGE
sriov-device-plugin   1/1       Running   0          7h
testpod1        	  1/1       Running   0          3s
````

#### Verify Pod network interfaces

````
$ kubectl exec -it testpod1 -- ip addr show

1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
3: eth0@if17511: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue state UP
    link/ether 0a:58:c0:a8:4a:b1 brd ff:ff:ff:ff:ff:ff link-netnsid 0
    inet 192.168.74.177/24 scope global eth0
       valid_lft forever preferred_lft forever
17508: net0: <NO-CARRIER,BROADCAST,MULTICAST,UP> mtu 1500 qdisc mq state DOWN qlen 1000
    link/ether ce:d8:06:08:e6:3f brd ff:ff:ff:ff:ff:ff
    inet 10.56.217.179/24 scope global net0
       valid_lft forever preferred_lft forever
````

#### Verify Pod routing table

````
$ kubectl exec -it testpod1 -- route -n

Kernel IP routing table
Destination     Gateway         Genmask         Flags Metric Ref    Use Iface
0.0.0.0         192.168.74.1    0.0.0.0         UG    0      0        0 eth0
10.56.217.0     0.0.0.0         255.255.255.0   U     0      0        0 net0
192.168.0.0     192.168.74.1    255.255.0.0     UG    0      0        0 eth0
192.168.74.0    0.0.0.0         255.255.255.0   U     0      0        0 eth0
````

### Pod device information

The allocated device information are exported in Container's environment variable. The variable name is `PCIDEVICE_` appended with full extended resource name(i.e. intel.com/sriov) which is capitailzed and any special characters(".", "/") are replaced with underscore("_"). In case of multiple devices from same extended resource pool, the device IDs are delimited with commas(",").

For example, if 2 devices are allocated from `intel.com/sriov` extended resource then the allocated device information will be found in following env variable:
`PCIDEVICE_INTEL_COM_SRIOV=0000:03:02.1,0000:03:04.3`

## New mdev device plugin

We extended this Kubernetes device plugin for mediate device (mdev) support, which is a recent addition to linux vfio framework that is currently used by, e.g. Intel Graphics Virtualization (GVT-g).

This plugin also creates device plugin endpoints based on the configurations given in file `/etc/pcidp/config.json`. This configuration file is in json format as shown below (with "mdevMode" parameter is true).

```json
{
    "resourceList":
    [
        {
            "resourceName": "vgpu",
            "rootDevices": [ "00:02.0" ],
            "mdevMode": true,
            "deviceType": "vfio"
        }
    ]
}

```

It will discover the root devices corresponding mediate devices by UUID. The you can follow the same steps of SRIOV device plugin to create Pod with mdev devices.

## Issues and Contributing

We welcome your feedback and contributions to this project. Please see the [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines. 

Copyright 2018 © Intel Corporation.
