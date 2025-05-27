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
	"maps"
	"reflect"
	"strings"
	"time"

	"dario.cat/mergo"
	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/cni"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

const (
	ControllerName = "kkp-cni-application-installation-controller"

	cniPluginNamespace = "kube-system"
)

// Cilium specific constants.
const (
	ciliumHelmChartName = "cilium"
	ciliumImageRegistry = "quay.io/cilium/"
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
	systemAppEnforceInterval      int
	overwriteRegistry             string
}

func Add(ctx context.Context, mgr manager.Manager, numWorkers int, workerName string, systemAppEnforceInterval int, userClusterConnectionProvider UserClusterClientProvider, log *zap.SugaredLogger, versions kubermatic.Versions, overwriteRegistry string) error {
	reconciler := &Reconciler{
		Client:                        mgr.GetClient(),
		workerName:                    workerName,
		systemAppEnforceInterval:      systemAppEnforceInterval,
		recorder:                      mgr.GetEventRecorderFor(ControllerName),
		userClusterConnectionProvider: userClusterConnectionProvider,
		log:                           log.Named(ControllerName),
		versions:                      versions,
		overwriteRegistry:             overwriteRegistry,
	}

	clusterEventPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// Process the event only if the cluster's CNI is managed by App Infra
			cluster := e.Object.(*kubermaticv1.Cluster)
			return cni.IsManagedByAppInfra(cluster.Spec.CNIPlugin.Type, cluster.Spec.CNIPlugin.Version)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldCluster := e.ObjectOld.(*kubermaticv1.Cluster)
			newCluster := e.ObjectNew.(*kubermaticv1.Cluster)
			// Process the event only if the new cluster's CNI is managed by App Infra
			if !cni.IsManagedByAppInfra(newCluster.Spec.CNIPlugin.Type, newCluster.Spec.CNIPlugin.Version) {
				return false
			}
			// Process the event only if CNIPlugin oy ClusterNetwork config changed, or if cluster Address changed
			if !reflect.DeepEqual(oldCluster.Spec.CNIPlugin, newCluster.Spec.CNIPlugin) {
				return true
			}
			if !reflect.DeepEqual(oldCluster.Spec.ClusterNetwork, newCluster.Spec.ClusterNetwork) {
				return true
			}
			if !reflect.DeepEqual(oldCluster.Status.Address, newCluster.Status.Address) {
				return true
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// No action needed on Delete
			return false
		},
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{}, builder.WithPredicates(clusterEventPredicate, workerlabel.Predicate(workerName))).
		Build(reconciler)

	return err
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.Name)
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
	result, err := util.ClusterReconcileWrapper(
		ctx,
		r,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionCNIControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, log, cluster)
		},
	)

	if result == nil || err != nil {
		result = &reconcile.Result{}
	}

	if err != nil {
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, logger *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	log := logger.With("CNIType", cluster.Spec.CNIPlugin.Type, "CNIVersion", cluster.Spec.CNIPlugin.Version)

	if !cni.IsManagedByAppInfra(cluster.Spec.CNIPlugin.Type, cluster.Spec.CNIPlugin.Version) {
		return &reconcile.Result{}, nil // in case that CNI changed since last requeue, skip if it is not managed by this controller
	}

	// Make sure that cluster is in a state when creating ApplicationInstallation is permissible
	if !cluster.Status.ExtendedHealth.ApplicationControllerHealthy() {
		log.Debug("Requeue CNI reconciliation as Application controller is not healthy")
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil // try reconciling later
	}

	log.Debug("Reconciling CNI")

	// Ensure legacy CNI addon is removed if it was deployed as older CNI version
	requeueAfter, err := r.ensureLegacyCNIAddonIsRemoved(ctx, cluster)
	if err != nil {
		return &reconcile.Result{}, err
	}
	if requeueAfter > 0 {
		return &reconcile.Result{RequeueAfter: requeueAfter}, nil
	}

	// Prepare initialValues for the CNI ApplicationInstallation. These values will be used if the ApplicationInstallation does not exist yet.
	initialValues := make(map[string]any)

	// Try to load the initial values form the annotation
	if err := r.parseCNIValuesAnnotation(cluster, initialValues); err != nil {
		return &reconcile.Result{}, err
	}
	removeAnnotation := len(initialValues) > 0

	// If initial values were not loaded from the annotation, use the default values from the ApplicationDefinition
	if len(initialValues) == 0 {
		initialValues, err = r.parseAppDefDefaultValues(ctx, cluster)
		if err != nil {
			return &reconcile.Result{}, err
		}
	}

	// Ensure ApplicationInstallation of the CNI
	if err := r.ensureCNIApplicationInstallation(ctx, cluster, initialValues); err != nil {
		return &reconcile.Result{}, err
	}

	if removeAnnotation {
		if err := r.removeCNIValuesAnnotation(ctx, cluster); err != nil {
			return nil, fmt.Errorf("failed to remove initial CNI values annotation: %w", err)
		}
	}

	result := &reconcile.Result{}
	if r.systemAppEnforceInterval != 0 {
		// Reconciliation was successful, but requeue in systemAppEnforceInterval minutes if set.
		// We do this to make sure that ApplicationInstallation in the user cluster is not modified in a wrong way / deleted accidentally.
		// Even though it is protected by a webhook, not all unwanted modifications can be easily prevented there.
		result.RequeueAfter = time.Duration(r.systemAppEnforceInterval) * time.Minute
	}

	return result, nil
}

