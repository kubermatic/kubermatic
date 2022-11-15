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

package cni

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/imdario/mergo"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"

	semverlib "github.com/Masterminds/semver/v3"
	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/version/cni"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kkp-cni-controller"

	cniPluginNamespace = "kube-system"
)

// UserClusterClientProvider provides functionality to get a user cluster client.
type UserClusterClientProvider interface {
	GetClient(ctx context.Context, c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

type Reconciler struct {
	ctrlruntimeclient.Client

	workerName                    string
	recorder                      record.EventRecorder
	seedGetter                    provider.SeedGetter
	userClusterConnectionProvider UserClusterClientProvider
	log                           *zap.SugaredLogger
	versions                      kubermatic.Versions
}

func Add(ctx context.Context, mgr manager.Manager, numWorkers int, workerName string, seedGetter provider.SeedGetter, userClusterConnectionProvider UserClusterClientProvider, log *zap.SugaredLogger, versions kubermatic.Versions) error {
	reconciler := &Reconciler{
		Client: mgr.GetClient(),

		workerName:                    workerName,
		recorder:                      mgr.GetEventRecorderFor(ControllerName),
		seedGetter:                    seedGetter,
		userClusterConnectionProvider: userClusterConnectionProvider,
		log:                           log,
		versions:                      versions,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	// React to cluster update events only when CNIPlugin or ClusterNetwork config changed
	clusterPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObj := e.ObjectOld.(*kubermaticv1.Cluster)
			newObj := e.ObjectNew.(*kubermaticv1.Cluster)
			if !reflect.DeepEqual(oldObj.Spec.CNIPlugin, newObj.Spec.CNIPlugin) {
				return true
			}
			if !reflect.DeepEqual(oldObj.Spec.ClusterNetwork, newObj.Spec.ClusterNetwork) {
				return true
			}
			return false
		},
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}, clusterPredicate); err != nil {
		return fmt.Errorf("failed to create watch: %w", err)
	}

	// TODO (rastislavs): enqueue cluster events for ApplicationDefinition changes

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if cluster.DeletionTimestamp != nil {
		// Cluster is queued for deletion; no action required
		r.log.Debugw("Cluster is queued for deletion; no action required", "cluster", cluster.Name)
		return reconcile.Result{}, nil
	}

	// Add a wrapping here, so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionCNIControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, cluster)
		},
	)
	if err != nil {
		r.log.Errorw("Failed to reconcile cluster", "cluster", cluster.Name, zap.Error(err))
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// Do not reconcile the cluster if the CNI is not managed by Applications infra
	if !cni.IsManagedByAppInfra(cluster.Spec.CNIPlugin.Type, cluster.Spec.CNIPlugin.Version) {
		return nil, nil
	}

	// Make sure that cluster is in a state when creating ApplicationInstallation is permissible
	if !cluster.Status.ExtendedHealth.ApplicationControllerHealthy() {
		r.log.Debug("Application controller not healthy")
		return &reconcile.Result{RequeueAfter: 30 * time.Second}, nil // try reconciling later
	}

	// Ensure CNI addon is removed if it was deployed before
	if err := r.ensureCNIAddonIsRemoved(ctx, cluster); err != nil {
		return &reconcile.Result{}, err
	}

	userClusterClient, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get user cluster client: %w", err)
	}

	if err := r.ensreCNIApplicationInstallation(ctx, userClusterClient, cluster); err != nil {
		return &reconcile.Result{}, err
	}

	return nil, nil
}

func (r *Reconciler) ensreCNIApplicationInstallation(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {

	cniVer, err := semverlib.NewVersion(cluster.Spec.CNIPlugin.Version)
	if err != nil {
		return fmt.Errorf("failed to parse CNI plugin version: %w", err)
	}

	cniAppInstallation := func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		var app *appskubermaticv1.ApplicationInstallation
		if existing == nil {
			app = &appskubermaticv1.ApplicationInstallation{}
		} else {
			app = existing.(*appskubermaticv1.ApplicationInstallation)
		}

		app.Labels = map[string]string{
			"apps.kubermatic.k8c.io/managed-by": "kkp",
			"apps.kubermatic.k8c.io/type":       "cni",
		}
		app.Spec.ApplicationRef = appskubermaticv1.ApplicationRef{
			Name:    cluster.Spec.CNIPlugin.Type.String(),
			Version: appskubermaticv1.Version{Version: *cniVer},
		}
		app.Spec.Namespace = appskubermaticv1.AppNamespaceSpec{
			Name: cniPluginNamespace,
		}

		// Unmarshall existing values
		values := make(map[string]interface{})
		if len(app.Spec.Values.Raw) > 0 {
			if err := json.Unmarshal(app.Spec.Values.Raw, &values); err != nil {
				return app, fmt.Errorf("failed to unmarshall CNI values: %w", err)
			}
		}

		// Override values with necessary CNI config
		overrideValues := r.getCNIOverrideValues(cluster)
		if err := mergo.Merge(&values, overrideValues, mergo.WithOverride); err != nil {
			return app, fmt.Errorf("failed to merge CNI values: %w", err)
		}

		// Set new values
		rawValues, err := json.Marshal(values)
		if err != nil {
			return app, fmt.Errorf("failed to marshall CNI values: %w", err)
		}
		app.Spec.Values = runtime.RawExtension{Raw: rawValues}

		return app, nil
	}

	return reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: cniPluginNamespace, Name: cluster.Spec.CNIPlugin.Type.String()}, cniAppInstallation, client, &appskubermaticv1.ApplicationInstallation{}, false)
}

func (r *Reconciler) getCNIOverrideValues(cluster *kubermaticv1.Cluster) map[string]interface{} {
	if cluster.Spec.CNIPlugin.Type == kubermaticv1.CNIPluginTypeCilium {
		return getCiliumOverrideValues(cluster)
	}
	return nil
}

func (r *Reconciler) ensureCNIAddonIsRemoved(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	addon := &kubermaticv1.Addon{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Spec.CNIPlugin.Type.String(),
			Namespace: cluster.Status.NamespaceName,
		},
	}
	err := r.Client.Delete(ctx, addon)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete CNI addon %s: %w", addon.GetName(), err)
	}
	return nil
}
