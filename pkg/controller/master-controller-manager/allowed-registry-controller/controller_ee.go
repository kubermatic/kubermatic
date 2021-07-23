// +build ee

/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package allowedregistrycontroller

import (
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	allowedregistrycontroller "k8c.io/kubermatic/v2/pkg/ee/allowed-registry-controller"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// This controller creates corresponding OPA Constraint Templates and Default Constraints based on AllowedRegistry data.
	ControllerName = "allowed_registry_controller"
)

func Add(mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	namespace string) error {

	reconciler := allowedregistrycontroller.NewReconciler(
		log.Named(ControllerName),
		mgr.GetEventRecorderFor(ControllerName),
		mgr.GetClient(),
		namespace,
	)

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.AllowedRegistry{}},
		&handler.EnqueueRequestForObject{},
	); err != nil {
		return fmt.Errorf("failed to create watch for allowedRegistries: %v", err)
	}
	return nil
}
