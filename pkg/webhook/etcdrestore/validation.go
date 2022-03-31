/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package etcdrestore

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// validator for mutating Kubermatic EtcdRestore CRD.
type validator struct {
	seedGetter       provider.SeedGetter
	seedClientGetter provider.SeedClientGetter
}

var _ admission.CustomValidator = &validator{}

// NewValidator returns a new EtcdRestore validator.
func NewValidator(seedGetter provider.SeedGetter,
	seedClientGetter provider.SeedClientGetter) admission.CustomValidator {
	return &validator{
		seedGetter:       seedGetter,
		seedClientGetter: seedClientGetter,
	}
}

func (v validator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	return v.validateEtcdLauncherEnabled(ctx, obj)
}

func (v validator) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) error {
	return v.validateEtcdLauncherEnabled(ctx, newObj)
}

func (v validator) ValidateDelete(_ context.Context, _ runtime.Object) error {
	return nil
}

// validateEtcdLauncherEnabled checks if cluster has etcd launcher enabled because it
// is required for restores to work properly.
func (v validator) validateEtcdLauncherEnabled(ctx context.Context, obj interface{}) error {
	etcdRestore, ok := obj.(*kubermaticv1.EtcdRestore)
	if !ok {
		return fmt.Errorf("object is not kubermaticv1.EtcdRestore")
	}

	c, err := v.getCluster(ctx, etcdRestore.Spec.Cluster)
	if err != nil {
		return err
	}

	if !c.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher] {
		return fmt.Errorf("etcd-launcher feature must be enabled for restoring clusters from etcd snapshots")
	}
	return nil
}

func (v validator) getCluster(ctx context.Context, clusterRef v1.ObjectReference) (*kubermaticv1.Cluster, error) {
	seed, err := v.seedGetter()
	if err != nil {
		return nil, fmt.Errorf("failed to get current Seed: %w", err)
	}
	if seed == nil {
		return nil, fmt.Errorf("webhook not configured for a Seed cluster, cannot validate EtcdRestore resources")
	}

	client, err := v.seedClientGetter(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to get Seed client: %w", err)
	}

	c := new(kubermaticv1.Cluster)
	err = client.Get(ctx, client2.ObjectKey{Namespace: clusterRef.Namespace, Name: clusterRef.Name}, c)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster object: %w", err)
	}

	return c, nil
}
