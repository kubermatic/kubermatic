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

package addon

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"go.uber.org/zap"

	addonutils "k8c.io/kubermatic/v2/pkg/addon"
	"k8c.io/kubermatic/v2/pkg/apis/equality"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/util/kubectl"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/yaml"
)

const (
	ControllerName = "kkp-addon-controller"

	addonLabelKey             = "kubermatic-addon"
	cleanupFinalizerName      = "cleanup-manifests"
	addonEnsureLabelKey       = "addons.kubermatic.io/ensure"
	migratedHetznerCSIAddon   = "kubermatic.k8c.io/migrated-hetzner-csi-addon"
	migratedVsphereCSIAddon   = "kubermatic.k8c.io/migrated-vsphere-csi-addon"
	csiAddonStorageClassLabel = "kubermatic-addon=csi"
	CSIAddonName              = "csi"
	pvMigrationAnnotation     = "pv.kubernetes.io/migrated-to"
	yes                       = "yes"
)

// KubeconfigProvider provides functionality to get a clusters admin kubeconfig.
type KubeconfigProvider interface {
	GetAdminKubeconfig(ctx context.Context, c *kubermaticv1.Cluster) ([]byte, error)
	GetClient(ctx context.Context, c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

// Reconciler stores necessary components that are required to manage in-cluster Add-On's.
type Reconciler struct {
	ctrlruntimeclient.Client

	log                  *zap.SugaredLogger
	workerName           string
	addonEnforceInterval int
	addonVariables       map[string]interface{}
	kubernetesAddonDir   string
	overwriteRegistry    string
	recorder             record.EventRecorder
	KubeconfigProvider   KubeconfigProvider
	versions             kubermatic.Versions
}

// Add creates a new Addon controller that is responsible for
// managing in-cluster addons.
func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	addonEnforceInterval int,
	addonCtxVariables map[string]interface{},
	kubernetesAddonDir,
	overwriteRegistry string,
	kubeconfigProvider KubeconfigProvider,
	versions kubermatic.Versions,
) error {
	log = log.Named(ControllerName)
	client := mgr.GetClient()

	reconciler := &Reconciler{
		Client: client,

		log:                  log,
		addonVariables:       addonCtxVariables,
		addonEnforceInterval: addonEnforceInterval,
		kubernetesAddonDir:   kubernetesAddonDir,
		KubeconfigProvider:   kubeconfigProvider,
		workerName:           workerName,
		recorder:             mgr.GetEventRecorderFor(ControllerName),
		overwriteRegistry:    overwriteRegistry,
		versions:             versions,
	}

	ctrlOptions := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	enqueueClusterAddons := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
		cluster := a.(*kubermaticv1.Cluster)
		if cluster.Status.NamespaceName == "" {
			return nil
		}

		addonList := &kubermaticv1.AddonList{}
		if err := client.List(ctx, addonList, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName)); err != nil {
			log.Errorw("Failed to get addons for cluster", zap.Error(err), "cluster", cluster.Name)
			return nil
		}
		var requests []reconcile.Request
		for _, addon := range addonList.Items {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: addon.Namespace, Name: addon.Name},
			})
		}
		return requests
	})

	clusterPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObj := e.ObjectOld.(*kubermaticv1.Cluster)
			newObj := e.ObjectNew.(*kubermaticv1.Cluster)

			reconcile, err := shouldReconcileCluster(oldObj, newObj)
			if err != nil {
				utilruntime.HandleError(fmt.Errorf("failed to diff clusters: %w", err))
				return true
			}

			return reconcile
		},
	}
	if err := c.Watch(source.Kind(mgr.GetCache(), &kubermaticv1.Cluster{}), enqueueClusterAddons, clusterPredicate); err != nil {
		return err
	}

	return c.Watch(source.Kind(mgr.GetCache(), &kubermaticv1.Addon{}), &handler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{})
}

