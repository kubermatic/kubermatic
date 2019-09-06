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

// CreateCredentialSecretForCluster creates a new secret for a credential
func CreateCredentialSecretForCluster(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, projectID string) error {
	if cluster.Spec.Cloud.AWS != nil {
		return createAWSSecret(ctx, seedClient, cluster, projectID)
	}
	if cluster.Spec.Cloud.Digitalocean != nil {
		return createDigitaloceanSecret(ctx, seedClient, cluster, projectID)
	}
	if cluster.Spec.Cloud.Hetzner != nil {
		return createHetznerSecret(ctx, seedClient, cluster, projectID)
	}
	if cluster.Spec.Cloud.Packet != nil {
		return createPacketSecret(ctx, seedClient, cluster, projectID)
	}
	if cluster.Spec.Cloud.Kubevirt != nil {
		return createKubevirtSecret(ctx, seedClient, cluster, projectID)
	}
	return nil
}

func createAWSSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, projectID string) error {
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

	if err := seedClient.Create(ctx, secret); err != nil {
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

func createDigitaloceanSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, projectID string) error {
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
			resources.DigitaloceanToken: []byte(cluster.Spec.Cloud.Digitalocean.Token),
		},
	}

	if err := seedClient.Create(ctx, secret); err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Digitalocean.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      secret.Name,
			Namespace: secret.Namespace,
		},
	}

	// remove credentials from cluster object
	cluster.Spec.Cloud.Digitalocean.Token = ""

	return nil
}

func createHetznerSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, projectID string) error {
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
			resources.HetznerToken: []byte(cluster.Spec.Cloud.Hetzner.Token),
		},
	}

	if err := seedClient.Create(ctx, secret); err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Hetzner.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      secret.Name,
			Namespace: secret.Namespace,
		},
	}

	// remove credentials from cluster object
	cluster.Spec.Cloud.Hetzner.Token = ""

	return nil
}

func createPacketSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, projectID string) error {
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

	if err := seedClient.Create(ctx, secret); err != nil {
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

func createKubevirtSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, projectID string) error {
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
			resources.KubevirtKubeConfig: []byte(cluster.Spec.Cloud.Kubevirt.Kubeconfig),
		},
	}

	if err := seedClient.Create(ctx, secret); err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Kubevirt.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      secret.Name,
			Namespace: secret.Namespace,
		},
	}

	// remove credentials from cluster object
	cluster.Spec.Cloud.Kubevirt.Kubeconfig = ""

	return nil
}