// ensureLegacyCNIAddonIsRemoved uninstalls CNI addons.
// It triggers the addon uninstall and checks if the addon has been uninstalled.
// If the addon has not been uninstalled, it will requeue after 5 seconds.
func (r *Reconciler) ensureLegacyCNIAddonIsRemoved(ctx context.Context, cluster *kubermaticv1.Cluster) (time.Duration, error) {
	addons := []string{cluster.Spec.CNIPlugin.Type.String()}
	if cluster.Spec.CNIPlugin.Type == kubermaticv1.CNIPluginTypeCilium {
		addons = append(addons, "hubble")
	}

	requeueAfter := time.Duration(0)
	for _, addon := range addons {
		cniAddon := &kubermaticv1.Addon{
			ObjectMeta: metav1.ObjectMeta{
				Name:      addon,
				Namespace: cluster.Status.NamespaceName,
			},
		}
		// trigger addon uninstall
		err := r.Delete(ctx, cniAddon)
		if err != nil && !apierrors.IsNotFound(err) {
			return 0, fmt.Errorf("failed to delete CNI addon %s: %w", cniAddon.GetName(), err)
		}

		// check addon has been uninstalled
		err = r.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(cniAddon), cniAddon)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return 0, fmt.Errorf("failed to check CNI addon %s has been uninstalled: %w", cniAddon.GetName(), err)
			}
		} else {
			requeueAfter = 5 * time.Second
		}
	}

	return requeueAfter, nil
}

func (r *Reconciler) parseCNIValuesAnnotation(cluster *kubermaticv1.Cluster, values map[string]any) error {
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

func (r *Reconciler) parseAppDefDefaultValues(ctx context.Context, cluster *kubermaticv1.Cluster) (map[string]any, error) {
	appDef := &appskubermaticv1.ApplicationDefinition{}
	if err := r.Get(ctx, types.NamespacedName{Name: cluster.Spec.CNIPlugin.Type.String()}, appDef); err != nil {
		return nil, ctrlruntimeclient.IgnoreNotFound(err)
	}

	values, err := appDef.Spec.GetParsedDefaultValues()
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal ApplicationDefinition default values: %w", err)
	}
	return values, nil
}

