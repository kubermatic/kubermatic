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

package constrainttemplatecontroller

import (
	"context"
	"fmt"
	"time"

	"github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"
	"go.uber.org/zap"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// This controller syncs the kubermatic constraint templates to gatekeeper constraint templates on the user cluster.
	ControllerName = "gatekeeper_constraint_template_controller"
)

// UserClusterClientProvider provides functionality to get a user cluster client.
type UserClusterClientProvider interface {
	GetClient(ctx context.Context, c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

type reconciler struct {
	log                       *zap.SugaredLogger
	workerNameLabelSelector   labels.Selector
	recorder                  record.EventRecorder
	seedClient                ctrlruntimeclient.Client
	userClusterClientProvider UserClusterClientProvider
	userClusterClients        map[string]ctrlruntimeclient.Client
}

func Add(ctx context.Context,
	mgr manager.Manager,
	clientProvider UserClusterClientProvider,
	log *zap.SugaredLogger,
	workerName string,
	numWorkers int,
) error {
	workerSelector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return fmt.Errorf("failed to build worker-name selector: %w", err)
	}

	reconciler := &reconciler{
		log:                       log.Named(ControllerName),
		workerNameLabelSelector:   workerSelector,
		recorder:                  mgr.GetEventRecorderFor(ControllerName),
		seedClient:                mgr.GetClient(),
		userClusterClientProvider: clientProvider,
		userClusterClients:        map[string]ctrlruntimeclient.Client{},
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.Cluster{}},
		enqueueAllConstraintTemplates(reconciler.seedClient, reconciler.log),
		workerlabel.Predicates(workerName),
	); err != nil {
		return fmt.Errorf("failed to establish watch for clusters: %w", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.ConstraintTemplate{}},
		&handler.EnqueueRequestForObject{},
	); err != nil {
		return fmt.Errorf("failed to create watch for constraintTemplates: %w", err)
	}

	return nil
}

// Reconcile reconciles the kubermatic constraint template on the seed cluster to all user clusters
// which have opa integration enabled.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Reconciling")

	constraintTemplate := &kubermaticv1.ConstraintTemplate{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, constraintTemplate); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("constraint template not found, returning")
			return reconcile.Result{}, nil
		}
		if controllerutil.IsCacheNotStarted(err) {
			return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get constraint template %s: %w", constraintTemplate.Name, err)
	}

	err := r.reconcile(ctx, log, constraintTemplate)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Eventf(constraintTemplate, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, constraintTemplate *kubermaticv1.ConstraintTemplate) error {
	if constraintTemplate.DeletionTimestamp != nil {
		if !kuberneteshelper.HasFinalizer(constraintTemplate, kubermaticapiv1.GatekeeperConstraintTemplateCleanupFinalizer) {
			return nil
		}

		err := r.syncAllClusters(ctx, log, constraintTemplate, func(userClusterClient ctrlruntimeclient.Client, ct *kubermaticv1.ConstraintTemplate) error {
			err := userClusterClient.Delete(ctx, &v1beta1.ConstraintTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: constraintTemplate.Name,
				},
			})
			if kerrors.IsNotFound(err) {
				return nil
			}
			return err
		})
		if err != nil {
			return err
		}

		oldConstraintTemplate := constraintTemplate.DeepCopy()
		kuberneteshelper.RemoveFinalizer(constraintTemplate, kubermaticapiv1.GatekeeperConstraintTemplateCleanupFinalizer)
		if err := r.seedClient.Patch(ctx, constraintTemplate, ctrlruntimeclient.MergeFrom(oldConstraintTemplate)); err != nil {
			return fmt.Errorf("failed to remove constraint template finalizer %s: %w", constraintTemplate.Name, err)
		}
		return nil
	}

	if !kuberneteshelper.HasFinalizer(constraintTemplate, kubermaticapiv1.GatekeeperConstraintTemplateCleanupFinalizer) {
		oldConstraintTemplate := constraintTemplate.DeepCopy()
		kuberneteshelper.AddFinalizer(constraintTemplate, kubermaticapiv1.GatekeeperConstraintTemplateCleanupFinalizer)
		if err := r.seedClient.Patch(ctx, constraintTemplate, ctrlruntimeclient.MergeFrom(oldConstraintTemplate)); err != nil {
			return fmt.Errorf("failed to set constraint template finalizer %s: %w", constraintTemplate.Name, err)
		}
	}

	ctCreatorGetters := []reconciling.NamedConstraintTemplateCreatorGetter{
		constraintTemplateCreatorGetter(constraintTemplate),
	}

	return r.syncAllClusters(ctx, log, constraintTemplate, func(userClusterClient ctrlruntimeclient.Client, ct *kubermaticv1.ConstraintTemplate) error {
		return reconciling.ReconcileConstraintTemplates(ctx, ctCreatorGetters, "", userClusterClient)
	})
}

