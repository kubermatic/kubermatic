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

package resources

import (
	"context"
	"errors"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/machine-controller/sdk/providerconfig"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Credentials struct {
	AWS                 AWSCredentials
	Azure               AzureCredentials
	Baremetal           BaremetalCredentials
	Digitalocean        DigitaloceanCredentials
	GCP                 GCPCredentials
	Hetzner             HetznerCredentials
	Openstack           OpenstackCredentials
	Kubevirt            KubevirtCredentials
	VSphere             VSphereCredentials
	Alibaba             AlibabaCredentials
	Anexia              AnexiaCredentials
	Nutanix             NutanixCredentials
	VMwareCloudDirector VMwareCloudDirectorCredentials
}

type AWSCredentials struct {
	AccessKeyID          string
	SecretAccessKey      string
	AssumeRoleARN        string
	AssumeRoleExternalID string
}

type AzureCredentials struct {
	TenantID       string
	SubscriptionID string
	ClientID       string
	ClientSecret   string
}

type BaremetalCredentials struct {
	Tinkerbell TinkerbellCredentials
}

type TinkerbellCredentials struct {
	// Admin kubeconfig for Tinkerbell cluster
	Kubeconfig string
}

type EKSCredentials struct {
	AccessKeyID          string
	SecretAccessKey      string
	AssumeRoleARN        string
	AssumeRoleExternalID string
}

type GKECredentials struct {
	ServiceAccount string
}
type AKSCredentials struct {
	TenantID       string
	SubscriptionID string
	ClientID       string
	ClientSecret   string
}

type EKSCredential struct {
	AccessKeyID          string
	SecretAccessKey      string
	Region               string
	AssumeRoleARN        string
	AssumeRoleExternalID string
}

type DigitaloceanCredentials struct {
	Token string
}

type GCPCredentials struct {
	ServiceAccount string
}

type HetznerCredentials struct {
	Token string
}

type OpenstackCredentials struct {
	Username                    string
	Password                    string
	Project                     string
	ProjectID                   string
	Domain                      string
	ApplicationCredentialID     string
	ApplicationCredentialSecret string
	Token                       string
}

type KubevirtCredentials struct {
	// Admin kubeconfig for KubeVirt cluster
	KubeConfig string
}

type VMwareCloudDirectorCredentials struct {
	Username     string
	Password     string
	APIToken     string
	Organization string
	VDC          string
}

type VSphereCredentials struct {
	Username string
	Password string
}

type AlibabaCredentials struct {
	AccessKeyID     string
	AccessKeySecret string
}

type AnexiaCredentials struct {
	Token string
}

type NutanixCredentials struct {
	Username    string
	Password    string
	CSIUsername string
	CSIPassword string
	ProxyURL    string
}

type CredentialsData interface {
	Cluster() *kubermaticv1.Cluster
	GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)
}

func NewCredentialsData(ctx context.Context, cluster *kubermaticv1.Cluster, client ctrlruntimeclient.Client) CredentialsData {
	return &credentialsData{
		cluster:                          cluster,
		globalSecretKeySelectorValueFunc: provider.SecretKeySelectorValueFuncFactory(ctx, client),
	}
}

type credentialsData struct {
	cluster                          *kubermaticv1.Cluster
	globalSecretKeySelectorValueFunc provider.SecretKeySelectorValueFunc
}

func (cd *credentialsData) Cluster() *kubermaticv1.Cluster {
	return cd.cluster
}

func (cd *credentialsData) GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error) {
	return cd.globalSecretKeySelectorValueFunc(configVar, key)
}

