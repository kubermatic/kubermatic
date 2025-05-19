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

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/util/edition"
	"k8c.io/machine-controller/sdk/providerconfig"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/test-infra/pkg/genyaml"
	"k8s.io/utils/ptr"
	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

const (
	sampledc = "<<exampledc>>"
)

func main() {
	flag.Parse()

	if flag.NArg() < 2 {
		log.Fatal("Usage: go run main.go SRC_ROOT TARGET")
	}

	ed := edition.KubermaticEdition
	log.Printf("Generating for %s", ed)

	root := flag.Arg(0)
	target := flag.Arg(1)

	if _, err := os.Stat(target); err != nil {
		if err := os.MkdirAll(target, 0o755); err != nil {
			log.Fatalf("Failed to create target directory %s: %v", target, err)
		}
	}

	// find all .go files in kubermatic/v1
	kubermaticFiles, err := filepath.Glob(filepath.Join(root, "vendor/k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1/*.go"))
	if err != nil {
		log.Fatalf("Failed to find go files: %v", err)
	}

	appsKubermaticFiles, err := filepath.Glob(filepath.Join(root, "vendor/k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1/*.go"))
	if err != nil {
		log.Fatalf("Failed to find appsKubermatic go files: %v", err)
	}

	var files []string
	files = append(files, kubermaticFiles...)
	files = append(files, appsKubermaticFiles...)
	files = append(files, filepath.Join(root, "vendor/k8s.io/api/core/v1/types.go"))

	cm, err := genyaml.NewCommentMap(nil, files...)
	if err != nil {
		log.Fatalf("Failed to create comment map: %v", err)
	}

	config := createExampleKubermaticConfiguration()

	examples := map[string]runtime.Object{
		"kubermaticConfiguration": config,
		"seed":                    createExampleSeed(config),
		"applicationDefinition":   createExampleApplicationDefinition(),
		"applicationInstallation": createExampleApplicationInstallation(),
	}

	for name, data := range examples {
		log.Printf("Creating example YAML for %s resources...", name)

		filename := filepath.Join(target, fmt.Sprintf("zz_generated.%s.%s.yaml", name, strings.ToLower(ed.ShortString())))

		f, err := os.Create(filename)
		if err != nil {
			log.Fatalf("Failed to create %s: %v", filename, err)
		}

		encoder := yaml.NewEncoder(f)
		encoder.SetIndent(2)

		err = cm.EncodeYaml(data, encoder)
		f.Close()

		if err != nil {
			log.Fatalf("Failed to write YAML: %v", err)
		}
	}
}

