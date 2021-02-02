package log

import (
	"context"
	"fmt"
	"io"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	containerHeader = `
================================================================
Container %d/%d: %s
================================================================
`
	podHeader = `
****************************************************************
Pod %s/%s started
****************************************************************
`
)

func LogContainer(client kubernetes.Interface, w io.Writer, namespace, podname, container string) error {
	pod := client.CoreV1().Pods(namespace)
	req := pod.GetLogs(podname, &v1.PodLogOptions{Container: container})
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		return err
	}
	defer podLogs.Close()
	_, err = io.Copy(w, podLogs)
	return err
}

func LogPod(client kubernetes.Interface, w io.Writer, namespace, podname string) error {
	pod := client.CoreV1().Pods(namespace)
	pods, _ := pod.List(context.Background(), metav1.ListOptions{})
	for _, item := range pods.Items {
		if item.Name == podname {
			io.WriteString(w, fmt.Sprintf(podHeader, namespace, podname))
			containerCount := len(item.Spec.Containers)
			for idx, container := range item.Spec.Containers {
				if containerCount > 1 {
					io.WriteString(w, fmt.Sprintf(containerHeader, idx+1, containerCount, container.Name))
				}
				err := LogContainer(client, w, namespace, podname, container.Name)
				if err != nil {
					return err
				}
				if containerCount > 1 && idx < containerCount-1 {
					io.WriteString(w, "\n\n")
				}
			}
			io.WriteString(w, "\n")
			break
		}
	}
	return nil
}

func LogPodToString(client kubernetes.Interface, namespace, podname string) (string, error) {
	var writer strings.Builder
	err := LogPod(client, &writer, namespace, podname)
	if err != nil {
		return "", err
	}
	return writer.String(), nil
}
