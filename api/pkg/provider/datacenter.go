package provider

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/util/restmapper"
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

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
	for _, os := range providerconfig.AllOperatingSystems {
		allOperatingSystems.Insert(string(os))
	}
}

// SeedGetter is a function to retrieve a single seed
type SeedGetter = func() (*kubermaticv1.Seed, error)

// SeedsGetter is a function to retrieve a list of seeds
type SeedsGetter = func() (map[string]*kubermaticv1.Seed, error)

// SeedKubeconfigGetter is used to fetch the kubeconfig for a given seed
type SeedKubeconfigGetter = func(seed *kubermaticv1.Seed) (*rest.Config, error)

// SeedClientGetter is used to get a ctrlruntimeclient for a given seed
type SeedClientGetter = func(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error)

// ClusterProviderGetter is used to get a clusterProvider
type ClusterProviderGetter = func(seed *kubermaticv1.Seed) (ClusterProvider, error)

// AddonProviderGetterr is used to get an AddonProvider
type AddonProviderGetter = func(seed *kubermaticv1.Seed) (AddonProvider, error)

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

// SeedGetterFactory returns a SeedGetter. It has validation of all its arguments
func SeedGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, seedName, dcFile, namespace string, dynamicDatacenters bool) (SeedGetter, error) {
	seedGetter, err := seedGetterFactory(ctx, client, seedName, dcFile, namespace, dynamicDatacenters)
	if err != nil {
		return nil, err
	}
	return func() (*kubermaticv1.Seed, error) {
		seed, err := seedGetter()
		if err != nil {
			return nil, err
		}
		seed.SetDefaults()
		return seed, nil
	}, nil
}

func seedGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, seedName, dcFile, namespace string, dynamicDatacenters bool) (SeedGetter, error) {
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

func SeedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, dcFile, namespace, workerName string, dynamicDatacenters bool) (SeedsGetter, error) {
	seedsGetter, err := seedsGetterFactory(ctx, client, dcFile, namespace, workerName, dynamicDatacenters)
	if err != nil {
		return nil, err
	}
	return func() (map[string]*kubermaticv1.Seed, error) {
		seeds, err := seedsGetter()
		if err != nil {
			return nil, err
		}
		for idx := range seeds {
			seeds[idx].SetDefaults()
		}
		return seeds, nil
	}, nil
}

func seedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, dcFile, namespace, workerName string, dynamicDatacenters bool) (SeedsGetter, error) {
	if dynamicDatacenters {
		labelSelector, err := workerlabel.LabelSelector(workerName)
		if err != nil {
			return nil, fmt.Errorf("failed to construct label selector for worker-name: %v", err)
		}
		// We only have a options func for raw *metav1.ListOpts as the rbac controller currently required that
		listOpts := &ctrlruntimeclient.ListOptions{
			LabelSelector: labelSelector,
			Namespace:     namespace,
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

func SeedKubeconfigGetterFactory(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	kubeconfigFilePath,
	namespace string,
	dynamicDatacenters bool) (SeedKubeconfigGetter, error) {

	if dynamicDatacenters {
		return func(seed *kubermaticv1.Seed) (*rest.Config, error) {
			secret := &corev1.Secret{}
			name := types.NamespacedName{
				Namespace: seed.Spec.Kubeconfig.Namespace,
				Name:      seed.Spec.Kubeconfig.Name,
			}
			if err := client.Get(ctx, name, secret); err != nil {
				return nil, fmt.Errorf("failed to get kubeconfig secret %q: %v", name.String(), err)
			}
			if _, exists := secret.Data[seed.Spec.Kubeconfig.FieldPath]; !exists {
				return nil, fmt.Errorf("secret %q has no key %q", name.String(), seed.Spec.Kubeconfig.FieldPath)
			}
			cfg, err := clientcmd.RESTConfigFromKubeConfig(secret.Data[seed.Spec.Kubeconfig.FieldPath])
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

// SeedClientGetterFactory returns a SeedClientGetter. It uses a RestMapperCache to cache
// the discovery data, which considerably speeds up client creation.
func SeedClientGetterFactory(kubeconfigGetter SeedKubeconfigGetter) SeedClientGetter {
	cache := restmapper.New()
	return func(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error) {
		cfg, err := kubeconfigGetter(seed)
		if err != nil {
			return nil, err
		}
		return cache.Client(cfg)
	}
}
