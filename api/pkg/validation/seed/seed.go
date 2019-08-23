package seed

import (
	"context"
	"fmt"
	"sync"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func newValidator(
	ctx context.Context,
	seedsGetter provider.SeedsGetter,
	seedClientGetter provider.SeedClientGetter,
	listOpts *ctrlruntimeclient.ListOptions) *seedValidator {

	return &seedValidator{
		ctx:              ctx,
		seedsGetter:      seedsGetter,
		seedClientGetter: seedClientGetter,
		lock:             &sync.Mutex{},
		listOpts:         listOpts,
	}
}

type seedValidator struct {
	ctx              context.Context
	seedsGetter      provider.SeedsGetter
	seedClientGetter provider.SeedClientGetter
	lock             *sync.Mutex
	// Can be used to insert a labelSelector
	listOpts *ctrlruntimeclient.ListOptions
}

func (sv *seedValidator) Validate(seed *kubermaticv1.Seed, isDelete bool) error {
	// We need locking to make the validation concurrency-safe
	sv.lock.Lock()
	defer sv.lock.Unlock()

	client, err := sv.seedClientGetter(seed)
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
