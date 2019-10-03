package migrations

import (
	"fmt"
	"github.com/golang/glog"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createSecretsForCredentials(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	var err error
	if cluster.Spec.Cloud.AWS != nil {
		err = createAWSSecret(cluster, ctx)
	}
	if cluster.Spec.Cloud.Azure != nil {
		err = createAzureSecret(cluster, ctx)
	}
	if cluster.Spec.Cloud.Digitalocean != nil {
		err = createDigitaloceanSecret(cluster, ctx)
	}
	if cluster.Spec.Cloud.GCP != nil {
		err = createGCPSecret(cluster, ctx)
	}
	if cluster.Spec.Cloud.Hetzner != nil {
		err = createHetznerSecret(cluster, ctx)
	}
	if cluster.Spec.Cloud.Openstack != nil {
		err = createOpenstackSecret(cluster, ctx)
	}
	if cluster.Spec.Cloud.Packet != nil {
		err = createPacketSecret(cluster, ctx)
	}
	if cluster.Spec.Cloud.Kubevirt != nil {
		err = createKubevirtSecret(cluster, ctx)
	}
	if err != nil {
		return err
	}

	kuberneteshelper.AddFinalizer(cluster, apiv1.CredentialsSecretsCleanupFinalizer)
	updatedCluster, err := ctx.kubermaticClient.KubermaticV1().Clusters().Update(cluster)
	if err != nil {
		return fmt.Errorf("failed to update cluster %q: %v", cluster.Name, err)
	}
	*cluster = *updatedCluster
	return nil
}

func createAWSSecret(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	spec := cluster.Spec.Cloud.AWS
	referenceDefined := spec.CredentialsReference != nil && spec.CredentialsReference.Name != ""
	valuesDefined := spec.AccessKeyID != "" || spec.SecretAccessKey != ""

	var secret *corev1.Secret
	if !referenceDefined && valuesDefined {
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				resources.AWSAccessKeyID:     []byte(spec.AccessKeyID),
				resources.AWSSecretAccessKey: []byte(spec.SecretAccessKey),
			},
		}

		if _, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Create(secret); err != nil {
			return err
		}
		glog.Infof("the secret = %s was created for cluster = %s resource", cluster.GetSecretName(), cluster.Name)
	}

	if !referenceDefined {
		cluster.Spec.Cloud.AWS.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
			ObjectReference: corev1.ObjectReference{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
		}
	}

	// treat the value provided as the sourch of truth and update the secret with this value
	if referenceDefined && valuesDefined {
		var updateNeeded bool
		existingSecret, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Get(cluster.GetSecretName(), metav1.GetOptions{})
		if err != nil {
			return err
		}
		if existingSecret.Data == nil {
			existingSecret.Data = map[string][]byte{}
		}

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
			if _, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Update(existingSecret); err != nil {
				return fmt.Errorf("error updating secret: %v", err)
			}
		}
	}

	if valuesDefined {
		cluster.Spec.Cloud.AWS.AccessKeyID = ""
		cluster.Spec.Cloud.AWS.SecretAccessKey = ""
	}

	return nil
}

func createAzureSecret(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	spec := cluster.Spec.Cloud.Azure
	referenceDefined := spec.CredentialsReference != nil && spec.CredentialsReference.Name != ""
	valuesDefined := spec.TenantID != "" || spec.SubscriptionID != "" || spec.ClientID != "" || spec.ClientSecret != ""

	var secret *corev1.Secret
	if !referenceDefined && valuesDefined {
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				resources.AzureTenantID:       []byte(spec.TenantID),
				resources.AzureSubscriptionID: []byte(spec.SubscriptionID),
				resources.AzureClientID:       []byte(spec.ClientID),
				resources.AzureClientSecret:   []byte(spec.ClientSecret),
			},
		}

		if _, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Create(secret); err != nil {
			return err
		}
		glog.Infof("the secret = %s was created for cluster = %s resource", cluster.GetSecretName(), cluster.Name)
	}

	if !referenceDefined {
		cluster.Spec.Cloud.Azure.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
			ObjectReference: corev1.ObjectReference{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
		}
	}

	// treat the value provided as the sourch of truth and update the secret with this value
	if referenceDefined && valuesDefined {
		var updateNeeded bool
		existingSecret, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Get(cluster.GetSecretName(), metav1.GetOptions{})
		if err != nil {
			return err
		}
		if existingSecret.Data == nil {
			existingSecret.Data = map[string][]byte{}
		}

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
			if _, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Update(existingSecret); err != nil {
				return fmt.Errorf("error updating secret: %v", err)
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

func createDigitaloceanSecret(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	spec := cluster.Spec.Cloud.Digitalocean
	referenceDefined := spec.CredentialsReference != nil && spec.CredentialsReference.Name != ""
	valuesDefined := spec.Token != ""

	var secret *corev1.Secret
	if !referenceDefined && valuesDefined {
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				resources.DigitaloceanToken: []byte(cluster.Spec.Cloud.Digitalocean.Token),
			},
		}

		if _, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Create(secret); err != nil {
			return err
		}
		glog.Infof("the secret = %s was created for cluster = %s resource", cluster.GetSecretName(), cluster.Name)
	}

	if !referenceDefined {
		cluster.Spec.Cloud.Digitalocean.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
			ObjectReference: corev1.ObjectReference{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
		}
	}

	// treat the value provided as the sourch of truth and update the secret with this value
	if referenceDefined && valuesDefined {
		var updateNeeded bool
		existingSecret, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Get(cluster.GetSecretName(), metav1.GetOptions{})
		if err != nil {
			return err
		}
		if existingSecret.Data == nil {
			existingSecret.Data = map[string][]byte{}
		}

		if existingSecret.Data != nil {
			if token, ok := existingSecret.Data[resources.DigitaloceanToken]; ok {
				if string(token) != spec.Token {
					existingSecret.Data[resources.DigitaloceanToken] = []byte(spec.Token)
					updateNeeded = true
				}
			}
		}
		if updateNeeded {
			if _, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Update(existingSecret); err != nil {
				return fmt.Errorf("error updating secret: %v", err)
			}
		}
	}

	if valuesDefined {
		cluster.Spec.Cloud.Digitalocean.Token = ""
	}

	return nil
}

