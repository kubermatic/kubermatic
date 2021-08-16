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
	"io/ioutil"
	"path"
	"strings"
	"text/template"

	"github.com/Masterminds/semver/v3"
	"github.com/Masterminds/sprig/v3"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/yaml"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"
)

const (
	ClusterTypeKubernetes = "kubernetes"
)

func txtFuncMap(overwriteRegistry string) template.FuncMap {
	funcs := sprig.TxtFuncMap()
	funcs["Registry"] = func(registry string) string {
		if overwriteRegistry != "" {
			return overwriteRegistry
		}
		return registry
	}

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
	variables map[string]interface{},
) (*TemplateData, error) {
	providerName, err := provider.ClusterCloudProviderName(cluster.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to determine cloud provider name: %v", err)
	}

	// Ensure IPVS configuration is set
	if cluster.Spec.ClusterNetwork.IPVS == nil {
		cluster.Spec.ClusterNetwork.IPVS = &kubermaticv1.IPVSConfiguration{StrictArp: pointer.BoolPtr(resources.IPVSStrictArp)}
	} else if cluster.Spec.ClusterNetwork.IPVS.StrictArp == nil {
		cluster.Spec.ClusterNetwork.IPVS.StrictArp = pointer.BoolPtr(resources.IPVSStrictArp)
	}

	if variables == nil {
		variables = make(map[string]interface{})
	}

	var cniPlugin CNIPlugin
	if cluster.Spec.CNIPlugin == nil {
		cniPlugin = CNIPlugin{
			Type: kubermaticv1.CNIPluginTypeCanal.String(),
			// This is to keep backward compatibility with clusters created before
			// those settings were introduced.
			Version: "v3.8",
		}
	} else {
		cniPlugin = CNIPlugin{
			Type:    cluster.Spec.CNIPlugin.Type.String(),
			Version: cluster.Spec.CNIPlugin.Version,
		}
	}

	return &TemplateData{
		DatacenterName: cluster.Spec.Cloud.DatacenterName,
		Variables:      variables,
		Credentials:    credentials,
		Cluster: ClusterData{
			Type:                 ClusterTypeKubernetes,
			Name:                 cluster.Name,
			HumanReadableName:    cluster.Spec.HumanReadableName,
			Namespace:            cluster.Status.NamespaceName,
			Labels:               cluster.Labels,
			Annotations:          cluster.Annotations,
			Kubeconfig:           kubeconfig,
			OwnerName:            cluster.Status.UserName,
			OwnerEmail:           cluster.Status.UserEmail,
			ApiserverExternalURL: cluster.Address.URL,
			ApiserverInternalURL: fmt.Sprintf("https://%s:%d", cluster.Address.InternalName, cluster.Address.Port),
			AdminToken:           cluster.Address.AdminToken,
			CloudProviderName:    providerName,
			Version:              semver.MustParse(cluster.Spec.Version.String()),
			MajorMinorVersion:    cluster.Spec.Version.MajorMinor(),
			Features:             sets.StringKeySet(cluster.Spec.Features),
			Network: ClusterNetwork{
				DNSDomain:         cluster.Spec.ClusterNetwork.DNSDomain,
				DNSClusterIP:      dnsClusterIP,
				DNSResolverIP:     dnsResolverIP,
				PodCIDRBlocks:     cluster.Spec.ClusterNetwork.Pods.CIDRBlocks,
				ServiceCIDRBlocks: cluster.Spec.ClusterNetwork.Services.CIDRBlocks,
				ProxyMode:         cluster.Spec.ClusterNetwork.ProxyMode,
				StrictArp:         *cluster.Spec.ClusterNetwork.IPVS.StrictArp,
			},
			CNIPlugin: cniPlugin,
			MLA: MLASettings{
				MonitoringEnabled: cluster.Spec.MLA != nil && cluster.Spec.MLA.MonitoringEnabled,
				LoggingEnabled:    cluster.Spec.MLA != nil && cluster.Spec.MLA.LoggingEnabled,
			},
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
	// ApiserverExternalURL is the full URL to the apiserver service from the
	// outside, including protocol and port number. It does not contain any
	// trailing slashes.
	ApiserverExternalURL string
	// ApiserverExternalURL is the full URL to the apiserver from within the
	// seed cluster itself. It does not contain any trailing slashes.
	ApiserverInternalURL string
	// AdminToken is the cluster's admin token.
	AdminToken string
	// CloudProviderName is the name of the cloud provider used, one of
	// "alibaba", "aws", "azure", "bringyourown", "digitalocean", "gcp",
	// "hetzner", "kubevirt", "openstack", "packet", "vsphere" depending on
	// the configured datacenters.
	CloudProviderName string
	// Version is the exact cluster version.
	Version *semver.Version
	// MajorMinorVersion is a shortcut for common testing on "Major.Minor".
	MajorMinorVersion string
	// Network contains DNS and CIDR settings for the cluster.
	Network ClusterNetwork
	// Features is a set of enabled features for this cluster.
	Features sets.String
	// CNIPlugin contains the CNIPlugin settings
	CNIPlugin CNIPlugin
	// MLA contains monitoring, logging and alerting related settings for the user cluster.
	MLA MLASettings
}

type ClusterNetwork struct {
	DNSDomain         string
	DNSClusterIP      string
	DNSResolverIP     string
	PodCIDRBlocks     []string
	ServiceCIDRBlocks []string
	ProxyMode         string
	StrictArp         bool
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

func ParseFromFolder(log *zap.SugaredLogger, overwriteRegistry string, manifestPath string, data *TemplateData) ([]runtime.RawExtension, error) {
	var allManifests []runtime.RawExtension

	infos, err := ioutil.ReadDir(manifestPath)
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

		infoLog.Debug("Processing file")

		fbytes, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %v", filename, err)
		}

		tpl, err := template.New(info.Name()).Funcs(txtFuncMap(overwriteRegistry)).Parse(string(fbytes))
		if err != nil {
			return nil, fmt.Errorf("failed to parse file %s: %v", filename, err)
		}

		bufferAll := bytes.NewBuffer([]byte{})
		if err := tpl.Execute(bufferAll, data); err != nil {
			return nil, fmt.Errorf("failed to execute templating on file %s: %v", filename, err)
		}

		sd := strings.TrimSpace(bufferAll.String())
		if len(sd) == 0 {
			infoLog.Debug("Skipping file as its empty after parsing")
			continue
		}

		addonManifests, err := yaml.ParseMultipleDocuments(bufio.NewReader(bufferAll))
		if err != nil {
			return nil, fmt.Errorf("decoding failed for file %s: %v", filename, err)
		}
		allManifests = append(allManifests, addonManifests...)
	}

	return allManifests, nil
}
