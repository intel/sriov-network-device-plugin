package util

import (
	"context"
	"errors"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
)

// GetNodeAllocatableResourceQuantinty returns number of allocatable resources of provided name
func GetNodeAllocatableResourceQuantinty(ci coreclient.CoreV1Interface, nodeName, resName string) (int, error) {
	node, err := ci.Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})

	if err != nil {
		return -1, err
	}
	for k, v := range node.Status.Allocatable {
		if strings.Contains(k.String(), resName) {
			vv, success := v.AsInt64()
			if !success {
				return -2, ErrResourceParsing
			}
			return int(vv), err
		}
	}
	return -3, ErrNoResourceFound
}

// ErrResourceParsing is error returned when resources could not be parsed
var ErrResourceParsing = errors.New("could not parse resource quantity")

// ErrNoResourceFound is error returned when resource was not found
var ErrNoResourceFound = errors.New("could not find provided resource")