func createGCPSecret(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	spec := cluster.Spec.Cloud.GCP
	referenceDefined := spec.CredentialsReference != nil && spec.CredentialsReference.Name != ""
	valuesDefined := spec.ServiceAccount != ""

	var secret *corev1.Secret
	if !referenceDefined && valuesDefined {
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				resources.GCPServiceAccount: []byte(spec.ServiceAccount),
			},
		}

		if _, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Create(secret); err != nil {
			return err
		}
		glog.Infof("the secret = %s was created for cluster = %s resource", cluster.GetSecretName(), cluster.Name)
	}

	if !referenceDefined {
		cluster.Spec.Cloud.GCP.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
			ObjectReference: corev1.ObjectReference{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
		}
	}

	// treat the value provided as the sourch of truth and update the secret with this value
	if referenceDefined && valuesDefined {
		var updateNeeded bool
		existingSecret, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Get(cluster.GetSecretName(), metav1.GetOptions{})
		if err != nil {
			return err
		}
		if existingSecret.Data == nil {
			existingSecret.Data = map[string][]byte{}
		}

		if existingSecret.Data != nil {
			if sa, ok := existingSecret.Data[resources.GCPServiceAccount]; ok {
				if string(sa) != spec.ServiceAccount {
					existingSecret.Data[resources.GCPServiceAccount] = []byte(spec.ServiceAccount)
					updateNeeded = true
				}
			}
		}
		if updateNeeded {
			if _, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Update(existingSecret); err != nil {
				return fmt.Errorf("error updating secret: %v", err)
			}
		}
	}

	if valuesDefined {
		cluster.Spec.Cloud.GCP.ServiceAccount = ""
	}

	return nil
}

func createHetznerSecret(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	spec := cluster.Spec.Cloud.Hetzner
	referenceDefined := spec.CredentialsReference != nil && spec.CredentialsReference.Name != ""
	valuesDefined := spec.Token != ""

	var secret *corev1.Secret
	if !referenceDefined && valuesDefined {
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				resources.HetznerToken: []byte(cluster.Spec.Cloud.Hetzner.Token),
			},
		}

		if _, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Create(secret); err != nil {
			return err
		}
		glog.Infof("the secret = %s was created for cluster = %s resource", cluster.GetSecretName(), cluster.Name)
	}

	if !referenceDefined {
		cluster.Spec.Cloud.Hetzner.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
			ObjectReference: corev1.ObjectReference{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
		}
	}

	// treat the value provided as the sourch of truth and update the secret with this value
	if referenceDefined && valuesDefined {
		var updateNeeded bool
		existingSecret, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Get(cluster.GetSecretName(), metav1.GetOptions{})
		if err != nil {
			return err
		}
		if existingSecret.Data == nil {
			existingSecret.Data = map[string][]byte{}
		}

		if existingSecret.Data != nil {
			if token, ok := existingSecret.Data[resources.HetznerToken]; ok {
				if string(token) != spec.Token {
					existingSecret.Data[resources.HetznerToken] = []byte(spec.Token)
					updateNeeded = true
				}
			}
		}
		if updateNeeded {
			if _, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Update(existingSecret); err != nil {
				return fmt.Errorf("error updating secret: %v", err)
			}
		}
	}

	if valuesDefined {
		cluster.Spec.Cloud.Hetzner.Token = ""
	}

	return nil
}