func createBaseExampleSeed(config *kubermaticv1.KubermaticConfiguration) *kubermaticv1.Seed {
	imageList := kubermaticv1.ImageList{}
	operatingSystemProfileList := kubermaticv1.OperatingSystemProfileList{}
	kubevirtHTTPSource := kubermaticv1.KubeVirtHTTPSource{
		OperatingSystems: map[providerconfig.OperatingSystem]kubermaticv1.OSVersions{},
	}

	tinkerbellHTTPsource := kubermaticv1.TinkerbellHTTPSource{
		OperatingSystems: map[providerconfig.OperatingSystem]kubermaticv1.OSVersions{},
	}
	for _, operatingSystem := range providerconfig.AllOperatingSystems {
		imageList[operatingSystem] = ""
		operatingSystemProfileList[operatingSystem] = ""
	}
	for supportedOS := range kubermaticv1.SupportedKubeVirtOS {
		kubevirtHTTPSource.OperatingSystems[supportedOS] = map[string]string{"vX.Y": "http://example.com/images/os.iso"}
	}

	for supportedOS := range kubermaticv1.SupportedTinkerbellOS {
		tinkerbellHTTPsource.OperatingSystems[supportedOS] = map[string]string{"vX.Y": "http://example.com/images/os.iso"}
	}

	proxySettings := kubermaticv1.ProxySettings{
		HTTPProxy: kubermaticv1.NewProxyValue(""),
		NoProxy:   kubermaticv1.NewProxyValue(""),
	}

	seed := &kubermaticv1.Seed{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubermatic.k8c.io/v1",
			Kind:       "Seed",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "<<exampleseed>>",
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.SeedSpec{
			Datacenters: map[string]kubermaticv1.Datacenter{
				sampledc: {
					Node: &kubermaticv1.NodeSettings{
						ProxySettings:      proxySettings,
						InsecureRegistries: []string{},
						RegistryMirrors:    []string{},
						ContainerdRegistryMirrors: &kubermaticv1.ContainerRuntimeContainerd{
							Registries: map[string]kubermaticv1.ContainerdRegistry{
								"docker.io": {
									Mirrors: []string{"mirror.gcr.io"},
								},
							},
						},
					},
					Spec: kubermaticv1.DatacenterSpec{
						ProviderReconciliationInterval: &metav1.Duration{Duration: defaulting.DefaultCloudProviderReconciliationInterval},
						Digitalocean:                   &kubermaticv1.DatacenterSpecDigitalocean{},
						Baremetal: &kubermaticv1.DatacenterSpecBaremetal{
							Tinkerbell: &kubermaticv1.DatacenterSpecTinkerbell{
								Images: kubermaticv1.TinkerbellImageSources{HTTP: &tinkerbellHTTPsource},
							},
						},
						BringYourOwn:                   &kubermaticv1.DatacenterSpecBringYourOwn{},
						Edge:                           &kubermaticv1.DatacenterSpecEdge{},
						RequiredEmails:                 []string{},
						DefaultOperatingSystemProfiles: operatingSystemProfileList,
						MachineFlavorFilter: &kubermaticv1.MachineFlavorFilter{
							MinCPU:    0,
							MaxCPU:    0,
							MinRAM:    0,
							MaxRAM:    0,
							EnableGPU: false,
						},
						AWS: &kubermaticv1.DatacenterSpecAWS{
							Images: imageList,
						},
						Azure: &kubermaticv1.DatacenterSpecAzure{},
						Openstack: &kubermaticv1.DatacenterSpecOpenstack{
							Images:               imageList,
							ManageSecurityGroups: ptr.To(true),
							UseOctavia:           ptr.To(true),
							DNSServers:           []string{},
							TrustDevicePath:      ptr.To(false),
							EnabledFlavors:       []string{},
							IPv6Enabled:          ptr.To(false),
							NodeSizeRequirements: &kubermaticv1.OpenstackNodeSizeRequirements{
								MinimumVCPUs:  0,
								MinimumMemory: 0,
							},
						},
						//nolint:staticcheck // Deprecated Packet provider is still used for backward compatibility until v2.29
						Packet: &kubermaticv1.DatacenterSpecPacket{
							Facilities: []string{},
							Metro:      "",
						},
						Hetzner: &kubermaticv1.DatacenterSpecHetzner{},
						VSphere: &kubermaticv1.DatacenterSpecVSphere{
							Templates:           imageList,
							InfraManagementUser: &kubermaticv1.VSphereCredentials{},
							IPv6Enabled:         ptr.To(false),
						},
						GCP: &kubermaticv1.DatacenterSpecGCP{
							ZoneSuffixes: []string{},
						},
						Kubevirt: &kubermaticv1.DatacenterSpecKubevirt{
							DNSPolicy:                    "",
							DNSConfig:                    &corev1.PodDNSConfig{},
							Images:                       kubermaticv1.KubeVirtImageSources{HTTP: &kubevirtHTTPSource},
							EnableDefaultNetworkPolicies: ptr.To(true),
							CustomNetworkPolicies: []kubermaticv1.CustomNetworkPolicy{
								{
									Name: "deny-ingress",
									Spec: networkingv1.NetworkPolicySpec{
										PodSelector: metav1.LabelSelector{},
										PolicyTypes: []networkingv1.PolicyType{
											networkingv1.PolicyTypeIngress,
										},
									},
								},
							},
							InfraStorageClasses: []kubermaticv1.KubeVirtInfraStorageClass{
								{
									Name:           "rook-ceph-block",
									IsDefaultClass: ptr.To(true),
								},
							},
						},
						Alibaba: &kubermaticv1.DatacenterSpecAlibaba{},
						Anexia:  &kubermaticv1.DatacenterSpecAnexia{},
						Nutanix: &kubermaticv1.DatacenterSpecNutanix{
							Images: imageList,
							Port:   ptr.To[int32](9440),
						},
						VMwareCloudDirector: &kubermaticv1.DatacenterSpecVMwareCloudDirector{
							Templates: imageList,
						},
						KubeLB: &kubermaticv1.KubeLBDatacenterSettings{
							Enabled:                  true,
							NodeAddressType:          "ExternalIP",
							UseLoadBalancerClass:     true,
							EnableGatewayAPI:         false,
							EnableSecretSynchronizer: true,
							DisableIngressClass:      false,
						},
					},
				},
			},
			ProxySettings: &proxySettings,
			NodeportProxy: kubermaticv1.NodeportProxyConfig{
				Annotations: map[string]string{},
				Envoy: kubermaticv1.NodePortProxyComponentEnvoy{
					LoadBalancerService: kubermaticv1.EnvoyLoadBalancerService{
						SourceRanges: []kubermaticv1.CIDR{},
					},
				},
			},
			Metering: &kubermaticv1.MeteringConfiguration{
				Enabled:          false,
				StorageClassName: "kubermatic-fast",
				ReportConfigurations: map[string]kubermaticv1.MeteringReportConfiguration{
					"weekly": {
						Schedule: "0 1 * * 6",
						Interval: 7,
					},
				},
			},
			MLA: &kubermaticv1.SeedMLASettings{},
			KubeLB: &kubermaticv1.KubeLBSettings{
				Kubeconfig: corev1.ObjectReference{
					Name:      "kubelb-management-kubeconfig",
					Namespace: "kubermatic",
				},
			},
		},
	}

	defaulted, err := defaulting.DefaultSeed(seed, config, zap.NewNop().Sugar())
	if err != nil {
		log.Fatalf("Failed to default Seed: %v", err)
	}

	if err := validateAllFieldsAreDefined(&defaulted.Spec); err != nil {
		log.Fatalf("Seed struct is incomplete: %v", err)
	}

	return defaulted
}

