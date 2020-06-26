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

func (sv *seedValidator) validate(subject *kubermaticv1.Seed, seedClient ctrlruntimeclient.Client, existingSeeds map[string]*kubermaticv1.Seed, isDelete bool) error {
	// this can be nil on new seed clusters
	existingSeed := existingSeeds[subject.Name]

	// remove the seed itself from the list, so uniqueness checks won't fail
	delete(existingSeeds, subject.Name)

	// collect datacenter names
	subjectDatacenters := sets.NewString()
	existingDatacenters := sets.NewString()

	if !isDelete {
		// this has no effect on the DC uniqueness check, but makes the
		// cluster-remaining-in-DC check easier
		subjectDatacenters = sets.StringKeySet(subject.Spec.Datacenters)
	}

	// check if the subject introduces a datacenter that already exists
	for _, existingSeed := range existingSeeds {
		datacenters := sets.StringKeySet(existingSeed.Spec.Datacenters)

		if duplicates := subjectDatacenters.Intersection(datacenters); duplicates.Len() > 0 {
			return fmt.Errorf("seed redefines existing datacenters %v from seed %q; datacenter names must be globally unique", duplicates.List(), existingSeed.Name)
		}

		existingDatacenters = existingDatacenters.Union(datacenters)
	}

	// check if all DCs have exactly one provider and that the provider
	// is never changed after it has been set once
	for dcName, dc := range subject.Spec.Datacenters {
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

	// list of all datacenters after the seed would have been persisted
	finalDatacenters := subjectDatacenters.Union(existingDatacenters)

	for _, cluster := range clusters.Items {
		if !finalDatacenters.Has(cluster.Spec.Cloud.DatacenterName) {
			return fmt.Errorf("datacenter %q is still in use by cluster %q, cannot delete it", cluster.Spec.Cloud.DatacenterName, cluster.Name)
		}
	}

	return nil
}
