#!/bin/sh

# This script is based on:
# https://github.com/intel/multus-cni/blob/17b24d5fd5bed65be3e46c7582f6644d73bb155c/e2e/get_tools.sh

set -o errexit

if [ ! -d bin ]; then
  /usr/bin/mkdir bin
fi

/usr/bin/curl -Lo ./bin/kind "https://github.com/kubernetes-sigs/kind/releases/download/v0.10.0/kind-$(uname)-amd64"
/usr/bin/chmod +x ./bin/kind

/usr/bin/curl -Lo ./bin/kubectl "https://storage.googleapis.com/kubernetes-release/release/$(/usr/bin/curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x ./bin/kubectl
