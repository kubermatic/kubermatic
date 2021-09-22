// +build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2020 Loodse GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package conversion

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	certmanagerv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/controller/operator/defaults"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/crd/migrations/util"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/ee/provider"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/yaml"
)

type dockerImage struct {
	Repository string `yaml:"repository"`
	Tag        string `yaml:"tag"`
	PullPolicy string `yaml:"pullPolicy"`
}

type addonValues struct {
	DefaultAddons     []string    `yaml:"defaultAddons"`
	DefaultAddonsFile string      `yaml:"defaultAddonsFile"`
	Image             dockerImage `yaml:"image"`
}

type helmValues struct {
	Kubermatic kubermaticValues `yaml:"kubermatic"`
}

type kubermaticValues struct {
	ImagePullSecretData string `yaml:"imagePullSecretData"`
	Auth                struct {
		ClientID                 string `yaml:"clientID"`
		TokenIssuer              string `yaml:"tokenIssuer"`
		IssuerRedirectURL        string `yaml:"issuerRedirectURL"`
		IssuerClientID           string `yaml:"issuerClientID"`
		IssuerClientSecret       string `yaml:"issuerClientSecret"`
		IssuerCookieKey          string `yaml:"issuerCookieKey"`
		CABundle                 string `yaml:"caBundle"`
		SkipTokenIssuerTLSVerify string `yaml:"skipTokenIssuerTLSVerify"`
		ServiceAccountKey        string `yaml:"serviceAccountKey"`
	} `yaml:"auth"`
	Datacenters                          string  `yaml:"datacenters"`
	Domain                               string  `yaml:"domain"`
	Kubeconfig                           string  `yaml:"kubeconfig"`
	MonitoringScrapeAnnotationPrefix     string  `yaml:"monitoringScrapeAnnotationPrefix"`
	KubermaticImage                      string  `yaml:"kubermaticImage"`
	DNATControllerImage                  string  `yaml:"dnatControllerImage"`
	ExposeStrategy                       string  `yaml:"exposeStrategy"`
	Presets                              string  `yaml:"presets"`
	APIServerDefaultReplicas             *string `yaml:"apiserverDefaultReplicas"`
	ControllerManagerDefaultReplicas     *string `yaml:"controllerManagerDefaultReplicas"`
	SchedulerDefaultReplicas             *string `yaml:"schedulerDefaultReplicas"`
	MaxParallelReconcile                 *string `yaml:"maxParallelReconcile"`
	APIServerEndpointReconcilingDisabled bool    `yaml:"apiserverEndpointReconcilingDisabled"`
	DynamicDatacenters                   bool    `yaml:"dynamicDatacenters"`
	DynamicPresets                       bool    `yaml:"dynamicPresets"`
	Etcd                                 struct {
		DiskSize string `yaml:"diskSize"`
	} `yaml:"etcd"`
	Controller struct {
		FeatureGates   string      `yaml:"featureGates"`
		DatacenterName string      `yaml:"datacenterName"`
		NodeportRange  string      `yaml:"nodeportRange"`
		Replicas       *string     `yaml:"replicas"`
		Image          dockerImage `yaml:"image"`
		PProfEndpoint  string      `yaml:"pprofEndpoint"`
		Addons         struct {
			Kubernetes addonValues `yaml:"kubernetes"`
		} `yaml:"addons"`
		OverwriteRegistry string                      `yaml:"overwriteRegistry"`
		WorkerCount       int                         `yaml:"workerCount"`
		Resources         corev1.ResourceRequirements `yaml:"resources"`
	} `yaml:"controller"`
	API struct {
		FeatureGates     string                      `yaml:"featureGates"`
		Replicas         *string                     `yaml:"replicas"`
		AccessibleAddons []string                    `yaml:"accessibleAddons"`
		Image            dockerImage                 `yaml:"image"`
		PProfEndpoint    string                      `yaml:"pprofEndpoint"`
		Resources        corev1.ResourceRequirements `yaml:"resources"`
	} `yaml:"api"`
	UI struct {
		Replicas  *string                     `yaml:"replicas"`
		Image     dockerImage                 `yaml:"image"`
		Config    string                      `yaml:"config"`
		Resources corev1.ResourceRequirements `yaml:"resources"`
	} `yaml:"ui"`
	MasterController struct {
		Replicas      *string                     `yaml:"replicas"`
		Image         dockerImage                 `yaml:"image"`
		DebugLog      bool                        `yaml:"debugLog"`
		PProfEndpoint string                      `yaml:"pprofEndpoint"`
		WorkerCount   int                         `yaml:"workerCount"`
		Resources     corev1.ResourceRequirements `yaml:"resources"`
	} `yaml:"masterController"`
	StoreContainer             string `yaml:"storeContainer"`
	CleanupContainer           string `yaml:"cleanupContainer"`
	ClusterNamespacePrometheus struct {
		DisableDefaultScrapingConfigs bool          `yaml:"disableDefaultScrapingConfigs"`
		ScrapingConfigs               []interface{} `yaml:"scrapingConfigs"`
		DisableDefaultRules           bool          `yaml:"disableDefaultRules"`
		Rules                         interface{}   `yaml:"rules"`
	} `yaml:"clusterNamespacePrometheus"`
}

