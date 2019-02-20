package resources

import (
	"context"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
)

// Openshift data contains all data required for Openshift control plane components
// It should be as small as possible
type openshiftData interface {
	Cluster(context.Context) *kubermaticv1.Cluster
	GetPodTemplateLabels(context.Context, string, []corev1.Volume, map[string]string) (map[string]string, error)
	GetApiserverExternalNodePort(context.Context) (int32, error)
	//TODO: Add ImageRegistry(string) string
	NodePortRange(context.Context) string
}
