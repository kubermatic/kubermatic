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

package cniapplicationinstallationcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/imdario/mergo"
	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/cni"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kkp-cni-application-installation-controller"

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
	userClusterConnectionProvider UserClusterClientProvider
	log                           *zap.SugaredLogger
	versions                      kubermatic.Versions
	overwriteRegistry             string
}

func Add(ctx context.Context, mgr manager.Manager, numWorkers int, workerName string, userClusterConnectionProvider UserClusterClientProvider, log *zap.SugaredLogger, versions kubermatic.Versions, overwriteRegistry string) error {
	reconciler := &Reconciler{
		Client:                        mgr.GetClient(),
		workerName:                    workerName,
		recorder:                      mgr.GetEventRecorderFor(ControllerName),
		userClusterConnectionProvider: userClusterConnectionProvider,
		log:                           log.Named(ControllerName),
		versions:                      versions,
		overwriteRegistry:             overwriteRegistry,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	// Predicate for reacting to cluster update events only when CNIPlugin or ClusterNetwork config changed, or cluster Address changed
	clusterUpdatePredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObj := e.ObjectOld.(*kubermaticv1.Cluster)
			newObj := e.ObjectNew.(*kubermaticv1.Cluster)
			if !reflect.DeepEqual(oldObj.Spec.CNIPlugin, newObj.Spec.CNIPlugin) {
				return true
			}
			if !reflect.DeepEqual(oldObj.Spec.ClusterNetwork, newObj.Spec.ClusterNetwork) {
				return true
			}
			if !reflect.DeepEqual(oldObj.Status.Address, newObj.Status.Address) {
				return true
			}
			return false
		},
	}

	// Watch on cluster events
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}, clusterUpdatePredicate, workerlabel.Predicates(workerName)); err != nil {
		return fmt.Errorf("failed to create watch: %w", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.NamespacedName.Name)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if cluster.DeletionTimestamp != nil {
		// Cluster is queued for deletion; no action required
		log.Debugw("Cluster is queued for deletion; skipping")
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
			return r.reconcile(ctx, log, cluster)
		},
	)
	if err != nil {
		log.Errorw("Failed to reconcile cluster", zap.Error(err))
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, logger *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	log := logger.With("CNIType", cluster.Spec.CNIPlugin.Type, "CNIVersion", cluster.Spec.CNIPlugin.Version)

	// Do not reconcile the cluster if the CNI is not managed by Applications infra
	if !cni.IsManagedByAppInfra(cluster.Spec.CNIPlugin.Type, cluster.Spec.CNIPlugin.Version) {
		log.Debug("CNI is not managed by Applications infra, skipping")
		return nil, nil
	}

	// Make sure that cluster is in a state when creating ApplicationInstallation is permissible
	if !cluster.Status.ExtendedHealth.ApplicationControllerHealthy() {
		log.Debug("Requeue CNI reconciliation as Application controller is not healthy")
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil // try reconciling later
	}

	log.Debugf("Reconciling CNI")

	// Ensure legacy CNI addon is removed if it was deployed as older CNI version
	if err := r.ensureLegacyCNIAddonIsRemoved(ctx, cluster); err != nil {
		return &reconcile.Result{}, err
	}

	// Prepare initialValues for the CNI ApplicationInstallation. These values will be used if the ApplicationInstallation does not exist yet.
	initialValues := make(map[string]interface{})

	// Try to load the initial values form the annotation
	if err := r.parseCNIValuesAnnotation(cluster, initialValues); err != nil {
		return &reconcile.Result{}, err
	}
	removeAnnotation := false
	if len(initialValues) > 0 {
		removeAnnotation = true
	}

	// If initial values were not loaded from the annotation, use the default values from the ApplicationDefinition
	if len(initialValues) == 0 {
		if err := r.parseAppDefDefaultValues(ctx, cluster, initialValues); err != nil {
			return &reconcile.Result{}, err
		}
	}

	// Ensure ApplicationInstallation of the CNI
	if err := r.ensreCNIApplicationInstallation(ctx, cluster, initialValues); err != nil {
		return &reconcile.Result{}, err
	}

	if removeAnnotation {
		if err := r.removeCNIValuesAnnotation(ctx, cluster); err != nil {
			return nil, fmt.Errorf("failed to remove initial CNI values annotation: %w", err)
		}
	}

	return nil, nil
}

