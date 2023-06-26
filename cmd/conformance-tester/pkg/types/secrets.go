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

	"k8c.io/kubermatic/v2/pkg/test"
)

type Secrets struct {
	Anexia struct {
		KKPDatacenter string
		Token         string
		TemplateID    string
		VlanID        string
	}
	AWS struct {
		KKPDatacenter   string
		AccessKeyID     string
		SecretAccessKey string
	}
	Azure struct {
		KKPDatacenter  string
		ClientID       string
		ClientSecret   string
		TenantID       string
		SubscriptionID string
	}
	Digitalocean struct {
		KKPDatacenter string
		Token         string
	}
	Hetzner struct {
		KKPDatacenter string
		Token         string
	}
	OpenStack struct {
		KKPDatacenter string
		Domain        string
		Project       string
		ProjectID     string
		Username      string
		Password      string
	}
	VSphere struct {
		KKPDatacenter string
		Username      string
		Password      string
	}
	Packet struct {
		KKPDatacenter string
		APIKey        string
		ProjectID     string
	}
	GCP struct {
		KKPDatacenter string
		// ServiceAccount is the plaintext Service account (as JSON) without any (base64) encoding.
		ServiceAccount string
		Network        string
		Subnetwork     string
	}
	Kubevirt struct {
		KKPDatacenter string
		// Kubeconfig is the plaintext kubeconfig without any (base64) encoding.
		Kubeconfig string
	}
	Alibaba struct {
		KKPDatacenter   string
		AccessKeyID     string
		AccessKeySecret string
	}
	Nutanix struct {
		KKPDatacenter string
		Username      string
		Password      string
		CSIUsername   string
		CSIPassword   string
		CSIEndpoint   string
		CSIPort       int32
		ProxyURL      string
		ClusterName   string
		ProjectName   string
		SubnetName    string
	}
	VMwareCloudDirector struct {
		KKPDatacenter string
		Username      string
		Password      string
		Organization  string
		VDC           string
		OVDCNetwork   string
	}
	RHEL struct {
		SubscriptionUser     string
		SubscriptionPassword string
		OfflineToken         string
	}
}

var (
	kubevirtKubeconfigFile string
)

