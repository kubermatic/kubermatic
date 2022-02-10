/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubernetes

import (
	"bytes"
	"context"
	"fmt"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/provider"
	awsprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/aws"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/azure"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/digitalocean"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gcp"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/openstack"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateOrUpdateCredentialSecretForCluster creates a new secret for a credential.
func CreateOrUpdateCredentialSecretForCluster(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, validate bool) error {
	if cluster.Spec.Cloud.AWS != nil {
		return createOrUpdateAWSSecret(ctx, seedClient, cluster, validate)
	}
	if cluster.Spec.Cloud.Azure != nil {
		return createOrUpdateAzureSecret(ctx, seedClient, cluster, validate)
	}
	if cluster.Spec.Cloud.Digitalocean != nil {
		return createOrUpdateDigitaloceanSecret(ctx, seedClient, cluster, validate)
	}
	if cluster.Spec.Cloud.GCP != nil {
		return createOrUpdateGCPSecret(ctx, seedClient, cluster, validate)
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
		return createAlibabaSecret(ctx, seedClient, cluster)
	}
	if cluster.Spec.Cloud.Anexia != nil {
		return createOrUpdateAnexiaSecret(ctx, seedClient, cluster)
	}
	if cluster.Spec.Cloud.Nutanix != nil {
		return createOrUpdateNutanixSecret(ctx, seedClient, cluster)
	}
	return nil
}

func ensureCredentialSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, secretData map[string][]byte) (*providerconfig.GlobalSecretKeySelector, error) {
	name := cluster.GetSecretName()

	namespacedName := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: name}
	existingSecret := &corev1.Secret{}
	if err := seedClient.Get(ctx, namespacedName, existingSecret); err != nil && !kerrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to probe for secret %q: %w", name, err)
	}

	if existingSecret.Name == "" {
		projectID := cluster.Labels[kubermaticv1.ProjectIDLabelKey]
		if len(projectID) == 0 {
			return nil, fmt.Errorf("cluster is missing '%s' label", kubermaticv1.ProjectIDLabelKey)
		}
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: resources.KubermaticNamespace,
				Labels: map[string]string{
					"name":                         name,
					kubermaticv1.ProjectIDLabelKey: projectID,
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: secretData,
		}

		if err := seedClient.Create(ctx, secret); err != nil {
			return nil, fmt.Errorf("failed to create credential secret: %w", err)
		}
	} else {
		if existingSecret.Data == nil {
			existingSecret.Data = map[string][]byte{}
		}

		requiresUpdate := false

		for k, v := range secretData {
			if !bytes.Equal(v, existingSecret.Data[k]) {
				requiresUpdate = true
				break
			}
		}

		if requiresUpdate {
			existingSecret.Data = secretData
			if err := seedClient.Update(ctx, existingSecret); err != nil {
				return nil, fmt.Errorf("failed to update credential secret: %w", err)
			}
		}
	}

	return &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      name,
			Namespace: resources.KubermaticNamespace,
		},
	}, nil
}

func createOrUpdateAWSSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, validate bool) error {
	spec := cluster.Spec.Cloud.AWS

	// already migrated
	if spec.AccessKeyID == "" && spec.SecretAccessKey == "" {
		return nil
	}

	if validate {
		if err := awsprovider.ValidateCredentials(spec.AccessKeyID, spec.SecretAccessKey); err != nil {
			return fmt.Errorf("invalid AWS credentials: %w", err)
		}
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.AWSAccessKeyID:     []byte(spec.AccessKeyID),
		resources.AWSSecretAccessKey: []byte(spec.SecretAccessKey),
	})
	if err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.AWS.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.AWS.AccessKeyID = ""
	cluster.Spec.Cloud.AWS.SecretAccessKey = ""

	return nil
}

func createOrUpdateAzureSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, validate bool) error {
	spec := cluster.Spec.Cloud.Azure

	// already migrated
	if spec.TenantID == "" && spec.SubscriptionID == "" && spec.ClientID == "" && spec.ClientSecret == "" {
		return nil
	}

	if validate {
		if err := azure.ValidateCredentials(ctx, azure.Credentials{
			TenantID:       spec.TenantID,
			SubscriptionID: spec.SubscriptionID,
			ClientID:       spec.ClientID,
			ClientSecret:   spec.ClientSecret,
		}); err != nil {
			return fmt.Errorf("invalid Azure credentials: %w", err)
		}
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.AzureTenantID:       []byte(spec.TenantID),
		resources.AzureSubscriptionID: []byte(spec.SubscriptionID),
		resources.AzureClientID:       []byte(spec.ClientID),
		resources.AzureClientSecret:   []byte(spec.ClientSecret),
	})
	if err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Azure.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.Azure.TenantID = ""
	cluster.Spec.Cloud.Azure.SubscriptionID = ""
	cluster.Spec.Cloud.Azure.ClientID = ""
	cluster.Spec.Cloud.Azure.ClientSecret = ""

	return nil
}

func createOrUpdateDigitaloceanSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, validate bool) error {
	spec := cluster.Spec.Cloud.Digitalocean

	// already migrated
	if spec.Token == "" {
		return nil
	}

	if validate {
		if err := digitalocean.ValidateCredentials(ctx, spec.Token); err != nil {
			return fmt.Errorf("invalid DigitalOcean token: %w", err)
		}
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.DigitaloceanToken: []byte(spec.Token),
	})
	if err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Digitalocean.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.Digitalocean.Token = ""

	return nil
}

func createOrUpdateGCPSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, validate bool) error {
	spec := cluster.Spec.Cloud.GCP

	// already migrated
	if spec.ServiceAccount == "" {
		return nil
	}

	if validate {
		if err := gcp.ValidateCredentials(ctx, spec.ServiceAccount); err != nil {
			return fmt.Errorf("invalid GCP credentials: %w", err)
		}
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.GCPServiceAccount: []byte(spec.ServiceAccount),
	})
	if err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.GCP.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.GCP.ServiceAccount = ""

	return nil
}

func createOrUpdateHetznerSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	spec := cluster.Spec.Cloud.Hetzner

	// already migrated
	if spec.Token == "" {
		return nil
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.HetznerToken: []byte(spec.Token),
	})
	if err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Hetzner.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.Hetzner.Token = ""

	return nil
}

func createOrUpdateOpenstackSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	spec := cluster.Spec.Cloud.Openstack

	// already migrated
	if spec.Username == "" && spec.Password == "" && spec.Project == "" && spec.ProjectID == "" && spec.Domain == "" && spec.ApplicationCredentialID == "" && spec.ApplicationCredentialSecret == "" && !spec.UseToken {
		return nil
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, seedClient)
	oldCred, err := openstack.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
	if err != nil {
		return err
	}
	if spec.Project == "" {
		spec.Project = oldCred.Project
	}
	if spec.ProjectID == "" {
		spec.ProjectID = oldCred.ProjectID
	}
	if spec.Domain == "" {
		spec.Domain = oldCred.Domain
	}
	authToken := ""
	if spec.UseToken {
		t := ctx.Value(middleware.RawTokenContextKey)
		token, ok := t.(string)
		if !ok || token == "" {
			return fmt.Errorf("failed to get authentication token")
		}
		authToken = token
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.OpenstackUsername:                    []byte(spec.Username),
		resources.OpenstackPassword:                    []byte(spec.Password),
		resources.OpenstackProject:                     []byte(spec.Project),
		resources.OpenstackProjectID:                   []byte(spec.ProjectID),
		resources.OpenstackDomain:                      []byte(spec.Domain),
		resources.OpenstackApplicationCredentialID:     []byte(spec.ApplicationCredentialID),
		resources.OpenstackApplicationCredentialSecret: []byte(spec.ApplicationCredentialSecret),
		resources.OpenstackToken:                       []byte(authToken),
	})
	if err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Openstack.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.Openstack.Username = ""
	cluster.Spec.Cloud.Openstack.Password = ""
	cluster.Spec.Cloud.Openstack.Project = ""
	cluster.Spec.Cloud.Openstack.ProjectID = ""
	cluster.Spec.Cloud.Openstack.Domain = ""
	cluster.Spec.Cloud.Openstack.ApplicationCredentialSecret = ""
	cluster.Spec.Cloud.Openstack.ApplicationCredentialID = ""
	cluster.Spec.Cloud.Openstack.UseToken = false

	return nil
}

func createOrUpdatePacketSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	spec := cluster.Spec.Cloud.Packet

	// already migrated
	if spec.APIKey == "" && spec.ProjectID == "" {
		return nil
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.PacketAPIKey:    []byte(spec.APIKey),
		resources.PacketProjectID: []byte(spec.ProjectID),
	})
	if err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Packet.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.Packet.APIKey = ""
	cluster.Spec.Cloud.Packet.ProjectID = ""

	return nil
}

func createOrUpdateKubevirtSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	spec := cluster.Spec.Cloud.Kubevirt
	// already migrated
	if spec.Kubeconfig == "" {
		return nil
	}

	// ensure that CSI driver on user cluster will have an access to KubeVirt cluster
	r, err := kubevirt.NewReconciler(spec.Kubeconfig, cluster.Name)
	if err != nil {
		return err
	}
	csiKubeconfig, err := r.ReconcileCSIAccess(ctx)
	if err != nil {
		return err
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.KubevirtKubeConfig:    []byte(spec.Kubeconfig),
		resources.KubevirtCSIKubeConfig: csiKubeconfig,
	})
	if err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Kubevirt.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.Kubevirt.Kubeconfig = ""

	return nil
}

func createVSphereSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	spec := cluster.Spec.Cloud.VSphere

	// already migrated
	if spec.Username == "" && spec.Password == "" && spec.InfraManagementUser.Username == "" && spec.InfraManagementUser.Password == "" {
		return nil
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.VsphereUsername:                    []byte(spec.Username),
		resources.VspherePassword:                    []byte(spec.Password),
		resources.VsphereInfraManagementUserUsername: []byte(spec.InfraManagementUser.Username),
		resources.VsphereInfraManagementUserPassword: []byte(spec.InfraManagementUser.Password),
	})
	if err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.VSphere.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.VSphere.Username = ""
	cluster.Spec.Cloud.VSphere.Password = ""
	cluster.Spec.Cloud.VSphere.InfraManagementUser.Username = ""
	cluster.Spec.Cloud.VSphere.InfraManagementUser.Password = ""

	return nil
}

func createAlibabaSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	spec := cluster.Spec.Cloud.Alibaba

	// already migrated
	if spec.AccessKeyID == "" && spec.AccessKeySecret == "" {
		return nil
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.AlibabaAccessKeyID:     []byte(spec.AccessKeyID),
		resources.AlibabaAccessKeySecret: []byte(spec.AccessKeySecret),
	})
	if err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Alibaba.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.Alibaba.AccessKeyID = ""
	cluster.Spec.Cloud.Alibaba.AccessKeySecret = ""

	return nil
}

func createOrUpdateAnexiaSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	spec := cluster.Spec.Cloud.Anexia

	// already migrated
	if spec.Token == "" {
		return nil
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.AnexiaToken: []byte(spec.Token),
	})
	if err != nil {
		return err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Anexia.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.Anexia.Token = ""

	return nil
}

func createOrUpdateNutanixSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	spec := cluster.Spec.Cloud.Nutanix

	// already migrated
	if spec.Username == "" && spec.Password == "" && spec.PeUsername == "" && spec.PePassword == "" && spec.ProxyURL == "" {
		return nil
	}

	secretData := map[string][]byte{
		resources.NutanixUsername:   []byte(spec.Username),
		resources.NutanixPassword:   []byte(spec.Password),
		resources.NutanixPeUsername: []byte(spec.PeUsername),
		resources.NutanixPePassword: []byte(spec.PePassword),
	}

	if spec.ProxyURL != "" {
		secretData[resources.NutanixProxyURL] = []byte(spec.ProxyURL)
	}

	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, secretData)
	if err != nil {
		return err
	}

	// add secret key reference to cluster object
	cluster.Spec.Cloud.Nutanix.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.Nutanix.Username = ""
	cluster.Spec.Cloud.Nutanix.Password = ""
	cluster.Spec.Cloud.Nutanix.PeUsername = ""
	cluster.Spec.Cloud.Nutanix.PePassword = ""
	cluster.Spec.Cloud.Nutanix.ProxyURL = ""

	return nil
}
