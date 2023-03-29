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
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	predicateutil "k8c.io/kubermatic/v3/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v3/pkg/provider"
	"k8c.io/kubermatic/v3/pkg/version/kubermatic"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
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
		versions:         kubermatic.NewDefaultVersions(),
	}

	ctrlOpts := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, masterManager, ctrlOpts)
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	seed := &kubermaticv1.Seed{}
	if err := c.Watch(&source.Kind{Type: seed}, &handler.EnqueueRequestForObject{}, predicateutil.ByNamespace(namespace), predicate.ResourceVersionChangedPredicate{}); err != nil {
		return fmt.Errorf("failed to create watcher for %T: %w", seed, err)
	}

	return nil
}
