package util

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
)

// DeployCm deploys configmap
func DeployCm(ci coreclient.CoreV1Interface, testCmName, testNs *string,
	cmData *map[string]string, timeout time.Duration) (bool, error) {
	configMap := prepareConfigMap(testCmName, testNs, cmData)
	err := createConfigMap(ci, configMap, timeout)
	if err != nil {
		return false, err
	}
	return waitForCmDeployment(ci, configMap, timeout)
}

// DeleteCm deletes ConfigMap
func DeleteCm(ci coreclient.CoreV1Interface, name *string, namespace *string) error {
	err := ci.ConfigMaps(*namespace).Delete(context.TODO(), *name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("DeleteCm: %w", err)
	}
	_, err = ci.ConfigMaps(*namespace).Get(context.TODO(), *name, metav1.GetOptions{})
	return err
}

// PrepareConfigMap prepare SR-IOV device plugin ConfigMap
func prepareConfigMap(name *string, namespace *string, data *map[string]string) *corev1.ConfigMap {
	configMap := &corev1.ConfigMap{}
	configMap.Name = *name
	configMap.Namespace = *namespace
	configMap.Data = *data
	return configMap
}

// CreateConfigMap create ConfigMap it is running
func createConfigMap(ci coreclient.CoreV1Interface, configMap *corev1.ConfigMap, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, err := ci.ConfigMaps(configMap.Namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// WaitForCmDeployment waits for ConfigMap to be deployed
func waitForCmDeployment(ci coreclient.CoreV1Interface, configMap *corev1.ConfigMap,
	timeout time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, err := ci.ConfigMaps(configMap.Namespace).Get(ctx, configMap.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}
