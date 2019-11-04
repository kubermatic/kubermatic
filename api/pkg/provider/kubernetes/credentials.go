package kubernetes

import (
	"context"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateCredentialSecretForCluster creates a new secret for a credential
func CreateOrUpdateCredentialSecretForCluster(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	if cluster.Spec.Cloud.AWS != nil {
		return createOrUpdateAWSSecret(ctx, seedClient, cluster)
	}
	if cluster.Spec.Cloud.Azure != nil {
		return createOrUpdateAzureSecret(ctx, seedClient, cluster)
	}
	if cluster.Spec.Cloud.Digitalocean != nil {
		return createOrUpdateDigitaloceanSecret(ctx, seedClient, cluster)
	}
	if cluster.Spec.Cloud.GCP != nil {
		return createOrUpdateGCPSecret(ctx, seedClient, cluster)
	}
	if cluster.Spec.Cloud.Hetzner != nil {
		return createOrUpdateHetznerSecret(ctx, seedClient, cluster)
	}
	if cluster.Spec.Cloud.Openstack != nil {
		return createOrUpdateOpenstackSecret(ctx, seedClient, cluster)
	}
	if cluster.Spec.Cloud.Packet != nil {
		return createOrUpdatePacketSecret(ctx, seedClient, cluster)
	}
	if cluster.Spec.Cloud.Kubevirt != nil {
		return createOrUpdateKubevirtSecret(ctx, seedClient, cluster)
	}
	if cluster.Spec.Cloud.VSphere != nil {
		return createVSphereSecret(ctx, seedClient, cluster)
	}
	if cluster.Spec.Cloud.Alibaba != nil {
		return createAlibabaSecret(ctx, seedClient, cluster, projectID)
	}
	return nil
}

func createOrUpdateAWSSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	spec := cluster.Spec.Cloud.AWS
	referenceDefined := spec.CredentialsReference != nil && spec.CredentialsReference.Name != ""
	valuesDefined := spec.AccessKeyID != "" || spec.SecretAccessKey != ""

	if !referenceDefined && valuesDefined {
		name := cluster.GetSecretName()
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: resources.KubermaticNamespace,
				Labels: map[string]string{
					"name": name,
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				resources.AWSAccessKeyID:     []byte(spec.AccessKeyID),
				resources.AWSSecretAccessKey: []byte(spec.SecretAccessKey),
			},
		}

		if err := seedClient.Create(ctx, secret); err != nil {
			return err
		}

		// add secret key selectors to cluster object
		cluster.Spec.Cloud.AWS.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
			ObjectReference: corev1.ObjectReference{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
		}
	}

	if referenceDefined && valuesDefined {
		namespacedName := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: cluster.GetSecretName()}
		existingSecret := &corev1.Secret{}
		if err := seedClient.Get(ctx, namespacedName, existingSecret); err != nil {
			return err
		}

		var updateNeeded bool
		if existingSecret.Data != nil {
			if accessKeyID, ok := existingSecret.Data[resources.AWSAccessKeyID]; ok {
				if string(accessKeyID) != spec.AccessKeyID {
					existingSecret.Data[resources.AWSAccessKeyID] = []byte(spec.AccessKeyID)
					updateNeeded = true
				}
			}
			if secretAccessKey, ok := existingSecret.Data[resources.AWSSecretAccessKey]; ok {
				if string(secretAccessKey) != spec.SecretAccessKey {
					existingSecret.Data[resources.AWSSecretAccessKey] = []byte(spec.SecretAccessKey)
					updateNeeded = true
				}
			}
		}
		if updateNeeded {
			if err := seedClient.Update(ctx, existingSecret); err != nil {
				return err
			}
		}
	}

	if valuesDefined {
		cluster.Spec.Cloud.AWS.AccessKeyID = ""
		cluster.Spec.Cloud.AWS.SecretAccessKey = ""
	}

	return nil
}

func createOrUpdateAzureSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	spec := cluster.Spec.Cloud.Azure
	referenceDefined := spec.CredentialsReference != nil && spec.CredentialsReference.Name != ""
	valuesDefined := spec.TenantID != "" || spec.SubscriptionID != "" || spec.ClientID != "" || spec.ClientSecret != ""

	if !referenceDefined && valuesDefined {
		name := cluster.GetSecretName()
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: resources.KubermaticNamespace,
				Labels: map[string]string{
					"name": name,
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				resources.AzureTenantID:       []byte(spec.TenantID),
				resources.AzureSubscriptionID: []byte(spec.SubscriptionID),
				resources.AzureClientID:       []byte(spec.ClientID),
				resources.AzureClientSecret:   []byte(spec.ClientSecret),
			},
		}

		if err := seedClient.Create(ctx, secret); err != nil {
			return err
		}

		// add secret key selectors to cluster object
		cluster.Spec.Cloud.Azure.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
			ObjectReference: corev1.ObjectReference{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
		}
	}

	if referenceDefined && valuesDefined {
		namespacedName := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: cluster.GetSecretName()}
		existingSecret := &corev1.Secret{}
		if err := seedClient.Get(ctx, namespacedName, existingSecret); err != nil {
			return err
		}

		var updateNeeded bool
		if existingSecret.Data != nil {
			if tenantID, ok := existingSecret.Data[resources.AzureTenantID]; ok {
				if string(tenantID) != spec.TenantID {
					existingSecret.Data[resources.AzureTenantID] = []byte(spec.TenantID)
					updateNeeded = true
				}
			}
			if subscriptionID, ok := existingSecret.Data[resources.AzureSubscriptionID]; ok {
				if string(subscriptionID) != spec.SubscriptionID {
					existingSecret.Data[resources.AzureSubscriptionID] = []byte(spec.SubscriptionID)
					updateNeeded = true
				}
			}
			if clientID, ok := existingSecret.Data[resources.AzureClientID]; ok {
				if string(clientID) != spec.ClientID {
					existingSecret.Data[resources.AzureClientID] = []byte(spec.ClientID)
					updateNeeded = true
				}
			}
			if clientSecret, ok := existingSecret.Data[resources.AzureClientSecret]; ok {
				if string(clientSecret) != spec.ClientSecret {
					existingSecret.Data[resources.AzureClientSecret] = []byte(spec.ClientSecret)
					updateNeeded = true
				}
			}
		}
		if updateNeeded {
			if err := seedClient.Update(ctx, existingSecret); err != nil {
				return err
			}
		}
	}

	if valuesDefined {
		cluster.Spec.Cloud.Azure.TenantID = ""
		cluster.Spec.Cloud.Azure.SubscriptionID = ""
		cluster.Spec.Cloud.Azure.ClientID = ""
		cluster.Spec.Cloud.Azure.ClientSecret = ""
	}

	return nil
}

func createOrUpdateDigitaloceanSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	spec := cluster.Spec.Cloud.Digitalocean
	referenceDefined := spec.CredentialsReference != nil && spec.CredentialsReference.Name != ""
	valuesDefined := spec.Token != ""

	if !referenceDefined && valuesDefined {
		name := cluster.GetSecretName()
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: resources.KubermaticNamespace,
				Labels: map[string]string{
					"name": name,
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				resources.DigitaloceanToken: []byte(spec.Token),
			},
		}

		if err := seedClient.Create(ctx, secret); err != nil {
			return err
		}

		// add secret key selectors to cluster object
		cluster.Spec.Cloud.Digitalocean.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
			ObjectReference: corev1.ObjectReference{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
		}
	}

	if referenceDefined && valuesDefined {
		namespacedName := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: cluster.GetSecretName()}
		existingSecret := &corev1.Secret{}
		if err := seedClient.Get(ctx, namespacedName, existingSecret); err != nil {
			return err
		}

		var updateNeeded bool
		if existingSecret.Data != nil {
			if token, ok := existingSecret.Data[resources.DigitaloceanToken]; ok {
				if string(token) != spec.Token {
					existingSecret.Data[resources.DigitaloceanToken] = []byte(spec.Token)
					updateNeeded = true
				}
			}
		}

		if updateNeeded {
			if err := seedClient.Update(ctx, existingSecret); err != nil {
				return err
			}
		}
	}

	if valuesDefined {
		cluster.Spec.Cloud.Digitalocean.Token = ""
	}

	return nil
}

func createOrUpdateGCPSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	spec := cluster.Spec.Cloud.GCP
	referenceDefined := spec.CredentialsReference != nil && spec.CredentialsReference.Name != ""
	valuesDefined := spec.ServiceAccount != ""

	if !referenceDefined && valuesDefined {
		name := cluster.GetSecretName()
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: resources.KubermaticNamespace,
				Labels: map[string]string{
					"name": name,
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				resources.GCPServiceAccount: []byte(spec.ServiceAccount),
			},
		}

		if err := seedClient.Create(ctx, secret); err != nil {
			return err
		}

		cluster.Spec.Cloud.GCP.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
			ObjectReference: corev1.ObjectReference{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
		}
	}

	if referenceDefined && valuesDefined {
		namespacedName := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: cluster.GetSecretName()}
		existingSecret := &corev1.Secret{}
		if err := seedClient.Get(ctx, namespacedName, existingSecret); err != nil {
			return err
		}

		var updateNeeded bool
		if existingSecret.Data != nil {
			if sa, ok := existingSecret.Data[resources.GCPServiceAccount]; ok {
				if string(sa) != spec.ServiceAccount {
					existingSecret.Data[resources.GCPServiceAccount] = []byte(spec.ServiceAccount)
					updateNeeded = true
				}
			}
		}

		if updateNeeded {
			if err := seedClient.Update(ctx, existingSecret); err != nil {
				return err
			}
		}
	}

	if valuesDefined {
		cluster.Spec.Cloud.GCP.ServiceAccount = ""
	}

	return nil
}

func createOrUpdateHetznerSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	spec := cluster.Spec.Cloud.Hetzner
	referenceDefined := spec.CredentialsReference != nil && spec.CredentialsReference.Name != ""
	valuesDefined := spec.Token != ""

	if !referenceDefined && valuesDefined {
		name := cluster.GetSecretName()
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: resources.KubermaticNamespace,
				Labels: map[string]string{
					"name": name,
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				resources.HetznerToken: []byte(spec.Token),
			},
		}

		if err := seedClient.Create(ctx, secret); err != nil {
			return err
		}

		cluster.Spec.Cloud.Hetzner.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
			ObjectReference: corev1.ObjectReference{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
		}
	}

	if referenceDefined && valuesDefined {
		namespacedName := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: cluster.GetSecretName()}
		existingSecret := &corev1.Secret{}
		if err := seedClient.Get(ctx, namespacedName, existingSecret); err != nil {
			return err
		}

		var updateNeeded bool
		if existingSecret.Data != nil {
			if token, ok := existingSecret.Data[resources.HetznerToken]; ok {
				if string(token) != spec.Token {
					existingSecret.Data[resources.HetznerToken] = []byte(spec.Token)
					updateNeeded = true
				}
			}
		}

		if updateNeeded {
			if err := seedClient.Update(ctx, existingSecret); err != nil {
				return err
			}
		}
	}

	if valuesDefined {
		cluster.Spec.Cloud.Hetzner.Token = ""
	}

	return nil
}

func createOrUpdateOpenstackSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	spec := cluster.Spec.Cloud.Openstack
	referenceDefined := spec.CredentialsReference != nil && spec.CredentialsReference.Name != ""
	valuesDefined := spec.Username != "" || spec.Password != "" || spec.Tenant != "" || spec.TenantID != "" || spec.Domain != ""

	if !referenceDefined && valuesDefined {
		name := cluster.GetSecretName()
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: resources.KubermaticNamespace,
				Labels: map[string]string{
					"name": name,
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				resources.OpenstackUsername: []byte(spec.Username),
				resources.OpenstackPassword: []byte(spec.Password),
				resources.OpenstackTenant:   []byte(spec.Tenant),
				resources.OpenstackTenantID: []byte(spec.TenantID),
				resources.OpenstackDomain:   []byte(spec.Domain),
			},
		}

		if err := seedClient.Create(ctx, secret); err != nil {
			return err
		}

		// add secret key selectors to cluster object
		cluster.Spec.Cloud.Openstack.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
			ObjectReference: corev1.ObjectReference{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
		}
	}

	if referenceDefined && valuesDefined {
		namespacedName := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: cluster.GetSecretName()}
		existingSecret := &corev1.Secret{}
		if err := seedClient.Get(ctx, namespacedName, existingSecret); err != nil {
			return err
		}

		var updateNeeded bool
		if existingSecret.Data != nil {
			if username, ok := existingSecret.Data[resources.OpenstackUsername]; ok {
				if string(username) != spec.Username {
					existingSecret.Data[resources.OpenstackUsername] = []byte(spec.Username)
					updateNeeded = true
				}
			}
			if password, ok := existingSecret.Data[resources.OpenstackPassword]; ok {
				if string(password) != spec.Password {
					existingSecret.Data[resources.OpenstackPassword] = []byte(spec.Password)
					updateNeeded = true
				}
			}
			if tenant, ok := existingSecret.Data[resources.OpenstackTenant]; ok {
				if string(tenant) != spec.Tenant {
					existingSecret.Data[resources.OpenstackTenant] = []byte(spec.Tenant)
					updateNeeded = true
				}
			}
			if tenantID, ok := existingSecret.Data[resources.OpenstackTenantID]; ok {
				if string(tenantID) != spec.TenantID {
					existingSecret.Data[resources.OpenstackTenantID] = []byte(spec.TenantID)
					updateNeeded = true
				}
			}
			if domain, ok := existingSecret.Data[resources.OpenstackDomain]; ok {
				if string(domain) != spec.Domain {
					existingSecret.Data[resources.OpenstackDomain] = []byte(spec.Domain)
					updateNeeded = true
				}
			}
		}
		if updateNeeded {
			if err := seedClient.Update(ctx, existingSecret); err != nil {
				return err
			}
		}
	}

	if valuesDefined {
		cluster.Spec.Cloud.Openstack.Username = ""
		cluster.Spec.Cloud.Openstack.Password = ""
		cluster.Spec.Cloud.Openstack.Tenant = ""
		cluster.Spec.Cloud.Openstack.TenantID = ""
		cluster.Spec.Cloud.Openstack.Domain = ""
	}

	return nil
}

func createOrUpdatePacketSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	spec := cluster.Spec.Cloud.Packet
	referenceDefined := spec.CredentialsReference != nil && spec.CredentialsReference.Name != ""
	valuesDefined := spec.APIKey != "" || spec.ProjectID != ""

	if !referenceDefined && valuesDefined {
		name := cluster.GetSecretName()
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: resources.KubermaticNamespace,
				Labels: map[string]string{
					"name": name,
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				resources.PacketAPIKey:    []byte(spec.APIKey),
				resources.PacketProjectID: []byte(spec.ProjectID),
			},
		}

		if err := seedClient.Create(ctx, secret); err != nil {
			return err
		}

		// add secret key selectors to cluster object
		cluster.Spec.Cloud.Packet.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
			ObjectReference: corev1.ObjectReference{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
		}
	}

	if referenceDefined && valuesDefined {
		namespacedName := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: cluster.GetSecretName()}
		existingSecret := &corev1.Secret{}
		if err := seedClient.Get(ctx, namespacedName, existingSecret); err != nil {
			return err
		}

		var updateNeeded bool
		if existingSecret.Data != nil {
			if apiKey, ok := existingSecret.Data[resources.PacketAPIKey]; ok {
				if string(apiKey) != spec.APIKey {
					existingSecret.Data[resources.PacketAPIKey] = []byte(spec.APIKey)
					updateNeeded = true
				}
			}
			if projectID, ok := existingSecret.Data[resources.PacketProjectID]; ok {
				if string(projectID) != spec.ProjectID {
					existingSecret.Data[resources.PacketProjectID] = []byte(spec.ProjectID)
					updateNeeded = true
				}
			}
		}
		if updateNeeded {
			if err := seedClient.Update(ctx, existingSecret); err != nil {
				return err
			}
		}
	}

	if valuesDefined {
		cluster.Spec.Cloud.Packet.APIKey = ""
		cluster.Spec.Cloud.Packet.ProjectID = ""
	}

	return nil
}

func createOrUpdateKubevirtSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	spec := cluster.Spec.Cloud.Kubevirt
	referenceDefined := spec.CredentialsReference != nil && spec.CredentialsReference.Name != ""
	valuesDefined := spec.Kubeconfig != ""

	if !referenceDefined && valuesDefined {
		name := cluster.GetSecretName()
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: resources.KubermaticNamespace,
				Labels: map[string]string{
					"name": name,
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				resources.KubevirtKubeConfig: []byte(spec.Kubeconfig),
			},
		}

		if err := seedClient.Create(ctx, secret); err != nil {
			return err
		}

		// add secret key selectors to cluster object
		cluster.Spec.Cloud.Kubevirt.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
			ObjectReference: corev1.ObjectReference{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
		}
	}

	if referenceDefined && valuesDefined {
		namespacedName := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: cluster.GetSecretName()}
		existingSecret := &corev1.Secret{}
		if err := seedClient.Get(ctx, namespacedName, existingSecret); err != nil {
			return err
		}

		var updateNeeded bool
		if existingSecret.Data != nil {
			if kubeconfig, ok := existingSecret.Data[resources.KubevirtKubeConfig]; ok {
				if string(kubeconfig) != spec.Kubeconfig {
					existingSecret.Data[resources.KubevirtKubeConfig] = []byte(spec.Kubeconfig)
					updateNeeded = true
				}
			}
		}
		if updateNeeded {
			if err := seedClient.Update(ctx, existingSecret); err != nil {
				return err
			}
		}
	}

	if valuesDefined {
		cluster.Spec.Cloud.Kubevirt.Kubeconfig = ""
	}

	return nil
}

func createVSphereSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	spec := cluster.Spec.Cloud.VSphere
	name := cluster.GetSecretName()
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: resources.KubermaticNamespace,
			Labels: map[string]string{
				"name": name,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			resources.VsphereUsername:                    []byte(spec.Username),
			resources.VspherePassword:                    []byte(spec.Password),
			resources.VsphereInfraManagementUserUsername: []byte(spec.InfraManagementUser.Username),
			resources.VsphereInfraManagementUserPassword: []byte(spec.InfraManagementUser.Password),
		},
	}

	if err := seedClient.Create(ctx, secret); err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.VSphere.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      secret.Name,
			Namespace: secret.Namespace,
		},
	}

	// remove credentials from cluster object
	cluster.Spec.Cloud.VSphere.Username = ""
	cluster.Spec.Cloud.VSphere.Password = ""
	cluster.Spec.Cloud.VSphere.InfraManagementUser.Username = ""
	cluster.Spec.Cloud.VSphere.InfraManagementUser.Password = ""

	return nil
}

func createAlibabaSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, projectID string) error {
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
			resources.AlibabaAccessKeyID:     []byte(cluster.Spec.Cloud.Alibaba.AccessKeyID),
			resources.AlibabaAccessKeySecret: []byte(cluster.Spec.Cloud.Alibaba.AccessKeySecret),
		},
	}

	if err := seedClient.Create(ctx, secret); err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Alibaba.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      secret.Name,
			Namespace: secret.Namespace,
		},
	}

	// remove credentials from cluster object
	cluster.Spec.Cloud.Alibaba.AccessKeyID = ""
	cluster.Spec.Cloud.Alibaba.AccessKeySecret = ""

	return nil
}
