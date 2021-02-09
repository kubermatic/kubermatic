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

package addoninstaller

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName  = "kubermatic_addoninstaller_controller"
	addonDefaultKey = ".spec.isDefault"
)

type Reconciler struct {
	ctrlruntimeclient.Client

	log              *zap.SugaredLogger
	kubernetesAddons kubermaticv1.AddonList
	workerName       string
	recorder         record.EventRecorder
	versions         kubermatic.Versions
}

func Add(
	log *zap.SugaredLogger,
	mgr manager.Manager,
	numWorkers int,
	workerName string,
	kubernetesAddons kubermaticv1.AddonList,
	versions kubermatic.Versions,
) error {
	log = log.Named(ControllerName)

	reconciler := &Reconciler{
		Client:           mgr.GetClient(),
		log:              log,
		workerName:       workerName,
		kubernetesAddons: kubernetesAddons,
		recorder:         mgr.GetEventRecorderFor(ControllerName),
		versions:         versions,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %v", err)
	}

	// Add index on IsDefault flag
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &kubermaticv1.Addon{}, addonDefaultKey, func(rawObj ctrlruntimeclient.Object) []string {
		a := rawObj.(*kubermaticv1.Addon)
		return []string{strconv.FormatBool(a.Spec.IsDefault)}
	}); err != nil {
		return fmt.Errorf("failed to add index on Addon IsDefault flag: %v", err)
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to create watch for clusters: %v", err)
	}

	enqueueClusterForNamespacedObject := handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		clusterList := &kubermaticv1.ClusterList{}
		if err := mgr.GetClient().List(context.Background(), clusterList); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list Clusters: %v", err))
			log.Errorw("Failed to list clusters", zap.Error(err))
			return []reconcile.Request{}
		}
		for _, cluster := range clusterList.Items {
			if cluster.Status.NamespaceName == a.GetNamespace() {
				return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: cluster.Name}}}
			}
		}
		return []reconcile.Request{}
	})
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Addon{}}, enqueueClusterForNamespacedObject); err != nil {
		return fmt.Errorf("failed to create watch for Addons: %v", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("Skipping because the cluster is already gone")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionAddonInstallerControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, log, cluster)
		},
	)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {

	// Wait until the Apiserver is running to ensure the namespace exists at least.
	// Just checking for cluster.status.namespaceName is not enough as it gets set before the namespace exists
	if cluster.Status.ExtendedHealth.Apiserver != kubermaticv1.HealthStatusUp {
		log.Debug("Skipping because the API server is not running")
		return &reconcile.Result{RequeueAfter: 1 * time.Second}, nil
	}

	return nil, r.ensureAddons(ctx, log, cluster, *r.kubernetesAddons.DeepCopy())
}

func (r *Reconciler) ensureAddons(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, addons kubermaticv1.AddonList) error {
	ensuredAddonsMap := map[string]struct{}{}
	for _, addon := range addons.Items {
		ensuredAddonsMap[addon.Name] = struct{}{}
		name := types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: addon.Name}
		addonLog := log.With("addon", name)
		existingAddon := &kubermaticv1.Addon{}
		err := r.Get(ctx, name, existingAddon)
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return fmt.Errorf("failed to get addon %q: %v", addon.Name, err)
			}
			if err := r.createAddon(ctx, addonLog, addon, cluster); err != nil {
				return fmt.Errorf("failed to create addon %q: %v", addon.Name, err)
			}
		} else {
			addonLog.Debug("Addon already exists")
			if !reflect.DeepEqual(addon.Labels, existingAddon.Labels) || !reflect.DeepEqual(addon.Annotations, existingAddon.Annotations) || !reflect.DeepEqual(addon.Spec.Variables, existingAddon.Spec.Variables) || !reflect.DeepEqual(addon.Spec.RequiredResourceTypes, existingAddon.Spec.RequiredResourceTypes) {
				updatedAddon := existingAddon.DeepCopy()
				updatedAddon.Labels = addon.Labels
				updatedAddon.Annotations = addon.Annotations
				updatedAddon.Spec.Name = addon.Name
				updatedAddon.Spec.Variables = addon.Spec.Variables
				updatedAddon.Spec.RequiredResourceTypes = addon.Spec.RequiredResourceTypes
				updatedAddon.Spec.IsDefault = true
				if err := r.Patch(ctx, updatedAddon, ctrlruntimeclient.MergeFrom(existingAddon)); err != nil {
					return fmt.Errorf("failed to update addon %q: %v", addon.Name, err)
				}
			}
		}
	}

	currentAddons := kubermaticv1.AddonList{}
	// only list default addons as user added addons should not be deleted
	if err := r.List(ctx, &currentAddons, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName), ctrlruntimeclient.MatchingFields{addonDefaultKey: "true"}); err != nil {
		return fmt.Errorf("failed to list cluster addons: %v", err)
	}
	for _, currentAddon := range currentAddons.Items {
		if _, ensured := ensuredAddonsMap[currentAddon.Name]; !ensured {
			// we found an installed Addon that shouldn't be
			if err := r.deleteAddon(ctx, log, currentAddon); err != nil {
				return fmt.Errorf("failed to delete cluster addon: %v", err)
			}
		}
	}
	return nil
}

func (r *Reconciler) createAddon(ctx context.Context, log *zap.SugaredLogger, addon kubermaticv1.Addon, cluster *kubermaticv1.Cluster) error {
	gv := kubermaticv1.SchemeGroupVersion

	addon.Namespace = cluster.Status.NamespaceName
	addon.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(cluster, gv.WithKind("Cluster"))}
	if addon.Labels == nil {
		addon.Labels = map[string]string{}
	}
	addon.Spec.Name = addon.Name
	addon.Spec.Cluster = corev1.ObjectReference{
		Name:       cluster.Name,
		Namespace:  "",
		UID:        cluster.UID,
		APIVersion: cluster.APIVersion,
		Kind:       "Cluster",
	}
	addon.Spec.IsDefault = true

	// Swallow IsAlreadyExists, we have predictable names and our cache may not be
	// up to date, leading us to think the addon wasn't installed yet.
	if err := r.Create(ctx, &addon); err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create addon %q: %v", addon.Name, err)
	}

	log.Info("Addon successfully created")

	err := wait.Poll(10*time.Millisecond, 10*time.Second, func() (bool, error) {
		err := r.Get(ctx, types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: addon.Name}, &kubermaticv1.Addon{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed waiting for addon %s to exist in the lister", addon.Name)
	}

	return nil
}

func (r *Reconciler) deleteAddon(ctx context.Context, log *zap.SugaredLogger, addon kubermaticv1.Addon) error {
	log.Infof("deleting addon %s from cluster %s", addon.Name, addon.Namespace)
	err := r.Delete(ctx, &addon)
	if err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete addon %s from cluster %s: %v", addon.Name, addon.ClusterName, err)
	}
	return nil
}
