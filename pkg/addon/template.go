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

package addon

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/Masterminds/sprig/v3"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/kubermatic/v2/pkg/util/yaml"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	ClusterTypeKubernetes = "kubernetes"
)

func txtFuncMap(overwriteRegistry string) template.FuncMap {
	funcs := sprig.TxtFuncMap()
	// Registry is deprecated and should not be used anymore.
	funcs["Registry"] = registry.GetOverwriteFunc(overwriteRegistry)
	funcs["Image"] = registry.GetImageRewriterFunc(overwriteRegistry)
	funcs["join"] = strings.Join
	return funcs
}

// This alias exists purely because it makes the go doc we generate easier to
// read, as it does not hint at a different package anymore.
type Credentials = resources.Credentials

// TemplateData is the root context injected into each addon manifest file.
type TemplateData struct {
	SeedName       string
	DatacenterName string
	Cluster        ClusterData
	Credentials    Credentials
	Variables      map[string]interface{}
}

func NewTemplateData(
	cluster *kubermaticv1.Cluster,
	credentials resources.Credentials,
	kubeconfig string,
	dnsClusterIP string,
	dnsResolverIP string,
	ipamAllocations *kubermaticv1.IPAMAllocationList,
	variables map[string]interface{},
) (*TemplateData, error) {
	providerName, err := kubermaticv1helper.ClusterCloudProviderName(cluster.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to determine cloud provider name: %w", err)
	}

	if variables == nil {
		variables = make(map[string]interface{})
	}

	if cluster.Spec.CNIPlugin == nil {
		return nil, fmt.Errorf("cniPlugin must not be nil")
	}

	var csiOptions CSIOptions
	if cluster.Spec.Cloud.VSphere != nil {
		csiOptions.StoragePolicy = cluster.Spec.Cloud.VSphere.StoragePolicy
	}

	if cluster.Spec.Cloud.Nutanix != nil && cluster.Spec.Cloud.Nutanix.CSI != nil {
		csiOptions.StorageContainer = cluster.Spec.Cloud.Nutanix.CSI.StorageContainer
		csiOptions.Fstype = cluster.Spec.Cloud.Nutanix.CSI.Fstype
		csiOptions.SsSegmentedIscsiNetwork = cluster.Spec.Cloud.Nutanix.CSI.SsSegmentedIscsiNetwork
	}

	if cluster.Spec.Cloud.VMwareCloudDirector != nil && cluster.Spec.Cloud.VMwareCloudDirector.CSI != nil {
		csiOptions.Filesystem = cluster.Spec.Cloud.VMwareCloudDirector.CSI.Filesystem
		csiOptions.StorageProfile = cluster.Spec.Cloud.VMwareCloudDirector.CSI.StorageProfile
	}

	if cluster.Spec.Cloud.Openstack != nil {
		csiOptions.CinderTopologyEnabled = cluster.Spec.Cloud.Openstack.CinderTopologyEnabled
	}

	csiMigration := metav1.HasAnnotation(cluster.ObjectMeta, kubermaticv1.CSIMigrationNeededAnnotation) || kubermaticv1helper.CCMMigrationCompleted(cluster)

	var ipvs kubermaticv1.IPVSConfiguration
	if cluster.Spec.ClusterNetwork.IPVS != nil {
		ipvs = *cluster.Spec.ClusterNetwork.IPVS
	}

	var kubeVirtStorageClasses []kubermaticv1.KubeVirtInfraStorageClass
	if cluster.Spec.Cloud.Kubevirt != nil {
		kubeVirtStorageClasses = cluster.Spec.Cloud.Kubevirt.StorageClasses
	}

	var ipamAllocationsData map[string]IPAMAllocation
	if ipamAllocations != nil {
		ipamAllocationsData = make(map[string]IPAMAllocation, len(ipamAllocations.Items))
		for _, ipamAllocation := range ipamAllocations.Items {
			ipamAllocationsData[ipamAllocation.Name] = IPAMAllocation{
				Type:      ipamAllocation.Spec.Type,
				CIDR:      ipamAllocation.Spec.CIDR,
				Addresses: ipamAllocation.Spec.Addresses,
			}
		}
	}

	var clusterVersion *semverlib.Version
	if s := cluster.Status.Versions.ControlPlane.Semver(); s != nil {
		clusterVersion = s
	} else {
		clusterVersion = cluster.Spec.Version.Semver()
	}

	return &TemplateData{
		DatacenterName: cluster.Spec.Cloud.DatacenterName,
		Variables:      variables,
		Credentials:    credentials,
		Cluster: ClusterData{
			Type:              ClusterTypeKubernetes,
			Name:              cluster.Name,
			HumanReadableName: cluster.Spec.HumanReadableName,
			Namespace:         cluster.Status.NamespaceName,
			Labels:            cluster.Labels,
			Annotations:       cluster.Annotations,
			Kubeconfig:        kubeconfig,
			//nolint:staticcheck
			OwnerName:         cluster.Status.UserName,
			OwnerEmail:        cluster.Status.UserEmail,
			Address:           cluster.Status.Address,
			CloudProviderName: providerName,
			Version:           clusterVersion,
			MajorMinorVersion: fmt.Sprintf("%d.%d", clusterVersion.Major(), clusterVersion.Minor()),
			Features:          sets.KeySet(cluster.Spec.Features),
			Network: ClusterNetwork{
				DNSDomain:            cluster.Spec.ClusterNetwork.DNSDomain,
				DNSClusterIP:         dnsClusterIP,
				DNSResolverIP:        dnsResolverIP,
				PodCIDRBlocks:        cluster.Spec.ClusterNetwork.Pods.CIDRBlocks,
				ServiceCIDRBlocks:    cluster.Spec.ClusterNetwork.Services.CIDRBlocks,
				ProxyMode:            cluster.Spec.ClusterNetwork.ProxyMode,
				StrictArp:            ipvs.StrictArp,
				DualStack:            cluster.IsDualStack(),
				PodCIDRIPv4:          cluster.Spec.ClusterNetwork.Pods.GetIPv4CIDR(),
				PodCIDRIPv6:          cluster.Spec.ClusterNetwork.Pods.GetIPv6CIDR(),
				NodeCIDRMaskSizeIPv4: resources.GetClusterNodeCIDRMaskSizeIPv4(cluster),
				NodeCIDRMaskSizeIPv6: resources.GetClusterNodeCIDRMaskSizeIPv6(cluster),
				IPAMAllocations:      ipamAllocationsData,
				NodePortRange:        cluster.Spec.ComponentsOverride.Apiserver.NodePortRange,
			},
			CNIPlugin: CNIPlugin{
				Type:    cluster.Spec.CNIPlugin.Type.String(),
				Version: cluster.Spec.CNIPlugin.Version,
			},
			CSI: csiOptions,
			MLA: MLASettings{
				MonitoringEnabled: cluster.Spec.MLA != nil && cluster.Spec.MLA.MonitoringEnabled,
				LoggingEnabled:    cluster.Spec.MLA != nil && cluster.Spec.MLA.LoggingEnabled,
			},
			CSIMigration:                csiMigration,
			KubeVirtInfraStorageClasses: kubeVirtStorageClasses,
		},
	}, nil
}

