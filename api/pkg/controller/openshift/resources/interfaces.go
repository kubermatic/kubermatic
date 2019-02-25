package resources

import (
	"context"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ServiceSignerCASecretName = "service-signer-ca"
)

// Openshift data contains all data required for Openshift control plane components
// It should be as small as possible
type openshiftData interface {
	Cluster() *kubermaticv1.Cluster
	GetPodTemplateLabels(context.Context, string, []corev1.Volume, map[string]string) (map[string]string, error)
	GetApiserverExternalNodePort(context.Context) (int32, error)
	//TODO: Add ImageRegistry(string) string
	NodePortRange(context.Context) string
	ClusterIPByServiceName(name string) (string, error)
	ImageRegistry(string) string
	NodeAccessNetwork() string
	GetClusterRef() metav1.OwnerReference
}

type SecretCreator = func(openshiftData, *corev1.Secret) (*corev1.Secret, error)

type NamedSecretCreator func(context.Context, openshiftData) (string, SecretCreator)
type NamedConfigMapCreator func(context.Context, openshiftData) (string, resources.ConfigMapCreator)
type NamedDeploymentCreator func(context.Context, openshiftData) (string, resources.DeploymentCreator)
