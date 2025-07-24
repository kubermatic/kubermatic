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
	"context"
	"errors"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/openstack"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/machine-controller/sdk/providerconfig"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateOrUpdateCredentialSecretForCluster creates a new secret for a credential.
func CreateOrUpdateCredentialSecretForCluster(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	_, err := createOrUpdateCredentialSecretForCluster(ctx, seedClient, cluster)
	return err
}

func createOrUpdateCredentialSecretForCluster(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
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
	if cluster.Spec.Cloud.Kubevirt != nil {
		return createOrUpdateKubevirtSecret(ctx, seedClient, cluster)
	}
	if cluster.Spec.Cloud.VSphere != nil {
		return createVSphereSecret(ctx, seedClient, cluster)
	}
	if cluster.Spec.Cloud.Baremetal != nil {
		return createOrUpdateBaremetalSecret(ctx, seedClient, cluster)
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
	if cluster.Spec.Cloud.VMwareCloudDirector != nil {
		return createOrUpdateVMwareCloudDirectorSecret(ctx, seedClient, cluster)
	}
	return false, nil
}

func CreateOrUpdateSecretForCluster(ctx context.Context, client ctrlruntimeclient.Client, externalcluster *kubermaticv1.ExternalCluster, secretData map[string][]byte, secretName, secretNamespace string) (*providerconfig.GlobalSecretKeySelector, error) {
	reconciler, err := credentialSecretReconcilerFactory(secretName, externalcluster.Labels, secretData)
	if err != nil {
		return nil, err
	}

	if err := reconciling.ReconcileSecrets(ctx, []reconciling.NamedSecretReconcilerFactory{reconciler}, secretNamespace, client); err != nil {
		return nil, fmt.Errorf("failed to ensure Secret: %w", err)
	}

	return &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      secretName,
			Namespace: secretNamespace,
		},
	}, nil
}

func ensureCredentialSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, secretData map[string][]byte) (*providerconfig.GlobalSecretKeySelector, error) {
	reconciler, err := credentialSecretReconcilerFactory(cluster.GetSecretName(), cluster.Labels, secretData)
	if err != nil {
		return nil, err
	}

	if err := reconciling.ReconcileSecrets(ctx, []reconciling.NamedSecretReconcilerFactory{reconciler}, resources.KubermaticNamespace, seedClient); err != nil {
		return nil, err
	}

	return &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      cluster.GetSecretName(),
			Namespace: resources.KubermaticNamespace,
		},
	}, nil
}

func credentialSecretReconcilerFactory(secretName string, clusterLabels map[string]string, secretData map[string][]byte) (reconciling.NamedSecretReconcilerFactory, error) {
	projectID := clusterLabels[kubermaticv1.ProjectIDLabelKey]
	if len(projectID) == 0 {
		return nil, fmt.Errorf("cluster is missing '%s' label", kubermaticv1.ProjectIDLabelKey)
	}

	return func() (name string, reconciler reconciling.SecretReconciler) {
		return secretName, func(existing *corev1.Secret) (*corev1.Secret, error) {
			if existing.Labels == nil {
				existing.Labels = map[string]string{}
			}

			existing.Labels[kubermaticv1.ProjectIDLabelKey] = projectID
			existing.Data = secretData

			return existing, nil
		}
	}, nil
}

func createOrUpdateAWSSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	spec := cluster.Spec.Cloud.AWS

	// already migrated
	if spec.AccessKeyID == "" && spec.SecretAccessKey == "" {
		return false, nil
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.AWSAccessKeyID:     []byte(spec.AccessKeyID),
		resources.AWSSecretAccessKey: []byte(spec.SecretAccessKey),
	})
	if err != nil {
		return false, err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.AWS.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.AWS.AccessKeyID = ""
	cluster.Spec.Cloud.AWS.SecretAccessKey = ""

	return true, nil
}

func createOrUpdateAzureSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	spec := cluster.Spec.Cloud.Azure

	// already migrated
	if spec.TenantID == "" && spec.SubscriptionID == "" && spec.ClientID == "" && spec.ClientSecret == "" {
		return false, nil
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.AzureTenantID:       []byte(spec.TenantID),
		resources.AzureSubscriptionID: []byte(spec.SubscriptionID),
		resources.AzureClientID:       []byte(spec.ClientID),
		resources.AzureClientSecret:   []byte(spec.ClientSecret),
	})
	if err != nil {
		return false, err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Azure.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.Azure.TenantID = ""
	cluster.Spec.Cloud.Azure.SubscriptionID = ""
	cluster.Spec.Cloud.Azure.ClientID = ""
	cluster.Spec.Cloud.Azure.ClientSecret = ""

	return true, nil
}

func createOrUpdateDigitaloceanSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	spec := cluster.Spec.Cloud.Digitalocean

	// already migrated
	if spec.Token == "" {
		return false, nil
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.DigitaloceanToken: []byte(spec.Token),
	})
	if err != nil {
		return false, err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Digitalocean.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.Digitalocean.Token = ""

	return true, nil
}

func createOrUpdateGCPSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	spec := cluster.Spec.Cloud.GCP

	// already migrated
	if spec.ServiceAccount == "" {
		return false, nil
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.GCPServiceAccount: []byte(spec.ServiceAccount),
	})
	if err != nil {
		return false, err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.GCP.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.GCP.ServiceAccount = ""

	return true, nil
}

func createOrUpdateHetznerSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	spec := cluster.Spec.Cloud.Hetzner

	// already migrated
	if spec.Token == "" {
		return false, nil
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.HetznerToken: []byte(spec.Token),
	})
	if err != nil {
		return false, err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Hetzner.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.Hetzner.Token = ""

	return true, nil
}

func createOrUpdateOpenstackSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	spec := cluster.Spec.Cloud.Openstack
	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, seedClient)

	// TenantID and Tenant fields are deprecated, if we find them in the credentials reference secret
	// then we have to trigger the secret update.
	tenantToProjectMigrated := true
	if val, _ := secretKeySelector(cluster.Spec.Cloud.Openstack.CredentialsReference, resources.OpenstackTenant); val != "" {
		tenantToProjectMigrated = false
	}
	if val, _ := secretKeySelector(cluster.Spec.Cloud.Openstack.CredentialsReference, resources.OpenstackTenantID); val != "" {
		tenantToProjectMigrated = false
	}
	clusterSpecMigrated := spec.Username == "" && spec.Password == "" && spec.Project == "" && spec.ProjectID == "" && spec.Domain == "" && spec.ApplicationCredentialID == "" && spec.ApplicationCredentialSecret == "" && !spec.UseToken

	// already migrated
	if tenantToProjectMigrated && clusterSpecMigrated {
		return false, nil
	}

	oldCred, err := openstack.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
	if err != nil {
		return false, err
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
	if spec.Username == "" {
		spec.Username = oldCred.Username
	}
	if spec.Password == "" {
		spec.Password = oldCred.Password
	}
	if spec.ApplicationCredentialID == "" {
		spec.ApplicationCredentialID = oldCred.ApplicationCredentialID
	}
	if spec.ApplicationCredentialSecret == "" {
		spec.ApplicationCredentialSecret = oldCred.ApplicationCredentialSecret
	}

	authToken := ""
	if spec.UseToken {
		if spec.Token == "" {
			return false, fmt.Errorf("failed to get authentication token")
		}
		authToken = spec.Token
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
		return false, err
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

	return true, nil
}

func createOrUpdateKubevirtSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	spec := cluster.Spec.Cloud.Kubevirt
	// already migrated
	if spec.Kubeconfig == "" {
		return false, nil
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.KubeVirtKubeconfig: []byte(spec.Kubeconfig),
	})
	if err != nil {
		return false, err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Kubevirt.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.Kubevirt.Kubeconfig = ""

	return true, nil
}

func createVSphereSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	spec := cluster.Spec.Cloud.VSphere

	// already migrated
	if spec.Username == "" && spec.Password == "" && spec.InfraManagementUser.Username == "" && spec.InfraManagementUser.Password == "" {
		return false, nil
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.VsphereUsername:                    []byte(spec.Username),
		resources.VspherePassword:                    []byte(spec.Password),
		resources.VsphereInfraManagementUserUsername: []byte(spec.InfraManagementUser.Username),
		resources.VsphereInfraManagementUserPassword: []byte(spec.InfraManagementUser.Password),
	})
	if err != nil {
		return false, err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.VSphere.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.VSphere.Username = ""
	cluster.Spec.Cloud.VSphere.Password = ""
	cluster.Spec.Cloud.VSphere.InfraManagementUser.Username = ""
	cluster.Spec.Cloud.VSphere.InfraManagementUser.Password = ""

	return true, nil
}

