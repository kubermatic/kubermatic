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
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"strconv"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/alibaba"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/anexia"
	awsprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/aws"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/azure"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/digitalocean"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gcp"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/hetzner"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/nutanix"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/openstack"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/packet"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/vmwareclouddirector"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/vsphere"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ValidateCredentials struct {
	Datacenter *kubermaticv1.Datacenter
	CABundle   *x509.CertPool
}

// CreateOrUpdateCredentialSecretForClusterWithValidation creates a new secret for a credential.
func CreateOrUpdateCredentialSecretForClusterWithValidation(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, validate *ValidateCredentials) (bool, error) {
	return createOrUpdateCredentialSecretForCluster(ctx, seedClient, cluster, validate)
}

// CreateOrUpdateCredentialSecretForCluster creates a new secret for a credential.
func CreateOrUpdateCredentialSecretForCluster(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	_, err := createOrUpdateCredentialSecretForCluster(ctx, seedClient, cluster, nil)
	return err
}

func createOrUpdateCredentialSecretForCluster(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, validate *ValidateCredentials) (bool, error) {
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
		return createOrUpdateHetznerSecret(ctx, seedClient, cluster, validate)
	}
	if cluster.Spec.Cloud.Openstack != nil {
		return createOrUpdateOpenstackSecret(ctx, seedClient, cluster, validate)
	}
	if cluster.Spec.Cloud.Packet != nil {
		return createOrUpdatePacketSecret(ctx, seedClient, cluster, validate)
	}
	if cluster.Spec.Cloud.Kubevirt != nil {
		return createOrUpdateKubevirtSecret(ctx, seedClient, cluster)
	}
	if cluster.Spec.Cloud.VSphere != nil {
		return createVSphereSecret(ctx, seedClient, cluster, validate)
	}
	if cluster.Spec.Cloud.Alibaba != nil {
		return createAlibabaSecret(ctx, seedClient, cluster, validate)
	}
	if cluster.Spec.Cloud.Anexia != nil {
		return createOrUpdateAnexiaSecret(ctx, seedClient, cluster, validate)
	}
	if cluster.Spec.Cloud.Nutanix != nil {
		return createOrUpdateNutanixSecret(ctx, seedClient, cluster, validate)
	}
	if cluster.Spec.Cloud.VMwareCloudDirector != nil {
		return createOrUpdateVMwareCloudDirectorSecret(ctx, seedClient, cluster, validate)
	}
	return false, nil
}

func ensureCredentialSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, secretData map[string][]byte) (*providerconfig.GlobalSecretKeySelector, error) {
	creator, err := credentialSecretCreatorGetter(cluster.GetSecretName(), cluster.Labels, secretData)
	if err != nil {
		return nil, err
	}

	if err := reconciling.ReconcileSecrets(ctx, []reconciling.NamedSecretCreatorGetter{creator}, resources.KubermaticNamespace, seedClient); err != nil {
		return nil, err
	}

	return &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      cluster.GetSecretName(),
			Namespace: resources.KubermaticNamespace,
		},
	}, nil
}