type Options struct {
	Namespace      string
	IncludeSeeds   bool
	IncludePresets bool
	PauseSeeds     bool
}

func HelmValuesFileToCRDs(yamlContent []byte, opt Options) ([]runtime.Object, error) {
	values := helmValues{}
	if err := yaml.Unmarshal(yamlContent, &values); err != nil {
		return nil, fmt.Errorf("failed to decode file: %v", err)
	}

	result := []runtime.Object{}

	config, err := convertKubermaticConfiguration(&values, opt.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create KubermaticConfiguration: %v", err)
	}
	result = append(result, config)

	caBundle, err := convertOIDCCABundle(&values, config, opt.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC ConfigMap: %v", err)
	}
	if caBundle != nil {
		result = append(result, caBundle)
	}

	if opt.IncludeSeeds {
		seeds, err := convertSeeds(&values, opt.Namespace, opt.PauseSeeds)
		if err != nil {
			return nil, fmt.Errorf("failed to create Seeds: %v", err)
		}
		result = append(result, seeds...)
	}

	if opt.IncludePresets {
		presets, err := convertPresets(&values, opt.Namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to create Presets: %v", err)
		}
		result = append(result, presets...)
	}

	return result, nil
}

func convertKubermaticConfiguration(values *helmValues, targetNamespace string) (*operatorv1alpha1.KubermaticConfiguration, error) {
	config := operatorv1alpha1.KubermaticConfiguration{}
	config.APIVersion = operatorv1alpha1.SchemeGroupVersion.String()
	config.Kind = "KubermaticConfiguration"
	config.Name = "kubermatic"
	config.Namespace = targetNamespace

	config.Spec.Ingress.Domain = values.Kubermatic.Domain

	// This is not actually the default, but anyone running our current stack has the
	// most recent cert-manager chart installed, which provides this ClusterIssuer by
	// default.
	config.Spec.Ingress.CertificateIssuer.Name = "letsencrypt-prod"
	config.Spec.Ingress.CertificateIssuer.Kind = certmanagerv1.ClusterIssuerKind

	if values.Kubermatic.ExposeStrategy != "" {
		if es, ok := kubermaticv1.ExposeStrategyFromString(values.Kubermatic.ExposeStrategy); ok {
			config.Spec.ExposeStrategy = es
		} else {
			return nil, fmt.Errorf("invalid expose strategy '%s', choose one of %v", values.Kubermatic.ExposeStrategy, kubermaticv1.AllExposeStrategies)
		}
	}

	pullSecret, err := base64.StdEncoding.DecodeString(values.Kubermatic.ImagePullSecretData)
	if err != nil {
		return &config, fmt.Errorf("invalid imagePullSecretData: %v", err)
	}
	config.Spec.ImagePullSecret = string(pullSecret)

	auth, err := convertAuth(&values.Kubermatic)
	if err != nil {
		return &config, fmt.Errorf("invalid auth section: %v", err)
	}
	config.Spec.Auth = *auth

	featureGates, err := convertFeatureGates(&values.Kubermatic)
	if err != nil {
		return &config, fmt.Errorf("invalid feature gates: %v", err)
	}
	config.Spec.FeatureGates = featureGates

	api, err := convertAPI(&values.Kubermatic)
	if err != nil {
		return &config, fmt.Errorf("invalid API section: %v", err)
	}
	config.Spec.API = *api

	seedController, err := convertSeedController(&values.Kubermatic)
	if err != nil {
		return &config, fmt.Errorf("invalid seedController section: %v", err)
	}
	config.Spec.SeedController = *seedController

	userCluster, err := convertUserCluster(&values.Kubermatic)
	if err != nil {
		// the "seedController" in the error msg is not a typo
		return &config, fmt.Errorf("invalid seedController section: %v", err)
	}
	config.Spec.UserCluster = *userCluster

	masterController, err := convertMasterController(&values.Kubermatic)
	if err != nil {
		return &config, fmt.Errorf("invalid masterController section: %v", err)
	}
	config.Spec.MasterController = *masterController

	ui, err := convertUI(&values.Kubermatic)
	if err != nil {
		return &config, fmt.Errorf("invalid UI section: %v", err)
	}
	config.Spec.UI = *ui

	return &config, nil
}

