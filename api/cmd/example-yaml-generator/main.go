package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	files, err := filepath.Glob(filepath.Join(root, "pkg/crd/kubermatic/v1/*.go"))
	if err != nil {
		log.Fatalf("Failed to find go files: %v", err)
	}

	files = append(
		files,
		filepath.Join(root, "vendor/k8s.io/api/core/v1/types.go"),
	)

	cm := genyaml.NewCommentMap(files...)
	examples := map[string]runtime.Object{
		"seed": createExampleSeed(),
	}

	for name, data := range examples {
		yaml, err := cm.GenYaml(data)
		if err != nil {
			log.Fatalf("Failed to create YAML: %v", err)
		}

		// reduce indentation
		yaml = strings.Replace(yaml, "    ", "  ", -1)

		filename := filepath.Join(target, fmt.Sprintf("zz_generated.%s.yaml", name))
		if err := ioutil.WriteFile(filename, []byte(yaml), 0644); err != nil {
			log.Fatalf("Failed to write %s: %v", filename, err)
		}
	}
}

func createExampleSeed() *kubermaticv1.Seed {
	imageList := kubermaticv1.ImageList{}

	for _, os := range providerconfig.AllOperatingSystems {
		imageList[os] = ""
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
					Node: kubermaticv1.NodeSettings{
						ProxySettings:      proxySettings,
						InsecureRegistries: []string{},
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
						Kubevirt: &kubermaticv1.DatacenterSpecKubevirt{},
					},
				},
			},
			ProxySettings: &proxySettings,
		},
	}

	if err := validateAllFieldsAreDefined(&seed.Spec); err != nil {
		log.Fatalf("Seed struct is incomplete: %v", err)
	}

	return seed
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