func shouldReconcileCluster(oldCluster, newCluster *kubermaticv1.Cluster) (bool, error) {
	// kubeconfig and credentials are external Secrets, so they can have no influence on this
	// decision and can be left with dummy values; in a more elaborate implementation, the real
	// kubeconfig/credentials resourceVersions could be remembered also in the AddonStatus, but at
	// that point we're re-implementing the Applications feature.
	// If kubeconfig/credentials change, we rely on the auto-resync behaviour of Kubernetes.

	createData := func(cluster *kubermaticv1.Cluster) (*addonutils.TemplateData, error) {
		return addonutils.NewTemplateData(
			cluster,
			resources.Credentials{},
			"<kubeconfig>",
			"1.2.3.4", // cluster DNS
			"5.6.7.8", // DNS resolver
			nil,
			nil,
		)
	}

	oldData, err := createData(oldCluster)
	if err != nil {
		return false, fmt.Errorf("failed to create template data for old cluster: %w", err)
	}

	newData, err := createData(newCluster)
	if err != nil {
		return false, fmt.Errorf("failed to create template data for new cluster: %w", err)
	}

	return !equality.Semantic.DeepEqual(oldData, newData), nil
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("addon", request)
	log.Debug("Processing")

	addon := &kubermaticv1.Addon{}
	if err := r.Get(ctx, request.NamespacedName, addon); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	cluster := &kubermaticv1.Cluster{}
	err := r.Get(ctx, types.NamespacedName{Name: addon.Spec.Cluster.Name}, cluster)
	isNotFound := apierrors.IsNotFound(err)
	if err != nil && !isNotFound {
		return reconcile.Result{}, err
	}

	// If the cluster is deleted, there is no point in cleaning up addons properly anymore.
	// Free the addon from its cleanup finalizer and delete it. This should happen regardless
	// of the control plane version being set in order to not prevent cleanups in defunct clusters.
	if isNotFound || cluster.DeletionTimestamp != nil {
		return reconcile.Result{}, r.garbageCollectAddon(ctx, log, addon)
	}

	log = r.log.With("cluster", cluster.Name, "addon", addon.Name)

	if cluster.Status.Versions.ControlPlane == "" {
		log.Debug("Skipping because the cluster has no version status yet, skipping")
		return reconcile.Result{}, nil
	}

	if cluster.Status.ExtendedHealth.Apiserver != kubermaticv1.HealthStatusUp {
		log.Debug("API server is not running, trying again in 3 seconds")
		return reconcile.Result{RequeueAfter: 3 * time.Second}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionAddonControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, log, addon, cluster)
		},
	)

	if err != nil {
		r.recorder.Event(addon, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError",
			"failed to reconcile Addon %q: %v", addon.Name, err)

		return reconcile.Result{}, err
	}

	if result == nil {
		// we check for this after the ClusterReconcileWrapper() call because otherwise the cluster would never reconcile since we always requeue
		result = &reconcile.Result{}
		if r.addonEnforceInterval != 0 { // addon enforce is enabled
			// All is well, requeue in addonEnforceInterval minutes. We do this to enforce default addons and prevent cluster admins from disabling them.
			// We only set this if err == nil, as controller-runtime would ignore it otherwise and log a warning.
			result.RequeueAfter = time.Duration(r.addonEnforceInterval) * time.Minute
		}
	}

	return *result, nil
}

// garbageCollectAddon is called when the cluster that owns the addon is gone
// or in deletion. The function ensures that the addon is removed without going
// through the normal cleanup procedure (i.e. no `kubectl delete`).
func (r *Reconciler) garbageCollectAddon(ctx context.Context, log *zap.SugaredLogger, addon *kubermaticv1.Addon) error {
	if addon.DeletionTimestamp == nil {
		if err := r.Delete(ctx, addon); err != nil {
			return fmt.Errorf("failed to delete Addon: %w", err)
		}
	}

	if err := r.removeCleanupFinalizer(ctx, log, addon); err != nil {
		return fmt.Errorf("failed to remove cleanup finalizer: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	reqeueAfter, err := r.ensureRequiredResourceTypesExist(ctx, log, addon, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to check if all required resources exist: %w", err)
	}
	if reqeueAfter != nil {
		return reqeueAfter, nil
	}

	if addon.DeletionTimestamp != nil {
		if err := r.cleanupManifests(ctx, log, addon, cluster); err != nil {
			return nil, fmt.Errorf("failed to delete manifests from cluster: %w", err)
		}
		if err := r.removeCleanupFinalizer(ctx, log, addon); err != nil {
			return nil, fmt.Errorf("failed to remove cleanup finalizer from addon: %w", err)
		}
		return nil, nil
	}

	// This is true when the addon: 1) is fully deployed, 2) doesn't have a `addonEnsureLabelKey` set to true.
	// we do this to allow users to "edit/delete" resources deployed by unlabeled addons,
	// while we enforce the labeled ones
	if addonResourcesCreated(addon) && !hasEnsureResourcesLabel(addon) {
		return nil, nil
	}

	// Reconciling
	if err := r.ensureFinalizerIsSet(ctx, addon); err != nil {
		return nil, fmt.Errorf("failed to ensure that the cleanup finalizer exists on the addon: %w", err)
	}
	if err := r.ensureIsInstalled(ctx, log, addon, cluster); err != nil {
		return nil, fmt.Errorf("failed to deploy the addon manifests into the cluster: %w", err)
	}
	if err := r.ensureResourcesCreatedConditionIsSet(ctx, addon); err != nil {
		return nil, fmt.Errorf("failed to set add ResourcesCreated Condition: %w", err)
	}
	return nil, nil
}

