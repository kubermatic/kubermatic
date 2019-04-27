package addon

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/sets"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
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
	workerName         string
	addonVariables     map[string]interface{}
	kubernetesAddonDir string
	openshiftAddonDir  string
	registryURI        string
	ctrlruntimeclient.Client
	recorder record.EventRecorder

	KubeconfigProvider KubeconfigProvider
}

// Add creates a new Addon controller that is responsible for
// managing in-cluster addons
func Add(
	mgr manager.Manager,
	numWorkers int,
	workerName string,
	addonCtxVariables map[string]interface{},
	kubernetesAddonDir string,
	openshiftAddonDir string,
	overwriteRegistey string,
	KubeconfigProvider KubeconfigProvider) error {

	client := mgr.GetClient()
	reconciler := &Reconciler{
		addonVariables:     addonCtxVariables,
		kubernetesAddonDir: kubernetesAddonDir,
		openshiftAddonDir:  openshiftAddonDir,
		KubeconfigProvider: KubeconfigProvider,
		Client:             client,
		workerName:         workerName,
		recorder:           mgr.GetRecorder(ControllerName),
	}

	if overwriteRegistey != "" {
		reconciler.registryURI = parseRegistryURI(overwriteRegistey)
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
		listOptions := &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}
		if err := client.List(context.Background(), listOptions, addonList); err != nil {
			glog.Errorf("failed to get addons for cluster %s: %v", cluster.Name, err)
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

	addon := &kubermaticv1.Addon{}
	if err := r.Get(ctx, request.NamespacedName, addon); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Add a wrapping here so we can emit an event on error
	err := r.reconcile(ctx, addon)
	if err != nil {
		glog.Errorf("Failed to reconcile addon %s: %v", addon.Name, err)
		r.recorder.Eventf(addon, corev1.EventTypeWarning, "ReconcilingError", "%v", err)
		reconcilingError := err
		//Get the cluster so we can report an event to it
		cluster := &kubermaticv1.Cluster{}
		if err := r.Get(ctx, types.NamespacedName{Name: addon.Spec.Cluster.Name}, cluster); err != nil {
			glog.Errorf("failed to get cluster for reporting error onto it: %v", err)
		} else {
			r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError",
				"failed to reconcile Addon %q: %v", addon.Name, reconcilingError)
		}
	}
	return reconcile.Result{}, err
}

func parseRegistryURI(uri string) string {
	return path.Clean(uri) + "/"
}

func (r *Reconciler) reconcile(ctx context.Context, addon *kubermaticv1.Addon) error {
	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: addon.Spec.Cluster.Name}, cluster); err != nil {
		// If its not a NotFound return it
		if !kerrors.IsNotFound(err) {
			return err
		}

		// Cluster does not exist - If the addon has the deletion timestamp - we shall delete it
		if addon.DeletionTimestamp != nil {
			if err := r.removeCleanupFinalizer(ctx, addon); err != nil {
				return fmt.Errorf("failed to ensure that the cleanup finalizer got removed from the addon: %v", err)
			}
		}
		return nil
	}

	if cluster.Spec.Pause {
		glog.V(4).Infof("skipping paused cluster %s", cluster.Name)
		return nil
	}

	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName {
		return nil
	}

	// When a cluster gets deleted - we can skip it - not worth the effort.
	// This could lead though to a potential leak of resources in case addons deploy LB's or PV's.
	// The correct way of handling it though should be a optional cleanup routine in the cluster controller, which will delete all PV's and LB's inside the cluster cluster.
	if cluster.DeletionTimestamp != nil {
		glog.V(4).Infof("cluster %s is already being deleted - no need to cleanup the manifests", cluster.Name)
		return nil
	}

	// When the apiserver is not healthy, we must skip it
	if !cluster.Status.Health.Apiserver {
		glog.V(4).Infof("API server of cluster %s is not running - not processing the addon", cluster.Name)
		return nil
	}

	// Addon got deleted - remove all manifests
	if addon.DeletionTimestamp != nil {
		if err := r.cleanupManifests(ctx, addon, cluster); err != nil {
			return fmt.Errorf("failed to delete manifests from cluster: %v", err)
		}
		if err := r.removeCleanupFinalizer(ctx, addon); err != nil {
			return fmt.Errorf("failed to ensure that the cleanup finalizer got removed from the addon: %v", err)
		}
		return nil
	}

	// Reconciling
	if err := r.ensureIsInstalled(ctx, addon, cluster); err != nil {
		return fmt.Errorf("failed to deploy the addon manifests into the cluster: %v", err)
	}
	if err := r.ensureFinalizerIsSet(ctx, addon); err != nil {
		return fmt.Errorf("failed to ensure that the cleanup finalizer existis on the addon: %v", err)
	}

	return nil
}

