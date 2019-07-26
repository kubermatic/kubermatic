package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const KubeconfigSecretKey = "kubeconfig"

var (
	// AllOperatingSystems defines all available operating systems
	AllOperatingSystems = sets.NewString(string(providerconfig.OperatingSystemCoreos), string(providerconfig.OperatingSystemCentOS), string(providerconfig.OperatingSystemUbuntu))
)

// SeedGetter is a function to retrieve a single seed
type SeedGetter = func() (*kubermaticv1.Seed, error)

// SeedsGetter is a function to retrieve a list of seeds
type SeedsGetter = func() (map[string]*kubermaticv1.Seed, error)

// SeedKubeconfigGetter is used to fetch the kubeconfig for a given seed
type SeedKubeconfigGetter = func(seedName string) (*rest.Config, error)

// DatacenterMeta describes a Kubermatic datacenter.
type DatacenterMeta struct {
	Location         string                      `json:"location"`
	Seed             string                      `json:"seed"`
	Country          string                      `json:"country"`
	Spec             kubermaticv1.DatacenterSpec `json:"spec"`
	IsSeed           bool                        `json:"is_seed"`
	SeedDNSOverwrite *string                     `json:"seed_dns_overwrite,omitempty"`
	Node             kubermaticv1.NodeSettings   `json:"node,omitempty"`
}

// datacentersMeta describes a number of Kubermatic datacenters.
type datacentersMeta struct {
	Datacenters map[string]DatacenterMeta `json:"datacenters"`
}

// loadDatacenters loads all Datacenters from the given path.
func loadSeeds(path string) (map[string]*kubermaticv1.Seed, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	dcMetas := datacentersMeta{}
	if err := yaml.UnmarshalStrict(bytes, &dcMetas); err != nil {
		return nil, err
	}

	if err := validateDatacenters(dcMetas.Datacenters); err != nil {
		return nil, err
	}

	dcs, err := DatacenterMetasToSeeds(dcMetas.Datacenters)
	if err != nil {
		return nil, err
	}

	return dcs, nil
}

func LoadSeed(path, datacenterName string) (*kubermaticv1.Seed, error) {
	seeds, err := loadSeeds(path)
	if err != nil {
		return nil, err
	}

	datacenter, exists := seeds[datacenterName]
	if !exists {
		return nil, fmt.Errorf("Datacenter %q is not in datacenters.yaml", datacenterName)
	}

	return datacenter, nil
}

func validateImageList(images kubermaticv1.ImageList) error {
	for s := range images {
		if !AllOperatingSystems.Has(string(s)) {
			return fmt.Errorf("invalid operating system defined '%s'. Possible values: %s", s, strings.Join(AllOperatingSystems.List(), ","))
		}
	}

	return nil
}

func validateDatacenters(datacenters map[string]DatacenterMeta) error {
	for name, dc := range datacenters {
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

	for datacenterName, datacenter := range datacenters {
		if !datacenter.IsSeed {
			continue
		}
		if datacenter.SeedDNSOverwrite != nil && *datacenter.SeedDNSOverwrite != "" {
			if errs := validation.IsDNS1123Subdomain(*datacenter.SeedDNSOverwrite); errs != nil {
				return fmt.Errorf("SeedDNS overwrite %q of datacenter %q is not a valid DNS name: %v",
					*datacenter.SeedDNSOverwrite, datacenterName, errs)
			}
			continue
		}
		if errs := validation.IsDNS1123Subdomain(datacenterName); errs != nil {
			return fmt.Errorf("Datacentername %q is not a valid DNS name: %v", datacenterName, errs)
		}
	}

	return nil
}

// SeedGetterFactory returns a SeedGetter. It has validation of all its arguments
func SeedGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, seedName, dcFile string, dynamicDatacenters bool) (SeedGetter, error) {
	if dcFile != "" && dynamicDatacenters {
		return nil, errors.New("--datacenters must be empty when --dynamic-datacenters is enabled")
	}

	if dynamicDatacenters {
		return func() (*kubermaticv1.Seed, error) {
			seed := &kubermaticv1.Seed{}
			if err := client.Get(ctx, types.NamespacedName{Name: seedName}, seed); err != nil {
				return nil, fmt.Errorf("failed to get seed %q: %v", seedName, err)
			}
			return seed, nil
		}, nil
	}

	if dcFile == "" {
		return nil, errors.New("--datacenters is required")
	}
	// Make sure we fail early, an error here is nor recoverable
	seed, err := LoadSeed(dcFile, seedName)
	if err != nil {
		return nil, err
	}
	return func() (*kubermaticv1.Seed, error) {
		return seed.DeepCopy(), nil
	}, nil

}

func SeedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, dcFile, workerName string, dynamicDatacenters bool) (SeedsGetter, error) {
	if dcFile != "" && dynamicDatacenters {
		return nil, errors.New("--datacenters must be empty when --dynamic-datacenters is enabled")
	}

	if dynamicDatacenters {
		selectorOpts, err := workerlabel.LabelSelector(workerName)
		if err != nil {
			return nil, fmt.Errorf("failed to construct label selector for worker-name: %v", err)
		}
		listOptsRaw := &metav1.ListOptions{}
		selectorOpts(listOptsRaw)
		listOpts := &ctrlruntimeclient.ListOptions{Raw: listOptsRaw}

		return func() (map[string]*kubermaticv1.Seed, error) {
			seeds := &kubermaticv1.SeedList{}
			if err := client.List(ctx, listOpts, seeds); err != nil {
				return nil, fmt.Errorf("failed to list the seeds: %v", err)
			}
			seedMap := map[string]*kubermaticv1.Seed{}
			for _, seed := range seeds.Items {
				seedMap[seed.Name] = &seed
			}
			return seedMap, nil
		}, nil
	}

	if dcFile == "" {
		return nil, errors.New("--datacenters is required")
	}
	// Make sure we fail early, an error here is nor recoverable
	seedMap, err := loadSeeds(dcFile)
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
	kubeconfigFilePath string,
	dynamicDatacenters bool) (SeedKubeconfigGetter, error) {

	if dynamicDatacenters {
		return func(seedName string) (*rest.Config, error) {
			seed := &kubermaticv1.Seed{}
			if err := client.Get(ctx, types.NamespacedName{Name: seedName}, seed); err != nil {
				return nil, fmt.Errorf("failed to get seed %q: %v", seedName, err)
			}
			secret := &corev1.Secret{}
			name := types.NamespacedName{
				Namespace: seed.Spec.Kubeconfig.Namespace,
				Name:      seed.Spec.Kubeconfig.Name,
			}
			if err := client.Get(ctx, name, secret); err != nil {
				return nil, fmt.Errorf("failed to get kubeconfig secret %q: %v", name.String(), err)
			}
			if _, exists := secret.Data[KubeconfigSecretKey]; !exists {
				return nil, fmt.Errorf("secret %q has no key %q", name.String(), KubeconfigSecretKey)
			}
			cfg := &rest.Config{}
			if err := json.Unmarshal(secret.Data[KubeconfigSecretKey], cfg); err != nil {
				return nil, fmt.Errorf("failed to unmarshal kubeconfig: %v", err)
			}
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
	return func(seedName string) (*rest.Config, error) {
		if _, exists := kubeconfig.Contexts[seedName]; !exists {
			return nil, fmt.Errorf("found no context with name %q in kubeconfig", seedName)
		}
		cfg, err := clientcmd.NewNonInteractiveClientConfig(*kubeconfig, seedName, &clientcmd.ConfigOverrides{}, nil).ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get restConfig for seed %q: %v", seedName, err)
		}
		return cfg, nil
	}, nil
}