func (r *Reconciler) removeCleanupFinalizer(ctx context.Context, log *zap.SugaredLogger, addon *kubermaticv1.Addon) error {
	return kuberneteshelper.TryRemoveFinalizer(ctx, r, addon, cleanupFinalizerName)
}

func (r *Reconciler) getAddonManifests(ctx context.Context, log *zap.SugaredLogger, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) ([]addonutils.Manifest, error) {
	addonDir := r.kubernetesAddonDir
	clusterIP, err := resources.UserClusterDNSResolverIP(cluster)
	if err != nil {
		return nil, err
	}
	dnsResolverIP := clusterIP
	if cluster.Spec.ClusterNetwork.NodeLocalDNSCacheEnabled == nil || *cluster.Spec.ClusterNetwork.NodeLocalDNSCacheEnabled {
		// NOTE: even if NodeLocalDNSCacheEnabled is nil, we assume it is enabled (backward compatibility for already existing clusters)
		dnsResolverIP = resources.NodeLocalDNSCacheAddress
	}

	kubeconfig, err := r.KubeconfigProvider.GetAdminKubeconfig(ctx, cluster)
	if err != nil {
		return nil, err
	}

	credentials, err := resources.GetCredentials(resources.NewCredentialsData(ctx, cluster, r.Client))
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	// Add addon variables if available.
	variables := make(map[string]interface{})

	if sub := r.addonVariables[addon.Spec.Name]; sub != nil {
		variables = sub.(map[string]interface{})
	}

	if addon.Spec.Variables != nil && len(addon.Spec.Variables.Raw) > 0 {
		if err = json.Unmarshal(addon.Spec.Variables.Raw, &variables); err != nil {
			return nil, err
		}
	}

	// listing IPAM allocations for cluster
	ipamAllocationList := &kubermaticv1.IPAMAllocationList{}
	if r.Client != nil {
		if err := r.Client.List(ctx, ipamAllocationList, &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}); err != nil {
			return nil, fmt.Errorf("failed to list IPAM allocations: %w", err)
		}
	}

	data, err := addonutils.NewTemplateData(
		cluster,
		credentials,
		string(kubeconfig),
		clusterIP,
		dnsResolverIP,
		ipamAllocationList,
		variables,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create template data for addon manifests: %w", err)
	}

	manifestPath := path.Join(addonDir, addon.Spec.Name)
	allManifests, err := addonutils.ParseFromFolder(log, r.overwriteRegistry, manifestPath, data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse addon templates in %s: %w", manifestPath, err)
	}

	return allManifests, nil
}

// combineManifests returns all manifests combined into a multi document yaml.
func (r *Reconciler) combineManifests(manifests []*bytes.Buffer) *bytes.Buffer {
	parts := make([]string, len(manifests))
	for i, m := range manifests {
		s := m.String()
		s = strings.TrimSuffix(s, "\n")
		s = strings.TrimSpace(s)
		parts[i] = s
	}

	return bytes.NewBufferString(strings.Join(parts, "\n---\n") + "\n")
}