// createOrUpdateBaremetalSecret checks and migrates Tinkerbell credentials from inline storage to a dedicated Kubernetes secret.
// Returns true if migration occurs, false otherwise along with any error encountered.
func createOrUpdateBaremetalSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	spec := cluster.Spec.Cloud.Baremetal

	// Ensure Tinkerbell provisioner is configured, as it is mandatory.
	if spec.Tinkerbell == nil {
		return false, errors.New("tinkerbell provisioner configuration is required but missing")
	}

	// Check if migration is necessary; return early if not.
	if spec.Tinkerbell.Kubeconfig == "" {
		return false, nil // Not an error, just no action needed.
	}

	// Move credentials into a dedicated Secret and retrieve reference.
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.TinkerbellKubeconfig: []byte(spec.Tinkerbell.Kubeconfig),
	})
	if err != nil {
		return false, fmt.Errorf("failed to ensure credential secret: %w", err)
	}

	// Update the cluster object with the new credentials reference.
	cluster.Spec.Cloud.Baremetal.CredentialsReference = credentialRef
	cluster.Spec.Cloud.Baremetal.Tinkerbell.Kubeconfig = "" // Clean old inline credentials.

	// Indicate successful migration.
	return true, nil
}

func createAlibabaSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	spec := cluster.Spec.Cloud.Alibaba

	// already migrated
	if spec.AccessKeyID == "" && spec.AccessKeySecret == "" {
		return false, nil
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.AlibabaAccessKeyID:     []byte(spec.AccessKeyID),
		resources.AlibabaAccessKeySecret: []byte(spec.AccessKeySecret),
	})
	if err != nil {
		return false, err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Alibaba.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.Alibaba.AccessKeyID = ""
	cluster.Spec.Cloud.Alibaba.AccessKeySecret = ""

	return true, nil
}

func createOrUpdateAnexiaSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	spec := cluster.Spec.Cloud.Anexia

	// already migrated
	if spec.Token == "" {
		return false, nil
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.AnexiaToken: []byte(spec.Token),
	})
	if err != nil {
		return false, err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Anexia.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.Anexia.Token = ""

	return true, nil
}

func createOrUpdateNutanixSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	spec := cluster.Spec.Cloud.Nutanix

	// already migrated
	if spec.Username == "" && spec.Password == "" && spec.ProxyURL == "" && (spec.CSI == nil || (spec.CSI.Username == "" && spec.CSI.Password == "")) {
		return false, nil
	}

	secretData := map[string][]byte{
		resources.NutanixUsername: []byte(spec.Username),
		resources.NutanixPassword: []byte(spec.Password),
	}

	if spec.ProxyURL != "" {
		secretData[resources.NutanixProxyURL] = []byte(spec.ProxyURL)
	}

	// clean old inline credentials
	cluster.Spec.Cloud.Nutanix.Username = ""
	cluster.Spec.Cloud.Nutanix.Password = ""
	cluster.Spec.Cloud.Nutanix.ProxyURL = ""

	if spec.CSI != nil {
		secretData[resources.NutanixCSIUsername] = []byte(spec.CSI.Username)
		secretData[resources.NutanixCSIPassword] = []byte(spec.CSI.Password)

		cluster.Spec.Cloud.Nutanix.CSI.Username = ""
		cluster.Spec.Cloud.Nutanix.CSI.Password = ""
	}

	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, secretData)
	if err != nil {
		return false, err
	}

	// add secret key reference to cluster object
	cluster.Spec.Cloud.Nutanix.CredentialsReference = credentialRef

	return true, nil
}

func createOrUpdateVMwareCloudDirectorSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	spec := cluster.Spec.Cloud.VMwareCloudDirector

	// already migrated
	if spec.Username == "" && spec.Password == "" && spec.Organization == "" && spec.VDC == "" && spec.APIToken == "" {
		return false, nil
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.VMwareCloudDirectorUsername:     []byte(spec.Username),
		resources.VMwareCloudDirectorPassword:     []byte(spec.Password),
		resources.VMwareCloudDirectorAPIToken:     []byte(spec.APIToken),
		resources.VMwareCloudDirectorOrganization: []byte(spec.Organization),
		resources.VMwareCloudDirectorVDC:          []byte(spec.VDC),
	})
	if err != nil {
		return false, err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.VMwareCloudDirector.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.VMwareCloudDirector.Username = ""
	cluster.Spec.Cloud.VMwareCloudDirector.Password = ""
	cluster.Spec.Cloud.VMwareCloudDirector.Organization = ""
	cluster.Spec.Cloud.VMwareCloudDirector.VDC = ""
	cluster.Spec.Cloud.VMwareCloudDirector.APIToken = ""

	return true, nil
}
