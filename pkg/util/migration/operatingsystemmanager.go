/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

// This should be removed with KKP 2.23.

package migration

import (
	"context"
	"fmt"

	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	cleanupFinalizer = "kubermatic.k8c.io/cleanup-kubermatic-operating-system-profiles"

	customOperatingSystemProfileAPIVersion = "operatingsystemmanager.k8c.io/v1alpha1"
	customOperatingSystemProfileKind       = "CustomOperatingSystemProfile"
	operatingSystemProfileKind             = "OperatingSystemProfile"
	customOperatingSystemProfileListKind   = "CustomOperatingSystemProfileList"
)

var defaultOSPs = map[string]bool{
	"osp-amzn2":      true,
	"osp-centos":     true,
	"osp-flatcar":    true,
	"osp-rhel":       true,
	"osp-rockylinux": true,
	"osp-ubuntu":     true,
	"osp-sles":       true,
}

func crdExists(ctx context.Context, client ctrlruntimeclient.Client, name string) (bool, error) {
	crd := apiextensionsv1.CustomResourceDefinition{}
	key := types.NamespacedName{Name: name}
	if err := client.Get(ctx, key, &crd); err != nil {
		if !apierrors.IsNotFound(err) {
			return false, fmt.Errorf("failed to retrieve %q CRD: %w", name, err)
		}
		return false, nil
	}
	return true, nil
}

func IsOperatingSystemProfileInstalled(ctx context.Context, client ctrlruntimeclient.Client) (bool, error) {
	return crdExists(ctx, client, "operatingsystemprofiles.operatingsystemmanager.k8c.io")
}

func IsOperatingSystemConfigInstalled(ctx context.Context, client ctrlruntimeclient.Client) (bool, error) {
	return crdExists(ctx, client, "operatingsystemconfigs.operatingsystemmanager.k8c.io")
}

func ConvertOSPsToCustomOSPs(ctx context.Context, client ctrlruntimeclient.Client, seedNamespace string) error {
	ospList := &osmv1alpha1.OperatingSystemProfileList{}
	err := client.List(ctx, ospList, &ctrlruntimeclient.ListOptions{Namespace: seedNamespace})
	if err != nil {
		return fmt.Errorf("failed to list operatingSystemProfiles: %w", err)
	}

	if len(ospList.Items) == 0 {
		return nil
	}

	// We use an error aggregate to make sure we process everything we can and keep aggregating errors that we run into along the way.
	var errs []error
	for _, osp := range ospList.Items {
		// step 1: convert OSP to CustomOperatingSytemProfile.
		customOSP, err := ospToCustomOSP(osp)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to convert OSP to Custom OSP: %w", err))
			continue
		}

		// Step 2: strip resource versioning.
		customOSP.SetResourceVersion("")
		customOSP.SetUID("")
		customOSP.SetSelfLink("")
		customOSP.SetGeneration(0)

		// step 3: create the converted CustomOperatingSystemProfile.
		if err := client.Create(ctx, customOSP); err != nil {
			errs = append(errs, fmt.Errorf("failed to create CustomOperatingSystemProfile: %w", err))
			continue
		}

		// At this point we have converted the OSP to Custom OSP. Now we need to delete the existing OSP.
		// step 4: remove finalizer from OperatingSystemProfile.
		if err := kuberneteshelper.TryRemoveFinalizer(ctx, client, &osp, cleanupFinalizer); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove OSP finalizer: %w", err))
			continue
		}

		// step 4: delete OperatingSystemProfile.
		if err := client.Delete(ctx, &osp); err != nil {
			errs = append(errs, fmt.Errorf("failed to delete CustomOperatingSystemProfile: %w", err))
			continue
		}
	}

	return kerrors.NewAggregate(errs)
}

func MoveOSPToUserClusters(ctx context.Context, seedClient ctrlruntimeclient.Client, client ctrlruntimeclient.Client, namespace string) error {
	ospList := &osmv1alpha1.OperatingSystemProfileList{}
	err := seedClient.List(ctx, ospList, &ctrlruntimeclient.ListOptions{Namespace: namespace})
	if err != nil {
		return fmt.Errorf("failed to list operatingSystemProfiles: %w", err)
	}

	if len(ospList.Items) == 0 {
		return nil
	}

	// We use an error aggregate to make sure we process everything we can and keep aggregating errors that we run into along the way.
	var errs []error
	for _, osp := range ospList.Items {
		// step 1: check if it's a default OSP
		_, ok := defaultOSPs[osp.Name]
		if !ok {
			exists := true
			// Step 2: check if the OSP already exists in the user cluster. Since OSPs are immutable we won't update the existing resource and just move forward.
			userClusterOSP := osmv1alpha1.OperatingSystemProfile{}
			if err := client.Get(ctx, types.NamespacedName{Name: osp.Name, Namespace: metav1.NamespaceSystem}, &userClusterOSP); err != nil {
				if !apierrors.IsNotFound(err) {
					errs = append(errs, fmt.Errorf("failed to get OperatingSystemProfile: %w", err))
					continue
				}
				exists = false
			}

			if !exists {
				// step 3: move custom OSP to user cluster if it doesn't already exist.
				userClusterOSP := osp.DeepCopy()
				userClusterOSP.Namespace = metav1.NamespaceSystem
				userClusterOSP.SetResourceVersion("")
				userClusterOSP.SetUID("")
				userClusterOSP.SetSelfLink("")
				userClusterOSP.SetGeneration(0)

				if err := client.Create(ctx, userClusterOSP); err != nil {
					errs = append(errs, fmt.Errorf("failed to create OperatingSystemProfile: %w", err))
					continue
				}
			}
		}

		// step 4: delete OperatingSystemProfile.
		if err := seedClient.Delete(ctx, &osp); err != nil {
			errs = append(errs, fmt.Errorf("failed to delete OperatingSystemProfile: %w", err))
			continue
		}
	}

	return kerrors.NewAggregate(errs)
}

// ospToCustomOSP converts an OSP to an Unstructured custom OSP.
func ospToCustomOSP(osp osmv1alpha1.OperatingSystemProfile) (*unstructured.Unstructured, error) {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(osp)
	if err != nil {
		return nil, err
	}
	u := &unstructured.Unstructured{Object: obj}
	u.SetAPIVersion(customOperatingSystemProfileAPIVersion)
	u.SetKind(customOperatingSystemProfileKind)
	return u, nil
}
