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

// Validate returns an error if the given seed does not pass all validation steps.
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
	// this can be nil on fresh seed clusters
	existingSeed := existingSeeds[seed.Name]

	// remove the seed itself from the list, so uniqueness checks won't fail
	delete(existingSeeds, seed.Name)

	ourDatacenters := sets.NewString()
	for datacenter := range seed.Spec.Datacenters {
		ourDatacenters.Insert(datacenter)
	}

	// check if all the DCs in the seed are unique
	for _, s := range existingSeeds {
		for dc := range s.Spec.Datacenters {
			if ourDatacenters.Has(dc) {
				return fmt.Errorf("datacenter %q already exists in seed %q, can only have one datacenter with a given name", dc, s.Name)
			}
		}
	}

	// check if all DCs have exactly one provider and that the provider
	// is never changed after it has been set once
	for dcName, dc := range seed.Spec.Datacenters {
		providerName, err := provider.DatacenterCloudProviderName(&dc.Spec)
		if err != nil {
			return fmt.Errorf("datacenter %q is invalid: %v", dcName, err)
		}
		if providerName == "" {
			return fmt.Errorf("datacenter %q has no provider defined", dcName)
		}

		if existingSeed == nil {
			continue
		}

		existingDC, ok := existingSeed.Spec.Datacenters[dcName]
		if !ok {
			continue
		}

		existingProvider, _ := provider.DatacenterCloudProviderName(&existingDC.Spec)
		if providerName != existingProvider {
			return fmt.Errorf("cannot change datacenter %q provider from %q to %q", dcName, existingProvider, providerName)
		}
	}

	// check if there are still clusters using DCs not defined anymore
	clusters := &kubermaticv1.ClusterList{}
	if err := seedClient.List(sv.ctx, clusters, sv.listOpts); err != nil {
		return fmt.Errorf("failed to list clusters: %v", err)
	}

	for _, cluster := range clusters.Items {
		if !ourDatacenters.Has(cluster.Spec.Cloud.DatacenterName) {
			return fmt.Errorf("datacenter %q is still in use, cannot delete it", cluster.Spec.Cloud.DatacenterName)
		}
	}

	if isDelete && len(clusters.Items) > 0 {
		return fmt.Errorf("can not delete seed, there are still %d clusters in it", len(clusters.Items))
	}

	return nil
}
