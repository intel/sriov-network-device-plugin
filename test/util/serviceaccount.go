package util

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
)

// CreateServiceAccount applies service account
func CreateServiceAccount(ci coreclient.CoreV1Interface, name, namespace string) error {
	sa := corev1.ServiceAccount{}
	sa.Name = name
	sa.Namespace = namespace
	_, err := ci.ServiceAccounts(namespace).Create(context.TODO(), &sa, metav1.CreateOptions{})
	return err
}

// DeleteServiceAccount deletes service account
func DeleteServiceAccount(ci coreclient.CoreV1Interface, name, namespace string) error {
	err := ci.ServiceAccounts(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	return err
}
