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

package util

import (
	"context"
	"fmt"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// RemoveOldApplicationInstallationCRD handles deletion of ApplicationInstallations CRD
// if the existing CRD has scope set to `Cluster`. This action is super safe since
// at the time of writing this method this feature has not been rolled out.
// We can just delete the CRD and install the new one.
// TODO: This should be removed after KKP 2.21 release.
func RemoveOldApplicationInstallationCRD(ctx context.Context, client ctrlruntimeclient.Client) error {
	crd := &apiextensionsv1.CustomResourceDefinition{}
	err := client.Get(ctx, types.NamespacedName{Name: appskubermaticv1.ApplicationInstallationsFQDNName}, crd)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to get CRD %s: %w", appskubermaticv1.ApplicationInstallationsFQDNName, err)
	}

	// No action is required
	if err != nil && apierrors.IsNotFound(err) {
		return nil
	}

	// CRD exists, now we need to check if it's cluster scoped
	if crd.Spec.Scope == apiextensionsv1.ClusterScoped {
		// We need to delete this CRD
		if err := client.Delete(ctx, crd); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete CRD %s: %w", crd.Name, err)
		}
	}
	return nil
}
