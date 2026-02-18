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
	"errors"
	"fmt"
	"strconv"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/cni"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

const (
	ControllerName  = "kkp-addoninstaller-controller"
	addonDefaultKey = ".spec.isDefault"

	kubeProxyAddonName = "kube-proxy"
	CSIAddonName       = "csi"
)

type Reconciler struct {
	ctrlruntimeclient.Client

	log          *zap.SugaredLogger
	configGetter provider.KubermaticConfigurationGetter
	workerName   string
	recorder     events.EventRecorder
	versions     kubermatic.Versions
}

func Add(
	log *zap.SugaredLogger,
	mgr manager.Manager,
	numWorkers int,
	workerName string,
	configGetter provider.KubermaticConfigurationGetter,
	versions kubermatic.Versions,
) error {
	// Add index on IsDefault flag
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &kubermaticv1.Addon{}, addonDefaultKey, func(rawObj ctrlruntimeclient.Object) []string {
		a := rawObj.(*kubermaticv1.Addon)
		return []string{strconv.FormatBool(a.Spec.IsDefault)}
	}); err != nil {
		return fmt.Errorf("failed to add index on Addon IsDefault flag: %w", err)
	}

	log = log.Named(ControllerName)

	reconciler := &Reconciler{
		Client:       mgr.GetClient(),
		log:          log,
		workerName:   workerName,
		configGetter: configGetter,
		recorder:     mgr.GetEventRecorder(ControllerName),
		versions:     versions,
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{}).
		Watches(&kubermaticv1.Addon{}, util.EnqueueClusterForNamespacedObject(mgr.GetClient())).
		Build(reconciler)

	return err
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.Name)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("Skipping because the cluster is already gone")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if cluster.DeletionTimestamp != nil {
		return reconcile.Result{}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := util.ClusterReconcileWrapper(
		ctx,
		r,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionAddonInstallerControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, log, cluster)
		},
	)

	if result == nil || err != nil {
		result = &reconcile.Result{}
	}

	if err != nil {
		r.recorder.Eventf(cluster, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
	}

	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// Wait until the kube-apiserver is running to ensure the namespace exists at least.
	// Just checking for cluster.status.namespaceName is not enough as it gets set before the namespace exists
	if cluster.Status.ExtendedHealth.Apiserver != kubermaticv1.HealthStatusUp {
		log.Debug("Skipping because the API server is not running")
		return &reconcile.Result{RequeueAfter: 1 * time.Second}, nil
	}

	// determine the addon list that is currently configured for clusters
	addons, err := r.getAddons(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to determine addons to install: %w", err)
	}

	return nil, r.ensureAddons(ctx, log, cluster, *addons)
}

func (r *Reconciler) getAddons(ctx context.Context) (*kubermaticv1.AddonList, error) {
	cfg, err := r.configGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get KubermaticConfiguration: %w", err)
	}

	c := cfg.Spec.UserCluster.Addons

	if len(c.Default) > 0 && c.DefaultManifests != "" {
		return nil, errors.New("default addons are configured both as a list and a default manifest, which are mutually exclusive; configure addons using one of the two mechanisms")
	}

	result := &kubermaticv1.AddonList{}

	if len(c.Default) > 0 {
		defaultManifests, err := getDefaultAddonManifests()
		if err != nil {
			return nil, fmt.Errorf("failed to get default addon manifests: %w", err)
		}

		for _, addonName := range c.Default {
			labels := map[string]string{}

			for _, addon := range defaultManifests.Items {
				if addon.Name == addonName {
					labels = addon.Labels
					break
				}
			}

			result.Items = append(result.Items, kubermaticv1.Addon{
				ObjectMeta: metav1.ObjectMeta{
					Name:   addonName,
					Labels: labels,
				},
			})
		}
	}

	if c.DefaultManifests != "" {
		if err := yaml.UnmarshalStrict([]byte(c.DefaultManifests), result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal default addon list: %w", err)
		}
	}

	// Validate the metrics-server addon is disabled, otherwise it creates conflicts with the resources
	// we create for the metrics-server running in the seed and will render the latter unusable
	for _, addon := range result.Items {
		if addon.Name == "metrics-server" {
			return nil, errors.New("the metrics-server addon must be disabled, it is now deployed inside the seed cluster")
		}
	}

	return result, nil
}

