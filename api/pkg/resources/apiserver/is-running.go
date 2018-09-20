package apiserver

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
)

// IsRunningInitContainer returns a init container which will wait until the apiserver is reachable via its ClusterIP
func IsRunningInitContainer(data resources.DeploymentDataProvider) (*corev1.Container, error) {
	// get clusterIP of apiserver
	url, err := data.InClusterApiserverURL()
	if err != nil {
		return nil, fmt.Errorf("failed to get the ClusterIP of the apiserver: %v", err)
	}

	return &corev1.Container{
		Name:            "apiserver-running",
		Image:           data.ImageRegistry(resources.RegistryQuay) + "/kubermatic/curl:v0.1",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/usr/bin/curl"},
		Args: []string{
			"--retry", "100",
			"--retry-delay", "2",
			"--insecure",
			"--silent",
			"--show-error",
			fmt.Sprintf("%s/healthz", url),
		},
		TerminationMessagePath:   corev1.TerminationMessagePathDefault,
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
	}, nil
}