// GetCredentialsReference returns the CredentialsReference for the cluster's chosen
// cloud provider (or nil if the provider is BYO). If an unknown provider is used, an
// error is returned.
func GetCredentialsReference(cluster *kubermaticv1.Cluster) (*providerconfig.GlobalSecretKeySelector, error) {
	if cluster.Spec.Cloud.AWS != nil {
		return cluster.Spec.Cloud.AWS.CredentialsReference, nil
	}
	if cluster.Spec.Cloud.Azure != nil {
		return cluster.Spec.Cloud.Azure.CredentialsReference, nil
	}
	if cluster.Spec.Cloud.Baremetal != nil {
		return cluster.Spec.Cloud.Baremetal.CredentialsReference, nil
	}
	if cluster.Spec.Cloud.Digitalocean != nil {
		return cluster.Spec.Cloud.Digitalocean.CredentialsReference, nil
	}
	if cluster.Spec.Cloud.GCP != nil {
		return cluster.Spec.Cloud.GCP.CredentialsReference, nil
	}
	if cluster.Spec.Cloud.Hetzner != nil {
		return cluster.Spec.Cloud.Hetzner.CredentialsReference, nil
	}
	if cluster.Spec.Cloud.Openstack != nil {
		return cluster.Spec.Cloud.Openstack.CredentialsReference, nil
	}
	if cluster.Spec.Cloud.Kubevirt != nil {
		return cluster.Spec.Cloud.Kubevirt.CredentialsReference, nil
	}
	if cluster.Spec.Cloud.VSphere != nil {
		return cluster.Spec.Cloud.VSphere.CredentialsReference, nil
	}
	if cluster.Spec.Cloud.Alibaba != nil {
		return cluster.Spec.Cloud.Alibaba.CredentialsReference, nil
	}
	if cluster.Spec.Cloud.Anexia != nil {
		return cluster.Spec.Cloud.Anexia.CredentialsReference, nil
	}
	if cluster.Spec.Cloud.Nutanix != nil {
		return cluster.Spec.Cloud.Nutanix.CredentialsReference, nil
	}
	if cluster.Spec.Cloud.VMwareCloudDirector != nil {
		return cluster.Spec.Cloud.VMwareCloudDirector.CredentialsReference, nil
	}
	if cluster.Spec.Cloud.BringYourOwn != nil {
		return nil, nil
	}
	if cluster.Spec.Cloud.Edge != nil {
		return nil, nil
	}

	return nil, errors.New("cluster has no known cloud provider spec set")
}

func GetCredentials(data CredentialsData) (Credentials, error) {
	credentials := Credentials{}
	var err error

	if data.Cluster().Spec.Cloud.AWS != nil {
		if credentials.AWS, err = GetAWSCredentials(data); err != nil {
			return Credentials{}, err
		}
	}
	if data.Cluster().Spec.Cloud.Azure != nil {
		if credentials.Azure, err = GetAzureCredentials(data); err != nil {
			return Credentials{}, err
		}
	}

	if data.Cluster().Spec.Cloud.Baremetal != nil {
		if credentials.Baremetal, err = GetBaremetalCredentials(data); err != nil {
			return Credentials{}, err
		}
	}

	if data.Cluster().Spec.Cloud.Digitalocean != nil {
		if credentials.Digitalocean, err = GetDigitaloceanCredentials(data); err != nil {
			return Credentials{}, err
		}
	}
	if data.Cluster().Spec.Cloud.GCP != nil {
		if credentials.GCP, err = GetGCPCredentials(data); err != nil {
			return Credentials{}, err
		}
	}
	if data.Cluster().Spec.Cloud.Hetzner != nil {
		if credentials.Hetzner, err = GetHetznerCredentials(data); err != nil {
			return Credentials{}, err
		}
	}
	if data.Cluster().Spec.Cloud.Openstack != nil {
		if credentials.Openstack, err = GetOpenstackCredentials(data); err != nil {
			return Credentials{}, err
		}
	}
	if data.Cluster().Spec.Cloud.Kubevirt != nil {
		if credentials.Kubevirt, err = GetKubevirtCredentials(data); err != nil {
			return Credentials{}, err
		}
	}
	if data.Cluster().Spec.Cloud.VSphere != nil {
		if credentials.VSphere, err = GetVSphereCredentials(data); err != nil {
			return Credentials{}, err
		}
	}
	if data.Cluster().Spec.Cloud.Alibaba != nil {
		if credentials.Alibaba, err = GetAlibabaCredentials(data); err != nil {
			return Credentials{}, err
		}
	}
	if data.Cluster().Spec.Cloud.Anexia != nil {
		if credentials.Anexia, err = GetAnexiaCredentials(data); err != nil {
			return Credentials{}, err
		}
	}

	if data.Cluster().Spec.Cloud.Nutanix != nil {
		if credentials.Nutanix, err = GetNutanixCredentials(data); err != nil {
			return Credentials{}, err
		}
	}

	if data.Cluster().Spec.Cloud.VMwareCloudDirector != nil {
		if credentials.VMwareCloudDirector, err = GetVMwareCloudDirectorCredentials(data); err != nil {
			return Credentials{}, err
		}
	}

	return credentials, err
}

