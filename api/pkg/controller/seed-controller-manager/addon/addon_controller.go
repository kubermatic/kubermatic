package addon

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/ghodss/yaml"

	"go.uber.org/zap"

	addonutils "github.com/kubermatic/kubermatic/api/pkg/addon"
	clusterclient "github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticv1helper "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1/helper"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machinecontroller"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
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
	ControllerName = "kubermatic_addon_controller"

	addonLabelKey        = "kubermatic-addon"
	cleanupFinalizerName = "cleanup-manifests"
)

// KubeconfigProvider provides functionality to get a clusters admin kubeconfig
type KubeconfigProvider interface {
	GetAdminKubeconfig(c *kubermaticv1.Cluster) ([]byte, error)
	GetClient(c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

// Reconciler stores necessary components that are required to manage in-cluster Add-On's
type Reconciler struct {
	log                  *zap.SugaredLogger
	workerName           string
	addonEnforceInterval int
	addonVariables       map[string]interface{}
	kubernetesAddonDir   string
	openshiftAddonDir    string
	overwriteRegistry    string
	ctrlruntimeclient.Client
	recorder                 record.EventRecorder
	KubeconfigProvider       KubeconfigProvider
	nodeLocalDNSCacheEnabled bool
}

// Add creates a new Addon controller that is responsible for
// managing in-cluster addons
func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	addonEnforceInterval int,
	addonCtxVariables map[string]interface{},
	kubernetesAddonDir,
	openshiftAddonDir,
	overwriteRegistey string,
	nodeLocalDNSCacheEnabled bool,
	kubeconfigProvider KubeconfigProvider,
) error {
	log = log.Named(ControllerName)
	client := mgr.GetClient()

	reconciler := &Reconciler{
		log:                      log,
		addonVariables:           addonCtxVariables,
		addonEnforceInterval:     addonEnforceInterval,
		kubernetesAddonDir:       kubernetesAddonDir,
		openshiftAddonDir:        openshiftAddonDir,
		KubeconfigProvider:       kubeconfigProvider,
		Client:                   client,
		workerName:               workerName,
		recorder:                 mgr.GetEventRecorderFor(ControllerName),
		overwriteRegistry:        overwriteRegistey,
		nodeLocalDNSCacheEnabled: nodeLocalDNSCacheEnabled,
	}

	ctrlOptions := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	enqueueClusterAddons := &handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
		cluster := a.Object.(*kubermaticv1.Cluster)
		if cluster.Status.NamespaceName == "" {
			return nil
		}

		addonList := &kubermaticv1.AddonList{}
		if err := client.List(context.Background(), addonList, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName)); err != nil {
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
	})}

	// Only react cluster update events when our condition changed
	clusterPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			old := e.ObjectOld.(*kubermaticv1.Cluster)
			new := e.ObjectNew.(*kubermaticv1.Cluster)
			_, oldCondition := kubermaticv1helper.GetClusterCondition(old, kubermaticv1.ClusterConditionAddonControllerReconcilingSuccess)
			_, newCondition := kubermaticv1helper.GetClusterCondition(new, kubermaticv1.ClusterConditionAddonControllerReconcilingSuccess)
			return !reflect.DeepEqual(oldCondition, newCondition)
		},
	}
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, enqueueClusterAddons, clusterPredicate); err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &kubermaticv1.Addon{}}, &handler.EnqueueRequestForObject{})
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := r.log.With("request", request)
	log.Debug("Processing")

	addon := &kubermaticv1.Addon{}
	if err := r.Get(ctx, request.NamespacedName, addon); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: addon.Spec.Cluster.Name}, cluster); err != nil {
		// If it's not a NotFound err, return it
		if !kerrors.IsNotFound(err) {
			return reconcile.Result{}, err
		}

		// Remove the cleanup finalizer if the cluster is gone, as we can not delete the addons manifests
		// from the cluster anymore
		if err := r.removeCleanupFinalizer(ctx, log, addon); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to remove addon cleanup finalizer: %v", err)
		}

		return reconcile.Result{}, nil
	}

	log = r.log.With("cluster", cluster.Name, "addon", addon.Name)

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		kubermaticv1.ClusterConditionAddonControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, log, addon, cluster)
		},
	)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(addon, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError",
			"failed to reconcile Addon %q: %v", addon.Name, err)
	}
	if result == nil {
		// we check for this after the ClusterReconcileWrapper() call because otherwise the cluster would never reconcile since we always requeue
		if r.addonEnforceInterval == 0 { // addon enforce is disabled.
			result = &reconcile.Result{}
		} else {
			// All is well, requeue in addonEnforceInterval minutes. We do this to enforce default addons and prevent cluster admins from disabling them.
			result = &reconcile.Result{RequeueAfter: time.Duration(r.addonEnforceInterval) * time.Minute}
		}

	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	if cluster.Status.ExtendedHealth.Apiserver != kubermaticv1.HealthStatusUp {
		log.Debug("Skipping because the API server is not running")
		return nil, nil
	}

	reqeueAfter, err := r.ensureRequiredResourceTypesExist(ctx, log, addon, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to check if all required resources exist: %v", err)
	}
	if reqeueAfter != nil {
		return reqeueAfter, nil
	}

	// Openshift needs some time to create this, so avoid getting into the backoff
	// while the admin-kubeconfig secret doesn't exist yet
	if _, err := r.KubeconfigProvider.GetAdminKubeconfig(cluster); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("Kubeconfig wasn't found, trying again in 10 seconds")
			return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
		return nil, err
	}

	if addon.DeletionTimestamp != nil {
		if err := r.cleanupManifests(ctx, log, addon, cluster); err != nil {
			return nil, fmt.Errorf("failed to delete manifests from cluster: %v", err)
		}
		if err := r.removeCleanupFinalizer(ctx, log, addon); err != nil {
			return nil, fmt.Errorf("failed to ensure that the cleanup finalizer got removed from the addon: %v", err)
		}
		return nil, nil
	}

	// Reconciling
	if err := r.ensureIsInstalled(ctx, log, addon, cluster); err != nil {
		return nil, fmt.Errorf("failed to deploy the addon manifests into the cluster: %v", err)
	}
	if err := r.ensureFinalizerIsSet(ctx, addon); err != nil {
		return nil, fmt.Errorf("failed to ensure that the cleanup finalizer existis on the addon: %v", err)
	}
	return nil, nil
}