// getDefaultAddonManifests returns the default addons, parsed as an AddonList.
// This is used to fill in labels for when the admin used the legacy configuration
// mechanism where instead of an AddonList, only a []string is given.
func getDefaultAddonManifests() (*kubermaticv1.AddonList, error) {
	defaultAddonList := kubermaticv1.AddonList{}
	if err := yaml.UnmarshalStrict([]byte(defaulting.DefaultKubernetesAddons), &defaultAddonList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal default addon list: %w", err)
	}

	return &defaultAddonList, nil
}

func (r *Reconciler) ensureAddons(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, addons kubermaticv1.AddonList) error {
	ensuredAddonsMap := sets.New[string]()
	creators := []reconciling.NamedAddonReconcilerFactory{}

	for i, addon := range addons.Items {
		if skipAddonInstallation(addon, cluster) {
			continue
		}

		ensuredAddonsMap.Insert(addon.Name)
		creators = append(creators, r.addonReconciler(cluster, addons.Items[i]))
	}

	if err := reconciling.ReconcileAddons(ctx, creators, cluster.Status.NamespaceName, r); err != nil {
		return fmt.Errorf("failed to ensure addons: %w", err)
	}

	currentAddons := kubermaticv1.AddonList{}
	// only list default addons as user added addons should not be deleted
	if err := r.List(ctx, &currentAddons, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName), ctrlruntimeclient.MatchingFields{addonDefaultKey: "true"}); err != nil {
		return fmt.Errorf("failed to list cluster addons: %w", err)
	}
	for _, currentAddon := range currentAddons.Items {
		if _, ensured := ensuredAddonsMap[currentAddon.Name]; !ensured {
			// we found an installed Addon that shouldn't be
			if err := r.deleteAddon(ctx, log, currentAddon); err != nil {
				return fmt.Errorf("failed to delete cluster addon: %w", err)
			}
		}
	}
	return nil
}

func (r *Reconciler) addonReconciler(cluster *kubermaticv1.Cluster, addon kubermaticv1.Addon) reconciling.NamedAddonReconcilerFactory {
	return func() (name string, create reconciling.AddonReconciler) {
		return addon.Name, func(existing *kubermaticv1.Addon) (*kubermaticv1.Addon, error) {
			existing.Labels = addon.Labels
			existing.Annotations = addon.Annotations

			existing.Spec.IsDefault = true
			existing.Spec.Variables = addon.Spec.Variables
			existing.Spec.RequiredResourceTypes = addon.Spec.RequiredResourceTypes
			existing.Spec.Name = addon.Name
			existing.Spec.Cluster = corev1.ObjectReference{
				APIVersion: cluster.APIVersion,
				Kind:       kubermaticv1.ClusterKindName,
				Name:       cluster.Name,
			}

			return existing, nil
		}
	}
}

func (r *Reconciler) deleteAddon(ctx context.Context, log *zap.SugaredLogger, addon kubermaticv1.Addon) error {
	log.Infow("Deleting addon", "addon", addon.Name)
	err := r.Delete(ctx, &addon)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete addon %s from cluster %s: %w", addon.Name, addon.Spec.Cluster.Name, err)
	}
	return nil
}

// skipAddonInstallation returns true if the addon installation should be skipped based on the Cluster spec, false otherwise.
func skipAddonInstallation(addon kubermaticv1.Addon, cluster *kubermaticv1.Cluster) bool {
	if cluster.Spec.CNIPlugin != nil {
		if addon.Name == string(kubermaticv1.CNIPluginTypeCanal) && cluster.Spec.CNIPlugin.Type == kubermaticv1.CNIPluginTypeCilium {
			return true // skip Canal if Cilium is used
		}
		if addon.Name == string(kubermaticv1.CNIPluginTypeCilium) && cluster.Spec.CNIPlugin.Type == kubermaticv1.CNIPluginTypeCanal {
			return true // skip Cilium if Canal is used
		}
		if addon.Name == string(kubermaticv1.CNIPluginTypeCanal) || addon.Name == string(kubermaticv1.CNIPluginTypeCilium) {
			if cni.IsManagedByAppInfra(cluster.Spec.CNIPlugin.Type, cluster.Spec.CNIPlugin.Version) {
				return true // skip if CNI is managed by App infra
			}
		}
	}
	if addon.Name == kubeProxyAddonName && cluster.Spec.ClusterNetwork.ProxyMode == resources.EBPFProxyMode {
		return true // skip kube-proxy if eBPF proxy mode is used
	}
	if addon.Name == CSIAddonName && cluster.Spec.DisableCSIDriver {
		return true // skip csi driver installation if DisableCSIDriver is true
	}
	return false
}
