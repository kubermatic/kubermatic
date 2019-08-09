package seed

import (
	"context"
	"fmt"
	"sync"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/restmapper"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func newValidator(
	ctx context.Context,
	seedsGetter provider.SeedsGetter,
	seedKubeconfigGetter provider.SeedKubeconfigGetter,
	listOpts *ctrlruntimeclient.ListOptions) *seedValidator {

	return &seedValidator{
		ctx:                  ctx,
		seedsGetter:          seedsGetter,
		seedKubeconfigGetter: seedKubeconfigGetter,
		lock:                 &sync.Mutex{},
		listOpts:             listOpts,
		restMapperCache:      &sync.Map{},
	}
}

type seedValidator struct {
	ctx                  context.Context
	seedsGetter          provider.SeedsGetter
	seedKubeconfigGetter provider.SeedKubeconfigGetter
	lock                 *sync.Mutex
	// Can be used to insert a labelSelector
	listOpts        *ctrlruntimeclient.ListOptions
	restMapperCache *sync.Map
}

func (sv *seedValidator) Validate(seed *kubermaticv1.Seed, isDelete bool) error {
	// We need locking to make the validation concurrency-safe
	sv.lock.Lock()
	defer sv.lock.Unlock()

	client, err := sv.clientForSeed(seed)
	if err != nil {
		return fmt.Errorf("failed to get client for seed %q: %v", seed.Name, err)
	}

	seeds, err := sv.seedsGetter()
	if err != nil {
		return fmt.Errorf("failed to get seeds: %v", err)
	}

	return sv.validate(seed, client, seeds, isDelete)
}

func (sv *seedValidator) validate(seed *kubermaticv1.Seed, seedClient ctrlruntimeclient.Client, existingSeeds map[string]*kubermaticv1.Seed, isDelete bool) error {

	newDatacenters := sets.NewString()
	for datacenter := range seed.Spec.Datacenters {
		newDatacenters.Insert(datacenter)
	}

	for _, existingSeed := range existingSeeds {
		for existingDatacenter := range existingSeed.Spec.Datacenters {
			if newDatacenters.Has(existingDatacenter) {
				return fmt.Errorf("datacenter %q already exists in seed %q, can only have one datacenter with a given name", existingDatacenter, existingSeed.Name)
			}
		}
	}

	clusters := &kubermaticv1.ClusterList{}
	if err := seedClient.List(sv.ctx, sv.listOpts, clusters); err != nil {
		return fmt.Errorf("failed to list clusters: %v", err)
	}

	for _, cluster := range clusters.Items {
		if !newDatacenters.Has(cluster.Spec.Cloud.DatacenterName) {
			return fmt.Errorf("datacenter %q is still in use, can not delete it",
				cluster.Spec.Cloud.DatacenterName)
		}
	}

	if isDelete && len(clusters.Items) > 0 {
		return fmt.Errorf("can not delete seed, there are still %d clusters in it", len(clusters.Items))
	}

	return nil
}

func (sv *seedValidator) clientForSeed(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error) {
	cfg, err := sv.seedKubeconfigGetter(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig for seed %q: %v", seed.Name, err)
	}

	var mapper meta.RESTMapper
	mapperKey := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s/%s/%s",
		seed.Name, cfg.Host, cfg.APIPath, cfg.Username, cfg.Password, cfg.BearerToken, cfg.BearerTokenFile,
		string(cfg.CertData), string(cfg.KeyData), string(cfg.CAData))
	rawMapper, exists := sv.restMapperCache.Load(mapperKey)
	if !exists {
		mapper, err = restmapper.NewDynamicRESTMapper(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create restMapper for seed %q: %v", seed.Name, err)
		}
		sv.restMapperCache.Store(mapperKey, mapper)
	} else {
		var ok bool
		mapper, ok = rawMapper.(meta.RESTMapper)
		if !ok {
			return nil, fmt.Errorf("didn't get a restMapper from the cache")
		}
	}
	client, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Mapper: mapper})
	if err != nil {
		return nil, fmt.Errorf("failed to construct client for seed %q: %v", seed.Name, err)
	}

	return client, nil
}