func credentialSecretCreatorGetter(secretName string, clusterLabels map[string]string, secretData map[string][]byte) (reconciling.NamedSecretCreatorGetter, error) {
	projectID := clusterLabels[kubermaticv1.ProjectIDLabelKey]
	if len(projectID) == 0 {
		return nil, fmt.Errorf("cluster is missing '%s' label", kubermaticv1.ProjectIDLabelKey)
	}

	return func() (name string, create reconciling.SecretCreator) {
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

func createOrUpdateAWSSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, validate *ValidateCredentials) (bool, error) {
	spec := cluster.Spec.Cloud.AWS

	// already migrated
	if spec.AccessKeyID == "" && spec.SecretAccessKey == "" {
		return false, nil
	}

	if validate != nil {
		if err := awsprovider.ValidateCredentials(ctx, spec.AccessKeyID, spec.SecretAccessKey); err != nil {
			return false, fmt.Errorf("invalid AWS credentials: %w", err)
		}
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

func createOrUpdateAzureSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, validate *ValidateCredentials) (bool, error) {
	spec := cluster.Spec.Cloud.Azure

	// already migrated
	if spec.TenantID == "" && spec.SubscriptionID == "" && spec.ClientID == "" && spec.ClientSecret == "" {
		return false, nil
	}

	if validate != nil {
		cred, err := azure.Credentials{
			TenantID:       spec.TenantID,
			SubscriptionID: spec.SubscriptionID,
			ClientID:       spec.ClientID,
			ClientSecret:   spec.ClientSecret,
		}.ToAzureCredential()
		if err != nil {
			return false, fmt.Errorf("invalid Azure credentials: %w", err)
		}

		if err := azure.ValidateCredentials(ctx, cred, spec.SubscriptionID); err != nil {
			return false, fmt.Errorf("invalid Azure credentials: %w", err)
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

func createOrUpdateDigitaloceanSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, validate *ValidateCredentials) (bool, error) {
	spec := cluster.Spec.Cloud.Digitalocean

	// already migrated
	if spec.Token == "" {
		return false, nil
	}

	if validate != nil {
		if err := digitalocean.ValidateCredentials(ctx, spec.Token); err != nil {
			return false, fmt.Errorf("invalid DigitalOcean token: %w", err)
		}
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

func createOrUpdateGCPSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, validate *ValidateCredentials) (bool, error) {
	spec := cluster.Spec.Cloud.GCP

	// already migrated
	if spec.ServiceAccount == "" {
		return false, nil
	}

	if validate != nil {
		if err := gcp.ValidateCredentials(ctx, spec.ServiceAccount); err != nil {
			return false, fmt.Errorf("invalid GCP credentials: %w", err)
		}
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

func createOrUpdateHetznerSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, validate *ValidateCredentials) (bool, error) {
	spec := cluster.Spec.Cloud.Hetzner

	// already migrated
	if spec.Token == "" {
		return false, nil
	}

	if validate != nil {
		if err := hetzner.ValidateCredentials(ctx, spec.Token); err != nil {
			return false, fmt.Errorf("invalid Hetzner credentials: %w", err)
		}
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

func createOrUpdateOpenstackSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, validate *ValidateCredentials) (bool, error) {
	spec := cluster.Spec.Cloud.Openstack

	// already migrated
	if spec.Username == "" && spec.Password == "" && spec.Project == "" && spec.ProjectID == "" && spec.Domain == "" && spec.ApplicationCredentialID == "" && spec.ApplicationCredentialSecret == "" && !spec.UseToken {
		return false, nil
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, seedClient)
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
	authToken := ""
	if spec.UseToken {
		t := ctx.Value("raw-auth-token") // TODO: This cannot work since the KKP API got removed
		token, ok := t.(string)
		if !ok || token == "" {
			return false, fmt.Errorf("failed to get authentication token")
		}
		authToken = token
	}

	if validate != nil {
		cred := &resources.OpenstackCredentials{
			Username:                    spec.Username,
			Password:                    spec.Password,
			Project:                     spec.Project,
			ProjectID:                   spec.ProjectID,
			Domain:                      spec.Domain,
			ApplicationCredentialID:     spec.ApplicationCredentialID,
			ApplicationCredentialSecret: spec.ApplicationCredentialSecret,
			Token:                       authToken,
		}

		dcSpec := validate.Datacenter.Spec.Openstack
		if err := openstack.ValidateCredentials(dcSpec.AuthURL, dcSpec.Region, cred, validate.CABundle); err != nil {
			return false, fmt.Errorf("invalid Openstack credentials: %w", err)
		}
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

func createOrUpdatePacketSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, validate *ValidateCredentials) (bool, error) {
	spec := cluster.Spec.Cloud.Packet

	// already migrated
	if spec.APIKey == "" && spec.ProjectID == "" {
		return false, nil
	}

	if validate != nil {
		if err := packet.ValidateCredentials(spec.APIKey, spec.ProjectID); err != nil {
			return false, fmt.Errorf("invalid Equinixmetal credentials: %w", err)
		}
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.PacketAPIKey:    []byte(spec.APIKey),
		resources.PacketProjectID: []byte(spec.ProjectID),
	})
	if err != nil {
		return false, err
	}

	// add secret key selectors to cluster object
	cluster.Spec.Cloud.Packet.CredentialsReference = credentialRef

	// clean old inline credentials
	cluster.Spec.Cloud.Packet.APIKey = ""
	cluster.Spec.Cloud.Packet.ProjectID = ""

	return true, nil
}

func createOrUpdateKubevirtSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	spec := cluster.Spec.Cloud.Kubevirt
	// already migrated
	if spec.Kubeconfig == "" {
		return false, nil
	}

	// ensure that CSI driver on user cluster will have access to KubeVirt cluster
	// RBAC reconciliation takes place in the kubevirt cloud provider
	csiKubeconfig, err := kubevirt.EnsureCSIInfraTokenAccess(ctx, spec.Kubeconfig)
	if err != nil {
		return false, err
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.KubevirtKubeConfig:    []byte(spec.Kubeconfig),
		resources.KubevirtCSIKubeConfig: csiKubeconfig,
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

func createVSphereSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, validate *ValidateCredentials) (bool, error) {
	spec := cluster.Spec.Cloud.VSphere

	// already migrated
	if spec.Username == "" && spec.Password == "" && spec.InfraManagementUser.Username == "" && spec.InfraManagementUser.Password == "" {
		return false, nil
	}

	if validate != nil {
		if err := vsphere.ValidateCredentials(ctx, validate.Datacenter.Spec.VSphere, spec.Username, spec.Password, validate.CABundle); err != nil {
			return false, fmt.Errorf("invalid VSphere credentials: %w", err)
		}
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

func createAlibabaSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, validate *ValidateCredentials) (bool, error) {
	spec := cluster.Spec.Cloud.Alibaba

	// already migrated
	if spec.AccessKeyID == "" && spec.AccessKeySecret == "" {
		return false, nil
	}

	if validate != nil {
		dcSpec := validate.Datacenter.Spec.Alibaba
		if err := alibaba.ValidateCredentials(dcSpec.Region, spec.AccessKeyID, spec.AccessKeySecret); err != nil {
			return false, fmt.Errorf("invalid Alibaba credentials: %w", err)
		}
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

func createOrUpdateAnexiaSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, validate *ValidateCredentials) (bool, error) {
	spec := cluster.Spec.Cloud.Anexia

	// already migrated
	if spec.Token == "" {
		return false, nil
	}

	if validate != nil {
		if err := anexia.ValidateCredentials(ctx, spec.Token, validate.Datacenter.Spec.Anexia.LocationID); err != nil {
			return false, fmt.Errorf("invalid Anexia credentials: %w", err)
		}
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

func createOrUpdateNutanixSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, validate *ValidateCredentials) (bool, error) {
	spec := cluster.Spec.Cloud.Nutanix

	// already migrated
	if spec.Username == "" && spec.Password == "" && spec.ProxyURL == "" && (spec.CSI == nil || (spec.CSI.Username == "" && spec.CSI.Password == "")) {
		return false, nil
	}

	if validate != nil {
		if err := nutanix.ValidateCredentials(ctx, validate.Datacenter.Spec.Nutanix.Endpoint, validate.Datacenter.Spec.Nutanix.Port, &validate.Datacenter.Spec.Nutanix.AllowInsecure, spec.ProxyURL, spec.Username, spec.Password); err != nil {
			return false, fmt.Errorf("invalid Nutanix credentials: %w", err)
		}
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

func createOrUpdateVMwareCloudDirectorSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, validate *ValidateCredentials) (bool, error) {
	spec := cluster.Spec.Cloud.VMwareCloudDirector

	// already migrated
	if spec.Username == "" && spec.Password == "" && spec.Organization == "" && spec.VDC == "" {
		return false, nil
	}

	if validate != nil {
		if err := vmwareclouddirector.ValidateCredentials(ctx, validate.Datacenter.Spec.VMwareCloudDirector, spec.Username, spec.Password, spec.Organization, spec.VDC); err != nil {
			return false, fmt.Errorf("invalid VMware Cloud Director credentials: %w", err)
		}
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialSecret(ctx, seedClient, cluster, map[string][]byte{
		resources.VMwareCloudDirectorUsername:     []byte(spec.Username),
		resources.VMwareCloudDirectorPassword:     []byte(spec.Password),
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

	return true, nil
}

func GetKubeOneNamespaceName(externalClusterName string) string {
	return fmt.Sprintf("%s%s", resources.KubeOneNamespacePrefix, externalClusterName)
}

func (p *ExternalClusterProvider) CreateKubeOneClusterNamespace(ctx context.Context, externalCluster *kubermaticv1.ExternalCluster) error {
	kubeOneNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: GetKubeOneNamespaceName(externalCluster.Name),
		},
	}
	if err := p.GetMasterClient().Create(ctx, kubeOneNamespace); err != nil {
		return fmt.Errorf("failed to create kubeone cluster namespace: %w", err)
	}

	return nil
}

func ensureCredentialKubeOneSecret(ctx context.Context, masterClient ctrlruntimeclient.Client, externalcluster *kubermaticv1.ExternalCluster, secretName string, secretData map[string][]byte) (*providerconfig.GlobalSecretKeySelector, error) {
	creator, err := credentialSecretCreatorGetter(secretName, externalcluster.Labels, secretData)
	if err != nil {
		return nil, err
	}

	kubeOneNamespaceName := GetKubeOneNamespaceName(externalcluster.Name)
	creators := []reconciling.NamedSecretCreatorGetter{creator}

	if err := reconciling.ReconcileSecrets(ctx, creators, kubeOneNamespaceName, masterClient); err != nil {
		return nil, err
	}

	return &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      secretName,
			Namespace: kubeOneNamespaceName,
		},
	}, nil
}

// CreateOrUpdateKubeOneCredentialSecret creates a new secret for a credential.
func (p *ExternalClusterProvider) CreateOrUpdateKubeOneCredentialSecret(ctx context.Context, cloud apiv2.KubeOneCloudSpec, externalCluster *kubermaticv1.ExternalCluster) error {
	secretName := GetKubeOneCredentialsSecretName(cloud)

	if cloud.AWS != nil {
		externalCluster.Spec.CloudSpec.KubeOne.ProviderName = resources.KubeOneAWS
		return createOrUpdateKubeOneAWSSecret(ctx, cloud, p.GetMasterClient(), secretName, externalCluster)
	}
	if cloud.GCP != nil {
		externalCluster.Spec.CloudSpec.KubeOne.ProviderName = resources.KubeOneGCP
		return createOrUpdateKubeOneGCPSecret(ctx, cloud, p.GetMasterClient(), secretName, externalCluster)
	}
	if cloud.Azure != nil {
		externalCluster.Spec.CloudSpec.KubeOne.ProviderName = resources.KubeOneAzure
		return createOrUpdateKubeOneAzureSecret(ctx, cloud, p.GetMasterClient(), secretName, externalCluster)
	}
	if cloud.DigitalOcean != nil {
		externalCluster.Spec.CloudSpec.KubeOne.ProviderName = resources.KubeOneDigitalOcean
		return createOrUpdateKubeOneDigitaloceanSecret(ctx, cloud, p.GetMasterClient(), secretName, externalCluster)
	}
	if cloud.VSphere != nil {
		externalCluster.Spec.CloudSpec.KubeOne.ProviderName = resources.KubeOneVSphere
		return createOrUpdateKubeOneVSphereSecret(ctx, cloud, p.GetMasterClient(), secretName, externalCluster)
	}
	if cloud.Hetzner != nil {
		externalCluster.Spec.CloudSpec.KubeOne.ProviderName = resources.KubeOneHetzner
		return createOrUpdateKubeOneHetznerSecret(ctx, cloud, p.GetMasterClient(), secretName, externalCluster)
	}
	if cloud.Equinix != nil {
		externalCluster.Spec.CloudSpec.KubeOne.ProviderName = resources.KubeOneEquinix
		return createOrUpdateKubeOneEquinixSecret(ctx, cloud, p.GetMasterClient(), secretName, externalCluster)
	}
	if cloud.OpenStack != nil {
		externalCluster.Spec.CloudSpec.KubeOne.ProviderName = resources.KubeOneOpenStack
		return createOrUpdateKubeOneOpenstackSecret(ctx, cloud, p.GetMasterClient(), secretName, externalCluster)
	}
	if cloud.Nutanix != nil {
		externalCluster.Spec.CloudSpec.KubeOne.ProviderName = resources.KubeOneNutanix
		return createOrUpdateKubeOneNutanixSecret(ctx, cloud, p.GetMasterClient(), secretName, externalCluster)
	}
	if cloud.VMwareCloudDirector != nil {
		externalCluster.Spec.CloudSpec.KubeOne.ProviderName = resources.KubeOneVMwareCloudDirector
		return createOrUpdateKubeOneVMwareCloudDirectorSecret(ctx, cloud, p.GetMasterClient(), secretName, externalCluster)
	}
	return nil
}

func createOrUpdateKubeOneAWSSecret(ctx context.Context, cloud apiv2.KubeOneCloudSpec, masterClient ctrlruntimeclient.Client, secretName string, externalcluster *kubermaticv1.ExternalCluster) error {
	if cloud.AWS.AccessKeyID == "" || cloud.AWS.SecretAccessKey == "" {
		return utilerrors.NewBadRequest("kubeone aws credentials missing")
	}

	if err := awsprovider.ValidateCredentials(ctx, cloud.AWS.AccessKeyID, cloud.AWS.SecretAccessKey); err != nil {
		return fmt.Errorf("invalid AWS credentials: %w", err)
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialKubeOneSecret(ctx, masterClient, externalcluster, secretName, map[string][]byte{
		resources.AWSAccessKeyID:     []byte(cloud.AWS.AccessKeyID),
		resources.AWSSecretAccessKey: []byte(cloud.AWS.SecretAccessKey),
	})
	if err != nil {
		return err
	}

	// add secret key selectors to externalCluster object
	externalcluster.Spec.CloudSpec.KubeOne.CredentialsReference = *credentialRef

	return nil
}

func createOrUpdateKubeOneGCPSecret(ctx context.Context, cloud apiv2.KubeOneCloudSpec, masterClient ctrlruntimeclient.Client, secretName string, externalCluster *kubermaticv1.ExternalCluster) error {
	encodedServiceAccount := cloud.GCP.ServiceAccount
	if encodedServiceAccount == "" {
		return utilerrors.NewBadRequest("kubeone gcp credentials missing")
	}

	if err := gcp.ValidateCredentials(ctx, encodedServiceAccount); err != nil {
		return fmt.Errorf("invalid GCP credentials: %w", err)
	}

	serviceAccount, err := base64.StdEncoding.DecodeString(encodedServiceAccount)
	if err != nil {
		return fmt.Errorf("failed to decode gcp credential: %w", err)
	}
	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialKubeOneSecret(ctx, masterClient, externalCluster, secretName, map[string][]byte{
		resources.GCPServiceAccount: serviceAccount,
	})
	if err != nil {
		return err
	}

	// add secret key selectors to cluster object
	externalCluster.Spec.CloudSpec.KubeOne.CredentialsReference = *credentialRef

	return nil
}

func createOrUpdateKubeOneAzureSecret(ctx context.Context, cloud apiv2.KubeOneCloudSpec, masterClient ctrlruntimeclient.Client, secretName string, externalCluster *kubermaticv1.ExternalCluster) error {
	tenantID := cloud.Azure.TenantID
	subscriptionID := cloud.Azure.SubscriptionID
	clientID := cloud.Azure.ClientID
	clientSecret := cloud.Azure.ClientSecret

	if tenantID == "" || subscriptionID == "" || clientID == "" || clientSecret == "" {
		return utilerrors.NewBadRequest("kubeone Azure credentials missing")
	}

	cred, err := azure.Credentials{
		TenantID:       tenantID,
		SubscriptionID: subscriptionID,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
	}.ToAzureCredential()
	if err != nil {
		return fmt.Errorf("invalid Azure credentials: %w", err)
	}

	if err := azure.ValidateCredentials(ctx, cred, subscriptionID); err != nil {
		return fmt.Errorf("invalid Azure credentials: %w", err)
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialKubeOneSecret(ctx, masterClient, externalCluster, secretName, map[string][]byte{
		resources.AzureTenantID:       []byte(tenantID),
		resources.AzureSubscriptionID: []byte(subscriptionID),
		resources.AzureClientID:       []byte(clientID),
		resources.AzureClientSecret:   []byte(clientSecret),
	})
	if err != nil {
		return err
	}

	// add secret key selectors to externalCluster object
	externalCluster.Spec.CloudSpec.KubeOne.CredentialsReference = *credentialRef

	return nil
}

func createOrUpdateKubeOneDigitaloceanSecret(ctx context.Context, cloud apiv2.KubeOneCloudSpec, masterClient ctrlruntimeclient.Client, secretName string, externalCluster *kubermaticv1.ExternalCluster) error {
	token := cloud.DigitalOcean.Token

	if token == "" {
		return utilerrors.NewBadRequest("kubeone DigitalOcean credentials missing")
	}

	if err := digitalocean.ValidateCredentials(ctx, token); err != nil {
		return fmt.Errorf("invalid DigitalOcean token: %w", err)
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialKubeOneSecret(ctx, masterClient, externalCluster, secretName, map[string][]byte{
		resources.DigitaloceanToken: []byte(token),
	})
	if err != nil {
		return err
	}

	// add secret key selectors to externalCluster object
	externalCluster.Spec.CloudSpec.KubeOne.CredentialsReference = *credentialRef

	return nil
}

func createOrUpdateKubeOneOpenstackSecret(ctx context.Context, cloud apiv2.KubeOneCloudSpec, masterClient ctrlruntimeclient.Client, secretName string, externalCluster *kubermaticv1.ExternalCluster) error {
	authUrl := cloud.OpenStack.AuthURL
	username := cloud.OpenStack.Username
	password := cloud.OpenStack.Password
	project := cloud.OpenStack.Project
	projectID := cloud.OpenStack.ProjectID
	domain := cloud.OpenStack.Domain
	region := cloud.OpenStack.Region

	if username == "" || password == "" || domain == "" || authUrl == "" || project == "" || projectID == "" || region == "" {
		return utilerrors.NewBadRequest("kubeone Openstack credentials missing")
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialKubeOneSecret(ctx, masterClient, externalCluster, secretName, map[string][]byte{
		resources.OpenstackAuthURL:   []byte(authUrl),
		resources.OpenstackUsername:  []byte(username),
		resources.OpenstackPassword:  []byte(password),
		resources.OpenstackProject:   []byte(project),
		resources.OpenstackProjectID: []byte(projectID),
		resources.OpenstackDomain:    []byte(domain),
		resources.OpenstackRegion:    []byte(region),
	})
	if err != nil {
		return err
	}

	// add secret key selectors to externalCluster object
	externalCluster.Spec.CloudSpec.KubeOne.CredentialsReference = *credentialRef

	return nil
}

func createOrUpdateKubeOneVSphereSecret(ctx context.Context, cloud apiv2.KubeOneCloudSpec, masterClient ctrlruntimeclient.Client, secretName string, externalCluster *kubermaticv1.ExternalCluster) error {
	username := cloud.VSphere.Username
	password := cloud.VSphere.Password
	server := cloud.VSphere.Server

	if username == "" || password == "" || server == "" {
		return utilerrors.NewBadRequest("kubeone VSphere credentials missing")
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialKubeOneSecret(ctx, masterClient, externalCluster, secretName, map[string][]byte{
		resources.VsphereUsername: []byte(username),
		resources.VspherePassword: []byte(password),
		resources.VsphereServer:   []byte(server),
	})
	if err != nil {
		return err
	}

	// add secret key selectors to externalCluster object
	externalCluster.Spec.CloudSpec.KubeOne.CredentialsReference = *credentialRef

	return nil
}

func createOrUpdateKubeOneEquinixSecret(ctx context.Context, cloud apiv2.KubeOneCloudSpec, masterClient ctrlruntimeclient.Client, secretName string, externalCluster *kubermaticv1.ExternalCluster) error {
	apiKey := cloud.Equinix.APIKey
	projectID := cloud.Equinix.ProjectID

	if apiKey == "" || projectID == "" {
		return utilerrors.NewBadRequest("kubeone Packet credentials missing")
	}

	if err := packet.ValidateCredentials(apiKey, projectID); err != nil {
		return fmt.Errorf("invalid Packet credentials: %w", err)
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialKubeOneSecret(ctx, masterClient, externalCluster, secretName, map[string][]byte{
		resources.PacketAPIKey:    []byte(apiKey),
		resources.PacketProjectID: []byte(projectID),
	})
	if err != nil {
		return err
	}

	// add secret key selectors to cluster object
	externalCluster.Spec.CloudSpec.KubeOne.CredentialsReference = *credentialRef

	return nil
}

func createOrUpdateKubeOneHetznerSecret(ctx context.Context, cloud apiv2.KubeOneCloudSpec, masterClient ctrlruntimeclient.Client, secretName string, externalCluster *kubermaticv1.ExternalCluster) error {
	token := cloud.Hetzner.Token

	if token == "" {
		return utilerrors.NewBadRequest("kubeone Hetzner credentials missing")
	}

	if err := hetzner.ValidateCredentials(ctx, token); err != nil {
		return fmt.Errorf("invalid Hetzner credentials: %w", err)
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialKubeOneSecret(ctx, masterClient, externalCluster, secretName, map[string][]byte{
		resources.HetznerToken: []byte(token),
	})
	if err != nil {
		return err
	}

	// add secret key selectors to cluster object
	externalCluster.Spec.CloudSpec.KubeOne.CredentialsReference = *credentialRef

	return nil
}

func createOrUpdateKubeOneNutanixSecret(ctx context.Context, cloud apiv2.KubeOneCloudSpec, masterClient ctrlruntimeclient.Client, secretName string, externalCluster *kubermaticv1.ExternalCluster) error {
	username := cloud.Nutanix.Username
	password := cloud.Nutanix.Password
	endpoint := cloud.Nutanix.Endpoint
	port := cloud.Nutanix.Port
	peEndpoint := cloud.Nutanix.PrismElementEndpoint
	peUsername := cloud.Nutanix.PrismElementUsername
	pePassword := cloud.Nutanix.PrismElementPassword
	proxyURL := cloud.Nutanix.ProxyURL
	clusterName := cloud.Nutanix.ClusterName
	allowInsecure := cloud.Nutanix.AllowInsecure

	if endpoint == "" || port == "" || username == "" || password == "" || peEndpoint == "" || peUsername == "" || pePassword == "" {
		return utilerrors.NewBadRequest("kubeone Nutanix credentials missing")
	}

	secretData := map[string][]byte{
		resources.NutanixUsername:    []byte(username),
		resources.NutanixPassword:    []byte(password),
		resources.NutanixEndpoint:    []byte(endpoint),
		resources.NutanixPort:        []byte(port),
		resources.NutanixCSIUsername: []byte(peUsername),
		resources.NutanixCSIPassword: []byte(pePassword),
		resources.NutanixCSIEndpoint: []byte(peEndpoint),
	}

	if proxyURL != "" {
		secretData[resources.NutanixProxyURL] = []byte(proxyURL)
	}
	if allowInsecure {
		secretData[resources.NutanixAllowInsecure] = []byte(strconv.FormatBool(allowInsecure))
	}
	if clusterName != "" {
		secretData[resources.NutanixClusterName] = []byte(clusterName)
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialKubeOneSecret(ctx, masterClient, externalCluster, secretName, secretData)
	if err != nil {
		return err
	}

	// add secret key selectors to cluster object
	externalCluster.Spec.CloudSpec.KubeOne.CredentialsReference = *credentialRef

	return nil
}

func createOrUpdateKubeOneVMwareCloudDirectorSecret(ctx context.Context, cloud apiv2.KubeOneCloudSpec, masterClient ctrlruntimeclient.Client, secretName string, externalCluster *kubermaticv1.ExternalCluster) error {
	username := cloud.VMwareCloudDirector.Username
	password := cloud.VMwareCloudDirector.Password
	url := cloud.VMwareCloudDirector.URL
	organization := cloud.VMwareCloudDirector.Organization
	vdc := cloud.VMwareCloudDirector.VDC

	if username == "" || password == "" || url == "" || organization == "" || vdc == "" {
		return utilerrors.NewBadRequest("kubeone VMware Cloud Director credentials missing")
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialKubeOneSecret(ctx, masterClient, externalCluster, secretName, map[string][]byte{
		resources.VMwareCloudDirectorUsername:     []byte(username),
		resources.VMwareCloudDirectorPassword:     []byte(password),
		resources.VMwareCloudDirectorOrganization: []byte(organization),
		resources.VMwareCloudDirectorVDC:          []byte(vdc),
		resources.VMwareCloudDirectorURL:          []byte(url),
	})
	if err != nil {
		return err
	}

	// add secret key selectors to externalCluster object
	externalCluster.Spec.CloudSpec.KubeOne.CredentialsReference = *credentialRef
	return nil
}
func GetKubeOneCredentialsSecretName(cloud apiv2.KubeOneCloudSpec) string {
	if cloud.AWS != nil {
		return "credential-aws"
	}
	if cloud.Azure != nil {
		return "credential-azure"
	}
	if cloud.DigitalOcean != nil {
		return "credential-digitalocean"
	}
	if cloud.GCP != nil {
		return "credential-gcp"
	}
	if cloud.Hetzner != nil {
		return "credential-hetzner"
	}
	if cloud.OpenStack != nil {
		return "credential-openstack"
	}
	if cloud.Equinix != nil {
		return "credential-equinix"
	}
	if cloud.VMwareCloudDirector != nil {
		return "credential-vmware-cloud-director"
	}
	if cloud.VSphere != nil {
		return "credential-vsphere"
	}
	if cloud.Nutanix != nil {
		return "credential-nutanix"
	}
	return ""
}