// ensureAddonLabelOnManifests adds the addonLabelKey label to all manifests.
// For this to happen we need to decode all yaml files to json, parse them, add the label and finally encode to yaml again.
func (r *Reconciler) ensureAddonLabelOnManifests(addon *kubermaticv1.Addon, manifests []addonutils.Manifest) ([]*bytes.Buffer, error) {
	var rawManifests []*bytes.Buffer

	wantLabels := r.getAddonLabel(addon)
	for _, m := range manifests {
		parsedUnstructuredObj := &metav1unstructured.Unstructured{}
		if _, _, err := metav1unstructured.UnstructuredJSONScheme.Decode(m.Content.Raw, nil, parsedUnstructuredObj); err != nil {
			return nil, fmt.Errorf("parsing unstructured failed: %w", err)
		}

		existingLabels := parsedUnstructuredObj.GetLabels()
		if existingLabels == nil {
			existingLabels = map[string]string{}
		}

		// Apply the wanted labels
		for k, v := range wantLabels {
			existingLabels[k] = v
		}
		parsedUnstructuredObj.SetLabels(existingLabels)

		jsonBuffer := &bytes.Buffer{}
		if err := metav1unstructured.UnstructuredJSONScheme.Encode(parsedUnstructuredObj, jsonBuffer); err != nil {
			return nil, fmt.Errorf("encoding json failed: %w", err)
		}

		// Must be encoding back to yaml, otherwise kubectl fails to apply because it tries to parse the whole
		// thing as json
		yamlBytes, err := yaml.JSONToYAML(jsonBuffer.Bytes())
		if err != nil {
			return nil, err
		}

		rawManifests = append(rawManifests, bytes.NewBuffer(yamlBytes))
	}

	return rawManifests, nil
}

func (r *Reconciler) getAddonLabel(addon *kubermaticv1.Addon) map[string]string {
	return map[string]string{
		addonLabelKey: addon.Spec.Name,
	}
}

type fileHandlingDone func()

func getFileDeleteFinalizer(log *zap.SugaredLogger, filename string) fileHandlingDone {
	return func() {
		if err := os.RemoveAll(filename); err != nil {
			log.Errorw("Failed to delete file", zap.Error(err), "file", filename)
		}
	}
}

func (r *Reconciler) writeCombinedManifest(log *zap.SugaredLogger, manifest *bytes.Buffer, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) (string, fileHandlingDone, error) {
	// Write combined Manifest to disk
	manifestFilename := path.Join("/tmp", fmt.Sprintf("cluster-%s-%s.yaml", cluster.Name, addon.Name))
	if err := os.WriteFile(manifestFilename, manifest.Bytes(), 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write combined manifest to %s: %w", manifestFilename, err)
	}
	log.Debugw("Wrote combined manifest", "file", manifestFilename)

	return manifestFilename, getFileDeleteFinalizer(log, manifestFilename), nil
}

func (r *Reconciler) writeAdminKubeconfig(ctx context.Context, log *zap.SugaredLogger, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) (string, fileHandlingDone, error) {
	// Write kubeconfig to disk
	kubeconfig, err := r.KubeconfigProvider.GetAdminKubeconfig(ctx, cluster)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get admin kubeconfig for cluster %s: %w", cluster.Name, err)
	}
	kubeconfigFilename := path.Join("/tmp", fmt.Sprintf("cluster-%s-addon-%s-kubeconfig", cluster.Name, addon.Name))
	if err := os.WriteFile(kubeconfigFilename, kubeconfig, 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write admin kubeconfig for cluster %s: %w", cluster.Name, err)
	}
	log.Debugw("Wrote admin kubeconfig", "file", kubeconfigFilename)

	return kubeconfigFilename, getFileDeleteFinalizer(log, kubeconfigFilename), nil
}

func (r *Reconciler) setupManifestInteraction(ctx context.Context, log *zap.SugaredLogger, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) (string, string, fileHandlingDone, error) {
	manifests, err := r.getAddonManifests(ctx, log, addon, cluster)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to get addon manifests: %w", err)
	}

	rawManifests, err := r.ensureAddonLabelOnManifests(addon, manifests)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to add the addon specific label to all addon resources: %w", err)
	}

	rawManifest := r.combineManifests(rawManifests)
	manifestFilename, manifestDone, err := r.writeCombinedManifest(log, rawManifest, addon, cluster)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to write all addon resources into a combined manifest file: %w", err)
	}

	kubeconfigFilename, kubeconfigDone, err := r.writeAdminKubeconfig(ctx, log, addon, cluster)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to write the admin kubeconfig to the local filesystem: %w", err)
	}

	done := func() {
		kubeconfigDone()
		manifestDone()
	}
	return kubeconfigFilename, manifestFilename, done, nil
}

func (r *Reconciler) getApplyCommand(ctx context.Context, kubeconfigFilename, manifestFilename string, selector fmt.Stringer, clusterVersion semver.Semver) (*exec.Cmd, error) {
	binary, err := kubectl.BinaryForClusterVersion(&clusterVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to determine kubectl binary to use: %w", err)
	}

	cmd := exec.CommandContext(
		ctx,
		binary,
		"--kubeconfig", kubeconfigFilename,
		"apply",
		"--prune",
		"--filename", manifestFilename,
		"--selector", selector.String(),
	)
	return cmd, nil
}

