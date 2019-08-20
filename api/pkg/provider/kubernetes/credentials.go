package kubernetes

import (
	"context"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewCredentialsProvider returns a credentials provider
func NewCredentialsProvider(ctx context.Context, client ctrlruntimeclient.Client) *CredentialsProvider {
	return &CredentialsProvider{
		ctx:    ctx,
		client: client,
	}
}

// CredentialsProvider manages secrets for credentials
type CredentialsProvider struct {
	ctx    context.Context
	client ctrlruntimeclient.Client
}

// Create creates a new secret for a credential
func (p *CredentialsProvider) Create(cluster *kubermaticv1.Cluster, projectID string) error {
	if cluster.Spec.Cloud.AWS != nil {
		return p.createAWSSecret(cluster, projectID)
	}
	if cluster.Spec.Cloud.Packet != nil {
		return p.createPacketSecret(cluster, projectID)
	}
	return nil
}

func (p *CredentialsProvider) createAWSSecret(cluster *kubermaticv1.Cluster, projectID string) error {
	// create secret for storing credentials
	name := cluster.GetSecretName()
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
			resources.AWSAccessKeyID:     []byte(cluster.Spec.Cloud.AWS.AccessKeyID),
			resources.AWSSecretAccessKey: []byte(cluster.Spec.Cloud.AWS.SecretAccessKey),
		},
	}

	if err := p.client.Create(p.ctx, secret); err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.AWS.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      secret.Name,
			Namespace: secret.Namespace,
		},
	}

	// remove credentials from cluster object
	cluster.Spec.Cloud.AWS.AccessKeyID = ""
	cluster.Spec.Cloud.AWS.SecretAccessKey = ""

	return nil
}

func (p *CredentialsProvider) createPacketSecret(cluster *kubermaticv1.Cluster, projectID string) error {
	// create secret for storing credentials
	name := cluster.GetSecretName()
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
			resources.PacketAPIKey:    []byte(cluster.Spec.Cloud.Packet.APIKey),
			resources.PacketProjectID: []byte(cluster.Spec.Cloud.Packet.ProjectID),
		},
	}

	if err := p.client.Create(p.ctx, secret); err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Packet.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      secret.Name,
			Namespace: secret.Namespace,
		},
	}

	// remove credentials from cluster object
	cluster.Spec.Cloud.Packet.APIKey = ""
	cluster.Spec.Cloud.Packet.ProjectID = ""

	return nil
}
