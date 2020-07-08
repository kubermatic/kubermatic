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
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ValidateFunc validates a Seed resource
// On DELETE, the only set fields of seed are Name and Namespace as
// admissionReview.Request.Object is unset
type ValidateFunc func(ctx context.Context, seed *kubermaticv1.Seed, op admissionv1beta1.Operation) error

func NewDefaultSeedValidator(
	workerName string,
	seedsGetter provider.SeedsGetter,
	seedClientGetter provider.SeedClientGetter,
) (*Validator, error) {
	labelSelector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return nil, err
	}
	listOpts := &ctrlruntimeclient.ListOptions{LabelSelector: labelSelector}
	return &Validator{
		seedsGetter:      seedsGetter,
		seedClientGetter: seedClientGetter,
		lock:             &sync.Mutex{},
		listOpts:         listOpts,
	}, nil
}

// Ensure that Validator.Validate implements ValidateFunc
var _ ValidateFunc = (&Validator{}).Validate

type Validator struct {
	seedsGetter      provider.SeedsGetter
	seedClientGetter provider.SeedClientGetter
	lock             *sync.Mutex
	// Can be used to insert a labelSelector
	listOpts *ctrlruntimeclient.ListOptions
}

// Validate returns an error if the given seed does not pass all validation steps.
func (sv *Validator) Validate(ctx context.Context, seed *kubermaticv1.Seed, op admissionv1beta1.Operation) error {
	// We need locking to make the validation concurrency-safe
	// (irozzo): this is acceptable as request rate is low, but is it
	// required?
	sv.lock.Lock()
	defer sv.lock.Unlock()

	// (irozzo): Double check why we do need this getter and we cannot rely on
	// the controller-runtime client only.
	seeds, err := sv.seedsGetter()
	if err != nil {
		return fmt.Errorf("failed to get seeds: %v", err)
	}
	if op == admissionv1beta1.Delete {
		// when a namespace is deleted, a DELETE call for all seeds in the namespace
		// is issued; this request has no .Request.Name set, so this check will make
		// sure that we exit cleanly and allow deleting namespaces without seeds
		if _, exists := seeds[seed.Name]; !exists && op == admissionv1beta1.Delete {
			return nil
		}
		// in case of delete request the seed is empty
		seed = seeds[seed.Name]
	}

	client, err := sv.seedClientGetter(seed)
	if err != nil {
		return fmt.Errorf("failed to get client for seed %q: %v", seed.Name, err)
	}

	return sv.validate(ctx, seed, client, seeds, op == admissionv1beta1.Delete)
}

func (sv *Validator) validate(ctx context.Context, subject *kubermaticv1.Seed, seedClient ctrlruntimeclient.Client, existingSeeds map[string]*kubermaticv1.Seed, isDelete bool) error {
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
	if err := seedClient.List(ctx, clusters, sv.listOpts); err != nil {
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

//EnsureSingleSeedValidator ensures that only the seed with the given Name and
//Namespace can be created.
type EnsureSingleSeedValidator struct {
	Name      string
	Namespace string
}

// Ensure that SeedValidator.Validate implements ValidateFunc
var _ ValidateFunc = EnsureSingleSeedValidator{}.Validate

func (e EnsureSingleSeedValidator) Validate(_ context.Context, seed *kubermaticv1.Seed, op admissionv1beta1.Operation) error {
	switch op {
	case admissionv1beta1.Create:
		if seed.Name != e.Name || seed.Namespace != e.Namespace {
			return fmt.Errorf("cannot create Seed %s/%s. It must be named %s and installed in the %s namespace", seed.Name, seed.Namespace, e.Name, e.Namespace)
		}
		return nil
	default:
		return nil
	}
}

// SingleSeedValidateFunc returns a SeedValidateFunc which ensures that a
// single Seed named with the default seed name in the given namespace is created.
// This validator is used for Kubermatic Community Edition.
func SingleSeedValidateFunc(namespace string) ValidateFunc {
	return EnsureSingleSeedValidator{Name: provider.DefaultSeedName, Namespace: namespace}.Validate
}

// CombineSeedValidateFuncs combines two or more SeedValidateFunc by running
// them sequentially in the order they are given and returning an error at the
// first validation failure.
func CombineSeedValidateFuncs(funcs ...ValidateFunc) ValidateFunc {
	return func(ctx context.Context, seed *kubermaticv1.Seed, op admissionv1beta1.Operation) error {
		for _, f := range funcs {
			if err := f(ctx, seed, op); err != nil {
				return err
			}
		}
		return nil
	}
}
