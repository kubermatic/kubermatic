// +build ee

package provider

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	providertypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var (
	allOperatingSystems = sets.NewString()
)

func init() {
	// build a quicker, convenient lookup mechanism
	for _, os := range providertypes.AllOperatingSystems {
		allOperatingSystems.Insert(string(os))
	}
}

func validateImageList(images kubermaticv1.ImageList) error {
	for s := range images {
		if !allOperatingSystems.Has(string(s)) {
			return fmt.Errorf("invalid operating system defined '%s'. Possible values: %s", s, strings.Join(allOperatingSystems.List(), ","))
		}
	}

	return nil
}

// ValidateSeed takes a seed and returns an error if the seed's spec is invalid.
func ValidateSeed(seed *kubermaticv1.Seed) error {
	for name, dc := range seed.Spec.Datacenters {
		if dc.Spec.VSphere != nil {
			if err := validateImageList(dc.Spec.VSphere.Templates); err != nil {
				return fmt.Errorf("invalid datacenter defined '%s': %v", name, err)
			}
		}
		if dc.Spec.Openstack != nil {
			if err := validateImageList(dc.Spec.Openstack.Images); err != nil {
				return fmt.Errorf("invalid datacenter defined '%s': %v", name, err)
			}
		}
	}

	// invalid DNS overwrites can happen when a seed was freshly converted from
	// the datacenters.yaml and has not yet been validated
	if seed.Spec.SeedDNSOverwrite != "" {
		if errs := validation.IsDNS1123Subdomain(seed.Spec.SeedDNSOverwrite); errs != nil {
			return fmt.Errorf("DNS overwrite %q is not a valid DNS name: %v", seed.Spec.SeedDNSOverwrite, errs)
		}
	} else {
		if errs := validation.IsDNS1123Subdomain(seed.Name); errs != nil {
			return fmt.Errorf("seed name %q is not a valid DNS name: %v", seed.Name, errs)
		}
	}

	return nil
}

// DatacenterMeta describes a Kubermatic datacenter.
type DatacenterMeta struct {
	Location         string                      `json:"location"`
	Seed             string                      `json:"seed"`
	Country          string                      `json:"country"`
	Spec             kubermaticv1.DatacenterSpec `json:"spec"`
	IsSeed           bool                        `json:"is_seed"`
	SeedDNSOverwrite string                      `json:"seed_dns_overwrite,omitempty"`
	Node             kubermaticv1.NodeSettings   `json:"node,omitempty"`
}

// datacentersMeta describes a number of Kubermatic datacenters.
type datacentersMeta struct {
	Datacenters map[string]DatacenterMeta `json:"datacenters"`
}

// LoadSeeds loads all Datacenters from the given path.
func LoadSeeds(path string) (map[string]*kubermaticv1.Seed, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read datacenter.yaml: %v", err)
	}

	dcMetas := datacentersMeta{}
	if err := yaml.UnmarshalStrict(bytes, &dcMetas); err != nil {
		return nil, fmt.Errorf("failed to parse datacenters.yaml: %v", err)
	}

	seeds, err := DatacenterMetasToSeeds(dcMetas.Datacenters)
	if err != nil {
		return nil, fmt.Errorf("failed to convert datacenters.yaml: %v", err)
	}

	for seedName, seed := range seeds {
		if err := ValidateSeed(seed); err != nil {
			return nil, fmt.Errorf("failed to validate datacenters.yaml: seed %q is invalid: %v", seedName, err)
		}
	}

	return seeds, nil
}

// LoadSeed loads an existing datacenters.yaml from disk and returns the given
// datacenter as a seed or an error if the datacenter does not exist.
func LoadSeed(path, datacenterName string) (*kubermaticv1.Seed, error) {
	seeds, err := LoadSeeds(path)
	if err != nil {
		return nil, err
	}

	datacenter, exists := seeds[datacenterName]
	if !exists {
		return nil, fmt.Errorf("datacenter %q is not in datacenters.yaml", datacenterName)
	}

	return datacenter, nil
}

type EESeedGetter func() (*kubermaticv1.Seed, error)

func SeedGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, seedName, dcFile, namespace string, dynamicDatacenters bool) (EESeedGetter, error) {
	if dynamicDatacenters {
		return func() (*kubermaticv1.Seed, error) {
			seed := &kubermaticv1.Seed{}
			if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: seedName}, seed); err != nil {
				// allow callers to handle this gracefully
				if kerrors.IsNotFound(err) {
					return nil, err
				}
				return nil, fmt.Errorf("failed to get seed %q: %v", seedName, err)
			}
			return seed, nil
		}, nil
	}

	if dcFile == "" {
		return nil, errors.New("--datacenters is required")
	}
	// Make sure we fail early, an error here is not recoverable
	seed, err := LoadSeed(dcFile, seedName)
	if err != nil {
		return nil, err
	}
	return func() (*kubermaticv1.Seed, error) {
		return seed.DeepCopy(), nil
	}, nil
}

type EESeedsGetter func() (map[string]*kubermaticv1.Seed, error)

func SeedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, dcFile, namespace string, dynamicDatacenters bool) (EESeedsGetter, error) {
	if dynamicDatacenters {
		// We only have a options func for raw *metav1.ListOpts as the rbac controller currently required that
		listOpts := &ctrlruntimeclient.ListOptions{
			Namespace: namespace,
		}

		return func() (map[string]*kubermaticv1.Seed, error) {
			seeds := &kubermaticv1.SeedList{}
			if err := client.List(ctx, seeds, listOpts); err != nil {
				return nil, fmt.Errorf("failed to list the seeds: %v", err)
			}
			seedMap := map[string]*kubermaticv1.Seed{}
			for idx, seed := range seeds.Items {
				seedMap[seed.Name] = &seeds.Items[idx]
			}
			return seedMap, nil
		}, nil
	}

	if dcFile == "" {
		return nil, errors.New("--datacenters is required")
	}
	// Make sure we fail early, an error here is not recoverable
	seedMap, err := LoadSeeds(dcFile)
	if err != nil {
		return nil, err
	}

	return func() (map[string]*kubermaticv1.Seed, error) {
		// copy it, just to be safe
		mapToReturn := map[string]*kubermaticv1.Seed{}
		for k, v := range seedMap {
			mapToReturn[k] = v.DeepCopy()
		}
		return mapToReturn, nil
	}, nil
}

type EESeedKubeconfigGetter = func(seed *kubermaticv1.Seed) (*rest.Config, error)

func SeedKubeconfigGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, kubeconfigFilePath, namespace string, dynamicDatacenters bool) (EESeedKubeconfigGetter, error) {
	if dynamicDatacenters {
		return func(seed *kubermaticv1.Seed) (*rest.Config, error) {
			secret := &corev1.Secret{}
			name := types.NamespacedName{
				Namespace: seed.Spec.Kubeconfig.Namespace,
				Name:      seed.Spec.Kubeconfig.Name,
			}

			fieldPath := seed.Spec.Kubeconfig.FieldPath
			if len(fieldPath) == 0 {
				fieldPath = DefaultKubeconfigFieldPath
			}
			if _, exists := secret.Data[fieldPath]; !exists {
				return nil, fmt.Errorf("secret %q has no key %q", name.String(), fieldPath)
			}

			cfg, err := clientcmd.RESTConfigFromKubeConfig(secret.Data[fieldPath])
			if err != nil {
				return nil, fmt.Errorf("failed to load kubeconfig: %v", err)
			}

			kubermaticlog.Logger.With("seed", seed.Name).Debug("Successfully got kubeconfig")
			return cfg, nil
		}, nil
	}

	if kubeconfigFilePath == "" {
		return nil, errors.New("--kubeconfig is required when --dynamic-datacenters=false")
	}

	kubeconfig, err := clientcmd.LoadFromFile(kubeconfigFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig from %q: %v", kubeconfigFilePath, err)
	}

	return func(seed *kubermaticv1.Seed) (*rest.Config, error) {
		if _, exists := kubeconfig.Contexts[seed.Name]; !exists {
			return nil, fmt.Errorf("found no context with name %q in kubeconfig", seed.Name)
		}
		cfg, err := clientcmd.NewNonInteractiveClientConfig(*kubeconfig, seed.Name, &clientcmd.ConfigOverrides{}, nil).ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get restConfig for seed %q: %v", seed.Name, err)
		}
		return cfg, nil
	}, nil
}
