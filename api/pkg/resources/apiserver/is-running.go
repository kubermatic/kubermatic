package apiserver

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
)

// IsRunningInitContainer returns a init container which will wait until the apiserver is reachable via its ClusterIP
type isRunningInitContainerData interface {
	ImageRegistry(string) string
	Cluster() *kubermaticv1.Cluster
}

func IsRunningInitContainer(data isRunningInitContainerData) (*corev1.Container, error) {
	// get clusterIP of apiserver
	return &corev1.Container{
		Name:            "apiserver-running",
		Image:           data.ImageRegistry(resources.RegistryQuay) + "/kubermatic/http-prober:v0.1",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/usr/local/bin/http-prober"},
		Args: []string{
			"-endpoint", fmt.Sprintf("https://%s:%d/healthz", data.Cluster().Address.InternalName, data.Cluster().Address.Port),
			"-insecure",
			"-retries", "100",
			"-retry-wait", "2",
			"-timeout", "1",
		},
		TerminationMessagePath:   corev1.TerminationMessagePathDefault,
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
	}, nil
}
