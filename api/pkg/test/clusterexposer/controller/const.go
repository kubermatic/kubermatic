package controller

import (
	"fmt"
	"time"

	utilrand "k8s.io/apimachinery/pkg/util/rand"
)

const (
	UserClusterAPIServerServiceName         = "apiserver-external"
	UserClusterAPIServerServiceSuffixLength = 6

	// Amount of time to wait until at least one pod is running
	DefaultPodPortForwardWaitTimeout = 60 * time.Second
)

var (
	// Service annotations that will be added to the expose service of the outer cluster.
	defaultServiceAnnotations = map[string]string{"nodeport-proxy.k8s.io/expose": "true"}
)

func GenerateName(base, buildID string) string {
	return fmt.Sprintf("%s-%s-%s", base, buildID, utilrand.String(UserClusterAPIServerServiceSuffixLength))
}