// Between v2.22 and v2.23, there was a change to hetzner CSI driver immutable field fsGroupPolicy
// as a result, the CSDriver resource has to be redeployed
// https://github.com/kubermatic/kubermatic/issues/12429
func (r *Reconciler) migrateHetznerCSIDriver(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	cl, err := r.KubeconfigProvider.GetClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get kube client: %w", err)
	}

	driver := &storagev1.CSIDriver{}
	if err := cl.Get(ctx, types.NamespacedName{Name: "csi.hetzner.cloud"}, driver); apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get CSIDriver: %w", err)
	}

	if driver.Spec.FSGroupPolicy == nil || *driver.Spec.FSGroupPolicy != storagev1.FileFSGroupPolicy {
		log.Debug("deleting hetzner CSIDriver to allow upgrade")
		if err := cl.Delete(ctx, driver); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete old CSIDriver: %w", err)
		}
	}
	return nil
}

// Between v2.23 and v2.24, there was a change to the vSphere CSI driver immutable field volumeLifecycleModes
// as a result, the CSDriver resource has to be redeployed
// https://github.com/kubermatic/kubermatic/issues/12801
func (r *Reconciler) migrateVsphereCSIDriver(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	cl, err := r.KubeconfigProvider.GetClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get kube client: %w", err)
	}

	driver := &storagev1.CSIDriver{}
	if err := cl.Get(ctx, types.NamespacedName{Name: "csi.vsphere.vmware.com"}, driver); apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get CSIDriver: %w", err)
	}

	if len(driver.Spec.VolumeLifecycleModes) > 1 || (len(driver.Spec.VolumeLifecycleModes) == 1 && driver.Spec.VolumeLifecycleModes[0] != storagev1.VolumeLifecyclePersistent) {
		log.Debug("deleting vSphere CSIDriver to allow upgrade")
		if err := cl.Delete(ctx, driver); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete old CSIDriver: %w", err)
		}
	}
	return nil
}

func (r *Reconciler) ensureIsInstalled(ctx context.Context, log *zap.SugaredLogger, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) error {
	kubeconfigFilename, manifestFilename, done, err := r.setupManifestInteraction(ctx, log, addon, cluster)
	if err != nil {
		return err
	}
	defer done()

	d, err := os.ReadFile(manifestFilename)
	if err != nil {
		return err
	}
	sd := strings.TrimSpace(string(d))
	if len(sd) == 0 {
		log.Debug("Skipping addon installation as the manifest is empty after parsing")
		return nil
	}

	// We delete all resources with this label which are not in the combined manifest
	selector := labels.SelectorFromSet(r.getAddonLabel(addon))
	cmd, err := r.getApplyCommand(ctx, kubeconfigFilename, manifestFilename, selector, cluster.Status.Versions.ControlPlane)
	if err != nil {
		return fmt.Errorf("failed to create command: %w", err)
	}

	if addon.Name == "csi" {
		if cluster.Spec.Cloud.Hetzner != nil &&
			cluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] &&
			cluster.Annotations[migratedHetznerCSIAddon] != yes {
			// Between v2.22 and v2.23, there was a change to hetzner CSI driver immutable field fsGroupPolicy
			// as a result, the CSDriver resource has to be redeployed
			// https://github.com/kubermatic/kubermatic/issues/12429
			if err := r.migrateHetznerCSIDriver(ctx, log, cluster); err != nil {
				return fmt.Errorf("failed to migrate CSI Driver: %w", err)
			}
			if cluster.Annotations == nil {
				cluster.Annotations = make(map[string]string)
			}
			cluster.Annotations[migratedHetznerCSIAddon] = yes
			if err := r.Update(ctx, cluster); err != nil {
				log.Errorw("failed to set cluster annotation", zap.Error(err), "annotation", migratedHetznerCSIAddon)
			}
		} else if cluster.Spec.Cloud.VSphere != nil &&
			cluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] &&
			cluster.Annotations[migratedVsphereCSIAddon] != yes {
			if err := r.migrateVsphereCSIDriver(ctx, log, cluster); err != nil {
				return fmt.Errorf("failed to migrate CSI Driver: %w", err)
			}
			if cluster.Annotations == nil {
				cluster.Annotations = make(map[string]string)
			}
			cluster.Annotations[migratedVsphereCSIAddon] = yes
			if err := r.Update(ctx, cluster); err != nil {
				log.Errorw("failed to set cluster annotation", zap.Error(err), "annotation", migratedVsphereCSIAddon)
			}
		}
	}

	cmdLog := log.With("cmd", strings.Join(cmd.Args, " "))
	cmdLog.Debug("Applying manifest...")
	out, err := cmd.CombinedOutput()
	cmdLog.Debugw("Finished executing command", "output", string(out))
	if err != nil {
		return fmt.Errorf("failed to execute '%s' for addon %s of cluster %s: %w\n%s", strings.Join(cmd.Args, " "), addon.Name, cluster.Name, err, string(out))
	}

	if addon.Name == CSIAddonName {
		err := r.csiAddonInUseStatus(ctx, log, cluster)
		if err != nil {
			return fmt.Errorf("failed to update csi addon status %w", err)
		}
	}
	return nil
}