func (r *Reconciler) ensureLegacyCNIAddonIsRemoved(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	cniAddon := &kubermaticv1.Addon{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Spec.CNIPlugin.Type.String(),
			Namespace: cluster.Status.NamespaceName,
		},
	}
	err := r.Client.Delete(ctx, cniAddon)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete CNI addon %s: %w", cniAddon.GetName(), err)
	}

	// In case of Cilium we also need to remove the Hubble addon
	if cluster.Spec.CNIPlugin.Type == kubermaticv1.CNIPluginTypeCilium {
		cniAddon := &kubermaticv1.Addon{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hubble",
				Namespace: cluster.Status.NamespaceName,
			},
		}
		err := r.Client.Delete(ctx, cniAddon)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete Hubble addon %s: %w", cniAddon.GetName(), err)
		}
	}
	return nil
}

func (r *Reconciler) parseCNIValuesAnnotation(cluster *kubermaticv1.Cluster, values map[string]interface{}) error {
	annotation := cluster.Annotations[kubermaticv1.InitialCNIValuesRequestAnnotation]
	if annotation != "" {
		if err := json.Unmarshal([]byte(annotation), &values); err != nil {
			return fmt.Errorf("cannot unmarshal initial CNI values annotation: %w", err)
		}
	}
	return nil
}

func (r *Reconciler) removeCNIValuesAnnotation(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	oldCluster := cluster.DeepCopy()
	delete(cluster.Annotations, kubermaticv1.InitialCNIValuesRequestAnnotation)
	return r.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}

func (r *Reconciler) parseAppDefDefaultValues(ctx context.Context, cluster *kubermaticv1.Cluster, values map[string]interface{}) error {
	appDef := &appskubermaticv1.ApplicationDefinition{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: cluster.Spec.CNIPlugin.Type.String()}, appDef); err != nil {
		return ctrlruntimeclient.IgnoreNotFound(err)
	}
	if appDef.Spec.DefaultValues != nil {
		if len(appDef.Spec.DefaultValues.Raw) > 0 {
			if err := json.Unmarshal(appDef.Spec.DefaultValues.Raw, &values); err != nil {
				return fmt.Errorf("failed to unmarshall ApplicationDefinition default values: %w", err)
			}
		}
	}
	return nil
}

func (r *Reconciler) ensreCNIApplicationInstallation(ctx context.Context, cluster *kubermaticv1.Cluster, initialValues map[string]interface{}) error {
	userClusterClient, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get user cluster client: %w", err)
	}

	creators := []reconciling.NamedAppsKubermaticV1ApplicationInstallationCreatorGetter{
		ApplicationInstallationCreator(cluster, r.overwriteRegistry, initialValues),
	}
	return reconciling.ReconcileAppsKubermaticV1ApplicationInstallations(ctx, creators, cniPluginNamespace, userClusterClient)
}

func ApplicationInstallationCreator(cluster *kubermaticv1.Cluster, overwriteRegistry string, initialValues map[string]interface{}) reconciling.NamedAppsKubermaticV1ApplicationInstallationCreatorGetter {
	return func() (string, reconciling.AppsKubermaticV1ApplicationInstallationCreator) {
		return cluster.Spec.CNIPlugin.Type.String(), func(app *appskubermaticv1.ApplicationInstallation) (*appskubermaticv1.ApplicationInstallation, error) {
			app.Labels = map[string]string{
				appskubermaticv1.ApplicationManagedByLabel: appskubermaticv1.ApplicationManagedByKKPValue,
				appskubermaticv1.ApplicationTypeLabel:      appskubermaticv1.ApplicationTypeCNIValue,
			}
			app.Spec.ApplicationRef = appskubermaticv1.ApplicationRef{
				Name:    cluster.Spec.CNIPlugin.Type.String(),
				Version: cluster.Spec.CNIPlugin.Version,
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

			// If (and only if) existing values is empty, use the initial values
			if len(values) == 0 {
				values = initialValues
			}

			// Override values with necessary CNI config
			overrideValues := getCNIOverrideValues(cluster, overwriteRegistry)
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
	}
}

func getCNIOverrideValues(cluster *kubermaticv1.Cluster, overwriteRegistry string) map[string]interface{} {
	if cluster.Spec.CNIPlugin.Type == kubermaticv1.CNIPluginTypeCilium {
		return cni.GetCiliumAppInstallOverrideValues(cluster, overwriteRegistry)
	}
	return nil
}
