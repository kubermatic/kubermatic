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

package applicationdefinitionsynchronizer

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	appkubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"

	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "application_definition_syncing_controller"
)

type reconciler struct {
	log          *zap.SugaredLogger
	recorder     record.EventRecorder
	masterClient ctrlruntimeclient.Client
	seedClients  map[string]ctrlruntimeclient.Client
}

func Add(
	masterManager manager.Manager,
	seedManagers map[string]manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
) error {

	r := &reconciler{
		log:          log.Named(ControllerName),
		recorder:     masterManager.GetEventRecorderFor(ControllerName),
		masterClient: masterManager.GetClient(),
		seedClients:  map[string]ctrlruntimeclient.Client{},
	}

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()
	}

	c, err := controller.New(ControllerName, masterManager, controller.Options{Reconciler: r, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}

	// Watch for changes to ApplicationDefinition
	if err := c.Watch(&source.Kind{Type: &appkubermaticv1.ApplicationDefinition{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to create watch for applicationDefinitions: %v", err)
	}

	return nil
}

// Reconcile reconciles Kubermatic Project objects on the master cluster to all seed clusters
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("resource", request.Name)
	log.Infof("start reconciling")

	return reconcile.Result{}, nil
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, applicationDefiniton *appkubermaticv1.ApplicationDefinition) error {
	log.Infof("remove")
	return nil
}
