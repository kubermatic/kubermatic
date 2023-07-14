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

package seedsync

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// This controller is responsible for synchronizing the `Seed` custom resources across all seed clusters.
	ControllerName = "kkp-seed-sync-controller"

	// ManagedByLabel is the label used to identify the resources
	// created by this controller.
	ManagedByLabel = "app.kubernetes.io/managed-by"

	// CleanupFinalizer is put on Seed CRs to facilitate proper
	// cleanup when a Seed is deleted.
	CleanupFinalizer = "kubermatic.k8c.io/cleanup-seed-sync"
)

// Add creates a new Seed-Sync controller and sets up Watches.
func Add(
	mgr manager.Manager,
	numWorkers int,
	log *zap.SugaredLogger,
	namespace string,
	seedKubeconfigGetter provider.SeedKubeconfigGetter,
	seedsGetter provider.SeedsGetter,
) error {
	reconciler := &Reconciler{
		Client:           mgr.GetClient(),
		recorder:         mgr.GetEventRecorderFor(ControllerName),
		log:              log.Named(ControllerName),
		seedClientGetter: kubernetesprovider.SeedClientGetterFactory(seedKubeconfigGetter),
	}

	ctrlOptions := controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	nsPredicate := predicate.ByNamespace(namespace)

	// watch all seeds in the given namespace
	if err := c.Watch(source.Kind(mgr.GetCache(), &kubermaticv1.Seed{}), &handler.EnqueueRequestForObject{}, nsPredicate); err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// watch all KubermaticConfigurations in the given namespace
	configHandler := func(_ context.Context, o ctrlruntimeclient.Object) []reconcile.Request {
		seeds, err := seedsGetter()
		if err != nil {
			log.Errorw("Failed to retrieve seeds", zap.Error(err))
			return nil
		}

		requests := []reconcile.Request{}
		for _, seed := range seeds {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: seed.GetNamespace(),
					Name:      seed.GetName(),
				},
			})
		}

		return requests
	}

	if err := c.Watch(source.Kind(mgr.GetCache(), &kubermaticv1.KubermaticConfiguration{}), handler.EnqueueRequestsFromMapFunc(configHandler), nsPredicate); err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	return nil
}
