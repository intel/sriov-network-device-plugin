## SR-IOV Device Plugin e2e test with kind

### How to test e2e

1. Setup kind cluster:

```
$ git clone https://github.com/k8snetworkplumbingwg/sriov-network-device-plugin.git
$ cd sriov-network-device-plugin
$ ./scripts/e2e_get_tools.sh
$ ./scripts/e2e_setup_cluster.sh
```

2. Setup hardware:

    The example setup was done on machine with 4 SR-IOV capable Intel X710 network interfaces named ens785f0-3.
    
    1. 6 VFs were created on ens785f2.

        ```
        $ echo 6 > /sys/class/net/ens785f2/device/sriov_numvfs
        ```

    2. Interface ens785f2 has been moved to kind container's netns:

        ```
        $ ip link set ens785f2 netns $(docker inspect $(docker ps | grep kind | awk '{printf $1}') | grep -o \[a-zA-Z0-9\/]*netns[a-zA-Z0-9\/]*)
        ```

    3. 2 VFs and interface ens785f3 has been binded to vfio-pci driver.

    That gives total of 4 PFs and 6 VFs. However interface that has been moved to kind container's netns cannot be discovered by deviceplugin, therefor for purpose of the test the total number of devices will be 9 (6 VFs + 3 PFs).

3. Run the test:

    ```
    $ cd test/e2e && go test
    ```

    The number of interfaces for test (in case of hardware configuration that differs from the presented) can be configured on test start via following configuration flags:

    - `pfnamefortest` - PF that was moved to kind container's netns (default ens785f2).
    - `numofpfnetdev` - number of PFs that use kernel-driver (e.g. `i40e` for Intel X700 series cards). In most cases this will be equal to total number of NICs - number of NICs that uses `vfio-pci` driver - 1 (for PF that was moved to container's networking namespace) - (default value = 2).
    - `numofpfvfio` - number of PFs that are using `vfio-pci` driver (default value = 1).
    - `numofvfnetdev` - number of VFs that are using kernel driver (e.g. `iavf` for Intel cards, default value = 4).
    - `numofvfvfio` - number of VFs that are using `vfio-pci` driver (default value = 2).
    - `numofvfiovfforselectedpf`- total number of VF's that was created on PF selected by `pfnamefortest` flag (default value = 6).

    Example:

    3 NICs available: `eth0`, `eth1` and `eth2`. 9 VFs ceated on `eth1` of which 5 uses `vfio-pci` driver. 1 PF also uses `vfio-pci`. Interface **eth1** was moved to kind container's netns. Therefore test should be executed with following flags:

    ```
    $ go test -pfnamefortest=eth1 -numofpfnetdev=1 -numofpfvfio=1 -numofvfnetdev=4 -numofvfvfio=5 -numofvfiovfforselectedpf=9
    ```

### How to teardown cluster

```
$ ./scripts/bin/kind delete cluster
```

### Current test cases
* Discover PF - vendor, device, driver - netdev
* Discover PF - vendor, device, driver - vfio
* Discover PF - vendor, device
* Discover VF - vendor, device, driver - netdev
* Discover VF - vendor, device, driver - vfio
* Discover VF - vendor, device
* Discover VF - vendor, device, driver, 
* Discover PF and VF - vendor, device, driver