func CopyCredentials(data CredentialsData, cluster *kubermaticv1.Cluster) error {
	credentials := Credentials{}
	var err error

	if data.Cluster().Spec.Cloud.AWS != nil {
		if credentials.AWS, err = GetAWSCredentials(data); err != nil {
			return err
		}
		cluster.Spec.Cloud.AWS.AccessKeyID = credentials.AWS.AccessKeyID
		cluster.Spec.Cloud.AWS.SecretAccessKey = credentials.AWS.SecretAccessKey
		cluster.Spec.Cloud.AWS.AssumeRoleARN = credentials.AWS.AssumeRoleARN
		cluster.Spec.Cloud.AWS.AssumeRoleExternalID = credentials.AWS.AssumeRoleExternalID
	}
	if data.Cluster().Spec.Cloud.Azure != nil {
		if credentials.Azure, err = GetAzureCredentials(data); err != nil {
			return err
		}
		cluster.Spec.Cloud.Azure.TenantID = credentials.Azure.TenantID
		cluster.Spec.Cloud.Azure.ClientID = credentials.Azure.ClientID
		cluster.Spec.Cloud.Azure.ClientSecret = credentials.Azure.ClientSecret
		cluster.Spec.Cloud.Azure.SubscriptionID = credentials.Azure.SubscriptionID
	}

	if data.Cluster().Spec.Cloud.Baremetal != nil {
		if credentials.Baremetal, err = GetBaremetalCredentials(data); err != nil {
			return err
		}
		if cluster.Spec.Cloud.Baremetal.Tinkerbell != nil {
			cluster.Spec.Cloud.Baremetal.Tinkerbell.Kubeconfig = credentials.Baremetal.Tinkerbell.Kubeconfig
		}
	}

	if data.Cluster().Spec.Cloud.Digitalocean != nil {
		if credentials.Digitalocean, err = GetDigitaloceanCredentials(data); err != nil {
			return err
		}
		cluster.Spec.Cloud.Digitalocean.Token = credentials.Digitalocean.Token
	}
	if data.Cluster().Spec.Cloud.GCP != nil {
		if credentials.GCP, err = GetGCPCredentials(data); err != nil {
			return err
		}
		cluster.Spec.Cloud.GCP.ServiceAccount = credentials.GCP.ServiceAccount
	}
	if data.Cluster().Spec.Cloud.Hetzner != nil {
		if credentials.Hetzner, err = GetHetznerCredentials(data); err != nil {
			return err
		}
		cluster.Spec.Cloud.Hetzner.Token = credentials.Hetzner.Token
	}
	if data.Cluster().Spec.Cloud.Openstack != nil {
		if credentials.Openstack, err = GetOpenstackCredentials(data); err != nil {
			return err
		}
		cluster.Spec.Cloud.Openstack.Token = credentials.Openstack.Token
		cluster.Spec.Cloud.Openstack.ProjectID = credentials.Openstack.ProjectID
		cluster.Spec.Cloud.Openstack.Project = credentials.Openstack.Project
		cluster.Spec.Cloud.Openstack.Domain = credentials.Openstack.Domain
		cluster.Spec.Cloud.Openstack.ApplicationCredentialID = credentials.Openstack.ApplicationCredentialID
		cluster.Spec.Cloud.Openstack.ApplicationCredentialSecret = credentials.Openstack.ApplicationCredentialSecret
		cluster.Spec.Cloud.Openstack.Password = credentials.Openstack.Password
		cluster.Spec.Cloud.Openstack.Username = credentials.Openstack.Username
	}

	if data.Cluster().Spec.Cloud.Kubevirt != nil {
		if credentials.Kubevirt, err = GetKubevirtCredentials(data); err != nil {
			return err
		}
		cluster.Spec.Cloud.Kubevirt.Kubeconfig = credentials.Kubevirt.KubeConfig
	}
	if data.Cluster().Spec.Cloud.VSphere != nil {
		if credentials.VSphere, err = GetVSphereCredentials(data); err != nil {
			return err
		}
		cluster.Spec.Cloud.VSphere.Username = credentials.VSphere.Username
		cluster.Spec.Cloud.VSphere.Password = credentials.VSphere.Password
	}
	if data.Cluster().Spec.Cloud.Alibaba != nil {
		if credentials.Alibaba, err = GetAlibabaCredentials(data); err != nil {
			return err
		}
		cluster.Spec.Cloud.Alibaba.AccessKeyID = credentials.Alibaba.AccessKeyID
		cluster.Spec.Cloud.Alibaba.AccessKeySecret = credentials.Alibaba.AccessKeySecret
	}
	if data.Cluster().Spec.Cloud.Anexia != nil {
		if credentials.Anexia, err = GetAnexiaCredentials(data); err != nil {
			return err
		}
		cluster.Spec.Cloud.Anexia.Token = credentials.Anexia.Token
	}

	return nil
}

