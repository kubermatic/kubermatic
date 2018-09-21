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
		Command:         []string{"/bin/sh"},
		Args: []string{
			"-c",
			fmt.Sprintf(`
name=%s
port=%s
# manual re-resolving enables use of trailing dot with curl
ip=$(getent hosts $name | cut -d" " -f1)
while ! curl --resolve $name:$port:$ip --insecure --silent --show-error --max-time 3 %s/healthz; do
    if [ $(( timeout+=1 )) -gt 100 ]; then
        echo "Giving up after 100 tries."
        exit 1
    fi
    sleep 2
    echo "Retry $timeout/100"
    ip=$(getent hosts $name | cut -d" " -f1)
done`, url.Hostname(), url.Port(), url),
		},
		TerminationMessagePath:   corev1.TerminationMessagePathDefault,
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
	}, nil
}
