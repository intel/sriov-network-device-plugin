## Dockerfile build

This is used for distribution of SR-IOV Device Plugin binary in a Docker image.

Typically you'd build this from the root of your SR-IOV network device plugin clone, and you'd set the `-f` flag to specify the Dockerfile during build time. This allows the addition of the entirety of the SR-IOV network device plugin git clone as part of the Docker context. Use the `-f` flag with the root of the clone as the context (e.g. your current work directory would be root of git clone), such as:

```
$ docker build -t ghcr.io/k8snetworkplumbingwg/sriov-network-device-plugin -f ./images/Dockerfile .
```
You can run `make image` to build the docker image as well.

---

## Daemonset deployment

You may wish to deploy SR-IOV device plugin as a daemonset, you can do so by starting with the example Daemonset shown here:

```
$ kubectl create -f ./deployments/k8s-v1.16/sriovdp-daemonset.yaml
```
> For K8s version v1.15 or older use `deployments/k8s-v1.10-v1.15/sriovdp-daemonset.yaml` instead.

Note: The likely best practice here is to build your own image given the Dockerfile, and then push it to your preferred registry, and change the `image` fields in the Daemonset YAML to reference that image.

---

### Development notes

Example docker run command:

```
$ docker run -it -v /var/lib/kubelet/:/var/lib/kubelet/ -v /sys/class/net:/sys/class/net --entrypoint=/bin/bash ghcr.io/k8snetworkplumbingwg/sriov-network-device-plugin
```

Originally inspired by and is a portmanteau of the [Flannel daemonset](https://github.com/coreos/flannel/blob/master/Documentation/kube-flannel.yml), the [Calico Daemonset](https://github.com/projectcalico/calico/blob/master/v2.0/getting-started/kubernetes/installation/hosted/k8s-backend-addon-manager/calico-daemonset.yaml), and the [Calico CNI install bash script](https://github.com/projectcalico/cni-plugin/blob/be4df4db2e47aa7378b1bdf6933724bac1f348d0/k8s-install/scripts/install-cni.sh#L104-L153).
