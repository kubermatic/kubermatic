package controller

import (
	"fmt"
	"time"

	utilrand "k8s.io/apimachinery/pkg/util/rand"
)

const (
	UserClusterAPIServerServiceName         = "apiserver"
	UserClusterAPIServerServiceSuffixLength = 6

	// Amount of time to wait until at least one pod is running
	DefaultPodPortForwardWaitTimeout = 60 * time.Second
)

func GenerateName(base, buildID string) string {
	return fmt.Sprintf("%s-%s-%s", base, buildID, utilrand.String(UserClusterAPIServerServiceSuffixLength))
}
