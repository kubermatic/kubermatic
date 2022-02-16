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
	"errors"
	"fmt"
	"sync"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type validator struct {
	seedsGetter      provider.SeedsGetter
	seedClientGetter provider.SeedClientGetter
	features         features.FeatureGate
	lock             *sync.Mutex
}

func newSeedValidator(
	seedsGetter provider.SeedsGetter,
	seedClientGetter provider.SeedClientGetter,
	features features.FeatureGate,
) (*validator, error) {
	return &validator{
		seedsGetter:      seedsGetter,
		seedClientGetter: seedClientGetter,
		features:         features,
		lock:             &sync.Mutex{},
	}, nil
}

var _ admission.CustomValidator = &validator{}

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	return v.validate(ctx, obj, false)
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	return v.validate(ctx, newObj, false)
}

func (v *validator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return v.validate(ctx, obj, true)
}

func (v *validator) validate(ctx context.Context, obj runtime.Object, isDelete bool) error {
	subject, ok := obj.(*kubermaticv1.Seed)
	if !ok {
		return errors.New("given object is not a Seed")
	}

	// We need locking to make the validation concurrency-safe
	// TODO: this is acceptable as request rate is low, but is it required?
	v.lock.Lock()
	defer v.lock.Unlock()

	// fetch all relevant Seeds; in CE this will only be the one supported Seed,
	// in EE this is a map of all Seed resources; since in CE naming a Seed
	// anything but "kubermatic" is forbidden, it's okay that we use the default
	// SeedsGetter, which will also only return this one Seed (i.e. the webhook
	// in CE never sees any other Seeds)
	existingSeeds, err := v.seedsGetter()
	if err != nil {
		return fmt.Errorf("failed to list Seeds: %w", err)
	}

	if isDelete {
		// when a namespace is deleted, a DELETE call for all Seeds in the namespace
		// is issued; this request has no .Request.Name set, so this check will make
		// sure that we exit cleanly and allow deleting namespaces without seeds
		if _, exists := existingSeeds[subject.Name]; !exists {
			return nil
		}

		// in case of delete request the seed is empty, so fetch the current one from
		// the cluster instead
		subject = existingSeeds[subject.Name]
	}

	// get a client for the Seed cluster; this uses a restmapper and is therefore cached for better performance
	seedClient, err := v.seedClientGetter(subject)
	if err != nil {
		return fmt.Errorf("failed to get Seed client: %w", err)
	}

	// check whether the expose strategy is allowed or not.
	if subject.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyTunneling && !v.features.Enabled(features.TunnelingExposeStrategy) {
		return errors.New("cannot create Seed using Tunneling as a default expose strategy, the TunnelingExposeStrategy feature gate is not enabled")
	}

	// this can be nil on new seed clusters
	existingSeed := existingSeeds[subject.Name]

	// remove the seed itself from the map, so uniqueness checks won't fail
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
	for _, existing := range existingSeeds {
		datacenters := sets.StringKeySet(existing.Spec.Datacenters)

		if duplicates := subjectDatacenters.Intersection(datacenters); duplicates.Len() > 0 {
			return fmt.Errorf("Seed redefines existing datacenters %v from Seed %q; datacenter names must be globally unique", duplicates.List(), existing.Name)
		}

		existingDatacenters = existingDatacenters.Union(datacenters)
	}

	// check if all DCs have exactly one provider and that the provider
	// is never changed after it has been set once
	for dcName, dc := range subject.Spec.Datacenters {
		providerName, err := provider.DatacenterCloudProviderName(&dc.Spec)
		if err != nil {
			return fmt.Errorf("datacenter %q is invalid: %w", dcName, err)
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
	if err := seedClient.List(ctx, clusters); err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
	}

	// list of all datacenters after the seed would have been persisted
	finalDatacenters := subjectDatacenters.Union(existingDatacenters)

	for _, cluster := range clusters.Items {
		if !finalDatacenters.Has(cluster.Spec.Cloud.DatacenterName) {
			return fmt.Errorf("datacenter %q is still in use by cluster %q, cannot delete it", cluster.Spec.Cloud.DatacenterName, cluster.Name)
		}
	}

	if subject.Spec.EtcdBackupRestore != nil && len(subject.Spec.EtcdBackupRestore.Destinations) > 0 &&
		subject.Spec.EtcdBackupRestore.DefaultDestination != nil && *subject.Spec.EtcdBackupRestore.DefaultDestination != "" {
		if _, ok := subject.Spec.EtcdBackupRestore.Destinations[*subject.Spec.EtcdBackupRestore.DefaultDestination]; !ok {
			return fmt.Errorf("default etcd backup destination %q has to match a destination in the backup destinations", *subject.Spec.EtcdBackupRestore.DefaultDestination)
		}
	}

	return nil
}
