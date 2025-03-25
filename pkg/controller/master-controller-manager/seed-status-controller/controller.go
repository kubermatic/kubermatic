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

package seedstatuscontroller

import (
	"context"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	// ControllerName is the name of this very controller.
	ControllerName = "kkp-seed-status-controller"
)

// Add creates a new seed status controller and sets up watches.
func Add(
	ctx context.Context,
	mgr manager.Manager,
	numWorkers int,
	log *zap.SugaredLogger,
	namespace string,
	seedKubeconfigGetter provider.SeedKubeconfigGetter,
	versions kubermatic.Versions,
) error {
	reconciler := &Reconciler{
		Client:               mgr.GetClient(),
		recorder:             mgr.GetEventRecorderFor(ControllerName),
		log:                  log.Named(ControllerName),
		seedKubeconfigGetter: seedKubeconfigGetter,
		seedClientGetter:     kubernetesprovider.SeedClientGetterFactory(seedKubeconfigGetter),
		versions:             versions,
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Seed{}, builder.WithPredicates(predicate.ByNamespace(namespace))).
		Build(reconciler)

	return err
}