func (r *Reconciler) ensureFinalizerIsSet(ctx context.Context, addon *kubermaticv1.Addon) error {
	return kuberneteshelper.TryAddFinalizer(ctx, r, addon, cleanupFinalizerName)
}

func (r *Reconciler) ensureResourcesCreatedConditionIsSet(ctx context.Context, addon *kubermaticv1.Addon) error {
	if addon.Status.Conditions[kubermaticv1.AddonResourcesCreated].Status == corev1.ConditionTrue {
		return nil
	}

	oldAddon := addon.DeepCopy()

	setAddonCondition(addon, kubermaticv1.AddonResourcesCreated, corev1.ConditionTrue)
	return r.Client.Status().Patch(ctx, addon, ctrlruntimeclient.MergeFrom(oldAddon))
}

func (r *Reconciler) cleanupManifests(ctx context.Context, log *zap.SugaredLogger, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) error {
	kubeconfigFilename, manifestFilename, done, err := r.setupManifestInteraction(ctx, log, addon, cluster)
	if err != nil {
		// FIXME: use a dedicated error type and proper error unwrapping when we have the technology to do it
		if strings.Contains(err.Error(), "no such file or directory") { // if the manifest is already deleted, that's ok
			log.Debugf("cleanupManifests failed for addon %s/%s: %v", addon.Namespace, addon.Name, err)
			return nil
		}
		return err
	}
	defer done()

	binary, err := kubectl.BinaryForClusterVersion(&cluster.Status.Versions.ControlPlane)
	if err != nil {
		return fmt.Errorf("failed to determine kubectl binary to use: %w", err)
	}

	cmd := exec.CommandContext(ctx, binary, "--kubeconfig", kubeconfigFilename, "delete", "-f", manifestFilename, "--ignore-not-found")
	cmdLog := log.With("cmd", strings.Join(cmd.Args, " "))

	cmdLog.Debug("Deleting resources...")
	out, err := cmd.CombinedOutput()
	cmdLog.Debugw("Finished executing command", "output", string(out))
	if err != nil {
		return fmt.Errorf("failed to execute '%s' for addon %s of cluster %s: %w\n%s", strings.Join(cmd.Args, " "), addon.Name, cluster.Name, err, string(out))
	}
	if addon.Name == CSIAddonName {
		oldCluster := cluster.DeepCopy()
		_, ok := cluster.Status.Conditions[kubermaticv1.ClusterConditionCSIAddonInUse]
		if ok {
			delete(cluster.Status.Conditions, kubermaticv1.ClusterConditionCSIAddonInUse)
		}
		err = r.Client.Status().Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
	}
	return err
}

func (r *Reconciler) ensureRequiredResourceTypesExist(ctx context.Context, log *zap.SugaredLogger, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	if len(addon.Spec.RequiredResourceTypes) == 0 {
		// Avoid constructing a client we don't need and just return early
		return nil, nil
	}
	userClusterClient, err := r.KubeconfigProvider.GetClient(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get client for usercluster: %w", err)
	}

	for _, requiredResource := range addon.Spec.RequiredResourceTypes {
		unstructuedList := &metav1unstructured.UnstructuredList{}
		unstructuedList.SetAPIVersion(requiredResource.Group + "/" + requiredResource.Version)
		unstructuedList.SetKind(requiredResource.Kind)

		// We do not care about the result, just if the resource is served, so make sure we only
		// get as little as possible.
		listOpts := &ctrlruntimeclient.ListOptions{Limit: 1}
		if err := userClusterClient.List(ctx, unstructuedList, listOpts); err != nil {
			if meta.IsNoMatchError(err) {
				// Try again later
				log.Infow("Required resource isn't served, trying again in 10 seconds", "resource", formatGVK(requiredResource))
				return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
			}
			return nil, fmt.Errorf("failed to check if type %q is served: %w", formatGVK(requiredResource), err)
		}
	}

	return nil, nil
}