type DatacentersMeta struct {
	Datacenters map[string]provider.DatacenterMeta `json:"datacenters"`
}

func convertSeeds(values *helmValues, targetNamespace string, pauseProvisioning bool) ([]runtime.Object, error) {
	if values.Kubermatic.Datacenters == "" {
		return nil, nil
	}

	datacenters, err := base64.StdEncoding.DecodeString(values.Kubermatic.Datacenters)
	if err != nil {
		return nil, fmt.Errorf("datacenters are not valid base64: %v", err)
	}

	dcMetas := DatacentersMeta{}
	if err := yaml.UnmarshalStrict(datacenters, &dcMetas); err != nil {
		return nil, fmt.Errorf("failed to parse datacenters.yaml: %v", err)
	}

	var kubeconfig *clientcmdapi.Config
	if values.Kubermatic.Kubeconfig != "" {
		kubeconfigBytes, err := base64.StdEncoding.DecodeString(values.Kubermatic.Kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("kubeconfig is not valid base64: %v", err)
		}

		kubeconfig, err = clientcmd.Load(kubeconfigBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the kubeconfig: %v", err)
		}
	}

	return ConvertDatacenters(dcMetas.Datacenters, kubeconfig, targetNamespace, pauseProvisioning)
}

func ConvertDatacenters(datacenterMeta map[string]provider.DatacenterMeta, globalKubeconfig *clientcmdapi.Config, targetNamespace string, pauseProvisioning bool) ([]runtime.Object, error) {
	result := []runtime.Object{}

	seeds, err := provider.DatacenterMetasToSeeds(datacenterMeta)
	if err != nil {
		return result, fmt.Errorf("failed to convert datacenters.yaml: %v", err)
	}

	for _, seed := range seeds {
		seed.APIVersion = kubermaticv1.SchemeGroupVersion.String()
		seed.Kind = "Seed"
		seed.Namespace = targetNamespace

		if pauseProvisioning {
			if seed.Annotations == nil {
				seed.Annotations = map[string]string{}
			}

			seed.Annotations[common.SkipReconcilingAnnotation] = ""
		}

		var seedKubeconfig *clientcmdapi.Config
		if globalKubeconfig != nil {
			seedKubeconfig, err = util.SingleSeedKubeconfig(globalKubeconfig, seed.Name)
			if err != nil {
				return result, fmt.Errorf("kubeconfig does not contain a valid context for seed %s: %v", seed.Name, err)
			}

			secretName := fmt.Sprintf("kubeconfig-%s", seed.Name)

			secret, fieldPath, err := util.CreateKubeconfigSecret(seedKubeconfig, secretName, targetNamespace)
			if err != nil {
				return result, fmt.Errorf("failed to create kubeconfig Secret for seed %s: %v", seed.Name, err)
			}
			secret.APIVersion = "v1"
			secret.Kind = "Secret"

			seed.Spec.Kubeconfig.Name = secretName
			seed.Spec.Kubeconfig.Namespace = targetNamespace
			seed.Spec.Kubeconfig.FieldPath = fieldPath

			result = append(result, secret)
		}

		result = append(result, seed)
	}

	return result, nil
}

