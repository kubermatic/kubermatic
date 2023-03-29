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

	kuberneteshelper "k8c.io/kubermatic/v3/pkg/kubernetes"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
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

	OperatingSystemManagerAPIVersion = "operatingsystemmanager.k8c.io/v1alpha1"
	customOperatingSystemProfileKind = "CustomOperatingSystemProfile"
	OperatingSystemProfileKind       = "OperatingSystemProfile"
	OperatingSystemConfigKind        = "OperatingSystemConfig"
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
	ospList := &unstructured.UnstructuredList{}
	ospList.SetAPIVersion(OperatingSystemManagerAPIVersion)
	ospList.SetKind(fmt.Sprintf("%sList", OperatingSystemProfileKind))

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
		customOSP, err := ospToCustomOSP(&osp)
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
	ospList := &unstructured.UnstructuredList{}
	ospList.SetAPIVersion(OperatingSystemManagerAPIVersion)
	ospList.SetKind(fmt.Sprintf("%sList", OperatingSystemProfileKind))

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
		_, ok := defaultOSPs[osp.GetName()]
		if !ok {
			exists := true
			// Step 2: check if the OSP already exists in the user cluster. Since OSPs are immutable we won't update the existing resource and just move forward.
			userClusterOSP := osmv1alpha1.OperatingSystemProfile{}
			if err := client.Get(ctx, types.NamespacedName{Name: osp.GetName(), Namespace: metav1.NamespaceSystem}, &userClusterOSP); err != nil {
				if !apierrors.IsNotFound(err) {
					errs = append(errs, fmt.Errorf("failed to get OperatingSystemProfile: %w", err))
					continue
				}
				exists = false
			}

			if !exists {
				// step 3: move custom OSP to user cluster if it doesn't already exist.
				userClusterOSP, err := unstructuredOSPToOSP(&osp)
				if err != nil {
					errs = append(errs, fmt.Errorf("failed to get convert Unstructured OperatingSystemProfile: %w", err))
					continue
				}

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

func IsKubeOneInstallation(ctx context.Context, seedClient ctrlruntimeclient.Client) (bool, error) {
	osmDeploy := &appsv1.Deployment{}
	osmDeploymentName := "operating-system-manager"
	if err := seedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: metav1.NamespaceSystem, Name: osmDeploymentName}, osmDeploy); err != nil {
		if !apierrors.IsNotFound(err) {
			return false, fmt.Errorf("failed to get %s deployment: %w", osmDeploymentName, err)
		}
		return false, nil
	}
	return true, nil
}

// ospToCustomOSP converts an OSP to an Unstructured custom OSP.
func ospToCustomOSP(osp *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	u := osp.DeepCopy()
	u.SetAPIVersion(OperatingSystemManagerAPIVersion)
	u.SetKind(customOperatingSystemProfileKind)
	return u, nil
}

func unstructuredOSPToOSP(u *unstructured.Unstructured) (*osmv1alpha1.OperatingSystemProfile, error) {
	osp := &osmv1alpha1.OperatingSystemProfile{}
	// Required for converting CustomOperatingSystemProfile to OperatingSystemProfile.
	obj := u.DeepCopy()
	obj.SetKind(OperatingSystemProfileKind)
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, osp); err != nil {
		return osp, fmt.Errorf("failed to decode OperatingSystemProfile: %w", err)
	}
	return osp, nil
}