// ClusterData contains data related to the user cluster
// the addon is rendered for.
type ClusterData struct {
	// Type is only "kubernetes"
	Type string
	// Name is the auto-generated, internal cluster name, e.g. "bbc8sc24wb".
	Name string
	// HumanReadableName is the user-specified cluster name.
	HumanReadableName string
	// Namespace is the full namespace for the cluster's control plane.
	Namespace string
	// OwnerName is the owner's full name.
	OwnerName string
	// OwnerEmail is the owner's e-mail address.
	OwnerEmail string
	// Labels are the labels users have configured for their cluster, including
	// system-defined labels like the project ID.
	Labels map[string]string
	// Annotations are the annotations on the cluster resource, usually
	// cloud-provider related information like regions.
	Annotations map[string]string
	// Kubeconfig is a YAML-encoded kubeconfig with cluster-admin permissions
	// inside the user-cluster. The kubeconfig uses the external URL to reach
	// the apiserver.
	Kubeconfig string

	// ClusterAddress stores access and address information of a cluster.
	Address kubermaticv1.ClusterAddress

	// CloudProviderName is the name of the cloud provider used, one of
	// "alibaba", "aws", "azure", "bringyourown", "digitalocean", "gcp",
	// "hetzner", "kubevirt", "openstack", "packet", "vsphere" depending on
	// the configured datacenters.
	CloudProviderName string
	// Version is the exact current cluster version.
	Version *semverlib.Version
	// MajorMinorVersion is a shortcut for common testing on "Major.Minor" on the
	// current cluster version.
	MajorMinorVersion string
	// Network contains DNS and CIDR settings for the cluster.
	Network ClusterNetwork
	// Features is a set of enabled features for this cluster.
	Features sets.Set[string]
	// CNIPlugin contains the CNIPlugin settings
	CNIPlugin CNIPlugin
	// CSI specific options, dependent on provider
	CSI CSIOptions
	// MLA contains monitoring, logging and alerting related settings for the user cluster.
	MLA MLASettings
	// CSIMigration indicates if the cluster needed the CSIMigration
	CSIMigration bool
	// KubeVirtInfraStorageClasses is a list of storage classes from KubeVirt infra cluster that are used for
	// initialization of user cluster storage classes by the CSI driver kubevirt (hot pluggable disks)
	KubeVirtInfraStorageClasses []kubermaticv1.KubeVirtInfraStorageClass
}