func (r *Reconciler) ensureCNIApplicationInstallation(ctx context.Context, cluster *kubermaticv1.Cluster, initialValues map[string]any) error {
	userClusterClient, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get user cluster client: %w", err)
	}

	reconcilers := []reconciling.NamedApplicationInstallationReconcilerFactory{
		ApplicationInstallationReconciler(cluster, r.overwriteRegistry, initialValues),
	}
	return reconciling.ReconcileApplicationInstallations(ctx, reconcilers, cniPluginNamespace, userClusterClient)
}

func ApplicationInstallationReconciler(cluster *kubermaticv1.Cluster, overwriteRegistry string, initialValues map[string]any) reconciling.NamedApplicationInstallationReconcilerFactory {
	return func() (string, reconciling.ApplicationInstallationReconciler) {
		return cluster.Spec.CNIPlugin.Type.String(), func(app *appskubermaticv1.ApplicationInstallation) (*appskubermaticv1.ApplicationInstallation, error) {
			app.Labels = map[string]string{
				appskubermaticv1.ApplicationManagedByLabel: appskubermaticv1.ApplicationManagedByKKPValue,
				appskubermaticv1.ApplicationTypeLabel:      appskubermaticv1.ApplicationTypeCNIValue,
			}
			app.Spec.ApplicationRef = appskubermaticv1.ApplicationRef{
				Name:    cluster.Spec.CNIPlugin.Type.String(),
				Version: cluster.Spec.CNIPlugin.Version,
			}
			app.Spec.Namespace = &appskubermaticv1.AppNamespaceSpec{
				Name: cniPluginNamespace,
			}
			app.Spec.DeployOptions = &appskubermaticv1.DeployOptions{
				Helm: &appskubermaticv1.HelmDeployOptions{
					// Use atomic deployment, as atomic (with fixed retries count) migitates breaking the etcd due to creating events
					// when retrying on failure without a limit
					Atomic: true,
					Wait:   true,
					Timeout: metav1.Duration{
						Duration: 10 * time.Minute, // use longer timeout, as it may take some time for the CNI to be fully up
					},
				},
			}
			app.Spec.ReconciliationInterval = metav1.Duration{
				Duration: 60 * time.Minute, // reconcile the app periodically
			}

			// Unmarshal existing values
			values, err := app.Spec.GetParsedValues()
			if err != nil {
				return app, fmt.Errorf("failed to unmarshal CNI values: %w", err)
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
			rawValues, err := yaml.Marshal(values)
			if err != nil {
				return app, fmt.Errorf("failed to marshall CNI values: %w", err)
			}
			app.Spec.ValuesBlock = string(rawValues)

			// Override .spec.values because values and valuesBlock must not be in the same
			// ApplicationInstallation spec. If an existing ApplicationInstallation gets reconciled,
			// both values (from pre-reconciliation) and valuesBlock (from this very reconciler func)
			// would otherwise be set and fail validation at the webhook.
			app.Spec.Values = runtime.RawExtension{
				Raw: []byte("{}"),
			}

			return app, nil
		}
	}
}

func getCNIOverrideValues(cluster *kubermaticv1.Cluster, overwriteRegistry string) map[string]any {
	if cluster.Spec.CNIPlugin.Type == kubermaticv1.CNIPluginTypeCilium {
		return getAppInstallOverrideValues(cluster, overwriteRegistry)
	}
	return nil
}

// getAppInstallOverrideValues returns Helm values to be enforced on the cluster's ApplicationInstallation
// of the Cilium CNI managed by KKP.
func getAppInstallOverrideValues(cluster *kubermaticv1.Cluster, overwriteRegistry string) map[string]any {
	podSecurityContext := map[string]any{
		"seccompProfile": map[string]any{
			"type": "RuntimeDefault",
		},
	}
	defaultValues := map[string]any{
		"podSecurityContext": podSecurityContext,
	}
	values := map[string]any{
		"podSecurityContext": podSecurityContext,
	}
	valuesEnvoy := map[string]any{
		"podSecurityContext": podSecurityContext,
	}

	valuesOperator := map[string]any{
		"securityContext": map[string]any{
			"seccompProfile": map[string]any{
				"type": "RuntimeDefault",
			},
		},
		"podSecurityContext": podSecurityContext,
	}
	valuesCni := map[string]any{
		// we run Cilium as non-exclusive CNI to allow for Multus use-cases
		"exclusive": false,
	}

	valuesCertGen := maps.Clone(defaultValues)
	valuesRelay := maps.Clone(defaultValues)
	valuesFrontend := map[string]any{}
	valuesBackend := map[string]any{}

	if cluster.Spec.ClusterNetwork.ProxyMode == resources.EBPFProxyMode {
		values["kubeProxyReplacement"] = "true"
		values["k8sServiceHost"] = cluster.Status.Address.ExternalName
		values["k8sServicePort"] = cluster.Status.Address.Port

		nodePortRange := cluster.Spec.ComponentsOverride.Apiserver.NodePortRange
		if nodePortRange != "" && nodePortRange != resources.DefaultNodePortRange {
			values["nodePort"] = map[string]any{
				"range": strings.ReplaceAll(nodePortRange, "-", ","),
			}
		}
	} else {
		values["kubeProxyReplacement"] = "false"
		values["sessionAffinity"] = true
		valuesCni["chainingMode"] = "portmap"
	}

	ipamOperator := map[string]any{
		"clusterPoolIPv4PodCIDRList": cluster.Spec.ClusterNetwork.Pods.GetIPv4CIDRs(),
	}

	if cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv4 != nil {
		ipamOperator["clusterPoolIPv4MaskSize"] = *cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv4
	}

	if cluster.IsDualStack() {
		values["ipv6"] = map[string]any{"enabled": true}
		ipamOperator["clusterPoolIPv6PodCIDRList"] = cluster.Spec.ClusterNetwork.Pods.GetIPv6CIDRs()
		if cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv6 != nil {
			ipamOperator["clusterPoolIPv6MaskSize"] = *cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv6
		}
	}

	values["ipam"] = map[string]any{"operator": ipamOperator}

	// Override images if registry override is configured
	if overwriteRegistry != "" {
		values["image"] = map[string]any{
			"repository": registry.Must(registry.RewriteImage(ciliumImageRegistry+"cilium", overwriteRegistry)),
			"useDigest":  false,
		}
		valuesOperator["image"] = map[string]any{
			"repository": registry.Must(registry.RewriteImage(ciliumImageRegistry+"operator", overwriteRegistry)),
			"useDigest":  false,
		}
		valuesRelay["image"] = map[string]any{
			"repository": registry.Must(registry.RewriteImage(ciliumImageRegistry+"hubble-relay", overwriteRegistry)),
			"useDigest":  false,
		}
		valuesBackend["image"] = map[string]any{
			"repository": registry.Must(registry.RewriteImage(ciliumImageRegistry+"hubble-ui-backend", overwriteRegistry)),
			"useDigest":  false,
		}
		valuesFrontend["image"] = map[string]any{
			"repository": registry.Must(registry.RewriteImage(ciliumImageRegistry+"hubble-ui", overwriteRegistry)),
			"useDigest":  false,
		}
		valuesCertGen["image"] = map[string]any{
			"repository": registry.Must(registry.RewriteImage(ciliumImageRegistry+"certgen", overwriteRegistry)),
			"useDigest":  false,
		}
		valuesEnvoy["image"] = map[string]any{
			"repository": registry.Must(registry.RewriteImage(ciliumImageRegistry+"cilium-envoy", overwriteRegistry)),
			"useDigest":  false,
		}
	}

	uiSecContext := maps.Clone(podSecurityContext)
	uiSecContext["enabled"] = true

	values["cni"] = valuesCni
	values["envoy"] = valuesEnvoy
	values["operator"] = valuesOperator
	values["certgen"] = valuesCertGen
	values["hubble"] = map[string]any{
		"relay": valuesRelay,
		"ui": map[string]any{
			"securityContext": uiSecContext,
			"frontend":        valuesFrontend,
			"backend":         valuesBackend,
		},
	}

	return values
}
