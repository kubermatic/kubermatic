package apiserver

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
)

// IsRunningInitContainer returns a init container which will wait until the apiserver is reachable via its ClusterIP
func IsRunningInitContainer(data *resources.TemplateData) (*corev1.Container, error) {
	// get clusterIP of apiserver
	url, err := data.InClusterApiserverURL()
	if err != nil {
		return nil, fmt.Errorf("failed to get the ClusterIP of the apiserver: %v", err)
	}

	return &corev1.Container{
		Name:            "apiserver-running",
		Image:           data.ImageRegistry(resources.RegistryDocker) + "/busybox",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{
			"/bin/sh",
			"-ec",
			fmt.Sprintf("until wget -T 1 %s/healthz; do echo waiting for apiserver; sleep 2; done;", url),
		},
		TerminationMessagePath:   corev1.TerminationMessagePathDefault,
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
	}, nil
}
