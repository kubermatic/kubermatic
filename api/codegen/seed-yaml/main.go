package main

import (
	"fmt"
	"log"
	"path/filepath"
	"reflect"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	"github.com/kubernetes/test-infra/pkg/genyaml"
	"k8s.io/utils/pointer"
)

func main() {
	// find all .go files in kubermatic/v1, just in case a definition is split
	// into multiple files
	files, err := filepath.Glob("pkg/crd/kubermatic/v1/*.go")
	if err != nil {
		log.Fatalf("Failed to find go files: %v", err)
	}
	cm := genyaml.NewCommentMap(files...)

	mySeed := kubermaticv1.Seed{
		Spec: kubermaticv1.SeedSpec{
			Country: "foo",
			Datacenters: map[string]kubermaticv1.Datacenter{
				"exampleseed1": {
					Node: kubermaticv1.NodeSettings{
						ProxySettings: kubermaticv1.ProxySettings{
							HTTPProxy: pointer.StringPtr("external.com"),
							NoProxy:   pointer.StringPtr("localhost,internal.example.com"),
						},
						InsecureRegistries: []string{},
					},
					Spec: kubermaticv1.DatacenterSpec{
						Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{},
						BringYourOwn: &kubermaticv1.DatacenterSpecBringYourOwn{},
						AWS: &kubermaticv1.DatacenterSpecAWS{
							Images: kubermaticv1.ImageList{
								providerconfig.OperatingSystemCoreos: "foo",
							},
						},
						Azure: &kubermaticv1.DatacenterSpecAzure{},
						Openstack: &kubermaticv1.DatacenterSpecOpenstack{
							Images: kubermaticv1.ImageList{
								providerconfig.OperatingSystemCoreos: "foo",
							},
							ManageSecurityGroups: pointer.BoolPtr(true),
							TrustDevicePath:      pointer.BoolPtr(true),
							DNSServers:           []string{},
						},
						Packet: &kubermaticv1.DatacenterSpecPacket{
							Facilities: []string{},
						},
						Hetzner: &kubermaticv1.DatacenterSpecHetzner{},
						VSphere: &kubermaticv1.DatacenterSpecVSphere{
							Templates: kubermaticv1.ImageList{
								providerconfig.OperatingSystemCoreos: "foo",
							},
							InfraManagementUser: &kubermaticv1.VSphereCredentials{},
						},
						GCP: &kubermaticv1.DatacenterSpecGCP{
							ZoneSuffixes: []string{},
						},
						Fake:     &kubermaticv1.DatacenterSpecFake{},
						Kubevirt: &kubermaticv1.DatacenterSpecKubevirt{},
					},
				},
			},
			SeedDNSOverwrite: pointer.StringPtr("foo.example.com"),
			ProxySettings: &kubermaticv1.ProxySettings{
				HTTPProxy: pointer.StringPtr("external.com"),
				NoProxy:   pointer.StringPtr("localhost,internal.example.com"),
			},
		},
	}

	err = validateAllFieldsAreDefined(&mySeed.Spec)
	if err != nil {
		log.Fatalf("Seed struct is incomplete: %v", err)
	}

	yaml, err := cm.GenYaml(mySeed.Spec)
	if err != nil {
		log.Fatalf("Failed to create YAML: %v", err)
	}

	// fix ugly double newlines
	yaml = strings.Replace(yaml, "\n\n\n", "\n\n", -1)

	// reduce indentation
	yaml = strings.Replace(yaml, "    ", "  ", -1)

	fmt.Println(yaml)
}

// validateSeed checks recursively that all fields relevant to the documentation
// are filled in with example values.
func validateAllFieldsAreDefined(item interface{}) error {
	return validateReflect(reflect.ValueOf(item), []string{})
}

func validateReflect(value reflect.Value, path []string) error {
	typ := value.Type()

	// resolve pointer types to their underlying value
	if typ.Kind() == reflect.Ptr {
		// nil-pointers are not allowed for complex types
		if value.IsNil() && isComplexType(typ) {
			return fmt.Errorf("%s is invalid: field is nil", strings.Join(path, "."))
		}

		value = value.Elem()
		typ = value.Type()
	}

	switch typ.Kind() {
	case reflect.Struct:
		for i := 0; i < typ.NumField(); i++ {
			p := append(path, typ.Field(i).Name)

			if err := validateReflect(value.Field(i), p); err != nil {
				return err
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
