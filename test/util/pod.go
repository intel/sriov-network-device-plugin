package util

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
)

const secondsToWait = 15 * time.Second

// deletePod will delete a pod
func deletePod(ci coreclient.CoreV1Interface, pod *corev1.Pod, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := ci.Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
	return err
}

// waitForPodRecreation waits for pod selected by selector to get deleted and recreated with daemonset
func waitForPodRecreation(core coreclient.CoreV1Interface, oldPodName, ns string, selectors map[string]string, timeout,
	 interval time.Duration) error {
	if err := waitForPodDeletion(core, oldPodName, ns, timeout, interval); err != nil {
		return err
	}
	pods, _ := getPodWithSelectors(core, &ns, selectors)
	return waitForPodStateRunning(core, pods.Items[0].Name, pods.Items[0].Namespace, timeout, interval)
}

// waitForPodStateRunning waits for pod to enter running state
func waitForPodStateRunning(core coreclient.CoreV1Interface, podName, ns string,
	timeout, interval time.Duration) error {
	return wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		pod, err := core.Pods(ns).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		switch pod.Status.Phase {
		case corev1.PodRunning:
			return true, nil
		case corev1.PodFailed, corev1.PodSucceeded:
			return false, ErrPodNotRunning
		default:
			return false, nil
		}
	})
}

// waitForPodDeletion waits for pod to be deleted
func waitForPodDeletion(core coreclient.CoreV1Interface, podName, ns string, timeout, interval time.Duration) error {
	result := wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		_, err = core.Pods(ns).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		return false, nil
	})

	if result != nil {
		return nil
	}

	return ErrPodsDeletion
}

// getPodWithSelectors returns list of pod that matches selectors
func getPodWithSelectors(ci coreclient.CoreV1Interface, namespace *string,
	selectors map[string]string) (*corev1.PodList, error) {
	labelSelector := prepareSelectors(selectors)
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}
	pods, err := ci.Pods(*namespace).List(context.TODO(), listOptions)
	if len(pods.Items) == 0 && err == nil {
		err = ErrNoPodsFound
	}
	return pods, err
}

// waitForDpResourceUpdate waits for DP to send update
func waitForDpResourceUpdate(ci coreclient.CoreV1Interface, pod corev1.Pod, timeout,
	interval time.Duration) (bool, error) {
	isUpdated := false

	var err error

	retries := int(timeout / interval)
	for i := 0; i < retries; i++ {
		isUpdated, err = checkForDpResourceUpdate(ci, pod)
		if isUpdated {
			time.Sleep(secondsToWait)

			break
		}
		time.Sleep(interval)
	}
	return isUpdated, err
}

// checkForDpResourceUpdate checks if device plugin's resource list has been updated
func checkForDpResourceUpdate(ci coreclient.CoreV1Interface, pod corev1.Pod) (bool, error) {
	isUpdated := false
	res := ci.Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})

	logsStream, err := res.Stream(context.TODO())
	if err != nil {
		return isUpdated, err
	}
	defer logsStream.Close()

	tmpBuf := new(bytes.Buffer)
	_, err = io.Copy(tmpBuf, logsStream)
	if err != nil {
		return isUpdated, err
	}

	if logStr := tmpBuf.String(); strings.Contains(logStr, "send devices") {
		isUpdated = true
	}

	return isUpdated, nil
}

// prepareSelectors perpares pod selectors to be used
func prepareSelectors(labels map[string]string) string {
	var result string
	index := 0
	for key, value := range labels {
		result += key + "=" + value
		if index < (len(labels) - 1) {
			result += ","
		}
		index++
	}
	return result
}

// ErrNoPodsFound error when no pods are found
var ErrNoPodsFound = errors.New("no suitable pods found")

// ErrPodsDeletion error when pod was not deleted
var ErrPodsDeletion = errors.New("pod not deleted")

// ErrPodNotRunning error when pod is not in running state
var ErrPodNotRunning = errors.New("pod is not running")
