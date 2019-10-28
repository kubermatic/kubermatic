package addon

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/ghodss/yaml"

	"go.uber.org/zap"

	addonutils "github.com/kubermatic/kubermatic/api/pkg/addon"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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
}

// Reconciler stores necessary components that are required to manage in-cluster Add-On's
type Reconciler struct {
	log                     *zap.SugaredLogger
	workerName              string
	addonVariables          map[string]interface{}
	defaultKubernetesAddons sets.String
	defaultOpenshiftAddons  sets.String
	kubernetesAddonDir      string
	openshiftAddonDir       string
	overwriteRegistry       string
	ctrlruntimeclient.Client
	recorder record.EventRecorder

	KubeconfigProvider KubeconfigProvider
}

// Add creates a new Addon controller that is responsible for
// managing in-cluster addons
func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	addonCtxVariables map[string]interface{},
	defaultKubernetesAddons,
	defaultOpenshiftAddons sets.String,
	kubernetesAddonDir,
	openshiftAddonDir,
	overwriteRegistey string,
	kubeconfigProvider KubeconfigProvider,
) error {
	log = log.Named(ControllerName)
	client := mgr.GetClient()

	reconciler := &Reconciler{
		log:                     log,
		addonVariables:          addonCtxVariables,
		defaultKubernetesAddons: defaultKubernetesAddons,
		defaultOpenshiftAddons:  defaultOpenshiftAddons,
		kubernetesAddonDir:      kubernetesAddonDir,
		openshiftAddonDir:       openshiftAddonDir,
		KubeconfigProvider:      kubeconfigProvider,
		Client:                  client,
		workerName:              workerName,
		recorder:                mgr.GetEventRecorderFor(ControllerName),
		overwriteRegistry:       overwriteRegistey,
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

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, enqueueClusterAddons); err != nil {
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
	log = r.log.With("cluster", addon.Spec.Cluster.Name)

	// Add a wrapping here so we can emit an event on error
	result, err := r.reconcile(ctx, log, addon)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Eventf(addon, corev1.EventTypeWarning, "ReconcilingError", "%v", err)
		reconcilingError := err
		//Get the cluster so we can report an event to it
		cluster := &kubermaticv1.Cluster{}
		if err := r.Get(ctx, types.NamespacedName{Name: addon.Spec.Cluster.Name}, cluster); err != nil {
			log.Errorw("failed to get cluster for reporting error onto it", zap.Error(err))
		} else {
			r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError",
				"failed to reconcile Addon %q: %v", addon.Name, reconcilingError)
		}
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, addon *kubermaticv1.Addon) (*reconcile.Result, error) {
	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: addon.Spec.Cluster.Name}, cluster); err != nil {
		// If its not a NotFound return it
		if !kerrors.IsNotFound(err) {
			return nil, err
		}

		// Cluster does not exist - If the addon has the deletion timestamp - we shall delete it
		if addon.DeletionTimestamp != nil {
			if err := r.removeCleanupFinalizer(ctx, log, addon); err != nil {
				return nil, fmt.Errorf("failed to ensure that the cleanup finalizer got removed from the addon: %v", err)
			}
		}
		return nil, nil
	}

	if cluster.DeletionTimestamp != nil {
		log.Debug("Skipping addon because cluster is deleted")
		return nil, nil
	}

	if cluster.Spec.Pause {
		log.Debug("Skipping because the cluster is paused")
		return nil, nil
	}

	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName {
		log.Debug("Skipping because the cluster has a different worker name set")
		return nil, nil
	}

	// When a cluster gets deleted - we can skip it - not worth the effort.
	// This could lead though to a potential leak of resources in case addons deploy LB's or PV's.
	// The correct way of handling it though should be a optional cleanup routine in the cluster controller, which will delete all PV's and LB's inside the cluster cluster.
	if cluster.DeletionTimestamp != nil {
		log.Debug("Skipping because the cluster is being deleted")
		return nil, nil
	}

	if err := r.markDefaultAddons(ctx, log, addon, cluster); err != nil {
		return nil, fmt.Errorf("failed to ensure that the isDefault field is up to date in the addon: %v", err)
	}

	// When the apiserver is not healthy, we must skip it
	if kubermaticv1.HealthStatusDown == cluster.Status.ExtendedHealth.Apiserver {
		log.Debug("Skipping because the API server is not running")
		return nil, nil
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

	// Addon got deleted - remove all manifests
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

func (r *Reconciler) markDefaultAddons(ctx context.Context, log *zap.SugaredLogger,
	addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) error {
	var defaultAddons sets.String
	if cluster.Annotations["kubermatic.io/openshift"] != "" {
		defaultAddons = r.defaultOpenshiftAddons
	} else {
		defaultAddons = r.defaultKubernetesAddons
	}

	// Update only when the value was incorrect
	if isDefault := defaultAddons.Has(addon.Name); addon.Spec.IsDefault != isDefault {
		oldAddon := addon.DeepCopy()
		addon.Spec.IsDefault = isDefault
		if err := r.Client.Patch(ctx, addon, ctrlruntimeclient.MergeFrom(oldAddon)); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) removeCleanupFinalizer(ctx context.Context, log *zap.SugaredLogger, addon *kubermaticv1.Addon) error {
	if kuberneteshelper.HasFinalizer(addon, cleanupFinalizerName) {
		oldAddon := addon.DeepCopy()
		kuberneteshelper.RemoveFinalizer(addon, cleanupFinalizerName)
		if err := r.Client.Patch(ctx, addon, ctrlruntimeclient.MergeFrom(oldAddon)); err != nil {
			return err
		}
		log.Infow("Removed the cleanup finalizer", "finalizer", cleanupFinalizerName)
	}
	return nil
}

func (r *Reconciler) getAddonManifests(log *zap.SugaredLogger, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) ([]runtime.RawExtension, error) {
	addonDir := r.kubernetesAddonDir
	if isOpenshift(cluster) {
		addonDir = r.openshiftAddonDir
	}

	clusterIP, err := resources.UserClusterDNSResolverIP(cluster)
	if err != nil {
		return nil, err
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
		Variables:    make(map[string]interface{}),
		Cluster:      cluster,
		Credentials:  credentials,
		Addon:        addon,
		Kubeconfig:   string(kubeconfig),
		DNSClusterIP: clusterIP,
		ClusterCIDR:  cluster.Spec.ClusterNetwork.Pods.CIDRBlocks[0],
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
	log.Debugw("Wrote combined manifest", "file", manifestFilename, "content", manifest.String())

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

func (r *Reconciler) getDeleteCommand(ctx context.Context, kubeconfigFilename, manifestFilename string, openshift bool) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfigFilename, "delete", "-f", manifestFilename)
	return cmd
}

func (r *Reconciler) getApplyCommand(ctx context.Context, kubeconfigFilename, manifestFilename string, selector fmt.Stringer, openshift bool) *exec.Cmd {
	//kubectl apply --prune -f manifest.yaml -l app=nginx
	cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfigFilename, "apply", "--prune", "-f", manifestFilename, "-l", selector.String())
	return cmd
}

func (r *Reconciler) ensureFinalizerIsSet(ctx context.Context, addon *kubermaticv1.Addon) error {
	if kuberneteshelper.HasFinalizer(addon, cleanupFinalizerName) {
		return nil
	}

	oldAddon := addon.DeepCopy()
	kuberneteshelper.AddFinalizer(addon, cleanupFinalizerName)
	return r.Client.Patch(ctx, addon, ctrlruntimeclient.MergeFrom(oldAddon))
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
	cmd := r.getApplyCommand(ctx, kubeconfigFilename, manifestFilename, selector, isOpenshift(cluster))
	cmdLog := log.With("cmd", strings.Join(cmd.Args, " "))

	cmdLog.Debug("Applying manifest...")
	out, err := cmd.CombinedOutput()
	cmdLog.Debugw("Finished executing command", "output", string(out))
	if err != nil {
		return fmt.Errorf("failed to execute '%s' for addon %s of cluster %s: %v\n%s", strings.Join(cmd.Args, " "), addon.Name, cluster.Name, err, string(out))
	}
	return err
}

func (r *Reconciler) cleanupManifests(ctx context.Context, log *zap.SugaredLogger, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) error {
	kubeconfigFilename, manifestFilename, done, err := r.setupManifestInteraction(log, addon, cluster)
	if err != nil {
		return err
	}
	defer done()

	cmd := r.getDeleteCommand(ctx, kubeconfigFilename, manifestFilename, isOpenshift(cluster))
	cmdLog := log.With("cmd", strings.Join(cmd.Args, " "))

	cmdLog.Debug("Deleting resources...")
	out, err := cmd.CombinedOutput()
	cmdLog.Debugw("Finished executing command", "output", string(out))
	if err != nil {
		if wasKubectlDeleteSuccessful(string(out)) {
			return nil
		}
		return fmt.Errorf("failed to execute '%s' for addon %s of cluster %s: %v\n%s", strings.Join(cmd.Args, " "), addon.Name, cluster.Name, err, string(out))
	}
	return nil
}

func isOpenshift(c *kubermaticv1.Cluster) bool {
	return c.Annotations["kubermatic.io/openshift"] != ""
}

func wasKubectlDeleteSuccessful(out string) bool {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		if !isKubectlDeleteSuccessful(line) {
			return false
		}
	}

	return true
}

func isKubectlDeleteSuccessful(message string) bool {
	// Resource got successfully deleted. Something like: apiservice.apiregistration.k8s.io "v1beta1.metrics.k8s.io" deleted
	if strings.HasSuffix(message, "\" deleted") {
		return true
	}

	// Something like: Error from server (NotFound): error when deleting "/tmp/cluster-rwhxp9j5j-metrics-server.yaml": serviceaccounts "metrics-server" not found
	if strings.HasSuffix(message, "\" not found") {
		return true
	}

	fmt.Printf("fail: %v", message)
	return false
}
