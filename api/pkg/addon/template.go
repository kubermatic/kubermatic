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

// TemplateData is injected into templates.
type TemplateData struct {
	SeedName       string
	DatacenterName string
	Cluster        ClusterData
	Addon          AddonData
	Credentials    Credentials
	Variables      map[string]interface{}
}

func NewTemplateData(
	cluster *kubermaticv1.Cluster,
	addon *kubermaticv1.Addon,
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

	return &TemplateData{
		DatacenterName: cluster.Spec.Cloud.DatacenterName,
		Variables:      variables,
		Addon: AddonData{
			Name:      addon.Name,
			IsDefault: addon.Spec.IsDefault,
		},
		Cluster: ClusterData{
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
			Network: ClusterNetwork{
				DNSClusterIP:      dnsClusterIP,
				DNSResolverIP:     dnsResolverIP,
				PodCIDRBlocks:     cluster.Spec.ClusterNetwork.Pods.CIDRBlocks,
				ServiceCIDRBlocks: cluster.Spec.ClusterNetwork.Services.CIDRBlocks,
				ProxyMode:         cluster.Spec.ClusterNetwork.ProxyMode,
			},
		},
		Credentials: credentials,
	}, nil
}

// ClusterData contains data related to the user cluster
// the addon is rendered for.
type ClusterData struct {
	Name                 string
	HumanReadableName    string
	Namespace            string
	OwnerName            string
	OwnerEmail           string
	Kubeconfig           string
	ApiserverExternalURL string
	ApiserverInternalURL string
	AdminToken           string
	CloudProviderName    string
	Version              *semver.Version
	MajorMinorVersion    string
	Network              ClusterNetwork
	Features             sets.String
}

type ClusterNetwork struct {
	DNSClusterIP      string
	DNSResolverIP     string
	PodCIDRBlocks     []string
	ServiceCIDRBlocks []string
	ProxyMode         string
}

//nolint:golint
type AddonData struct {
	Name      string
	IsDefault bool
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