func (r *Reconciler) removeCleanupFinalizer(ctx context.Context, log *zap.SugaredLogger, addon *kubermaticv1.Addon) error {
	if kuberneteshelper.HasFinalizer(addon, cleanupFinalizerName) {
		oldAddon := addon.DeepCopy()
		kuberneteshelper.RemoveFinalizer(addon, cleanupFinalizerName)
		if err := r.Client.Patch(ctx, addon, ctrlruntimeclient.MergeFrom(oldAddon)); err != nil {
			return err
		}
		log.Debugw("Removed the cleanup finalizer", "finalizer", cleanupFinalizerName)
	}
	return nil
}

func (r *Reconciler) getAddonManifests(log *zap.SugaredLogger, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) ([]runtime.RawExtension, error) {
	addonDir := r.kubernetesAddonDir
	if cluster.IsOpenshift() {
		addonDir = r.openshiftAddonDir
	}
	clusterIP, err := resources.UserClusterDNSResolverIP(cluster)
	if err != nil {
		return nil, err
	}
	dnsResolverIP := clusterIP
	if r.nodeLocalDNSCacheEnabled {
		dnsResolverIP = machinecontroller.NodeLocalDNSCacheAddress
	}

	kubeconfig, err := r.KubeconfigProvider.GetAdminKubeconfig(cluster)
	if err != nil {
		return nil, err
	}

	credentials, err := resources.GetCredentials(resources.NewCredentialsData(context.Background(), cluster, r.Client))
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %v", err)
	}

	data := &addonutils.TemplateData{
		Variables:         make(map[string]interface{}),
		Cluster:           cluster,
		Credentials:       credentials,
		MajorMinorVersion: cluster.Spec.Version.MajorMinor(),
		Addon:             addon,
		Kubeconfig:        string(kubeconfig),
		DNSClusterIP:      clusterIP,
		DNSResolverIP:     dnsResolverIP,
		ClusterCIDR:       cluster.Spec.ClusterNetwork.Pods.CIDRBlocks[0],
	}

	// Add addon variables if available.
	if sub := r.addonVariables[addon.Spec.Name]; sub != nil {
		data.Variables = sub.(map[string]interface{})
	}

	if len(addon.Spec.Variables.Raw) > 0 {
		if err = json.Unmarshal(addon.Spec.Variables.Raw, &data.Variables); err != nil {
			return nil, err
		}
	}
	manifestPath := path.Join(addonDir, addon.Spec.Name)

	allManifests, err := addonutils.ParseFromFolder(log, r.overwriteRegistry, manifestPath, data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse addon templates in %s: %v", manifestPath, err)
	}

	return allManifests, nil
}

