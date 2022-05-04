/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package types

import (
	"flag"
	"fmt"
	"os"
)

type Secrets struct {
	AWS struct {
		AccessKeyID     string
		SecretAccessKey string
	}
	Azure struct {
		ClientID       string
		ClientSecret   string
		TenantID       string
		SubscriptionID string
	}
	Digitalocean struct {
		Token string
	}
	Hetzner struct {
		Token string
	}
	OpenStack struct {
		Domain    string
		Project   string
		ProjectID string
		Username  string
		Password  string
	}
	VSphere struct {
		Username  string
		Password  string
		Datastore string
	}
	Packet struct {
		APIKey    string
		ProjectID string
	}
	GCP struct {
		ServiceAccount string
		Network        string
		Subnetwork     string
		Zone           string
	}
	Kubevirt struct {
		Kubeconfig string
	}
	Alibaba struct {
		AccessKeyID     string
		AccessKeySecret string
	}
	Nutanix struct {
		Username    string
		Password    string
		CSIUsername string
		CSIPassword string
		CSIEndpoint string
		CSIPort     int32
		ProxyURL    string
		ClusterName string
		ProjectName string
		SubnetName  string
	}
}

var (
	kubevirtKubeconfigFile string
)

func (s *Secrets) AddFlags() {
	flag.StringVar(&s.AWS.AccessKeyID, "aws-access-key-id", "", "AWS: AccessKeyID")
	flag.StringVar(&s.AWS.SecretAccessKey, "aws-secret-access-key", "", "AWS: SecretAccessKey")
	flag.StringVar(&s.Digitalocean.Token, "digitalocean-token", "", "Digitalocean: API Token")
	flag.StringVar(&s.Hetzner.Token, "hetzner-token", "", "Hetzner: API Token")
	flag.StringVar(&s.OpenStack.Domain, "openstack-domain", "", "OpenStack: Domain")
	flag.StringVar(&s.OpenStack.Project, "openstack-project", "", "OpenStack: Project")
	flag.StringVar(&s.OpenStack.ProjectID, "openstack-project-id", "", "OpenStack: Project ID")
	flag.StringVar(&s.OpenStack.Username, "openstack-username", "", "OpenStack: Username")
	flag.StringVar(&s.OpenStack.Password, "openstack-password", "", "OpenStack: Password")
	flag.StringVar(&s.VSphere.Username, "vsphere-username", "", "vSphere: Username")
	flag.StringVar(&s.VSphere.Password, "vsphere-password", "", "vSphere: Password")
	flag.StringVar(&s.VSphere.Datastore, "vsphere-datastore", "", "vSphere: Datastore")
	flag.StringVar(&s.Azure.ClientID, "azure-client-id", "", "Azure: ClientID")
	flag.StringVar(&s.Azure.ClientSecret, "azure-client-secret", "", "Azure: ClientSecret")
	flag.StringVar(&s.Azure.TenantID, "azure-tenant-id", "", "Azure: TenantID")
	flag.StringVar(&s.Azure.SubscriptionID, "azure-subscription-id", "", "Azure: SubscriptionID")
	flag.StringVar(&s.Packet.APIKey, "packet-api-key", "", "Packet: APIKey")
	flag.StringVar(&s.Packet.ProjectID, "packet-project-id", "", "Packet: ProjectID")
	flag.StringVar(&s.GCP.ServiceAccount, "gcp-service-account", "", "GCP: Service Account")
	flag.StringVar(&s.GCP.Zone, "gcp-zone", "europe-west3-c", "GCP: Zone")
	flag.StringVar(&s.GCP.Network, "gcp-network", "", "GCP: Network")
	flag.StringVar(&s.GCP.Subnetwork, "gcp-subnetwork", "", "GCP: Subnetwork")
	flag.StringVar(&kubevirtKubeconfigFile, "kubevirt-kubeconfig", "", "Kubevirt: Cluster Kubeconfig filename")
	flag.StringVar(&s.Alibaba.AccessKeyID, "alibaba-access-key-id", "", "Alibaba: AccessKeyID")
	flag.StringVar(&s.Alibaba.AccessKeySecret, "alibaba-access-key-secret", "", "Alibaba: AccessKeySecret")
	flag.StringVar(&s.Nutanix.Username, "nutanix-username", "", "Nutanix: Username")
	flag.StringVar(&s.Nutanix.Password, "nutanix-password", "", "Nutanix: Password")
	flag.StringVar(&s.Nutanix.CSIUsername, "nutanix-csi-username", "", "Nutanix CSI Prism Element: Username")
	flag.StringVar(&s.Nutanix.CSIPassword, "nutanix-csi-password", "", "Nutanix CSI Prism Element: Password")
	flag.StringVar(&s.Nutanix.CSIEndpoint, "nutanix-csi-endpoint", "", "Nutanix CSI Prism Element: Endpoint")
	flag.StringVar(&s.Nutanix.ProxyURL, "nutanix-proxy-url", "", "Nutanix: HTTP Proxy URL to access endpoint")
	flag.StringVar(&s.Nutanix.ClusterName, "nutanix-cluster-name", "", "Nutanix: Cluster Name")
	flag.StringVar(&s.Nutanix.ProjectName, "nutanix-project-name", "", "Nutanix: Project Name")
	flag.StringVar(&s.Nutanix.SubnetName, "nutanix-subnet-name", "", "Nutanix: Subnet Name")
}

func (s *Secrets) ParseFlags() error {
	if kubevirtKubeconfigFile != "" {
		content, err := os.ReadFile(kubevirtKubeconfigFile)
		if err != nil {
			return fmt.Errorf("failed to read kubevirt kubeconfig file: %w", err)
		}

		s.Kubevirt.Kubeconfig = string(content)
	}

	return nil
}