func createExampleKubermaticConfiguration() *kubermaticv1.KubermaticConfiguration {
	cfg := &kubermaticv1.KubermaticConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
			Kind:       "KubermaticConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "<<mykubermatic>>",
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "example.com",
			},
			FeatureGates: map[string]bool{},
			API:          kubermaticv1.KubermaticAPIConfiguration{},
			SeedController: kubermaticv1.KubermaticSeedControllerConfiguration{
				BackupStoreContainer:  defaulting.DefaultBackupStoreContainer,
				BackupDeleteContainer: defaulting.DefaultBackupDeleteContainer,
			},
		},
	}

	defaulted, err := defaulting.DefaultConfiguration(cfg, zap.NewNop().Sugar())
	if err != nil {
		log.Fatalf("Failed to default KubermaticConfiguration: %v", err)
	}

	// ensure that all fields for updates are documented, even though we explicitly
	// omit them in all but the first array item
	setUpdateDefaults := func(cfg *kubermaticv1.KubermaticVersioningConfiguration) {
		if len(cfg.Updates) > 0 {
			if cfg.Updates[0].Automatic == nil {
				cfg.Updates[0].Automatic = ptr.To(false)
			}

			if cfg.Updates[0].AutomaticNodeUpdate == nil {
				cfg.Updates[0].AutomaticNodeUpdate = ptr.To(false)
			}
		}
	}

	setUpdateDefaults(&defaulted.Spec.Versions)
	return defaulted
}