func (s *Secrets) AddFlags() {
	flag.StringVar(&s.Anexia.Token, "anexia-token", "", "Anexia: API Token")
	flag.StringVar(&s.Anexia.TemplateID, "anexia-template-id", "", "Anexia: Template ID")
	flag.StringVar(&s.Anexia.VlanID, "anexia-vlan-id", "", "Anexia: VLAN ID")
	flag.StringVar(&s.Anexia.KKPDatacenter, "anexia-kkp-datacenter", "", "Anexia: KKP datacenter to use")
	flag.StringVar(&s.AWS.AccessKeyID, "aws-access-key-id", "", "AWS: AccessKeyID")
	flag.StringVar(&s.AWS.SecretAccessKey, "aws-secret-access-key", "", "AWS: SecretAccessKey")
	flag.StringVar(&s.AWS.KKPDatacenter, "aws-kkp-datacenter", "", "AWS: KKP datacenter to use")
	flag.StringVar(&s.Digitalocean.Token, "digitalocean-token", "", "Digitalocean: API Token")
	flag.StringVar(&s.Digitalocean.KKPDatacenter, "digitalocean-kkp-datacenter", "", "Digitalocean: KKP datacenter to use")
	flag.StringVar(&s.Hetzner.Token, "hetzner-token", "", "Hetzner: API Token")
	flag.StringVar(&s.Hetzner.KKPDatacenter, "hetzner-kkp-datacenter", "", "Hetzner: KKP datacenter to use")
	flag.StringVar(&s.OpenStack.Domain, "openstack-domain", "", "OpenStack: Domain")
	flag.StringVar(&s.OpenStack.Project, "openstack-project", "", "OpenStack: Project")
	flag.StringVar(&s.OpenStack.ProjectID, "openstack-project-id", "", "OpenStack: Project ID")
	flag.StringVar(&s.OpenStack.Username, "openstack-username", "", "OpenStack: Username")
	flag.StringVar(&s.OpenStack.Password, "openstack-password", "", "OpenStack: Password")
	flag.StringVar(&s.OpenStack.KKPDatacenter, "openstack-kkp-datacenter", "", "OpenStack: KKP datacenter to use")
	flag.StringVar(&s.VSphere.Username, "vsphere-username", "", "vSphere: Username")
	flag.StringVar(&s.VSphere.Password, "vsphere-password", "", "vSphere: Password")
	flag.StringVar(&s.VSphere.KKPDatacenter, "vsphere-kkp-datacenter", "", "vSphere: KKP datacenter to use")
	flag.StringVar(&s.Azure.ClientID, "azure-client-id", "", "Azure: ClientID")
	flag.StringVar(&s.Azure.ClientSecret, "azure-client-secret", "", "Azure: ClientSecret")
	flag.StringVar(&s.Azure.TenantID, "azure-tenant-id", "", "Azure: TenantID")
	flag.StringVar(&s.Azure.SubscriptionID, "azure-subscription-id", "", "Azure: SubscriptionID")
	flag.StringVar(&s.Azure.KKPDatacenter, "azure-kkp-datacenter", "", "Azure: KKP datacenter to use")
	flag.StringVar(&s.Packet.APIKey, "packet-api-key", "", "Packet: APIKey")
	flag.StringVar(&s.Packet.ProjectID, "packet-project-id", "", "Packet: ProjectID")
	flag.StringVar(&s.Packet.KKPDatacenter, "packet-kkp-datacenter", "", "Packet: KKP datacenter to use")
	flag.StringVar(&s.GCP.ServiceAccount, "gcp-service-account", "", "GCP: Service Account")
	flag.StringVar(&s.GCP.Network, "gcp-network", "", "GCP: Network")
	flag.StringVar(&s.GCP.Subnetwork, "gcp-subnetwork", "", "GCP: Subnetwork")
	flag.StringVar(&s.GCP.KKPDatacenter, "gcp-kkp-datacenter", "", "GCP: KKP datacenter to use")
	flag.StringVar(&kubevirtKubeconfigFile, "kubevirt-kubeconfig", "", "Kubevirt: Cluster Kubeconfig filename")
	flag.StringVar(&s.Kubevirt.KKPDatacenter, "kubevirt-kkp-datacenter", "", "Kubevirt: KKP datacenter to use")
	flag.StringVar(&s.Alibaba.AccessKeyID, "alibaba-access-key-id", "", "Alibaba: AccessKeyID")
	flag.StringVar(&s.Alibaba.AccessKeySecret, "alibaba-access-key-secret", "", "Alibaba: AccessKeySecret")
	flag.StringVar(&s.Alibaba.KKPDatacenter, "alibaba-kkp-datacenter", "", "Alibaba: KKP datacenter to use")
	flag.StringVar(&s.Nutanix.Username, "nutanix-username", "", "Nutanix: Username")
	flag.StringVar(&s.Nutanix.Password, "nutanix-password", "", "Nutanix: Password")
	flag.StringVar(&s.Nutanix.CSIUsername, "nutanix-csi-username", "", "Nutanix CSI Prism Element: Username")
	flag.StringVar(&s.Nutanix.CSIPassword, "nutanix-csi-password", "", "Nutanix CSI Prism Element: Password")
	flag.StringVar(&s.Nutanix.CSIEndpoint, "nutanix-csi-endpoint", "", "Nutanix CSI Prism Element: Endpoint")
	flag.StringVar(&s.Nutanix.ProxyURL, "nutanix-proxy-url", "", "Nutanix: HTTP Proxy URL to access endpoint")
	flag.StringVar(&s.Nutanix.ClusterName, "nutanix-cluster-name", "", "Nutanix: Cluster Name")
	flag.StringVar(&s.Nutanix.ProjectName, "nutanix-project-name", "", "Nutanix: Project Name")
	flag.StringVar(&s.Nutanix.SubnetName, "nutanix-subnet-name", "", "Nutanix: Subnet Name")
	flag.StringVar(&s.Nutanix.KKPDatacenter, "nutanix-kkp-datacenter", "", "Nutanix: KKP datacenter to use")
	flag.StringVar(&s.VMwareCloudDirector.Username, "vmware-cloud-director-username", "", "VMware Cloud Director: Username")
	flag.StringVar(&s.VMwareCloudDirector.Password, "vmware-cloud-director-password", "", "VMware Cloud Director: Password")
	flag.StringVar(&s.VMwareCloudDirector.Organization, "vmware-cloud-director-organization", "", "VMware Cloud Director: Organization")
	flag.StringVar(&s.VMwareCloudDirector.VDC, "vmware-cloud-director-vdc", "", "VMware Cloud Director: Organizational VDC")
	flag.StringVar(&s.VMwareCloudDirector.OVDCNetwork, "vmware-cloud-director-ovdc-network", "", "VMware Cloud Director: Organizational VDC network name")
	flag.StringVar(&s.VMwareCloudDirector.KKPDatacenter, "vmware-cloud-director-kkp-datacenter", "", "VMware Cloud Director: KKP datacenter to use")
	flag.StringVar(&s.RHEL.SubscriptionUser, "rhel-subscription-user", "", "RedHat Enterprise subscription user")
	flag.StringVar(&s.RHEL.SubscriptionPassword, "rhel-subscription-password", "", "RedHat Enterprise subscription password")
	flag.StringVar(&s.RHEL.OfflineToken, "rhel-offline-token", "", "RedHat Enterprise offlien token")
}

func (s *Secrets) ParseFlags() error {
	if kubevirtKubeconfigFile != "" {
		content, err := os.ReadFile(kubevirtKubeconfigFile)
		if err != nil {
			return fmt.Errorf("failed to read kubevirt kubeconfig file: %w", err)
		}

		s.Kubevirt.Kubeconfig = test.SafeBase64Decoding(string(content))
	}

	if s.GCP.ServiceAccount != "" {
		s.GCP.ServiceAccount = test.SafeBase64Decoding(s.GCP.ServiceAccount)
	}

	return nil
}