func formatGVK(gvk kubermaticv1.GroupVersionKind) string {
	return fmt.Sprintf("%s/%s %s", gvk.Group, gvk.Version, gvk.Kind)
}

func setAddonCondition(a *kubermaticv1.Addon, condType kubermaticv1.AddonConditionType, status corev1.ConditionStatus) {
	now := metav1.Now()

	condition, exists := a.Status.Conditions[condType]
	if exists && condition.Status != status {
		condition.LastTransitionTime = now
	}

	condition.Status = status
	condition.LastHeartbeatTime = now

	if a.Status.Conditions == nil {
		a.Status.Conditions = map[kubermaticv1.AddonConditionType]kubermaticv1.AddonCondition{}
	}
	a.Status.Conditions[condType] = condition
}

func addonResourcesCreated(addon *kubermaticv1.Addon) bool {
	return addon.Status.Conditions[kubermaticv1.AddonResourcesCreated].Status == corev1.ConditionTrue
}

func hasEnsureResourcesLabel(addon *kubermaticv1.Addon) bool {
	return addon.Labels[addonEnsureLabelKey] == "true"
}

func (r *Reconciler) csiAddonInUseStatus(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	status, reason := r.checkCSIAddonInUse(ctx, log, cluster)
	csiAddonInUse := kubermaticv1.ClusterCondition{
		Status:            status,
		KubermaticVersion: r.versions.Kubermatic,
		LastHeartbeatTime: metav1.Now(),
		Reason:            reason,
	}
	oldCluster := cluster.DeepCopy()
	cluster.Status.Conditions[kubermaticv1.ClusterConditionCSIAddonInUse] = csiAddonInUse
	err := r.Client.Status().Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
	return err
}