func convertPresets(values *helmValues, targetNamespace string) ([]runtime.Object, error) {
	if values.Kubermatic.Presets == "" {
		return nil, nil
	}

	presetsYAML, err := base64.StdEncoding.DecodeString(values.Kubermatic.Presets)
	if err != nil {
		return nil, fmt.Errorf("presets are not valid base64: %v", err)
	}

	presets, err := kubernetes.LoadPresets(presetsYAML)
	if err != nil {
		return nil, fmt.Errorf("failed to parse presets as YAML: %v", err)
	}

	result := []runtime.Object{}

	for idx := range presets.Items {
		preset := presets.Items[idx]
		preset.APIVersion = kubermaticv1.SchemeGroupVersion.String()
		preset.Kind = "Preset"
		preset.Namespace = targetNamespace

		result = append(result, &preset)
	}

	return result, nil
}

func convertOIDCCABundle(values *helmValues, config *operatorv1alpha1.KubermaticConfiguration, targetNamespace string) (*corev1.ConfigMap, error) {
	if values.Kubermatic.Auth.CABundle == "" {
		return nil, nil
	}

	caBundle, err := base64.StdEncoding.DecodeString(values.Kubermatic.Auth.CABundle)
	if err != nil {
		return nil, fmt.Errorf("invalid CA bundle: %v", err)
	}

	config.Spec.CABundle.Name = resources.CABundleConfigMapName

	return &corev1.ConfigMap{
		TypeMeta: v1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      config.Spec.CABundle.Name,
			Namespace: targetNamespace,
		},
		Data: map[string]string{
			resources.CABundleConfigMapKey: string(caBundle),
		},
	}, nil
}

func convertAuth(values *kubermaticValues) (*operatorv1alpha1.KubermaticAuthConfiguration, error) {
	effectiveClientID := values.Auth.ClientID
	if effectiveClientID == "" {
		effectiveClientID = defaults.DefaultAuthClientID
	}

	return &operatorv1alpha1.KubermaticAuthConfiguration{
		ClientID:                 strIfChanged(values.Auth.ClientID, defaults.DefaultAuthClientID),
		TokenIssuer:              strIfChanged(values.Auth.TokenIssuer, fmt.Sprintf("https://%s/dex", values.Domain)),
		IssuerClientID:           strIfChanged(values.Auth.IssuerClientID, fmt.Sprintf("%sIssuer", effectiveClientID)),
		IssuerRedirectURL:        strIfChanged(values.Auth.IssuerRedirectURL, fmt.Sprintf("https://%s/api/v1/kubeconfig", values.Domain)),
		IssuerClientSecret:       values.Auth.IssuerClientSecret,
		IssuerCookieKey:          values.Auth.IssuerCookieKey,
		SkipTokenIssuerTLSVerify: values.Auth.SkipTokenIssuerTLSVerify == "true",
		ServiceAccountKey:        values.Auth.ServiceAccountKey,
	}, nil
}

func convertAPI(values *kubermaticValues) (*operatorv1alpha1.KubermaticAPIConfiguration, error) {
	replicas, err := getReplicas(values.API.Replicas, defaults.DefaultAPIReplicas)
	if err != nil {
		return nil, fmt.Errorf("invalid replicas: %v", err)
	}

	return &operatorv1alpha1.KubermaticAPIConfiguration{
		DockerRepository: strIfChanged(values.API.Image.Repository, defaults.DefaultKubermaticImage),
		AccessibleAddons: values.API.AccessibleAddons,
		PProfEndpoint:    getPProfEndpoint(values.API.PProfEndpoint),
		Replicas:         replicas,
		Resources:        convertResources(values.API.Resources, defaults.DefaultAPIResources),
	}, nil
}