type ClusterNetwork struct {
	DNSDomain            string
	DNSClusterIP         string
	DNSResolverIP        string
	PodCIDRBlocks        []string
	ServiceCIDRBlocks    []string
	ProxyMode            string
	StrictArp            *bool
	DualStack            bool
	PodCIDRIPv4          string
	PodCIDRIPv6          string
	NodeCIDRMaskSizeIPv4 int32
	NodeCIDRMaskSizeIPv6 int32
	IPAMAllocations      map[string]IPAMAllocation
	NodePortRange        string
}

type IPAMAllocation struct {
	Type      kubermaticv1.IPAMPoolAllocationType
	CIDR      kubermaticv1.SubnetCIDR
	Addresses []string
}

type CNIPlugin struct {
	Type    string
	Version string
}

type MLASettings struct {
	// MonitoringEnabled is the flag for enabling monitoring in user cluster.
	MonitoringEnabled bool
	// LoggingEnabled is the flag for enabling logging in user cluster.
	LoggingEnabled bool
}

type CSIOptions struct {

	// vsphere
	// StoragePolicy is the storage policy to use for vsphere csi addon
	StoragePolicy string

	// nutanix
	StorageContainer        string
	Fstype                  string
	SsSegmentedIscsiNetwork *bool

	// vmware Cloud Director
	StorageProfile string
	Filesystem     string

	// openstack
	CinderTopologyEnabled bool
}

type Manifest struct {
	Content    runtime.RawExtension
	SourceFile string
}

func ParseFromFolder(log *zap.SugaredLogger, overwriteRegistry string, manifestPath string, data *TemplateData) ([]Manifest, error) {
	var allManifests []Manifest

	infos, err := os.ReadDir(manifestPath)
	if err != nil {
		return nil, err
	}

	for _, info := range infos {
		filename := path.Join(manifestPath, info.Name())
		infoLog := log.With("file", filename)

		// recurse into subdirectory
		if info.IsDir() {
			subManifests, err := ParseFromFolder(log, overwriteRegistry, filename, data)
			if err != nil {
				return nil, err
			}
			allManifests = append(allManifests, subManifests...)
			continue
		}

		if !strings.HasSuffix(filename, ".yaml") {
			infoLog.Debug("Ignoring non-YAML file")
			continue
		}

		infoLog.Debug("Processing file")

		fbytes, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
		}

		tpl, err := template.New(info.Name()).Funcs(txtFuncMap(overwriteRegistry)).Parse(string(fbytes))
		if err != nil {
			return nil, fmt.Errorf("failed to parse file %s: %w", filename, err)
		}

		bufferAll := bytes.NewBuffer([]byte{})
		if err := tpl.Execute(bufferAll, data); err != nil {
			return nil, fmt.Errorf("failed to execute templating on file %s: %w", filename, err)
		}

		sd := strings.TrimSpace(bufferAll.String())
		if len(sd) == 0 {
			infoLog.Debug("Skipping file as its empty after parsing")
			continue
		}

		addonManifests, err := yaml.ParseMultipleDocuments(bufio.NewReader(bufferAll))
		if err != nil {
			return nil, fmt.Errorf("decoding failed for file %s: %w", filename, err)
		}

		for _, m := range addonManifests {
			allManifests = append(allManifests, Manifest{
				Content:    m,
				SourceFile: filename,
			})
		}
	}

	return allManifests, nil
}