func GetAWSCredentials(data CredentialsData) (AWSCredentials, error) {
	spec := data.Cluster().Spec.Cloud.AWS
	awsCredentials := AWSCredentials{}
	var err error

	if spec.AccessKeyID != "" {
		awsCredentials.AccessKeyID = spec.AccessKeyID
	} else if awsCredentials.AccessKeyID, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, AWSAccessKeyID); err != nil {
		return AWSCredentials{}, err
	}

	if spec.SecretAccessKey != "" {
		awsCredentials.SecretAccessKey = spec.SecretAccessKey
	} else if awsCredentials.SecretAccessKey, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, AWSSecretAccessKey); err != nil {
		return AWSCredentials{}, err
	}

	// AssumeRole credentials are optional. They are allowed to be empty
	awsCredentials.AssumeRoleARN = spec.AssumeRoleARN
	awsCredentials.AssumeRoleExternalID = spec.AssumeRoleExternalID

	return awsCredentials, nil
}

func GetGKECredentials(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.ExternalCluster) (GKECredentials, error) {
	spec := cluster.Spec.CloudSpec.GKE
	gkeCredentials := GKECredentials{}
	GetGlobalSecretKeySelectorValue := provider.SecretKeySelectorValueFuncFactory(ctx, client)
	var err error

	if spec.ServiceAccount != "" {
		gkeCredentials.ServiceAccount = spec.ServiceAccount
	} else if gkeCredentials.ServiceAccount, err = GetGlobalSecretKeySelectorValue(spec.CredentialsReference, GCPServiceAccount); err != nil {
		return GKECredentials{}, err
	}

	return gkeCredentials, nil
}

func GetEKSCredentials(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.ExternalCluster) (EKSCredentials, error) {
	spec := cluster.Spec.CloudSpec.EKS
	eksCredentials := EKSCredentials{}
	GetGlobalSecretKeySelectorValue := provider.SecretKeySelectorValueFuncFactory(ctx, client)
	var err error

	if spec.AccessKeyID != "" {
		eksCredentials.AccessKeyID = spec.AccessKeyID
	} else if eksCredentials.AccessKeyID, err = GetGlobalSecretKeySelectorValue(spec.CredentialsReference, AWSAccessKeyID); err != nil {
		return EKSCredentials{}, err
	}

	if spec.SecretAccessKey != "" {
		eksCredentials.SecretAccessKey = spec.SecretAccessKey
	} else if eksCredentials.SecretAccessKey, err = GetGlobalSecretKeySelectorValue(spec.CredentialsReference, AWSSecretAccessKey); err != nil {
		return EKSCredentials{}, err
	}

	return eksCredentials, nil
}