func convertSeedController(values *kubermaticValues) (*operatorv1alpha1.KubermaticSeedControllerConfiguration, error) {
	replicas, err := getReplicas(values.Controller.Replicas, defaults.DefaultSeedControllerMgrReplicas)
	if err != nil {
		return nil, fmt.Errorf("invalid replicas: %v", err)
	}

	storeContainer := strings.TrimSpace(values.StoreContainer)
	if storeContainer == strings.TrimSpace(defaults.DefaultBackupStoreContainer) {
		storeContainer = ""
	}

	cleanupContainer := strings.TrimSpace(values.CleanupContainer)
	if cleanupContainer == strings.TrimSpace(defaults.DefaultBackupCleanupContainer) {
		cleanupContainer = ""
	}

	maxParallelReconciles := 0
	if values.MaxParallelReconcile != nil && *values.MaxParallelReconcile != "" {
		maxParallelReconciles, err = numericValue(*values.MaxParallelReconcile)
		if err != nil {
			return nil, fmt.Errorf("invalid maxParallelReconcile: %v", err)
		}
	}

	return &operatorv1alpha1.KubermaticSeedControllerConfiguration{
		MaximumParallelReconciles: maxParallelReconciles,
		DockerRepository:          strIfChanged(values.Controller.Image.Repository, defaults.DefaultKubermaticImage),
		BackupStoreContainer:      storeContainer,
		BackupCleanupContainer:    cleanupContainer,
		PProfEndpoint:             getPProfEndpoint(values.Controller.PProfEndpoint),
		Replicas:                  replicas,
		Resources:                 convertResources(values.Controller.Resources, defaults.DefaultSeedControllerMgrResources),
	}, nil
}

func convertUserCluster(values *kubermaticValues) (*operatorv1alpha1.KubermaticUserClusterConfiguration, error) {
	kubernetesAddonCfg, err := convertAddonConfig(&values.Controller.Addons.Kubernetes, common.KubernetesAddonsFileName, defaults.DefaultKubernetesAddonImage)
	if err != nil {
		return nil, fmt.Errorf("invalid kubernetes addons: %v", err)
	}

	customRules := ""
	if values.ClusterNamespacePrometheus.Rules != nil {
		encoded, err := yaml.Marshal(values.ClusterNamespacePrometheus.Rules)
		if err != nil {
			return nil, fmt.Errorf("failed to encode custom Prometheus rules as YAML: %v", err)
		}

		customRules = string(encoded)
	}

	customScrapingConfigs := ""
	if values.ClusterNamespacePrometheus.ScrapingConfigs != nil {
		encoded, err := yaml.Marshal(values.ClusterNamespacePrometheus.ScrapingConfigs)
		if err != nil {
			return nil, fmt.Errorf("failed to encode custom Prometheus scraping configs as YAML: %v", err)
		}

		customScrapingConfigs = string(encoded)
	}

	apiserverReplicas, err := getReplicas(values.APIServerDefaultReplicas, defaults.DefaultAPIServerReplicas)
	if err != nil {
		return nil, fmt.Errorf("invalid apiserverDefaultReplicas: %v", err)
	}

	return &operatorv1alpha1.KubermaticUserClusterConfiguration{
		KubermaticDockerRepository:     strIfChanged(values.KubermaticImage, defaults.DefaultKubermaticImage),
		DNATControllerDockerRepository: strIfChanged(values.DNATControllerImage, defaults.DefaultDNATControllerImage),
		NodePortRange:                  strIfChanged(values.Controller.NodeportRange, defaults.DefaultNodePortRange),
		EtcdVolumeSize:                 strIfChanged(values.Etcd.DiskSize, defaults.DefaultEtcdVolumeSize),
		OverwriteRegistry:              values.Controller.OverwriteRegistry,
		Addons: operatorv1alpha1.KubermaticAddonsConfiguration{
			Kubernetes: *kubernetesAddonCfg,
		},
		Monitoring: operatorv1alpha1.KubermaticUserClusterMonitoringConfiguration{
			ScrapeAnnotationPrefix:        values.MonitoringScrapeAnnotationPrefix,
			DisableDefaultRules:           values.ClusterNamespacePrometheus.DisableDefaultRules,
			DisableDefaultScrapingConfigs: values.ClusterNamespacePrometheus.DisableDefaultScrapingConfigs,
			CustomRules:                   customRules,
			CustomScrapingConfigs:         customScrapingConfigs,
		},
		DisableAPIServerEndpointReconciling: values.APIServerEndpointReconcilingDisabled,
		APIServerReplicas:                   apiserverReplicas,
	}, nil
}

func convertAddonConfig(values *addonValues, defaultManifestFile string, defaultRepo string) (*operatorv1alpha1.KubermaticAddonConfiguration, error) {
	if len(values.DefaultAddons) > 0 && values.DefaultAddonsFile != "" {
		return nil, fmt.Errorf("both defaultAddons and defaultAddonsFile are configured, but they are mutually exclusive")
	}

	defaultManifests := ""
	if values.DefaultAddonsFile != "" && values.DefaultAddonsFile != defaultManifestFile {
		defaultManifests = fmt.Sprintf("!! insert the contents of %s here !!", values.DefaultAddonsFile)
	}

	return &operatorv1alpha1.KubermaticAddonConfiguration{
		DockerRepository: strIfChanged(values.Image.Repository, defaultRepo),
		Default:          values.DefaultAddons,
		DefaultManifests: defaultManifests,
	}, nil
}

