
# SR-IOV Network device plugin for Kubernetes

[![Travis CI](https://travis-ci.org/intel/sriov-network-device-plugin.svg?branch=master)](https://travis-ci.org/intel/sriov-network-device-plugin/builds) [![Go Report Card](https://goreportcard.com/badge/github.com/intel/sriov-network-device-plugin)](https://goreportcard.com/report/github.com/intel/sriov-network-device-plugin)

## Table of Contents

- [SR-IOV Network device plugin](#sr-iov-network-device-plugin)
- [Features](#features)
  - [Supported SR-IOV NICs](#supported-sr-iov-nics)
- [Quick Start](#quick-start)
  - [Build SR-IOV CNI](#build-sr-iov-cni)
  - [Build and run SR-IOV network device plugin](#build-and-run-sr-iov-network-device-plugin)
  - [Install one compatible CNI meta plugin](#install-one-compatible-cni-meta-plugin)
      - [Option 1 - Multus](#option-1---multus)
        - [Install Multus](#install-multus)
        - [Network Object CRDs](#network-object-crds)
      - [Option 2 - DANM](#option-2---danm)
        - [Install DANM](#install-danm)
        - [Create SR-IOV type networks](#create-sr-iov-type-networks)
- [Configurations](#configurations)
  - [Config parameters](#config-parameters)
  - [Command line arguments](#command-line-arguments)
  - [Assumptions](#assumptions)
  - [Workflow](#workflow)
- [Example deployments](#example-deployments)
    - [Deploy the Device Plugin](#deploy-the-device-plugin)
    - [Deploy SR-IOV workloads when Multus is used](#deploy-sr-iov-workloads-when-multus-is-used)
      - [Deploy test Pod connecting to pre-created SR-IOV network](#deploy-test-pod-connecting-to-pre-created-sr-iov-network)
      - [Verify Pod network interfaces](#verify-pod-network-interfaces)
      - [Verify Pod routing table](#verify-pod-routing-table)
    - [Deploy SR-IOV workloads when DANM is used](#deploy-sr-iov-workloads-when-danm-is-used)
      - [Verify the existence of the example SR-IOV networks](#verify-the-existence-of-the-example-sr-iov-networks)
      - [Connect your networks to existing SR-IOV Device Pools](#connect-your-networks-to-existing-sr-iov-device-pools)
      - [Deploy demo Pod connecting to pre-created SR-IOV networks](#deploy-demo-pod-connecting-to-pre-created-sr-iov-networks)
      - [Verify status and the network connections of the demo Pod](#verify-status-and-the-network-connections-of-the-demo-pod)
    - [Pod device information](#pod-device-information)
- [Multi Architecture Support](#multi-architecture-support)
- [Issues and Contributing](#issues-and-contributing)

## SR-IOV Network Device Plugin

The SR-IOV network device plugin is Kubernetes device plugin for discovering and advertising SR-IOV virtual functions (VFs) available on a Kubernetes host.

## Features

- Handles SR-IOV capable/not-capable devices (NICs and Accelerators alike)
- Supports devices with both Kernel and userspace (UIO and VFIO) drivers
- Allows resource grouping using "Selector"
- User configurable resourceName
- Detects Kubelet restarts and auto-re-register
- Detects Link status (for Linux network devices) and updates associated VFs health accordingly
- Extensible to support new device types with minimal effort if not already supported

To deploy workloads with SR-IOV VF this plugin needs to work together with the following two CNI components:

- Any CNI meta plugin supporting Device Plugin based network provisioning (Multus CNI, or DANM)

  - Retrieves allocated network device information of a Pod

- SR-IOV CNI

  - During Pod creation, plumbs allocated SR-IOV VF to a Pods network namespace using VF information given by the meta plugin

  - On Pod deletion, reset and release the VF from the Pod


Please follow the [Quick Start](#quick-start) for multi network interface support in Kubernetes.

### Supported SR-IOV NICs

The following  NICs were tested with this implementation. However, other SR-IOV capable NICs should work as well.
-  Intel® Ethernet Controller X710 Series 4x10G
		- PF driver : v2.4.6
		- VF driver: v3.5.6
> please refer to Intel download center for installing latest [Intel Ethernet Controller-X710-Series](https://downloadcenter.intel.com/product/82947/Intel-Ethernet-Controller-X710-Series) drivers
 - Intel® 82599ES 10 Gigabit Ethernet Controller
	- PF driver : v4.4.0-k
	- VF driver: v3.2.2-k
> please refer to Intel download center for installing latest [Intel-® 82599ES 10 Gigabit Ethernet](https://ark.intel.com/products/41282/Intel-82599ES-10-Gigabit-Ethernet-Controller) drivers

- Mellanox ConnectX®-4 Lx EN Adapter
- Mellanox ConnectX®-5 Adapter
> Network card drivers are available as a part of the various linux distributions and upstream.
To download the latest Mellanox NIC drivers, click [here](http://www.mellanox.com/page/software_overview_eth).

## Quick Start

### Creating SR-IOV Virtual Functions

Before starting the SR-IOV device plugin you will need to create SR-IOV Virtual Functions on your system. [The VF Setup doc will guide you through that process.](docs/vf-setup.md)

### Build SR-IOV CNI

1. Compile SR-IOV-CNI (supported from release 2.0+):
```
$ git clone https://github.com/intel/sriov-cni.git
$ cd sriov-cni
$ make
$ cp build/sriov /opt/cni/bin
```

### Build and run SR-IOV network device plugin

You can either build the docker image locally or pull it from [docker hub](https://hub.docker.com/r/nfvpe/sriov-device-plugin/).

If you want to build the docker image locally then follow the following steps:

 1. Clone the sriov-network-device-plugin
 ```
$ git clone https://github.com/intel/sriov-network-device-plugin.git
$ cd sriov-network-device-plugin
 ```
 2. Build docker image binary using `make`
 ```
$ make image
```
> On a successful build, a docker image with tag `nfvpe/sriov-device-plugin:latest` will be created. You will need to build this image on each node. Alternatively, you could use a local docker registry to host this image.

 3. Create a ConfigMap that defines SR-IOV resrouce pool configuration
 
 > Make sure to update the 'config.json' entry in the configMap data to reflect your resource configuration for the device plugin. See [Configurations](#configurations) section for supported configuration parameters.

 ```
$ kubectl create -f deployments/configMap.yaml
```
 4. Deploy SR-IOV network device plugin Daemonset
```
$ kubectl create -f deployments/k8s-v1.16/sriovdp-daemonset.yaml
```
> For K8s version v1.15 or older use `deployments/k8s-v1.10-v1.15/sriovdp-daemonset.yaml` instead.


### Install one compatible CNI meta plugin
A compatible CNI meta-plugin installation is required for SR-IOV CNI plugin to be able to get allocated VF's deviceID in order to configure it.  

#### Option 1 - Multus

##### Install Multus
Please refer to Multus [Quickstart Installation Guide](https://github.com/intel/multus-cni#quickstart-installation-guide) to install Multus.

##### Network Object CRDs

Multus uses Custom Resource Definitions(CRDs) for defining additional network attachements. These network attachment CRDs follow the standards defined by K8s Network Plumbing Working Group(NPWG). Please refer to [Multus documentation](https://github.com/intel/multus-cni/blob/master/README.md) for more information.
1. Create the SR-IOV Network CRD
```
$ kubectl create -f deployments/sriov-crd.yaml
```

#### Option 2 - DANM
This section explains an example deployment of SR-IOV Network device plugin in Kubernetes if you choose DANM as your meta plugin.

##### Install DANM
Refer to [DANM documentation](https://github.com/nokia/danm#getting-started) for detailed instructions.

##### Create SR-IOV type networks
DANM supports the Device Plugin based SR-IOV provisioning with the dynamic level.
This means that all DANM API features seamlessly work together with the SR-IOV setup described above, whether you use the [lightweight](https://github.com/nokia/danm#lightweight-network-management-experience), or the [production grade](https://github.com/nokia/danm#production-grade-network-management-experience) network management APIs.
For example manifest objects refer to [SR-IOV demo](https://github.com/nokia/danm/tree/master/example/device_plugin_demo)

> See following sections on how to configure and run SR-IOV device plugin.

## Configurations

### Config parameters

This plugin creates device plugin endpoints based on the configurations given in  the config map associated with the SR-IOV device plugin. In json format this file appears as shown below:

```json
{
    "resourceList": [{
            "resourceName": "intel_sriov_netdevice",
            "selectors": {
                "vendors": ["8086"],
                "devices": ["154c", "10ed"],
                "drivers": ["i40evf", "ixgbevf"]
            }
        },
        {
            "resourceName": "intel_sriov_dpdk",
            "resourcePrefix": "intel.com",
            "selectors": {
                "vendors": ["8086"],
                "devices": ["154c", "10ed"],
                "drivers": ["vfio-pci"],
                "pfNames": ["enp0s0f0","enp2s2f1"]
            }
        },
        {
            "resourceName": "mlnx_sriov_rdma",
            "resourcePrefix": "mellanox.com",
            "selectors": {
                "vendors": ["15b3"],
                "devices": ["1018"],
                "drivers": ["mlx5_ib"],
                "isRdma": true
            }
        },
        {
            "resourceName": "infiniband_rdma_netdevs",
            "selectors": {
                "linkTypes": ["infiniband"],
                "isRdma": true
            }
        },
        {
            "resourceName": "intel_fpga",
            "deviceType": "accelerator",
            "selectors": {
                    "vendors": ["8086"],
                    "devices": ["0d90"]
            }
        }

    ]
}
```

`"resourceList"` should contain a list of config objects. Each config object may consist of following fields:


|      Field       | Required |                                                    Description                                                    |                     Type/Defaults                     |                     Example/Accepted values                     |
|------------------|----------|-------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------|-----------------------------------------------------------------|
| "resourceName"   | Y        | Endpoint resource name. Should not contain special characters, must be unique in the scope of the resource prefix | string                                                | "sriov_net_A"                                                   |
| "resourcePrefix" | N        | Endpoint resource prefix name override. Should not contain special characters                                     | string Default : "intel.com"                          | "yourcompany.com"                                               |
| "deviceType"     | N        | Device Type for a resource pool.                                                                                  | string value of supported types. Default: "netDevice" | Currently supported values: "accelerator", "netDevice"          |
| "selectors"      | N        | A map of device selectors. The "deviceType" value determines the "selectors" options.                             | json object as string Default: null                   | Example: "selectors": {"vendors": ["8086"],"devices": ["154c"]} |



Note: "resourceName" must be unique only in the scope of a given prefix, including the one specified globally in the CLI params, e.g. "example.com/10G", "acme.com/10G" and "acme.com/40G" are perfectly valid names.

#### Device selectors
The "deviceType" value determines which selectors are supported for that device. Each selector evaluated in order as listed in selector tables below.

#### Common selectors
All device types support following common device selectors.

|   Field   | Required |                Description                |         Type/Defaults          |   Example/Accepted values   |
|-----------|----------|-------------------------------------------|--------------------------------|-----------------------------|
| "vendors" | N        | Target device's vendor Hex code as string | `string` list Default: `null`  | "vendors": ["8086"]         |
| "devices" | N        | Target Devices' device Hex code as string | `string` list Default: `null`  | "devices": ["154c", "10ed"] |
| "drivers" | N        | Target device driver names as string      | `string` list Default: `null`  | "drivers": ["vfio-pci"]     |


#### Extended selectors for device type "netDevice"
This selector is applicable when "deviceType" is "netDevice"(note: this is default). In addition to the common selectors from above table, the "netDevice" also supports following selectors.

|     Field     | Required |                          Description                           |                   Type/Defaults                   |                               Example/Accepted values                                |
|---------------|----------|----------------------------------------------------------------|---------------------------------------------------|--------------------------------------------------------------------------------------|
| "pfNames"     | N        | VFs from PF matches list of PF names                           | `string` list Default: `null`                     | "pfNames": ["enp2s2f0"] (See follow-up sections for some advance usage of "pfNames") |
| "linkTypes"   | N        | The link type of the net device associated with the PCI device | `string` list Default: `null`                     | "linkTypes": ["ether"]                                                               |
| "ddpProfiles" | N        | A map of device selectors                                      | `string` list Default: `null`                     | "ddpProfiles": ["GTPv1-C/U IPv4/IPv6 payload"]                                       |
| "isRdma"      | N        | Mount RDMA resources                                           | `bool`  values `true` or `false` Default: `false` | "isRdma": `true`                                                                     |


[//]: # (The tables above generated using: https://ozh.github.io/ascii-tables/)

#### Extended selectors for device type "accelerator"
This selector is applicable when "deviceType" is "accelerator". The "accelerator" device type currently supports only the common selectors.

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

This plugin does not bind or unbind any driver to any device whether it's PFs or VFs. It also doesn't create Virtual functions either. Usually, the virtual functions are created at boot time when kernel module for the device is loaded. Required device drivers could be loaded on system boot-up time by white-listing/black-listing the right modules. But plugin needs to be aware of the driver type of the resources (i.e. devices) that it is registering as K8s extended resource so that it's able to create appropriate Device Specs for the requested resource.

For example, if the driver type is uio (i.e. igb_uio.ko) then there are specific device files to add in Device Spec. For vfio-pci, device files are different. And if it is Linux kernel network driver then there is no device file to be added.

The idea here is, user creates a resource config for each resource pool as shown in [Config parameters](#config-parameters) by specifying the resource name, a list resource "selectors".

The device plugin will initially discover all PCI network resources in the host and populate an initial "device list". Each "resource pool" then applies its selectors on this list and add devices that satisfies the selector's constraints. Each selector narrows down the list of devices for the resource pool. Currently, the selectors are applied in following order:

1. "vendors" - The vendor hex code of device
2. "devices" - The device hex code of device
3. "drivers" - The driver name the device is registered with
4. "pfNames" - The Physical function name
5. "linkTypes" - The link type of the net device associated with the PCI device.

The "pfNames" selector can be used to specify a list and/or range of VFs for a pool in the below format:
````
"<PFName>#<SingleVF>,<FirstVF>-<LastVF>,<SingleVF>,<SingleVF>,<FirstVF>-<LastVF>"
````

Where:

    `<PFName>`   - is the PF interface name
    `<SingleVF>` - is a single VF index (0-based) that is included into the pool
    `<FirstVF>`  - is the first VF index (0-based) that is included into the range
    `<LastVF>`   - is the last VF index (0-based) that is included into the range

Example:

The selector for interface named `netpf0` and VF 0, 2 upto 7 (included 2 and 7) and 9 will look like:
````
"pfNames": ["netpf0#0,2-7,9"]
````
If only PF network interface specified in the selector, then assuming that all VFs of this interface are going to the pool.


### Workflow

- Load device's (Physical function if it is SR-IOV capable) kernel module and bind the driver to the PF
- Create required Virtual functions
- Bind all VF with right drivers
- Create a resource config map
- Run SR-IOV device plugin (as daemonset)

On successful run, the allocatable resource list for the node should be updated with resource discovered by the plugin as shown below. Note that the resource name is appended with the `-resource-prefix` i.e. `"intel.com/sriov_net_A"`.

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

We assume that you have working K8s cluster configured with one of the supported meta plugins for multi-network support. Please see [Features](#features) and [Quick Start](#quick-start) sections for more information on required CNI plugins.

### Deploy the Device Plugin
The [images](./images) directory contains example Dockerfile, sample specs along with build scripts to deploy the SR-IOV device plugin as daemonset. Please see [README.md](./images/README.md) for more information about the Docker images.

````
# Create ConfigMap
$ kubectl create -f deployments/configMap.yaml
configmap/sriovdp-config created

# Create sriov-device-plugin-daemonset
$ kubectl create -f deployments/k8s-v1.16/sriovdp-daemonset.yaml
serviceaccount/sriov-device-plugin created
daemonset.apps/kube-sriov-device-plugin-amd64 created

$ kubectl -n kube-system get pods
NAMESPACE     NAME                                   READY   STATUS    RESTARTS   AGE
kube-system   kube-sriov-device-plugin-amd64-46wpv   1/1     Running   0          4s

````

### Deploy SR-IOV workloads when Multus is used
There are some example Pod specs and related network CRD yaml files in [deployments](./deployments) directory for a sample deployment with Multus.

Leave the SR-IOV device plugin running and open a new terminal session for following steps.

#### Deploy test Pod connecting to pre-created SR-IOV network

````
$ kubectl create -f pod-tc1.yaml
pod "testpod1" created

$ kubectl get pods
NAME                  READY     STATUS    RESTARTS   AGE
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

### Deploy SR-IOV workloads when DANM is used
#### Verify the existence of the example SR-IOV networks

````
[cloudadmin@controller-1 ~]$ kubectl get dnet -n example-sriov
NAME         AGE
management   6s
sriov-a      14m
sriov-b      13m
````

#### Connect your networks to existing SR-IOV Device Pools
The Spec.Options.device_pool mandatory parameter denotes the Device Pool used by the network.
Make sure this parameter is set to the name(s) of your existing SR-IOV Device Pool(s)!

````
[cloudadmin@controller-1 ~]$ kubectl describe node 172.31.3.154 | grep -A8 Allocatable
Allocatable:
 cpu:                          6
 ephemeral-storage:            50189Mi
 hugepages-1Gi:                0
 hugepages-2Mi:                0
 memory:                       249150992Ki
 nokia.k8s.io/exclusive_caas:  16
 nokia.k8s.io/shared_caas:     32k
 nokia.k8s.io/sriov_ens2f1:    32

[cloudadmin@controller-1 ~]$ kubectl describe dnet sriov-a -n example-sriov | grep device_pool
    device_pool:       nokia.k8s.io/sriov_ens2f1
[cloudadmin@controller-1 ~]$ kubectl describe dnet sriov-b -n example-sriov | grep device_pool
    device_pool:       nokia.k8s.io/sriov_ens2f1

````

#### Deploy demo Pod connecting to pre-created SR-IOV networks
First, make sure that your Pod asks appropriate number of Devices from the right Device Pools:

````
[cloudadmin@controller-1 ~]$ grep -B1 sriov_ sriov_pod.yaml
      requests:
        nokia.k8s.io/sriov_ens2f1: '2'
      limits:
        nokia.k8s.io/sriov_ens2f1: '2'
````

Then instantiate the Pod:

````
[cloudadmin@controller-1 ~]$ kubectl create -f sriov_pod.yaml
pod/sriov-pod created
````

#### Verify status and the network connections of the demo Pod

````
[cloudadmin@controller-1 ~]$ kubectl get pod sriov-pod -n example-sriov
NAME        READY   STATUS    RESTARTS   AGE
sriov-pod   1/1     Running   0          111s

[cloudadmin@controller-1 ~]$ kubectl exec -n example-sriov -it sriov-pod -- ip addr show
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host
       valid_lft forever preferred_lft forever
3: eth0@if49: <BROADCAST,MULTICAST,UP,LOWER_UP,M-DOWN> mtu 8950 qdisc noqueue
    link/ether 8a:74:fd:e0:ee:fa brd ff:ff:ff:ff:ff:ff
    inet 10.244.3.8/24 brd 10.244.3.255 scope global eth0
       valid_lft forever preferred_lft forever
9: second_path2: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc mq qlen 1000
    link/ether e2:19:e0:1b:91:44 brd ff:ff:ff:ff:ff:ff
26: first_path1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc mq qlen 1000
    link/ether 7e:0d:fa:eb:83:8c brd ff:ff:ff:ff:ff:ff
````

### Pod device information

The allocated device information is exported in Container's environment variable. The variable name is `PCIDEVICE_` appended with full extended resource name (e.g. intel.com/sriov etc.) which is capitailzed and any special characters (".", "/") are replaced with underscore ("_"). In case of multiple devices from same extended resource pool, the device IDs are delimited with commas (",").

For example, if 2 devices are allocated from `intel.com/sriov` extended resource then the allocated device information will be found in following env variable:
`PCIDEVICE_INTEL_COM_SRIOV=0000:03:02.1,0000:03:04.3`

## Multi Architecture Support

The supported architectures:
* AMD64
* PPC64LE

Buiding image for AMD64:
```
$ DOCKERFILE=Dockerfile make image 
```

Buiding image for PPC64LE:
```
$ DOCKERFILE=images/Dockerfile.ppc64le TAG=nfvpe/sriov-device-plugin:ppc64le make image        
```

## Issues and Contributing

We welcome your feedback and contributions to this project. Please see the [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

Copyright 2018 © Intel Corporation.
