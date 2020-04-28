package addon

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strings"
	"text/template"

	"github.com/Masterminds/semver"
	"github.com/Masterminds/sprig"
	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
)

const (
	ClusterTypeKubernetes = "kubernetes"
	ClusterTypeOpenshift  = "openshift"
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

	if variables == nil {
		variables = make(map[string]interface{})
	}

	clusterType := ClusterTypeKubernetes
	if cluster.IsOpenshift() {
		clusterType = ClusterTypeOpenshift
	}

	return &TemplateData{
		DatacenterName: cluster.Spec.Cloud.DatacenterName,
		Variables:      variables,
		Credentials:    credentials,
		Cluster: ClusterData{
			Type:                 clusterType,
			Name:                 cluster.Name,
			HumanReadableName:    cluster.Spec.HumanReadableName,
			Namespace:            cluster.Status.NamespaceName,
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
				DNSClusterIP:      dnsClusterIP,
				DNSResolverIP:     dnsResolverIP,
				PodCIDRBlocks:     cluster.Spec.ClusterNetwork.Pods.CIDRBlocks,
				ServiceCIDRBlocks: cluster.Spec.ClusterNetwork.Services.CIDRBlocks,
				ProxyMode:         cluster.Spec.ClusterNetwork.ProxyMode,
			},
		},
	}, nil
}

// ClusterData contains data related to the user cluster
// the addon is rendered for.
type ClusterData struct {
	// Type is either "kubernetes" or "openshift".
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
}

type ClusterNetwork struct {
	DNSClusterIP      string
	DNSResolverIP     string
	PodCIDRBlocks     []string
	ServiceCIDRBlocks []string
	ProxyMode         string
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

		if info.IsDir() {
			infoLog.Debug("Found directory in manifest path. Ignoring.")
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

		reader := kyaml.NewYAMLReader(bufio.NewReader(bufferAll))
		for {
			b, err := reader.Read()
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, fmt.Errorf("failed reading from YAML reader for file %s: %v", filename, err)
			}
			b = bytes.TrimSpace(b)
			if len(b) == 0 {
				continue
			}
			decoder := kyaml.NewYAMLToJSONDecoder(bytes.NewBuffer(b))
			raw := runtime.RawExtension{}
			if err := decoder.Decode(&raw); err != nil {
				return nil, fmt.Errorf("decoding failed for file %s: %v", filename, err)
			}
			if len(raw.Raw) == 0 {
				// This can happen if the manifest contains only comments, e.G. because it comes from Helm
				// something like `# Source: istio/charts/galley/templates/validatingwebhookconfiguration.yaml.tpl`
				continue
			}
			allManifests = append(allManifests, raw)
		}
	}

	return allManifests, nil
}