func convertMasterController(values *kubermaticValues) (*operatorv1alpha1.KubermaticMasterControllerConfiguration, error) {
	replicas, err := getReplicas(values.MasterController.Replicas, defaults.DefaultMasterControllerMgrReplicas)
	if err != nil {
		return nil, fmt.Errorf("invalid replicas: %v", err)
	}

	return &operatorv1alpha1.KubermaticMasterControllerConfiguration{
		DockerRepository: strIfChanged(values.MasterController.Image.Repository, defaults.DefaultKubermaticImage),
		PProfEndpoint:    getPProfEndpoint(values.MasterController.PProfEndpoint),
		Replicas:         replicas,
		Resources:        convertResources(values.MasterController.Resources, defaults.DefaultMasterControllerMgrResources),
	}, nil
}

func convertUI(values *kubermaticValues) (*operatorv1alpha1.KubermaticUIConfiguration, error) {
	replicas, err := getReplicas(values.UI.Replicas, defaults.DefaultUIReplicas)
	if err != nil {
		return nil, fmt.Errorf("invalid replicas: %v", err)
	}

	return &operatorv1alpha1.KubermaticUIConfiguration{
		DockerRepository: strIfChanged(values.UI.Image.Repository, defaults.DefaultDashboardImage),
		Config:           values.UI.Config,
		Replicas:         replicas,
		Resources:        convertResources(values.UI.Resources, defaults.DefaultUIResources),
	}, nil
}

func convertResources(values corev1.ResourceRequirements, defaults corev1.ResourceRequirements) corev1.ResourceRequirements {
	result := corev1.ResourceRequirements{}

	for _, r := range []corev1.ResourceName{corev1.ResourceMemory, corev1.ResourceCPU} {
		specified, exists := values.Requests[r]
		if exists {
			defaulted, exists := defaults.Requests[r]
			if !exists || specified.Cmp(defaulted) != 0 {
				if result.Requests == nil {
					result.Requests = corev1.ResourceList{}
				}
				result.Requests[r] = specified
			}
		}

		specified, exists = values.Limits[r]
		if exists {
			defaulted, exists := defaults.Limits[r]
			if !exists || specified.Cmp(defaulted) != 0 {
				if result.Limits == nil {
					result.Limits = corev1.ResourceList{}
				}
				result.Limits[r] = specified
			}
		}
	}

	return result
}

func convertFeatureGates(values *kubermaticValues) (sets.String, error) {
	set := sets.NewString()

	for _, gates := range []string{values.API.FeatureGates, values.Controller.FeatureGates} {
		features, err := features.NewFeatures(gates)
		if err != nil {
			return set, fmt.Errorf("failed to parse feature gates: %v", err)
		}

		for feature, enabled := range features {
			if enabled {
				set.Insert(feature)
			}
		}
	}

	return set, nil
}

func strIfChanged(value string, defaultValue string) string {
	if value == defaultValue {
		return ""
	}

	return value
}

func getReplicas(yamlValue *string, defaultValue int) (*int32, error) {
	if yamlValue == nil || *yamlValue == "" {
		return nil, nil
	}

	parsed, err := numericValue(*yamlValue)
	if err != nil {
		return nil, fmt.Errorf("invalid numeric value: %v", err)
	}

	if parsed != defaultValue {
		return pointer.Int32Ptr(int32(parsed)), nil
	}

	return nil, nil
}

func getPProfEndpoint(yamlValue string) *string {
	if yamlValue == "" || yamlValue == defaults.DefaultPProfEndpoint {
		return nil
	}

	return pointer.StringPtr(yamlValue)
}

func numericValue(value interface{}) (int, error) {
	switch t := value.(type) {
	case int:
		return t, nil

	case string:
		parsed, err := strconv.ParseInt(t, 10, 32)
		return int(parsed), err

	default:
		return 0, fmt.Errorf("cannot parse '%v' (%T) as an integer", t, t)
	}
}