// combineManifests returns all manifests combined into a multi document yaml
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
// For this to happen we need to decode all yaml files to json, parse them, add the label and finally encode to yaml again
func (r *Reconciler) ensureAddonLabelOnManifests(addon *kubermaticv1.Addon, manifests []runtime.RawExtension) ([]*bytes.Buffer, error) {
	var rawManifests []*bytes.Buffer

	wantLabels := r.getAddonLabel(addon)
	for _, m := range manifests {
		parsedUnstructuredObj := &metav1unstructured.Unstructured{}
		if _, _, err := metav1unstructured.UnstructuredJSONScheme.Decode(m.Raw, nil, parsedUnstructuredObj); err != nil {
			return nil, fmt.Errorf("parsing unstructured failed: %v", err)
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
			return nil, fmt.Errorf("encoding json failed: %v", err)
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
	//Write combined Manifest to disk
	manifestFilename := path.Join("/tmp", fmt.Sprintf("cluster-%s-%s.yaml", cluster.Name, addon.Name))
	if err := ioutil.WriteFile(manifestFilename, manifest.Bytes(), 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write combined manifest to %s: %v", manifestFilename, err)
	}
	log.Debugw("Wrote combined manifest", "file", manifestFilename)

	return manifestFilename, getFileDeleteFinalizer(log, manifestFilename), nil
}

func (r *Reconciler) writeAdminKubeconfig(log *zap.SugaredLogger, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) (string, fileHandlingDone, error) {
	// Write kubeconfig to disk
	kubeconfig, err := r.KubeconfigProvider.GetAdminKubeconfig(cluster)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get admin kubeconfig for cluster %s: %v", cluster.Name, err)
	}
	kubeconfigFilename := path.Join("/tmp", fmt.Sprintf("cluster-%s-addon-%s-kubeconfig", cluster.Name, addon.Name))
	if err := ioutil.WriteFile(kubeconfigFilename, kubeconfig, 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write admin kubeconfig for cluster %s: %v", cluster.Name, err)
	}
	log.Debugw("Wrote admin kubeconfig", "file", kubeconfigFilename)

	return kubeconfigFilename, getFileDeleteFinalizer(log, kubeconfigFilename), nil
}

func (r *Reconciler) setupManifestInteraction(log *zap.SugaredLogger, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) (string, string, fileHandlingDone, error) {
	manifests, err := r.getAddonManifests(log, addon, cluster)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to get addon manifests: %v", err)
	}

	rawManifests, err := r.ensureAddonLabelOnManifests(addon, manifests)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to add the addon specific label to all addon resources: %v", err)
	}

	rawManifest := r.combineManifests(rawManifests)
	manifestFilename, manifestDone, err := r.writeCombinedManifest(log, rawManifest, addon, cluster)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to write all addon resources into a combined manifest file: %v", err)
	}

	kubeconfigFilename, kubeconfigDone, err := r.writeAdminKubeconfig(log, addon, cluster)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to write the admin kubeconfig to the local filesystem: %v", err)
	}

	done := func() {
		kubeconfigDone()
		manifestDone()
	}
	return kubeconfigFilename, manifestFilename, done, nil
}