func (r *Reconciler) checkCSIAddonInUse(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (corev1.ConditionStatus, string) {
	userClusterClient, err := r.KubeconfigProvider.GetClient(ctx, cluster)
	if err != nil {
		return corev1.ConditionUnknown, fmt.Sprintf("failed to get client for usercluster: %v", err)
	}
	csiDriverList := &storagev1.CSIDriverList{}
	csiDriverListOption := &ctrlruntimeclient.ListOptions{
		Raw: &metav1.ListOptions{
			LabelSelector: csiAddonStorageClassLabel,
		},
	}

	// Get CSI drivers created by the csi addon
	if err := userClusterClient.List(ctx, csiDriverList, csiDriverListOption); apierrors.IsNotFound(err) {
		return corev1.ConditionUnknown, ""
	} else if err != nil {
		return corev1.ConditionUnknown, fmt.Sprintf("failed to list csi drivers with %v label : %v", csiAddonStorageClassLabel, err)
	}
	// map to hold csi drivers to storage classes list
	csiToSC := make(map[string][]string)
	csiToPV := make(map[string][]string)
	errMsg := []string{}
	for i := 0; i < len(csiDriverList.Items); i++ {
		if !strings.Contains(csiDriverList.Items[i].Name, "csi") {
			continue
		}
		// get all the storage classes that are using the csi driver created by csi addon as provisioner
		storageCLassList, err := r.storageClassesForProvisioner(ctx, cluster, csiDriverList.Items[i].Name)
		if err != nil {
			return corev1.ConditionUnknown, fmt.Sprintf("failed to get the list of storage classes using %v provisioner : %v", csiDriverList.Items[i].Name, err)
		}
		for j := 0; j < len(storageCLassList); j++ {
			pvcList, err := r.pvcsForStorageClass(ctx, cluster, storageCLassList[j])
			if err != nil {
				return corev1.ConditionUnknown, fmt.Sprintf("failed to get the list of PVCs referring the %v storage class : %v", storageCLassList[j], err)
			}
			if len(pvcList) > 0 {
				csiToSC[csiDriverList.Items[i].Name] = append(csiToSC[csiDriverList.Items[i].Name], storageCLassList[j])
			}
		}
		if len(csiToSC[csiDriverList.Items[i].Name]) != 0 {
			errMsg = append(errMsg, fmt.Sprintf("csidriver %s is being used by storage classes %s", csiDriverList.Items[i].Name, csiToSC[csiDriverList.Items[i].Name]))
		} else {
			// Check for the corner case where admin removes the SC without removing the PVs & tries to disable CSI drivers used by them.
			pvList, err := r.pvsForProvisioner(ctx, cluster, csiDriverList.Items[i].Name)
			if err != nil {
				return corev1.ConditionUnknown, fmt.Sprintf("failed to get the list of PVs having %v as provisioner : %v", csiDriverList.Items[i].Name, err)
			}
			if len(pvList) > 0 {
				csiToPV[csiDriverList.Items[i].Name] = append(csiToPV[csiDriverList.Items[i].Name], pvList[0])
				errMsg = append(errMsg, fmt.Sprintf("csidriver %s is being used by PV %s & %d other PVs", csiDriverList.Items[i].Name, pvList[0], (len(pvList)-1)))
			}
		}
	}
	if len(csiToSC) == 0 && len(csiToPV) == 0 {
		return corev1.ConditionFalse, ""
	}

	return corev1.ConditionTrue, fmt.Sprintf("%v", errMsg)
}

func (r *Reconciler) storageClassesForProvisioner(ctx context.Context, cluster *kubermaticv1.Cluster, provisionerName string) ([]string, error) {
	storageCLassesForProvisioner := []string{}
	cl, err := r.KubeconfigProvider.GetClient(ctx, cluster)
	if err != nil {
		return storageCLassesForProvisioner, fmt.Errorf("failed to get client for usercluster: %w", err)
	}
	storageCLassList := &storagev1.StorageClassList{}
	if err := cl.List(ctx, storageCLassList); apierrors.IsNotFound(err) {
		return storageCLassesForProvisioner, nil
	} else if err != nil {
		return storageCLassesForProvisioner, fmt.Errorf("failed to get storage classes using provisioner %v : %w", provisionerName, err)
	}
	for i := 0; i < len(storageCLassList.Items); i++ {
		if storageCLassList.Items[i].Provisioner == provisionerName {
			storageCLassesForProvisioner = append(storageCLassesForProvisioner, storageCLassList.Items[i].Name)
		}
	}
	return storageCLassesForProvisioner, nil
}

func (r *Reconciler) pvcsForStorageClass(ctx context.Context, cluster *kubermaticv1.Cluster, storageClassName string) ([]string, error) {
	pvcsForStorageClass := []string{}
	cl, err := r.KubeconfigProvider.GetClient(ctx, cluster)
	if err != nil {
		return pvcsForStorageClass, fmt.Errorf("failed to get client for usercluster: %w", err)
	}
	pvcList := &corev1.PersistentVolumeClaimList{}
	if err := cl.List(ctx, pvcList); apierrors.IsNotFound(err) {
		return pvcsForStorageClass, nil
	} else if err != nil {
		return pvcsForStorageClass, fmt.Errorf("failed to get the list of PVCs referring the %v storage class : %w", storageClassName, err)
	}
	for i := 0; i < len(pvcList.Items); i++ {
		if *pvcList.Items[i].Spec.StorageClassName == storageClassName {
			pvcsForStorageClass = append(pvcsForStorageClass, pvcList.Items[i].Name)
		}
	}
	return pvcsForStorageClass, nil
}

func (r *Reconciler) pvsForProvisioner(ctx context.Context, cluster *kubermaticv1.Cluster, provisionerName string) ([]string, error) {
	pvsForProvisioner := []string{}
	cl, err := r.KubeconfigProvider.GetClient(ctx, cluster)
	if err != nil {
		return pvsForProvisioner, fmt.Errorf("failed to get client for usercluster: %w", err)
	}
	pvList := &corev1.PersistentVolumeList{}
	if err := cl.List(ctx, pvList); apierrors.IsNotFound(err) {
		return pvsForProvisioner, nil
	} else if err != nil {
		return pvsForProvisioner, fmt.Errorf("failed to get the list of PVs having %v as provisioner : %w", provisionerName, err)
	}
	for i := 0; i < len(pvList.Items); i++ {
		if pvList.Items[i].Spec.CSI != nil {
			if pvList.Items[i].Spec.CSI.Driver == provisionerName {
				pvsForProvisioner = append(pvsForProvisioner, pvList.Items[i].Name)
			}
		} else {
			if pvList.Items[i].ObjectMeta.Annotations[pvMigrationAnnotation] == provisionerName {
				pvsForProvisioner = append(pvsForProvisioner, pvList.Items[i].Name)
			}
		}
	}
	return pvsForProvisioner, nil
}
