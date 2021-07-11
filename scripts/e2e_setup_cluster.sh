#!/bin/bash
set -eo pipefail

root="$(dirname "$0")/../"
export PATH="${PATH}:${root:?}/bin"
RETRY_MAX=10
INTERVAL=10
TIMEOUT=300
test_pf="$1"

check_requirements() {
  for cmd in docker kind kubectl ip; do
    if ! command -v "$cmd" &> /dev/null; then
      echo "$cmd is not available"
      exit 1
    fi
  done
  echo "### verify no existing KinD cluster is running"
  kind_clusters=$(kind get clusters)
  if [[ "$kind_clusters" == *"kind"* ]]; then
    echo "ERROR: Please teardown existing KinD cluster"
    exit 2
  fi

  if [ -z "$test_pf" ]
  then
    echo "ERROR: SR-IOV physical function netdev name must be specified as an argument"
    exit 3
  fi

  echo "### verify test device is a netdevice"
  ip link show "$test_pf" 2>&1 > /dev/null
}

retry() {
  local status=0
  local retries=${RETRY_MAX:=5}
  local delay=${INTERVAL:=5}
  local to=${TIMEOUT:=20}
  cmd="$*"

  while [ $retries -gt 0 ]
  do
    status=0
    timeout $to bash -c "echo $cmd && $cmd" || status=$?
    if [ $status -eq 0 ]; then
      break;
    fi
    echo "Exit code: '$status'. Sleeping '$delay' seconds before retrying"
    sleep $delay
    let retries--
  done
  return $status
}

echo "## checking requirements"
check_requirements
echo "## deploy single node control/data plane cluster with KinD"
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
EOF
echo "## export kube config for utilising locally"
kind export kubeconfig
echo "## wait for coredns"
retry kubectl -n kube-system wait --for=condition=available deploy/coredns --timeout=${TIMEOUT}s
echo "## find KinD container"
kind_container="$(docker ps | grep kind | awk '{print $1}')"
echo "## validate KinD cluster formed"
[ -z "$kind_container" ] && echo "ERROR: Could not find a KinD container" && exit 4
echo "## find KinD's container network namespace"
kind_netns="$(docker inspect "$kind_container" | grep netns | awk '{print $2}' | sed 's/[\"\,]//g')"
[ -z "${kind_netns}" ] && echo "could not find KinD's network namespace"
echo "## move test PF to KinD's container"
ip link set dev "$test_pf" netns "$kind_netns"
[ "$?" -ne 0 ] && echo "ERROR: Failed to move netdev to KinD network namespace" && exit 5
echo "## moved '$test_pf' into kind's container's netns '$kind_netns'"
echo "## label KinD's control-plane-node as worker"
kubectl label node kind-control-plane node-role.kubernetes.io/worker= --overwrite
