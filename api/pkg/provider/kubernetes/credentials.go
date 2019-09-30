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
	if cluster.Spec.Cloud.Azure != nil {
		return createAzureSecret(ctx, seedClient, cluster, projectID)
	}
	if cluster.Spec.Cloud.Digitalocean != nil {
		return createDigitaloceanSecret(ctx, seedClient, cluster, projectID)
	}
	if cluster.Spec.Cloud.GCP != nil {
		return createGCPSecret(ctx, seedClient, cluster, projectID)
	}
	if cluster.Spec.Cloud.Hetzner != nil {
		return createHetznerSecret(ctx, seedClient, cluster, projectID)
	}
	if cluster.Spec.Cloud.Openstack != nil {
		return createOpenstackSecret(ctx, seedClient, cluster, projectID)
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

func createAzureSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, projectID string) error {
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
			resources.AzureTenantID:       []byte(cluster.Spec.Cloud.Azure.TenantID),
			resources.AzureSubscriptionID: []byte(cluster.Spec.Cloud.Azure.SubscriptionID),
			resources.AzureClientID:       []byte(cluster.Spec.Cloud.Azure.ClientID),
			resources.AzureClientSecret:   []byte(cluster.Spec.Cloud.Azure.ClientSecret),
		},
	}

	if err := seedClient.Create(ctx, secret); err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Azure.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      secret.Name,
			Namespace: secret.Namespace,
		},
	}

	// remove credentials from cluster object
	cluster.Spec.Cloud.Azure.TenantID = ""
	cluster.Spec.Cloud.Azure.SubscriptionID = ""
	cluster.Spec.Cloud.Azure.ClientID = ""
	cluster.Spec.Cloud.Azure.ClientSecret = ""

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

func createGCPSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, projectID string) error {
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
			resources.GCPServiceAccount: []byte(cluster.Spec.Cloud.GCP.ServiceAccount),
		},
	}

	if err := seedClient.Create(ctx, secret); err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.GCP.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      secret.Name,
			Namespace: secret.Namespace,
		},
	}

	// remove credentials from cluster object
	cluster.Spec.Cloud.GCP.ServiceAccount = ""

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

func createOpenstackSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, projectID string) error {
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
			resources.OpenstackUsername: []byte(cluster.Spec.Cloud.Openstack.Username),
			resources.OpenstackPassword: []byte(cluster.Spec.Cloud.Openstack.Password),
			resources.OpenstackTenant:   []byte(cluster.Spec.Cloud.Openstack.Tenant),
			resources.OpenstackTenantID: []byte(cluster.Spec.Cloud.Openstack.TenantID),
			resources.OpenstackDomain:   []byte(cluster.Spec.Cloud.Openstack.Domain),
		},
	}

	if err := seedClient.Create(ctx, secret); err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Openstack.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      secret.Name,
			Namespace: secret.Namespace,
		},
	}

	// remove credentials from cluster object
	cluster.Spec.Cloud.Openstack.Username = ""
	cluster.Spec.Cloud.Openstack.Password = ""
	cluster.Spec.Cloud.Openstack.Tenant = ""
	cluster.Spec.Cloud.Openstack.TenantID = ""
	cluster.Spec.Cloud.Openstack.Domain = ""

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
