/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package applicationcatalogmanager

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	catalogv1alpha1 "k8c.io/application-catalog-manager/pkg/apis/applicationcatalog/v1alpha1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// DefaultApplicationCatalogName is the name of the default ApplicationCatalog CR
	DefaultApplicationCatalogName = "default-catalog"
)

// ReconcileDefaultApplicationCatalog ensures the default ApplicationCatalog CR exists
// when the ExternalApplicationCatalogManager feature gate is enabled.
func ReconcileDefaultApplicationCatalog(
	ctx context.Context,
	cfg *kubermaticv1.KubermaticConfiguration,
	client ctrlruntimeclient.Client,
	scheme *runtime.Scheme,
	logger *zap.SugaredLogger,
) error {
	logger.Debug("Reconciling default ApplicationCatalog CR")

	// Create the default ApplicationCatalog CR with empty spec.
	// The application-catalog webhook will mutate it to inject default Helm charts
	catalog := &catalogv1alpha1.ApplicationCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: DefaultApplicationCatalogName,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "kubermatic-operator",
				"app.kubernetes.io/component":  "application-catalog",
			},
		},
		Spec: catalogv1alpha1.ApplicationCatalogSpec{
			Helm: &catalogv1alpha1.HelmSpec{},
		},
	}

	if err := controllerutil.SetControllerReference(cfg, catalog, scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	existingCatalog := &catalogv1alpha1.ApplicationCatalog{}

	err := client.Get(ctx, types.NamespacedName{Name: DefaultApplicationCatalogName}, existingCatalog)
	if err != nil {
		if errors.IsNotFound(err) {
			return client.Create(ctx, catalog)
		}

		return err
	}

	if existingCatalog.Labels == nil {
		existingCatalog.Labels = make(map[string]string)
	}

	needsUpdate := false

	if existingCatalog.Labels["app.kubernetes.io/managed-by"] != "kubermatic-operator" {
		existingCatalog.Labels["app.kubernetes.io/managed-by"] = "kubermatic-operator"
		needsUpdate = true
	}

	if existingCatalog.Labels["app.kubernetes.io/component"] != "application-catalog" {
		existingCatalog.Labels["app.kubernetes.io/component"] = "application-catalog"
		needsUpdate = true
	}

	if needsUpdate {
		return client.Update(ctx, existingCatalog)
	}

	return nil
}

func CleanupApplicationCatalogs(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	logger *zap.SugaredLogger,
) error {
	catalogList := &catalogv1alpha1.ApplicationCatalogList{}
	if err := client.List(ctx, catalogList); err != nil {
		return err
	}

	var errs []error
	for _, catalog := range catalogList.Items {
		if catalog.Labels["app.kubernetes.io/managed-by"] != "kubermatic-operator" {
			continue
		}

		err := client.Delete(ctx, &catalog)
		if err != nil && !errors.IsNotFound(err) {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to cleanup ApplicationCatalogs: %v", errs)
	}

	return nil
}
