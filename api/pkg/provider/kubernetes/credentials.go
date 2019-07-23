package kubernetes

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
)

const CredentialPrefix = "credential"

// NewCredentialsProvider returns a credentials provider
func NewCredentialsProvider(kubernetesImpersonationClient kubernetesImpersonationClient, secretLister kubev1.SecretLister) (*CredentialsProvider, error) {
	kubernetesClient, err := kubernetesImpersonationClient(rest.ImpersonationConfig{})
	if err != nil {
		return nil, err
	}

	return &CredentialsProvider{
		secretLister:               secretLister,
		kubernetesClientPrivileged: kubernetesClient,
	}, nil
}

// CredentialsProvider manages secrets for credentials
type CredentialsProvider struct {
	secretLister               kubev1.SecretLister
	kubernetesClientPrivileged kubernetes.Interface
}

// Create creates a new secret for a credential
func (p *CredentialsProvider) Create(cluster *kubermaticv1.Cluster, projectID string) (*corev1.Secret, error) {
	if cluster.Spec.Cloud.Packet != nil {
		return p.createPacketSecret(cluster, projectID)
	}
	return nil, nil
}

func (p *CredentialsProvider) createPacketSecret(cluster *kubermaticv1.Cluster, projectID string) (*corev1.Secret, error) {
	// create secret for storing credentials
	name := fmt.Sprintf("%s-packet-%s", CredentialPrefix, cluster.Name)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: resources.KubermaticNamespace,
			Labels: map[string]string{
				kubermaticv1.ProjectIDLabelKey: projectID,
				"name":                         name,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"apiKey":    []byte(cluster.Spec.Cloud.Packet.APIKey),
			"projectID": []byte(cluster.Spec.Cloud.Packet.ProjectID),
		},
	}

	createdSecret, err := p.kubernetesClientPrivileged.CoreV1().Secrets(resources.KubermaticNamespace).Create(secret)
	if err != nil {
		return nil, err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Packet.APIKeyReference = &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      secret.Name,
			Namespace: secret.Namespace,
		},
		Key: "apiKey",
	}

	cluster.Spec.Cloud.Packet.ProjectIDReference = &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      secret.Name,
			Namespace: secret.Namespace,
		},
		Key: "projectID",
	}

	// remove credentials from cluster object
	cluster.Spec.Cloud.Packet.APIKey = ""
	cluster.Spec.Cloud.Packet.ProjectID = ""

	return createdSecret, nil
}
