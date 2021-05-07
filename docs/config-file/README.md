# Using node specific config file for running device plugin DaemonSet

To allow granular and accurate control over which SR-IOV devices can be exposed as kubernetes extended resource, it is sometimes required to define a per-node config file when launching SR-IOV device plugin DaemonSet in a heterogeneous cluster. Since SR-IOV device plugin provides a command line option `--config-file`, the node specific config file can be achieved by running the following steps:

1. Generate configMap with node specific sections:
```
apiVersion: v1
kind: ConfigMap
metadata:
  name: sriovdp-config
data:
  sriov-node-0: '{"resourceList":[{"resourceName":"sriovnics","selectors":{"pfNames":["ens785f0#0-4","ens785f1#0-9"],"IsRdma":false,"NeedVhostNet":false},"SelectorObj":null}]}'
  sriov-node-1: '{"resourceList":[{"resourceName":"sriovnics","selectors":{"pfNames":["ens785f0#0-9","ens785f1#0-4"],"IsRdma":false,"NeedVhostNet":false},"SelectorObj":null}]}'
``` 

`sriov-node-0` and `sriov-node-1` match the kubernetes node names.

2. Launch device plugin DaemonSet:
```
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: sriov-device-plugin
  namespace: kube-system

---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: kube-sriov-device-plugin-amd64
  namespace: kube-system
  labels:
    tier: node
    app: sriovdp
spec:
  selector:
    matchLabels:
      name: sriov-device-plugin
  template:
    metadata:
      labels:
        name: sriov-device-plugin
        tier: node
        app: sriovdp
    spec:
      hostNetwork: true
      nodeSelector:
        beta.kubernetes.io/arch: amd64
      tolerations:
      - key: node-role.kubernetes.io/master
        operator: Exists
        effect: NoSchedule
      serviceAccountName: sriov-device-plugin
      containers:
      - name: kube-sriovdp
        image: ghcr.io/k8snetworkplumbingwg/sriov-network-device-plugin:stable-amd64
        imagePullPolicy: IfNotPresent
        args:
        - --log-dir=sriovdp
        - --log-level=10
        - --config-file=/etc/pcidp/$(NODE_NAME)
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        securityContext:
          privileged: true
        volumeMounts:
        - name: devicesock
          mountPath: /var/lib/kubelet/
          readOnly: false
        - name: log
          mountPath: /var/log
        - name: config-volume
          mountPath: /etc/pcidp
      volumes:
        - name: devicesock
          hostPath:
            path: /var/lib/kubelet/
        - name: log
          hostPath:
            path: /var/log
        - name: config-volume
          configMap:
            name: sriovdp-config
```

`sriovdp-config` configMap maps node specific config data to device plugin container volume as separate files such as `sriov-node-0` and `sriov-node-1`.
`NODE_NAME` environment variable is defined from `.spec.nodeName` and is equal to the node name which matches with data entry in `sriovdp-config` configMap.
`--config-file` argument specifies the node specific config file.