func GetAKSCredentials(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.ExternalCluster) (AKSCredentials, error) {
	spec := cluster.Spec.CloudSpec.AKS
	aksCredentials := AKSCredentials{}
	GetGlobalSecretKeySelectorValue := provider.SecretKeySelectorValueFuncFactory(ctx, client)
	var err error

	if spec.TenantID != "" {
		aksCredentials.TenantID = spec.TenantID
	} else if aksCredentials.TenantID, err = GetGlobalSecretKeySelectorValue(spec.CredentialsReference, AzureTenantID); err != nil {
		return AKSCredentials{}, err
	}

	if spec.SubscriptionID != "" {
		aksCredentials.SubscriptionID = spec.SubscriptionID
	} else if aksCredentials.SubscriptionID, err = GetGlobalSecretKeySelectorValue(spec.CredentialsReference, AzureSubscriptionID); err != nil {
		return AKSCredentials{}, err
	}

	if spec.ClientID != "" {
		aksCredentials.ClientID = spec.ClientID
	} else if aksCredentials.ClientID, err = GetGlobalSecretKeySelectorValue(spec.CredentialsReference, AzureClientID); err != nil {
		return AKSCredentials{}, err
	}

	if spec.ClientSecret != "" {
		aksCredentials.ClientSecret = spec.ClientSecret
	} else if aksCredentials.ClientSecret, err = GetGlobalSecretKeySelectorValue(spec.CredentialsReference, AzureClientSecret); err != nil {
		return AKSCredentials{}, err
	}

	return aksCredentials, nil
}

func GetAzureCredentials(data CredentialsData) (AzureCredentials, error) {
	spec := data.Cluster().Spec.Cloud.Azure
	azureCredentials := AzureCredentials{}
	var err error

	if spec.TenantID != "" {
		azureCredentials.TenantID = spec.TenantID
	} else if azureCredentials.TenantID, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, AzureTenantID); err != nil {
		return AzureCredentials{}, err
	}

	if spec.SubscriptionID != "" {
		azureCredentials.SubscriptionID = spec.SubscriptionID
	} else if azureCredentials.SubscriptionID, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, AzureSubscriptionID); err != nil {
		return AzureCredentials{}, err
	}

	if spec.ClientID != "" {
		azureCredentials.ClientID = spec.ClientID
	} else if azureCredentials.ClientID, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, AzureClientID); err != nil {
		return AzureCredentials{}, err
	}

	if spec.ClientSecret != "" {
		azureCredentials.ClientSecret = spec.ClientSecret
	} else if azureCredentials.ClientSecret, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, AzureClientSecret); err != nil {
		return AzureCredentials{}, err
	}

	return azureCredentials, nil
}

func GetBaremetalCredentials(data CredentialsData) (BaremetalCredentials, error) {
	spec := data.Cluster().Spec.Cloud.Baremetal
	baremetalCredentials := BaremetalCredentials{}
	var err error

	if spec.Tinkerbell != nil && spec.Tinkerbell.Kubeconfig != "" {
		baremetalCredentials.Tinkerbell.Kubeconfig = spec.Tinkerbell.Kubeconfig
	} else if baremetalCredentials.Tinkerbell.Kubeconfig, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, TinkerbellKubeconfig); err != nil {
		return BaremetalCredentials{}, err
	}

	return baremetalCredentials, nil
}

func GetDigitaloceanCredentials(data CredentialsData) (DigitaloceanCredentials, error) {
	spec := data.Cluster().Spec.Cloud.Digitalocean
	digitaloceanCredentials := DigitaloceanCredentials{}
	var err error

	if spec.Token != "" {
		digitaloceanCredentials.Token = spec.Token
	} else if digitaloceanCredentials.Token, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, DigitaloceanToken); err != nil {
		return DigitaloceanCredentials{}, err
	}

	return digitaloceanCredentials, nil
}

func GetGCPCredentials(data CredentialsData) (GCPCredentials, error) {
	spec := data.Cluster().Spec.Cloud.GCP
	gcpCredentials := GCPCredentials{}
	var err error

	if spec.ServiceAccount != "" {
		gcpCredentials.ServiceAccount = spec.ServiceAccount
	} else if gcpCredentials.ServiceAccount, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, GCPServiceAccount); err != nil {
		return GCPCredentials{}, err
	}

	return gcpCredentials, nil
}

func GetHetznerCredentials(data CredentialsData) (HetznerCredentials, error) {
	spec := data.Cluster().Spec.Cloud.Hetzner
	hetznerCredentials := HetznerCredentials{}
	var err error

	if spec.Token != "" {
		hetznerCredentials.Token = spec.Token
	} else if hetznerCredentials.Token, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, HetznerToken); err != nil {
		return HetznerCredentials{}, err
	}

	return hetznerCredentials, nil
}