func createExampleApplicationDefinition() *appskubermaticv1.ApplicationDefinition {
	return &appskubermaticv1.ApplicationDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appskubermaticv1.SchemeGroupVersion.String(),
			Kind:       appskubermaticv1.ApplicationDefinitionKindName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "<<appdef-name>>",
		},
		Spec: appskubermaticv1.ApplicationDefinitionSpec{
			Description: "",
			Method:      appskubermaticv1.HelmTemplateMethod,
			Versions: []appskubermaticv1.ApplicationVersion{
				{
					Version: "v1.2.3",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								URL:          "https://charts.example.com || oci://localhost:5000/myrepo",
								ChartName:    "my-app",
								ChartVersion: "v13.9.0",
								Credentials: &appskubermaticv1.HelmCredentials{
									Username: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: "<<secret-name>>"},
										Key:                  "user",
										Optional:             ptr.To(false),
									},
									Password: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: "<<secret-name>>"},
										Key:                  "pass",
										Optional:             ptr.To(false),
									},
									RegistryConfigFile: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: "<<secret-name>>"},
										Key:                  ".dockerconfigjson",
										Optional:             ptr.To(false),
									},
								},
							},
							Git: &appskubermaticv1.GitSource{
								Remote: "https://git.example.com/repo || git@example.com/repo",
								Ref: appskubermaticv1.GitReference{
									Branch: "main",
									Commit: "8061ceb738db42fe82b4c305b7aa5459d926d03e",
									Tag:    "v1.2.3",
								},
								Path: "charts/apache",
								Credentials: &appskubermaticv1.GitCredentials{
									Method: "password || token || ssh-key",
									Username: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: "<<secret-name>>"},
										Key:                  "user",
										Optional:             ptr.To(false),
									},
									Password: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: "<<secret-name>>"},
										Key:                  "pass",
										Optional:             ptr.To(false),
									},
									Token: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: "<<secret-name>>"},
										Key:                  "token",
										Optional:             ptr.To(false),
									},
									SSHKey: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: "<<secret-name>>"},
										Key:                  "private-key",
										Optional:             ptr.To(false),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func createExampleApplicationInstallation() *appskubermaticv1.ApplicationInstallation {
	return &appskubermaticv1.ApplicationInstallation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appskubermaticv1.SchemeGroupVersion.String(),
			Kind:       appskubermaticv1.ApplicationInstallationKindName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "<<appInstallation-name>>",
		},
		Spec: appskubermaticv1.ApplicationInstallationSpec{
			Namespace: &appskubermaticv1.AppNamespaceSpec{
				Name:        "my-namespace",
				Create:      true,
				Labels:      map[string]string{"env": "dev"},
				Annotations: map[string]string{"project-code": "azerty"},
			},
			ApplicationRef: appskubermaticv1.ApplicationRef{
				Name:    "apache",
				Version: "1.2.3",
			},
			Values: runtime.RawExtension{Raw: []byte(`{ "commonLabels": {"owner": "somebody"}}`)},
			ValuesBlock: `
commonLabels:
  owner: somebody`[1:],
		},
	}
}

// validateAllFieldsAreDefined recursively checks that all fields relevant
// to the documentation are filled in with example values.
func validateAllFieldsAreDefined(item interface{}) error {
	return validateReflect(reflect.ValueOf(item), []string{})
}

func validateReflect(value reflect.Value, path []string) error {
	typ := value.Type()

	// resolve pointer types to their underlying value
	if typ.Kind() == reflect.Ptr {
		if value.IsNil() {
			// nil-pointers are not allowed for complex types
			if isComplexType(typ) {
				return fmt.Errorf("%s is invalid: field is nil", strings.Join(path, "."))
			}

			return nil
		}

		value = value.Elem()
		typ = value.Type()
	}

	p := path

	switch typ.Kind() {
	case reflect.Struct:
		for i := range typ.NumField() {
			fieldName := typ.Field(i).Name

			p = append(p, fieldName)

			if err := validateReflect(value.Field(i), p); err != nil {
				// super special exception: allow not defining the Fake cloud provider
				if typ.Name() == "DatacenterSpec" && fieldName != "Fake" {
					return err
				}
			}
		}

	case reflect.Map:
		mapKeys := value.MapKeys()

		// enforce non-empty maps so there is always an example for a valid key in the generated YAML
		if len(mapKeys) == 0 {
			return fmt.Errorf("%s is invalid: maps must contain at least one element", strings.Join(path, "."))
		}

		for _, mapKey := range mapKeys {
			p = append(p, fmt.Sprintf("[%s]", mapKey))

			if err := validateReflect(value.MapIndex(mapKey), p); err != nil {
				return err
			}
		}

	case reflect.Slice:
		itemType := value.Type().Elem()

		// nil slices are rendered as `null`, which may be valid but would still be confusing
		if value.IsNil() {
			return fmt.Errorf("%s is invalid: slices not be nil in order to create nicer-looking YAML", strings.Join(path, "."))
		}

		// enforce non-empty maps if the items themselves can have sub-fields with documentation
		if value.Len() == 0 && isComplexType(itemType) {
			return fmt.Errorf("%s is invalid: slices of complex types must contain at least one item", strings.Join(path, "."))
		}

		for i := range value.Len() {
			p = append(p, fmt.Sprintf("[%d]", i))

			if err := validateReflect(value.Index(i), p); err != nil {
				return err
			}
		}
	}

	return nil
}

func isComplexType(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return isComplexKind(t.Kind())
}

func isComplexKind(t reflect.Kind) bool {
	return t == reflect.Struct || t == reflect.Map || t == reflect.Slice || t == reflect.Interface
}