func (r *Reconciler) removeCleanupFinalizer(ctx context.Context, addon *kubermaticv1.Addon) error {
	finalizers := sets.NewString(addon.Finalizers...)
	if finalizers.Has(cleanupFinalizerName) {
		finalizers.Delete(cleanupFinalizerName)
		addon.Finalizers = finalizers.List()
		if err := r.Client.Update(ctx, addon); err != nil {
			return err
		}
		glog.V(2).Infof("Removed the cleanup finalizer from the addon %s/%s", addon.Namespace, addon.Name)
	}
	return nil
}

type templateData struct {
	Addon             *kubermaticv1.Addon
	Kubeconfig        string
	Cluster           *kubermaticv1.Cluster
	Variables         map[string]interface{}
	OverwriteRegistry string
	DNSClusterIP      string
	ClusterCIDR       string
}

func (r *Reconciler) getAddonManifests(addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) ([]*bytes.Buffer, error) {
	var allManifests []*bytes.Buffer

	addonDir := r.kubernetesAddonDir
	if isOpenshift(cluster) {
		addonDir = r.openshiftAddonDir
	}
	manifestPath := path.Join(addonDir, addon.Spec.Name)
	infos, err := ioutil.ReadDir(manifestPath)
	if err != nil {
		return nil, err
	}

	clusterIP, err := resources.UserClusterDNSResolverIP(cluster)
	if err != nil {
		return nil, err
	}

	kubeconfig, err := r.KubeconfigProvider.GetAdminKubeconfig(cluster)
	if err != nil {
		return nil, err
	}

	data := &templateData{
		Variables:         make(map[string]interface{}),
		Cluster:           cluster,
		Addon:             addon,
		Kubeconfig:        string(kubeconfig),
		OverwriteRegistry: r.registryURI,
		DNSClusterIP:      clusterIP,
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

	for _, info := range infos {
		if info.IsDir() {
			glog.V(4).Infof("found directory in manifest path %s for %s/%s. Ignoring.", manifestPath, addon.Namespace, addon.Name)
			continue
		}

		filename := path.Join(manifestPath, info.Name())
		glog.V(4).Infof("Processing file %s for addon %s/%s", filename, addon.Namespace, addon.Name)

		fbytes, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %v", filename, err)
		}

		tplName := fmt.Sprintf("%s-%s", addon.Name, info.Name())
		tpl, err := template.New(tplName).Funcs(sprig.TxtFuncMap()).Parse(string(fbytes))
		if err != nil {
			return nil, fmt.Errorf("failed to parse file %s: %v", filename, err)
		}

		bufferAll := bytes.NewBuffer([]byte{})
		if err := tpl.Execute(bufferAll, data); err != nil {
			return nil, fmt.Errorf("failed to execute templating on file %s: %v", filename, err)
		}

		sd := strings.TrimSpace(bufferAll.String())
		if len(sd) == 0 {
			glog.V(4).Infof("skipping %s/%s as its empty after parsing", cluster.Status.NamespaceName, addon.Name)
			continue
		}

		reader := kyaml.NewYAMLReader(bufio.NewReader(bufferAll))
		for {
			b, err := reader.Read()
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, fmt.Errorf("failed reading from YAML reader: %v", err)
			}
			if len(b) == 0 {
				break
			}
			allManifests = append(allManifests, bytes.NewBuffer(bytes.TrimSpace(b)))
		}
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
func (r *Reconciler) ensureAddonLabelOnManifests(addon *kubermaticv1.Addon, manifests []*bytes.Buffer) ([]*bytes.Buffer, error) {
	wantLabels := r.getAddonLabel(addon)
	for _, m := range manifests {
		decoder := kyaml.NewYAMLToJSONDecoder(m)
		raw := runtime.RawExtension{}
		if err := decoder.Decode(&raw); err != nil {
			return nil, err
		}

		parsedUnstructuredObj := &metav1unstructured.Unstructured{}
		if _, _, err := metav1unstructured.UnstructuredJSONScheme.Decode(raw.Raw, nil, parsedUnstructuredObj); err != nil {
			return nil, err
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
			return nil, err
		}

		yamlBytes, err := yaml.JSONToYAML(jsonBuffer.Bytes())
		if err != nil {
			return nil, err
		}

		m.Reset()
		if _, err := m.Write(yamlBytes); err != nil {
			return nil, err
		}
	}

	return manifests, nil
}

func (r *Reconciler) getAddonLabel(addon *kubermaticv1.Addon) map[string]string {
	return map[string]string{
		addonLabelKey: addon.Spec.Name,
	}
}

type fileHandlingDone func()

func getFileDeleteFinalizer(filename string) fileHandlingDone {
	return func() {
		if err := os.RemoveAll(filename); err != nil {
			glog.Errorf("failed to remove file %s: %v", filename, err)
		}
	}
}

func (r *Reconciler) writeCombinedManifest(manifest *bytes.Buffer, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) (string, fileHandlingDone, error) {
	//Write combined Manifest to disk
	manifestFilename := path.Join("/tmp", fmt.Sprintf("cluster-%s-%s.yaml", cluster.Name, addon.Name))
	if err := ioutil.WriteFile(manifestFilename, manifest.Bytes(), 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write combined manifest to %s: %v", manifestFilename, err)
	}
	glog.V(4).Infof("wrote combined manifest for addon %s/%s to %s\n%s", addon.Name, addon.Namespace, manifestFilename, manifest.String())

	return manifestFilename, getFileDeleteFinalizer(manifestFilename), nil
}

func (r *Reconciler) writeAdminKubeconfig(addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) (string, fileHandlingDone, error) {
	// Write kubeconfig to disk
	kubeconfig, err := r.KubeconfigProvider.GetAdminKubeconfig(cluster)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get admin kubeconfig for cluster %s: %v", cluster.Name, err)
	}
	kubeconfigFilename := path.Join("/tmp", fmt.Sprintf("cluster-%s-addon-%s-kubeconfig", cluster.Name, addon.Name))
	if err := ioutil.WriteFile(kubeconfigFilename, kubeconfig, 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write admin kubeconfig for cluster %s: %v", cluster.Name, err)
	}
	glog.V(4).Infof("wrote admin kubeconfig for cluster %s to %s", cluster.Name, kubeconfigFilename)

	return kubeconfigFilename, getFileDeleteFinalizer(kubeconfigFilename), nil
}

