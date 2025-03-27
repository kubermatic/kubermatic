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

package seedinit

import (
	"context"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	// This controller is responsible for the initial setup of new Seed clusters.
	ControllerName = "kkp-seed-init-operator"
)

func Add(
	ctx context.Context,
	log *zap.SugaredLogger,
	namespace string,
	masterManager manager.Manager,
	seedClientGetter provider.SeedClientGetter,
	numWorkers int,
	workerName string,
) error {
	reconciler := &Reconciler{
		log:              log.Named(ControllerName),
		masterClient:     masterManager.GetClient(),
		masterRecorder:   masterManager.GetEventRecorderFor(ControllerName),
		seedClientGetter: seedClientGetter,
		workerName:       workerName,
		versions:         kubermatic.GetVersions(),
	}

	_, err := builder.ControllerManagedBy(masterManager).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Seed{}, builder.WithPredicates(predicateutil.ByNamespace(namespace), predicate.ResourceVersionChangedPredicate{})).
		Build(reconciler)

	return err
}