func createOpenstackSecret(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	spec := cluster.Spec.Cloud.Openstack
	referenceDefined := spec.CredentialsReference != nil && spec.CredentialsReference.Name != ""
	valuesDefined := spec.Username != "" || spec.Password != "" || spec.Tenant != "" || spec.TenantID != "" || spec.Domain != ""

	var secret *corev1.Secret
	if !referenceDefined && valuesDefined {
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
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

		if _, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Create(secret); err != nil {
			return err
		}
		glog.Infof("the secret = %s was created for cluster = %s resource", cluster.GetSecretName(), cluster.Name)
	}

	if !referenceDefined {
		cluster.Spec.Cloud.Openstack.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
			ObjectReference: corev1.ObjectReference{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
		}
	}

	// treat the value provided as the sourch of truth and update the secret with this value
	if referenceDefined && valuesDefined {
		var updateNeeded bool
		existingSecret, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Get(cluster.GetSecretName(), metav1.GetOptions{})
		if err != nil {
			return err
		}
		if existingSecret.Data == nil {
			existingSecret.Data = map[string][]byte{}
		}

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
			if _, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Update(existingSecret); err != nil {
				return fmt.Errorf("error updating secret: %v", err)
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

func createPacketSecret(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	spec := cluster.Spec.Cloud.Packet
	referenceDefined := spec.CredentialsReference != nil && spec.CredentialsReference.Name != ""
	valuesDefined := spec.APIKey != "" || spec.ProjectID != ""

	var secret *corev1.Secret
	if !referenceDefined && valuesDefined {
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				resources.PacketAPIKey:    []byte(cluster.Spec.Cloud.Packet.APIKey),
				resources.PacketProjectID: []byte(cluster.Spec.Cloud.Packet.ProjectID),
			},
		}

		if _, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Create(secret); err != nil {
			return err
		}
		glog.Infof("the secret = %s was created for cluster = %s resource", cluster.GetSecretName(), cluster.Name)
	}

	if !referenceDefined {
		cluster.Spec.Cloud.Hetzner.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
			ObjectReference: corev1.ObjectReference{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
		}
	}

	// treat the value provided as the sourch of truth and update the secret with this value
	if referenceDefined && valuesDefined {
		var updateNeeded bool
		existingSecret, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Get(cluster.GetSecretName(), metav1.GetOptions{})
		if err != nil {
			return err
		}
		if existingSecret.Data == nil {
			existingSecret.Data = map[string][]byte{}
		}

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
			if _, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Update(existingSecret); err != nil {
				return fmt.Errorf("error updating secret: %v", err)
			}
		}
	}

	if valuesDefined {
		cluster.Spec.Cloud.Packet.APIKey = ""
		cluster.Spec.Cloud.Packet.ProjectID = ""
	}

	return nil
}

func createKubevirtSecret(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	spec := cluster.Spec.Cloud.Kubevirt
	referenceDefined := spec.CredentialsReference != nil && spec.CredentialsReference.Name != ""
	valuesDefined := spec.Kubeconfig != ""

	var secret *corev1.Secret
	if !referenceDefined && valuesDefined {
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				resources.KubevirtKubeConfig: []byte(cluster.Spec.Cloud.Kubevirt.Kubeconfig),
			},
		}

		if _, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Create(secret); err != nil {
			return err
		}
		glog.Infof("the secret = %s was created for cluster = %s resource", cluster.GetSecretName(), cluster.Name)
	}

	if !referenceDefined {
		cluster.Spec.Cloud.Kubevirt.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
			ObjectReference: corev1.ObjectReference{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
		}
	}

	// treat the value provided as the sourch of truth and update the secret with this value
	if referenceDefined && valuesDefined {
		var updateNeeded bool
		existingSecret, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Get(cluster.GetSecretName(), metav1.GetOptions{})
		if err != nil {
			return err
		}
		if existingSecret.Data == nil {
			existingSecret.Data = map[string][]byte{}
		}

		if existingSecret.Data != nil {
			if kubeconfig, ok := existingSecret.Data[resources.KubevirtKubeConfig]; ok {
				if string(kubeconfig) != spec.Kubeconfig {
					existingSecret.Data[resources.KubevirtKubeConfig] = []byte(spec.Kubeconfig)
					updateNeeded = true
				}
			}
		}
		if updateNeeded {
			if _, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Update(existingSecret); err != nil {
				return fmt.Errorf("error updating secret: %v", err)
			}
		}
	}

	if valuesDefined {
		cluster.Spec.Cloud.Kubevirt.Kubeconfig = ""
	}

	return nil
}
