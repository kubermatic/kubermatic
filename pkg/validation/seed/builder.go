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

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
)

type ValidationHandlerBuilder struct {
	client              ctrlruntimeclient.Client
	workerName          string
	seedName            string
	singleSeedValidator *ensureSingleSeedValidatorWrapper
}

// Client sets the client to get API resources on the cluster the handler is
// used for.
func (v *ValidationHandlerBuilder) Client(client ctrlruntimeclient.Client) *ValidationHandlerBuilder {
	v.client = client
	return v
}

// WorkerName sets the workerName value to be used to list clusters.
// TODO(irozzo) check how this is useful.
func (v *ValidationHandlerBuilder) WorkerName(workerName string) *ValidationHandlerBuilder {
	v.workerName = workerName
	return v
}

// SeedName sets the name of the Seed cluster the admission handler is used for.
func (v *ValidationHandlerBuilder) SeedName(seedName string) *ValidationHandlerBuilder {
	v.seedName = seedName
	return v
}

// AllowedSeed sets name and namespace of the single seed that will be allowed to be
// created by the admission handler.
func (v *ValidationHandlerBuilder) AllowedSeed(namespace string, name string) *ValidationHandlerBuilder {
	v.singleSeedValidator = &ensureSingleSeedValidatorWrapper{Namespace: namespace, Name: name}
	return v
}

// Build returns an AdmissionHandler for Seed CRs.
func (v *ValidationHandlerBuilder) Build(ctx context.Context) (AdmissionHandler, error) {
	if v.client == nil {
		return nil, errors.New("cannot build admission handler without setting client")
	}

	var seedClientGetter provider.SeedClientGetter
	if v.seedName == "" {
		// Handler used in master cluster
		seedKubeconfigGetter, err := provider.SeedKubeconfigGetterFactory(ctx, v.client)
		if err != nil {
			return nil, fmt.Errorf("error occurred while creating seed kubeconfig getter: %v", err)
		}
		seedClientGetter = provider.SeedClientGetterFactory(seedKubeconfigGetter)
	} else {
		// Handler used in seed cluster
		seedClientGetter = (&identitySeedClientGetter{client: v.client, seedName: v.seedName}).Get
	}

	sv, err := newSeedValidator(
		v.workerName,
		v.client,
		seedClientGetter,
	)
	if err != nil {
		return nil, fmt.Errorf("error occurred while creating seed validator: %v", err)
	}

	if v.singleSeedValidator != nil {
		v.singleSeedValidator.validateFunc = sv.Validate
		return &seedAdmissionHandler{
			validateFunc: sv.Validate,
		}, nil
	}
	return &seedAdmissionHandler{validateFunc: sv.Validate}, nil

}

// identitySeedClientGetter is a Seed Client getter used when the webhook is
// deployed on the Seed cluster.
type identitySeedClientGetter struct {
	client   ctrlruntimeclient.Client
	seedName string
}

func (i *identitySeedClientGetter) Get(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error) {
	if seed.Name != i.seedName {
		return nil, fmt.Errorf("can only return kubeconfig for our own seed (%q), got request for %q", i.seedName, seed.Name)
	}
	return i.client, nil
}