func (r *Reconciler) setupManifestInteraction(addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) (string, string, fileHandlingDone, error) {
	manifests, err := r.getAddonManifests(addon, cluster)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to get addon manifests: %v", err)
	}

	manifests, err = r.ensureAddonLabelOnManifests(addon, manifests)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to add the addon specific label to all addon resources: %v", err)
	}

	manifest := r.combineManifests(manifests)
	manifestFilename, manifestDone, err := r.writeCombinedManifest(manifest, addon, cluster)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to write all addon resources into a combined manifest file: %v", err)
	}

	kubeconfigFilename, kubeconfigDone, err := r.writeAdminKubeconfig(addon, cluster)
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
	binary := "kubectl"
	if openshift {
		binary = "oc"
	}
	cmd := exec.CommandContext(ctx, binary, "--kubeconfig", kubeconfigFilename, "delete", "-f", manifestFilename)
	return cmd
}

func (r *Reconciler) getApplyCommand(ctx context.Context, kubeconfigFilename, manifestFilename string, selector fmt.Stringer, openshift bool) *exec.Cmd {
	//kubectl apply --prune -f manifest.yaml -l app=nginx
	binary := "kubectl"
	if openshift {
		binary = "oc"
	}
	cmd := exec.CommandContext(ctx, binary, "--kubeconfig", kubeconfigFilename, "apply", "--prune", "-f", manifestFilename, "-l", selector.String())
	return cmd
}

func (r *Reconciler) ensureFinalizerIsSet(ctx context.Context, addon *kubermaticv1.Addon) error {
	finalizers := sets.NewString(addon.Finalizers...)
	if finalizers.Has(cleanupFinalizerName) {
		return nil
	}

	addon.Finalizers = append(addon.Finalizers, cleanupFinalizerName)
	return r.Client.Update(ctx, addon)
}

func (r *Reconciler) ensureIsInstalled(ctx context.Context, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) error {
	kubeconfigFilename, manifestFilename, done, err := r.setupManifestInteraction(addon, cluster)
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
		glog.V(4).Infof("skipping %s/%s as its empty after parsing", cluster.Status.NamespaceName, addon.Name)
		return nil
	}

	// We delete all resources with this label which are not in the combined manifest
	selector := labels.SelectorFromSet(r.getAddonLabel(addon))
	cmd := r.getApplyCommand(ctx, kubeconfigFilename, manifestFilename, selector, isOpenshift(cluster))

	glog.V(4).Infof("applying addon %s to cluster %s: %s ...", addon.Name, cluster.Name, strings.Join(cmd.Args, " "))
	out, err := cmd.CombinedOutput()
	glog.V(4).Infof("executed '%s' for addon %s of cluster %s: \n%s", strings.Join(cmd.Args, " "), addon.Name, cluster.Name, string(out))
	if err != nil {
		return fmt.Errorf("failed to execute '%s' for addon %s of cluster %s: %v\n%s", strings.Join(cmd.Args, " "), addon.Name, cluster.Name, err, string(out))
	}
	return err
}

func (r *Reconciler) cleanupManifests(ctx context.Context, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) error {
	kubeconfigFilename, manifestFilename, done, err := r.setupManifestInteraction(addon, cluster)
	if err != nil {
		return err
	}
	defer done()

	cmd := r.getDeleteCommand(ctx, kubeconfigFilename, manifestFilename, isOpenshift(cluster))
	glog.V(4).Infof("deleting addon (%s) manifests from cluster %s: %s ...", addon.Name, cluster.Name, strings.Join(cmd.Args, " "))
	out, err := cmd.CombinedOutput()
	glog.V(4).Infof("executed '%s' for addon %s of cluster %s: \n%s", strings.Join(cmd.Args, " "), addon.Name, cluster.Name, string(out))
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