func (r *Reconciler) getApplyCommand(ctx context.Context, kubeconfigFilename, manifestFilename string, selector fmt.Stringer, openshift bool) *exec.Cmd {
	// kubectl apply --prune -f manifest.yaml -l app=nginx
	cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfigFilename, "apply", "--prune", "-f", manifestFilename, "-l", selector.String())
	return cmd
}

func (r *Reconciler) ensureIsInstalled(ctx context.Context, log *zap.SugaredLogger, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) error {
	kubeconfigFilename, manifestFilename, done, err := r.setupManifestInteraction(log, addon, cluster)
	if err != nil {
		return err
	}
	defer done()

	d, err := ioutil.ReadFile(manifestFilename)
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
	cmd := r.getApplyCommand(ctx, kubeconfigFilename, manifestFilename, selector, cluster.IsOpenshift())
	cmdLog := log.With("cmd", strings.Join(cmd.Args, " "))

	cmdLog.Debug("Applying manifest...")
	out, err := cmd.CombinedOutput()
	cmdLog.Debugw("Finished executing command", "output", string(out))
	if err != nil {
		return fmt.Errorf("failed to execute '%s' for addon %s of cluster %s: %v\n%s", strings.Join(cmd.Args, " "), addon.Name, cluster.Name, err, string(out))
	}
	return err
}

func (r *Reconciler) ensureFinalizerIsSet(ctx context.Context, addon *kubermaticv1.Addon) error {
	if kuberneteshelper.HasFinalizer(addon, cleanupFinalizerName) {
		return nil
	}

	oldAddon := addon.DeepCopy()
	kuberneteshelper.AddFinalizer(addon, cleanupFinalizerName)
	return r.Client.Patch(ctx, addon, ctrlruntimeclient.MergeFrom(oldAddon))
}

func (r *Reconciler) cleanupManifests(ctx context.Context, log *zap.SugaredLogger, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) error {
	kubeconfigFilename, manifestFilename, done, err := r.setupManifestInteraction(log, addon, cluster)
	if err != nil {
		return err
	}
	defer done()

	cmd := deleteCommand(ctx, kubeconfigFilename, manifestFilename)
	cmdLog := log.With("cmd", strings.Join(cmd.Args, " "))

	cmdLog.Debug("Deleting resources...")
	out, err := cmd.CombinedOutput()
	cmdLog.Debugw("Finished executing command", "output", string(out))
	if err != nil {
		return fmt.Errorf("failed to execute '%s' for addon %s of cluster %s: %v\n%s", strings.Join(cmd.Args, " "), addon.Name, cluster.Name, err, string(out))
	}
	return nil
}

func (r *Reconciler) ensureRequiredResourceTypesExist(ctx context.Context, log *zap.SugaredLogger, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {

	if len(addon.Spec.RequiredResourceTypes) == 0 {
		// Avoid constructing a client we don't need and just return early
		return nil, nil
	}
	userClusterClient, err := r.KubeconfigProvider.GetClient(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get client for usercluster: %v", err)
	}

	for _, requiredResource := range addon.Spec.RequiredResourceTypes {
		unstructuedList := &metav1unstructured.UnstructuredList{}
		unstructuedList.SetAPIVersion(requiredResource.Group + "/" + requiredResource.Version)
		unstructuedList.SetKind(requiredResource.Kind)

		// We do not care about the result, just if the resource is served, so make sure we only
		// get as little as possible.
		listOpts := &ctrlruntimeclient.ListOptions{Limit: 1}
		if err := userClusterClient.List(ctx, unstructuedList, listOpts); err != nil {
			if _, ok := err.(*meta.NoKindMatchError); ok {
				// Try again later
				log.Infow("Required resource isn't served, trying again in 10 seconds", "resource", requiredResource.String())
				return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
			}
			return nil, fmt.Errorf("failed to check if type %q is served: %v", requiredResource.String(), err)
		}
	}

	return nil, nil
}

func deleteCommand(ctx context.Context, kubeconfigFilename, manifestFilename string) *exec.Cmd {
	return exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfigFilename, "delete", "-f", manifestFilename, "--ignore-not-found")
}