func GetOpenstackCredentials(data CredentialsData) (OpenstackCredentials, error) {
	spec := data.Cluster().Spec.Cloud.Openstack
	openstackCredentials := OpenstackCredentials{}
	var err error

	// needed for cluster creation with other credentials
	if spec.Domain != "" {
		openstackCredentials.Domain = spec.Domain
	} else if openstackCredentials.Domain, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, OpenstackDomain); err != nil {
		return OpenstackCredentials{}, err
	}

	if spec.ApplicationCredentialID != "" {
		openstackCredentials.ApplicationCredentialID = spec.ApplicationCredentialID
		openstackCredentials.ApplicationCredentialSecret = spec.ApplicationCredentialSecret
		return openstackCredentials, err
	} else if openstackCredentials.ApplicationCredentialID, _ = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, OpenstackApplicationCredentialID); openstackCredentials.ApplicationCredentialID != "" {
		openstackCredentials.ApplicationCredentialSecret, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, OpenstackApplicationCredentialSecret)
		if err != nil {
			return OpenstackCredentials{}, err
		}
		return openstackCredentials, err
	}

	if spec.Username != "" {
		openstackCredentials.Username = spec.Username
	} else if openstackCredentials.Username, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, OpenstackUsername); err != nil {
		return OpenstackCredentials{}, err
	}

	if spec.Password != "" {
		openstackCredentials.Password = spec.Password
	} else if openstackCredentials.Password, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, OpenstackPassword); err != nil {
		return OpenstackCredentials{}, err
	}

	if spec.Project != "" {
		openstackCredentials.Project = spec.Project
	} else if spec.CredentialsReference != nil && spec.CredentialsReference.Name != "" {
		if openstackCredentials.Project, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, OpenstackProject); err != nil {
			// fallback to tenant
			if openstackCredentials.Project, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, OpenstackTenant); err != nil {
				return OpenstackCredentials{}, err
			}
		}
	}

	if spec.ProjectID != "" {
		openstackCredentials.ProjectID = spec.ProjectID
	} else if spec.CredentialsReference != nil && spec.CredentialsReference.Name != "" {
		if openstackCredentials.ProjectID, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, OpenstackProjectID); err != nil {
			// fallback to tenantID
			if openstackCredentials.ProjectID, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, OpenstackTenantID); err != nil {
				return OpenstackCredentials{}, err
			}
		}
	}

	return openstackCredentials, nil
}

func GetKubevirtCredentials(data CredentialsData) (KubevirtCredentials, error) {
	spec := data.Cluster().Spec.Cloud.Kubevirt
	kubevirtCredentials := KubevirtCredentials{}
	var err error

	if spec.Kubeconfig != "" {
		kubevirtCredentials.KubeConfig = spec.Kubeconfig
	} else if kubevirtCredentials.KubeConfig, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, KubeVirtKubeconfig); err != nil {
		return KubevirtCredentials{}, err
	}

	return kubevirtCredentials, nil
}

func GetVSphereCredentials(data CredentialsData) (VSphereCredentials, error) {
	spec := data.Cluster().Spec.Cloud.VSphere
	var username, password string
	var err error

	if spec.Username != "" {
		username = spec.Username
	} else if spec.CredentialsReference != nil && spec.CredentialsReference.Name != "" {
		username, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, VsphereUsername)
		if err != nil {
			return VSphereCredentials{}, err
		}
	}

	if username == "" {
		if spec.InfraManagementUser.Username != "" {
			username = spec.InfraManagementUser.Username
		} else if spec.CredentialsReference != nil && spec.CredentialsReference.Name != "" {
			username, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, VsphereInfraManagementUserUsername)
			if err != nil {
				return VSphereCredentials{}, err
			}
		}
	}

	if spec.Password != "" {
		password = spec.Password
	} else if spec.CredentialsReference != nil && spec.CredentialsReference.Name != "" {
		password, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, VspherePassword)
		if err != nil {
			return VSphereCredentials{}, err
		}
	}

	if password == "" {
		if spec.InfraManagementUser.Password != "" {
			password = spec.InfraManagementUser.Password
		} else if spec.CredentialsReference != nil && spec.CredentialsReference.Name != "" {
			password, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, VsphereInfraManagementUserPassword)
			if err != nil {
				return VSphereCredentials{}, err
			}
		}
	}

	return VSphereCredentials{
		Username: username,
		Password: password,
	}, nil
}

