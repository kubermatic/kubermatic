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

package datacenterstatuscontroller

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/version/kubermatic"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// ControllerName is the name of this very controller.
	ControllerName = "kkp-datacenter-status-controller"
)

// Add creates a new datacenter status controller and sets up watches.
func Add(
	ctx context.Context,
	mgr manager.Manager,
	numWorkers int,
	log *zap.SugaredLogger,
	versions kubermatic.Versions,
) error {
	reconciler := &Reconciler{
		seedClient: mgr.GetClient(),
		recorder:   mgr.GetEventRecorderFor(ControllerName),
		log:        log.Named(ControllerName),
		versions:   versions,
	}

	ctrlOptions := controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	// watch all datacenters
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Datacenter{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	return nil
}
