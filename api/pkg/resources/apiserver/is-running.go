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
		Image:           data.ImageRegistry(resources.RegistryQuay) + "/kubermatic/http-prober:v0.1",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/usr/local/bin/http-prober"},
		Args: []string{
			"-endpoint", fmt.Sprintf("%s/healthz", url.String()),
			"-insecure",
			"-retries", "100",
			"-retry-wait", "2",
			"-timeout", "1",
		},
		TerminationMessagePath:   corev1.TerminationMessagePathDefault,
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
	}, nil
}
