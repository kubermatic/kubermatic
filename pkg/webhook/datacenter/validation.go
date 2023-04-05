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

package datacenter

import (
	"context"
	"errors"
	"fmt"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/api/v3/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v3/pkg/provider"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type validator struct {
	datacentersGetter provider.DatacentersGetter
	seedClient        ctrlruntimeclient.Client
}

func NewValidator(seedClient ctrlruntimeclient.Client, datacentersGetter provider.DatacentersGetter) admission.CustomValidator {
	return &validator{
		seedClient:        seedClient,
		datacentersGetter: datacentersGetter,
	}
}

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	return v.validate(ctx, nil, obj)
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	return v.validate(ctx, oldObj, newObj)
}

func (v *validator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	datacenter, ok := obj.(*kubermaticv1.Datacenter)
	if !ok {
		return errors.New("given object is not a *Datacenter")
	}

	if err := validateNoClustersRemaining(ctx, v.seedClient, datacenter); err != nil {
		return err
	}

	return nil
}

func (v *validator) validate(ctx context.Context, oldObj, newObj runtime.Object) error {
	newDatacenter, ok := newObj.(*kubermaticv1.Datacenter)
	if !ok {
		return errors.New("given new object is not a *Datacenter")
	}

	newProviderName, err := kubermaticv1helper.DatacenterCloudProviderName(&newDatacenter.Spec.Provider)
	if err != nil {
		return fmt.Errorf("datacenter is invalid: %w", err)
	}
	if newProviderName == "" {
		return errors.New("datacenter has no provider spec defined")
	}

	if oldObj != nil {
		oldDatacenter, ok := oldObj.(*kubermaticv1.Datacenter)
		if !ok {
			return errors.New("given old object is not a *Datacenter")
		}

		oldProviderName, err := kubermaticv1helper.DatacenterCloudProviderName(&oldDatacenter.Spec.Provider)

		// make sure invalid old datacenters can be fixed with new ones,
		// i.e. only validate provider changes if the previous version
		// had a valid provider to begin with
		if err == nil && oldProviderName != newProviderName {
			return fmt.Errorf("cannot change datacenter provider from %s to %s", oldProviderName, newProviderName)
		}
	}

	switch newProviderName {
	case kubermaticv1.CloudProviderKubeVirt:
		if err := validateKubeVirtSupportedOS(newDatacenter.Spec.Provider.KubeVirt); err != nil {
			return err
		}
	}

	return nil
}

func validateNoClustersRemaining(ctx context.Context, seedClient ctrlruntimeclient.Client, subject *kubermaticv1.Datacenter) error {
	// new seed clusters might not yet have the CRDs installed into them,
	// which for the purpose of this validation is not a problem and simply
	// means there can be no Clusters on that seed
	crd := apiextensionsv1.CustomResourceDefinition{}
	key := types.NamespacedName{Name: "clusters.kubermatic.k8c.io"}
	crdExists := true
	if err := seedClient.Get(ctx, key, &crd); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to probe for Cluster CRD: %w", err)
		}

		crdExists = false
	}

	if crdExists {
		// check if there are still clusters using the given datacenter
		clusters := &kubermaticv1.ClusterList{}
		if err := seedClient.List(ctx, clusters); err != nil {
			return fmt.Errorf("failed to list clusters: %w", err)
		}

		// list of all datacenters after the seed would have been persisted
		inUse := sets.Set[string]{}

		for _, cluster := range clusters.Items {
			if cluster.Spec.Cloud.DatacenterName == subject.Name {
				inUse.Insert(cluster.Name)
			}
		}

		if inUse.Len() > 0 {
			return fmt.Errorf("datacenter is still in use by clusters: %v", sets.List(inUse))
		}
	}

	return nil
}

func validateKubeVirtSupportedOS(datacenterSpec *kubermaticv1.DatacenterSpecKubeVirt) error {
	if datacenterSpec != nil && datacenterSpec.Images.HTTP != nil {
		for os := range datacenterSpec.Images.HTTP.OperatingSystems {
			if _, exist := kubermaticv1.SupportedKubeVirtOS[os]; !exist {
				return fmt.Errorf("invalid/unsupported operating system specified: %s", os)
			}
		}
	}

	return nil
}
