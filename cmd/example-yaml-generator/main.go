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
	"gopkg.in/yaml.v3"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/pkg/genyaml"
	"k8s.io/utils/pointer"
)

func main() {
	flag.Parse()

	if flag.NArg() < 2 {
		log.Fatal("Usage: go run main.go SRC_ROOT TARGET")
	}

	root := flag.Arg(0)
	target := flag.Arg(1)

	if _, err := os.Stat(target); err != nil {
		if err := os.MkdirAll(target, 0755); err != nil {
			log.Fatalf("Failed to create target directory %s: %v", target, err)
		}
	}

	// find all .go files in kubermatic/v1
	kubermaticFiles, err := filepath.Glob(filepath.Join(root, "pkg/crd/kubermatic/v1/*.go"))
	if err != nil {
		log.Fatalf("Failed to find go files: %v", err)
	}

	// find all .go files in operator/v1alpha1
	operatorFiles, err := filepath.Glob(filepath.Join(root, "pkg/crd/operator/v1alpha1/*.go"))
	if err != nil {
		log.Fatalf("Failed to find go files: %v", err)
	}

	files := append(kubermaticFiles, operatorFiles...)
	files = append(files, filepath.Join(root, "vendor/k8s.io/api/core/v1/types.go"))

	cm := genyaml.NewCommentMap(files...)
	examples := map[string]runtime.Object{
		"seed":                    createExampleSeed(),
		"kubermaticConfiguration": createExampleKubermaticConfiguration(),
	}

	for name, data := range examples {
		filename := filepath.Join(target, fmt.Sprintf("zz_generated.%s.yaml", name))

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

func createExampleSeed() *kubermaticv1.Seed {
	imageList := kubermaticv1.ImageList{}

	for _, operatingSystem := range providerconfig.AllOperatingSystems {
		imageList[operatingSystem] = ""
	}

	proxySettings := kubermaticv1.ProxySettings{
		HTTPProxy: kubermaticv1.NewProxyValue(""),
		NoProxy:   kubermaticv1.NewProxyValue(""),
	}

	seed := &kubermaticv1.Seed{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubermatic.k8s.io/v1",
			Kind:       "Seed",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "<<exampleseed>>",
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.SeedSpec{
			Datacenters: map[string]kubermaticv1.Datacenter{
				"<<exampledc>>": {
					Node: &kubermaticv1.NodeSettings{
						ProxySettings:      proxySettings,
						InsecureRegistries: []string{},
						RegistryMirrors:    []string{},
					},
					Spec: kubermaticv1.DatacenterSpec{
						Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{},
						BringYourOwn: &kubermaticv1.DatacenterSpecBringYourOwn{},
						AWS: &kubermaticv1.DatacenterSpecAWS{
							Images: imageList,
						},
						Azure: &kubermaticv1.DatacenterSpecAzure{},
						Openstack: &kubermaticv1.DatacenterSpecOpenstack{
							Images:               imageList,
							ManageSecurityGroups: pointer.BoolPtr(true),
							UseOctavia:           pointer.BoolPtr(true),
							DNSServers:           []string{},
							TrustDevicePath:      pointer.BoolPtr(false),
						},
						Packet: &kubermaticv1.DatacenterSpecPacket{
							Facilities: []string{},
						},
						Hetzner: &kubermaticv1.DatacenterSpecHetzner{},
						VSphere: &kubermaticv1.DatacenterSpecVSphere{
							Templates:           imageList,
							InfraManagementUser: &kubermaticv1.VSphereCredentials{},
						},
						GCP: &kubermaticv1.DatacenterSpecGCP{
							ZoneSuffixes: []string{},
						},
						Kubevirt: &kubermaticv1.DatacenterSpecKubevirt{
							DNSPolicy: "",
							DNSConfig: &corev1.PodDNSConfig{},
						},
						Alibaba: &kubermaticv1.DatacenterSpecAlibaba{},
					},
				},
			},
			ProxySettings: &proxySettings,
		},
	}

	defaulted, err := common.DefaultSeed(seed, zap.NewNop().Sugar())
	if err != nil {
		log.Fatalf("Failed to default Seed: %v", err)
	}

	if err := validateAllFieldsAreDefined(&defaulted.Spec); err != nil {
		log.Fatalf("Seed struct is incomplete: %v", err)
	}

	return defaulted
}

func createExampleKubermaticConfiguration() *operatorv1alpha1.KubermaticConfiguration {
	cfg := &operatorv1alpha1.KubermaticConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operatorv1alpha1.SchemeGroupVersion.String(),
			Kind:       "KubermaticConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "<<mykubermatic>>",
			Namespace: "kubermatic",
		},
		Spec: operatorv1alpha1.KubermaticConfigurationSpec{
			Ingress: operatorv1alpha1.KubermaticIngressConfiguration{
				Domain: "example.com",
			},
			FeatureGates: sets.NewString(),
			API: operatorv1alpha1.KubermaticAPIConfiguration{
				AccessibleAddons: []string{},
			},
		},
	}

	defaulted, err := common.DefaultConfiguration(cfg, zap.NewNop().Sugar())
	if err != nil {
		log.Fatalf("Failed to default KubermaticConfiguration: %v", err)
	}

	// ensure that all fields for updates are documented, even though we explicitly
	// omit them in all but the first array item
	setUpdateDefaults := func(cfg *operatorv1alpha1.KubermaticVersioningConfiguration) {
		if len(cfg.Updates) > 0 {
			if cfg.Updates[0].Automatic == nil {
				cfg.Updates[0].Automatic = pointer.BoolPtr(false)
			}

			if cfg.Updates[0].AutomaticNodeUpdate == nil {
				cfg.Updates[0].AutomaticNodeUpdate = pointer.BoolPtr(false)
			}
		}
	}

	setUpdateDefaults(&defaulted.Spec.Versions.Kubernetes)
	setUpdateDefaults(&defaulted.Spec.Versions.Openshift)

	return defaulted
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

	switch typ.Kind() {
	case reflect.Struct:
		for i := 0; i < typ.NumField(); i++ {
			fieldName := typ.Field(i).Name
			p := append(path, fieldName)

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
			p := append(path, fmt.Sprintf("[%s]", mapKey))

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

		for i := 0; i < value.Len(); i++ {
			p := append(path, fmt.Sprintf("[%d]", i))

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