func GetAlibabaCredentials(data CredentialsData) (AlibabaCredentials, error) {
	spec := data.Cluster().Spec.Cloud.Alibaba
	alibabaCredentials := AlibabaCredentials{}
	var err error

	if spec.AccessKeyID != "" {
		alibabaCredentials.AccessKeyID = spec.AccessKeyID
	} else if alibabaCredentials.AccessKeyID, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, AlibabaAccessKeyID); err != nil {
		return AlibabaCredentials{}, err
	}

	if spec.AccessKeySecret != "" {
		alibabaCredentials.AccessKeySecret = spec.AccessKeySecret
	} else if alibabaCredentials.AccessKeySecret, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, AlibabaAccessKeySecret); err != nil {
		return AlibabaCredentials{}, err
	}

	return alibabaCredentials, nil
}

func GetAnexiaCredentials(data CredentialsData) (AnexiaCredentials, error) {
	spec := data.Cluster().Spec.Cloud.Anexia
	anexiaCredentials := AnexiaCredentials{}
	var err error

	if spec.Token != "" {
		anexiaCredentials.Token = spec.Token
	} else if anexiaCredentials.Token, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, AnexiaToken); err != nil {
		return AnexiaCredentials{}, err
	}

	return anexiaCredentials, nil
}

func GetNutanixCredentials(data CredentialsData) (NutanixCredentials, error) {
	spec := data.Cluster().Spec.Cloud.Nutanix
	credentials := NutanixCredentials{}
	var err error

	if spec.Username != "" {
		credentials.Username = spec.Username
	} else if credentials.Username, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, NutanixUsername); err != nil {
		return NutanixCredentials{}, err
	}

	if spec.Password != "" {
		credentials.Password = spec.Password
	} else if credentials.Password, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, NutanixPassword); err != nil {
		return NutanixCredentials{}, err
	}

	if spec.ProxyURL != "" {
		credentials.ProxyURL = spec.ProxyURL
	} else {
		credentials.ProxyURL, _ = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, NutanixProxyURL)
	}

	if spec.CSI == nil {
		return credentials, nil
	}

	if spec.CSI.Username != "" {
		credentials.CSIUsername = spec.CSI.Username
	} else if credentials.CSIUsername, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, NutanixCSIUsername); err != nil {
		return NutanixCredentials{}, err
	}

	if spec.CSI.Password != "" {
		credentials.CSIPassword = spec.CSI.Password
	} else if credentials.CSIPassword, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, NutanixCSIPassword); err != nil {
		return NutanixCredentials{}, err
	}

	return credentials, nil
}

func GetVMwareCloudDirectorCredentials(data CredentialsData) (VMwareCloudDirectorCredentials, error) {
	spec := data.Cluster().Spec.Cloud.VMwareCloudDirector
	credentials := VMwareCloudDirectorCredentials{}
	var err error

	if spec.Organization != "" {
		credentials.Organization = spec.Organization
	} else if credentials.Organization, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, VMwareCloudDirectorOrganization); err != nil {
		return VMwareCloudDirectorCredentials{}, err
	}

	if spec.VDC != "" {
		credentials.VDC = spec.VDC
	} else if credentials.VDC, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, VMwareCloudDirectorVDC); err != nil {
		return VMwareCloudDirectorCredentials{}, err
	}

	if spec.APIToken != "" {
		credentials.APIToken = spec.APIToken
	} else {
		credentials.APIToken, _ = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, VMwareCloudDirectorAPIToken)
	}

	if credentials.APIToken != "" {
		return credentials, nil
	}

	if spec.Username != "" {
		credentials.Username = spec.Username
	} else if credentials.Username, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, VMwareCloudDirectorUsername); err != nil {
		return VMwareCloudDirectorCredentials{}, err
	}

	if spec.Password != "" {
		credentials.Password = spec.Password
	} else if credentials.Password, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, VMwareCloudDirectorPassword); err != nil {
		return VMwareCloudDirectorCredentials{}, err
	}

	return credentials, nil
}