func (r *reconciler) syncAllClusters(
	ctx context.Context,
	log *zap.SugaredLogger,
	constraintTemplate *kubermaticv1.ConstraintTemplate,
	action func(userClusterClient ctrlruntimeclient.Client, ct *kubermaticv1.ConstraintTemplate) error,
) error {
	clusterList, err := r.getClustersForConstraintTemplate(ctx, constraintTemplate)
	if err != nil {
		return fmt.Errorf("failed listing clusters: %w", err)
	}

	for _, userCluster := range clusterList.Items {
		if userCluster.Spec.Pause {
			log.Debugw("Cluster paused, skipping", "cluster", userCluster.Spec.HumanReadableName)
			continue
		}

		if userCluster.Spec.OPAIntegration != nil && userCluster.Spec.OPAIntegration.Enabled {
			// Get user cluster client from map, if it does not exist yet, create it
			var err error
			userClusterClient, ok := r.userClusterClients[userCluster.Name]
			if !ok {
				userClusterClient, err = r.userClusterClientProvider.GetClient(ctx, &userCluster)
				if err != nil {
					return fmt.Errorf("error getting client for cluster %s: %w", userCluster.Spec.HumanReadableName, err)
				}
				r.userClusterClients[userCluster.Name] = userClusterClient
			}

			err = action(userClusterClient, constraintTemplate)
			if err != nil {
				return fmt.Errorf("failed syncing constraint template for cluster %s: %w", userCluster.Spec.HumanReadableName, err)
			}
			log.Debugw("Reconciled constraint template with cluster", "cluster", userCluster.Spec.HumanReadableName)
		} else {
			log.Debugw("Cluster does not integrate with OPA, skipping", "cluster", userCluster.Spec.HumanReadableName)
		}
	}

	return nil
}

func constraintTemplateCreatorGetter(kubeCT *kubermaticv1.ConstraintTemplate) reconciling.NamedConstraintTemplateCreatorGetter {
	return func() (string, reconciling.ConstraintTemplateCreator) {
		return kubeCT.Name, func(ct *v1beta1.ConstraintTemplate) (*v1beta1.ConstraintTemplate, error) {
			ct.Name = kubeCT.Name
			ct.Spec = v1beta1.ConstraintTemplateSpec{
				CRD:     kubeCT.Spec.CRD,
				Targets: kubeCT.Spec.Targets,
			}

			return ct, nil
		}
	}
}

func enqueueAllConstraintTemplates(client ctrlruntimeclient.Client, log *zap.SugaredLogger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request

		cluster, ok := a.(*kubermaticv1.Cluster)
		if !ok {
			err := fmt.Errorf("object was not a cluster but a %T", a)
			log.Error(err)
			utilruntime.HandleError(err)
			return nil
		}
		if cluster.Spec.OPAIntegration == nil || !cluster.Spec.OPAIntegration.Enabled {
			return nil
		}

		ctList := &kubermaticv1.ConstraintTemplateList{}
		if err := client.List(context.Background(), ctList); err != nil {
			log.Error(err)
			utilruntime.HandleError(fmt.Errorf("failed to list constraint templates: %w", err))
		}
		for _, ct := range ctList.Items {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
				Name: ct.Name,
			}})
		}
		return requests
	})
}
