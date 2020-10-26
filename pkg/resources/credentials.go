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

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Credentials struct {
	AWS          AWSCredentials
	Azure        AzureCredentials
	Digitalocean DigitaloceanCredentials
	GCP          GCPCredentials
	Hetzner      HetznerCredentials
	Openstack    OpenstackCredentials
	Packet       PacketCredentials
	Kubevirt     KubevirtCredentials
	VSphere      VSphereCredentials
	Alibaba      AlibabaCredentials
	Anexia       AnexiaCredentials
}

type AWSCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
}

type AzureCredentials struct {
	TenantID       string
	SubscriptionID string
	ClientID       string
	ClientSecret   string
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
	Username string
	Password string
	Tenant   string
	TenantID string
	Domain   string
}

type PacketCredentials struct {
	APIKey    string
	ProjectID string
}

type KubevirtCredentials struct {
	KubeConfig string
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
	if data.Cluster().Spec.Cloud.Packet != nil {
		if credentials.Packet, err = GetPacketCredentials(data); err != nil {
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

	return credentials, err
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

	return awsCredentials, nil
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

	if spec.Tenant != "" {
		openstackCredentials.Tenant = spec.Tenant
	} else if spec.CredentialsReference != nil && spec.CredentialsReference.Name != "" {
		if openstackCredentials.Tenant, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, OpenstackTenant); err != nil {
			return OpenstackCredentials{}, err
		}
	}

	if spec.TenantID != "" {
		openstackCredentials.TenantID = spec.TenantID
	} else if spec.CredentialsReference != nil && spec.CredentialsReference.Name != "" {
		if openstackCredentials.TenantID, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, OpenstackTenantID); err != nil {
			return OpenstackCredentials{}, err
		}
	}

	if spec.Domain != "" {
		openstackCredentials.Domain = spec.Domain
	} else if openstackCredentials.Domain, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, OpenstackDomain); err != nil {
		return OpenstackCredentials{}, err
	}

	return openstackCredentials, nil
}

func GetPacketCredentials(data CredentialsData) (PacketCredentials, error) {
	spec := data.Cluster().Spec.Cloud.Packet
	packetCredentials := PacketCredentials{}
	var err error

	if spec.APIKey != "" {
		packetCredentials.APIKey = spec.APIKey
	} else if packetCredentials.APIKey, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, PacketAPIKey); err != nil {
		return PacketCredentials{}, err
	}

	if spec.ProjectID != "" {
		packetCredentials.ProjectID = spec.ProjectID
	} else if packetCredentials.ProjectID, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, PacketProjectID); err != nil {
		return PacketCredentials{}, err
	}

	return packetCredentials, nil
}

func GetKubevirtCredentials(data CredentialsData) (KubevirtCredentials, error) {
	spec := data.Cluster().Spec.Cloud.Kubevirt
	kubevirtCredentials := KubevirtCredentials{}
	var err error

	if spec.Kubeconfig != "" {
		kubevirtCredentials.KubeConfig = spec.Kubeconfig
	} else if kubevirtCredentials.KubeConfig, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, KubevirtKubeConfig); err != nil {
		return KubevirtCredentials{}, err
	}

	return kubevirtCredentials, nil
}

func GetVSphereCredentials(data CredentialsData) (VSphereCredentials, error) {
	spec := data.Cluster().Spec.Cloud.VSphere
	var username, password string
	var err error

	if spec.InfraManagementUser.Username != "" {
		username = spec.InfraManagementUser.Username
	} else if spec.CredentialsReference != nil && spec.CredentialsReference.Name != "" {
		username, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, VsphereInfraManagementUserUsername)
		if err != nil {
			return VSphereCredentials{}, err
		}
	}

	if spec.InfraManagementUser.Password != "" {
		password = spec.InfraManagementUser.Password
	} else if spec.CredentialsReference != nil && spec.CredentialsReference.Name != "" {
		password, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, VsphereInfraManagementUserPassword)
		if err != nil {
			return VSphereCredentials{}, err
		}
	}

	if username == "" && spec.Username != "" {
		username = spec.Username
	} else if spec.CredentialsReference != nil && spec.CredentialsReference.Name != "" {
		username, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, VsphereUsername)
		if err != nil {
			return VSphereCredentials{}, err
		}
	}

	if password == "" && spec.Password != "" {
		password = spec.Password
	} else if spec.CredentialsReference != nil && spec.CredentialsReference.Name != "" {
		password, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, VspherePassword)
		if err != nil {
			return VSphereCredentials{}, err
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
