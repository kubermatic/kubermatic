package addon

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1informers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions/kubermatic/v1"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
	corev1informer "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	addonLabelKey        = "kubermatic-addon"
	cleanupFinalizerName = "cleanup-manifests"
)

// Metrics contains metrics that this controller will collect and expose
type Metrics struct {
	Workers prometheus.Gauge
}

// NewMetrics creates a new Metrics
// with default values initialized, so metrics always show up.
func NewMetrics() *Metrics {
	cm := &Metrics{
		Workers: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "kubermatic",
			Subsystem: "update_controller",
			Name:      "workers",
			Help:      "The number of running Update controller workers",
		}),
	}

	cm.Workers.Set(0)
	return cm
}

// KubeconfigProvider provides functionality to get a clusters admin kubeconfig
type KubeconfigProvider interface {
	GetAdminKubeconfig(c *kubermaticv1.Cluster) ([]byte, error)
}

// Controller stores necessary components that are required to manage in-cluster Add-On's
type Controller struct {
	queue       workqueue.RateLimitingInterface
	metrics     *Metrics
	workerName  string
	addonDir    string
	registryURI string

	KubeconfigProvider KubeconfigProvider

	client        kubermaticclientset.Interface
	clusterLister kubermaticv1lister.ClusterLister
	clusterSynced cache.InformerSynced
	addonLister   kubermaticv1lister.AddonLister
	addonSynced   cache.InformerSynced
	secretsSynced cache.InformerSynced
}

// New creates a new Addon controller that is responsible for
// managing in-cluster addons
func New(
	metrics *Metrics,
	workerName string,
	addonDir string,
	overwriteRegistey string,
	KubeconfigProvider KubeconfigProvider,
	client kubermaticclientset.Interface,
	secretInformer corev1informer.SecretInformer,
	addonInformer kubermaticv1informers.AddonInformer,
	clusterInformer kubermaticv1informers.ClusterInformer) (*Controller, error) {

	c := &Controller{
		queue:              workqueue.NewNamedRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(1*time.Second, 5*time.Minute), "Addon"),
		metrics:            metrics,
		workerName:         workerName,
		addonDir:           addonDir,
		KubeconfigProvider: KubeconfigProvider,
		client:             client,
	}

	prometheus.MustRegister(metrics.Workers)

	if overwriteRegistey != "" {
		c.registryURI = parceRegistryURI(overwriteRegistey)
	}

	addonInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueue(obj.(*kubermaticv1.Addon))
		},
		UpdateFunc: func(old, cur interface{}) {
			c.enqueue(cur.(*kubermaticv1.Addon))
		},
		DeleteFunc: func(obj interface{}) {
			addon, ok := obj.(*kubermaticv1.Addon)
			if !ok {
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					utilruntime.HandleError(fmt.Errorf("couldn't get object from tombstone %#v", obj))
					return
				}
				addon, ok = tombstone.Obj.(*kubermaticv1.Addon)
				if !ok {
					utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not a Addon %#v", obj))
					return
				}
			}
			c.enqueue(addon)
		},
	})

	clusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueueClusterAddons(obj)
		},
		UpdateFunc: func(old, cur interface{}) {
			c.enqueueClusterAddons(cur)
		},
		DeleteFunc: func(obj interface{}) {
			c.enqueueClusterAddons(obj)
		},
	})
	c.addonLister = addonInformer.Lister()
	c.addonSynced = addonInformer.Informer().HasSynced
	c.clusterLister = clusterInformer.Lister()
	c.clusterSynced = clusterInformer.Informer().HasSynced
	c.secretsSynced = secretInformer.Informer().HasSynced

	return c, nil
}

func parceRegistryURI(uri string) string {
	return path.Clean(uri) + "/"
}

func (c *Controller) enqueueClusterAddons(i interface{}) {
	obj, ok := i.(metav1.Object)
	//Object might be a tombstone
	if !ok {
		tombstone, ok := i.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("couldn't get obj from tombstone %#v", obj))
			return
		}
		obj = tombstone.Obj.(metav1.Object)
	}

	addons, err := c.addonLister.List(labels.Everything())
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to list addons: %v", err))
		return
	}

	for _, addon := range addons {
		if addon.Spec.Cluster.UID == obj.GetUID() {
			c.enqueue(addon)
		}
	}
}

// Run starts the controller's worker routines. This method is blocking and ends when stopCh gets closed
func (c *Controller) Run(workerCount int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	glog.Infof("Starting Add-On controller with %d workers", workerCount)
	defer glog.Info("Shutting down Add-On controller")

	if !cache.WaitForCacheSync(stopCh, c.clusterSynced, c.addonSynced, c.secretsSynced) {
		utilruntime.HandleError(errors.New("unable to sync caches for Add-On controller"))
		return
	}

	for i := 0; i < workerCount; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	c.metrics.Workers.Set(float64(workerCount))
	<-stopCh
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

func (c *Controller) processNextItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	err := c.sync(key.(string))

	c.handleErr(err, key)
	return true
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < 5 {
		glog.V(0).Infof("Error syncing %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	utilruntime.HandleError(err)
	glog.V(0).Infof("Dropping %q out of the queue: %v", key, err)
}

func (c *Controller) enqueue(addon *kubermaticv1.Addon) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(addon)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for object %#v: %v", addon, err))
		return
	}

	c.queue.Add(key)
}

func (c *Controller) sync(key string) error {
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("failed to split namespace & name from key: %s: %v", key, err)
	}
	addonFromCache, err := c.addonLister.Addons(ns).Get(name)
	if err != nil {
		if kerrors.IsNotFound(err) {
			glog.V(2).Infof("addon '%s' in work queue no longer exists", key)
			return nil
		}
		return err
	}

	addon := addonFromCache.DeepCopy()
	clusterFromCache, err := c.clusterLister.Get(addon.Spec.Cluster.Name)
	if err != nil {
		if kerrors.IsNotFound(err) {
			//Cluster got deleted
			return c.cleanupManifests(addon, nil)
		}
		return fmt.Errorf("failed to get cluster %s: %v", addon.Spec.Cluster.Name, err)
	}
	cluster := clusterFromCache.DeepCopy()

	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != c.workerName {
		glog.V(8).Infof("skipping cluster %s due to different worker assigned to it", key)
		return nil
	}

	// Deletion
	if addon.DeletionTimestamp != nil {
		return c.cleanupManifests(addon, cluster)
	}

	//Reconciling
	if !cluster.Status.Health.Apiserver {
		return nil
	}
	if err := c.ensureIsInstalled(addon, cluster); err != nil {
		return err
	}
	if err := c.ensureFinalizerIsSet(addon); err != nil {
		return err
	}

	return err
}

type templateData struct {
	Addon             *kubermaticv1.Addon
	Cluster           *kubermaticv1.Cluster
	Variables         map[string]interface{}
	OverwriteRegistry string
	DNSClusterIP      string
	ClusterCIDR       string
}

func (c *Controller) getAddonManifests(addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) ([]*bytes.Buffer, error) {
	var allManifests []*bytes.Buffer

	manifestPath := path.Join(c.addonDir, addon.Spec.Name)
	infos, err := ioutil.ReadDir(manifestPath)
	if err != nil {
		return nil, err
	}

	clusterIP, err := getKubeDNSClusterIP(cluster.Spec.ClusterNetwork.Services.CIDRBlocks)
	if err != nil {
		return nil, err
	}

	data := &templateData{
		Variables:         make(map[string]interface{}),
		Cluster:           cluster,
		Addon:             addon,
		OverwriteRegistry: c.registryURI,
		DNSClusterIP:      clusterIP,
		ClusterCIDR:       cluster.Spec.ClusterNetwork.Pods.CIDRBlocks[0],
	}

	if len(addon.Spec.Variables.Raw) > 0 {
		if err = json.Unmarshal(addon.Spec.Variables.Raw, &data.Variables); err != nil {
			return nil, err
		}
	}

	for _, info := range infos {
		if info.IsDir() {
			glog.V(6).Infof("found directory in manifest path %s for %s/%s. Ignoring.", manifestPath, addon.Namespace, addon.Name)
			continue
		}

		filename := path.Join(manifestPath, info.Name())
		glog.V(6).Infof("Processing file %s for addon %s/%s", filename, addon.Namespace, addon.Name)

		fbytes, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}

		tplName := fmt.Sprintf("%s-%s", addon.Name, info.Name())
		tpl, err := template.New(tplName).Funcs(sprig.TxtFuncMap()).Parse(string(fbytes))
		if err != nil {
			return nil, err
		}

		bufferAll := bytes.NewBuffer([]byte{})
		if err := tpl.Execute(bufferAll, data); err != nil {
			return nil, err
		}

		reader := kyaml.NewDocumentDecoder(ioutil.NopCloser(bufferAll))
		size := bufferAll.Len()
		for {
			fullB := make([]byte, size)
			n, err := reader.Read(fullB)
			if err != nil {
				if err == io.ErrShortBuffer {
					size = n
					continue
				}
				if err == io.EOF {
					break
				}
				return nil, err
			}
			croppedB := fullB[:n]
			allManifests = append(allManifests, bytes.NewBuffer(croppedB))
		}
	}

	return allManifests, nil
}

func getKubeDNSClusterIP(blocks []string) (string, error) {
	if len(blocks) == 0 {
		return "", fmt.Errorf("empty Services.CIDRBlocks")
	}
	block := blocks[0]
	ip, _, err := net.ParseCIDR(block)
	if err != nil {
		return "", err
	}
	ip[len(ip)-1] = ip[len(ip)-1] + 10
	return ip.String(), nil
}

// combineManifests returns all manifests combined into a multi document yaml
func (c *Controller) combineManifests(manifests []*bytes.Buffer) *bytes.Buffer {
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
func (c *Controller) ensureAddonLabelOnManifests(addon *kubermaticv1.Addon, manifests []*bytes.Buffer) ([]*bytes.Buffer, error) {
	wantLabels := c.getAddonLabel(addon)
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

func (c *Controller) getAddonLabel(addon *kubermaticv1.Addon) map[string]string {
	return map[string]string{
		addonLabelKey: addon.Spec.Name,
	}
}

type fileHandlingDone func()

func getFileDeleteFinalizer(filename string) fileHandlingDone {
	return func() {
		if err := os.RemoveAll(filename); err != nil {
			glog.V(0).Infof("failed to remove file %s: %v", filename, err)
		}
	}
}

func (c *Controller) writeCombinedManifest(manifest *bytes.Buffer, addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) (string, fileHandlingDone, error) {
	//Write combined Manifest to disk
	manifestFilename := path.Join("/tmp", fmt.Sprintf("cluster-%s-%s.yaml", cluster.Name, addon.Name))
	if err := ioutil.WriteFile(manifestFilename, manifest.Bytes(), 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write combined manifest to %s: %v", manifestFilename, err)
	}
	glog.V(8).Infof("wrote combined manifest for addon %s/%s to %s\n%s", addon.Name, addon.Namespace, manifestFilename, manifest.String())

	return manifestFilename, getFileDeleteFinalizer(manifestFilename), nil
}

func (c *Controller) writeAdminKubeconfig(addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) (string, fileHandlingDone, error) {
	// Write kubeconfig to disk
	kubeconfig, err := c.KubeconfigProvider.GetAdminKubeconfig(cluster)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get admin kubeconfig for cluster %s: %v", cluster.Name, err)
	}
	kubeconfigFilename := path.Join("/tmp", fmt.Sprintf("cluster-%s-addon-%s-kubeconfig", cluster.Name, addon.Name))
	if err := ioutil.WriteFile(kubeconfigFilename, kubeconfig, 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write admin kubeconfig for cluster %s: %v", cluster.Name, err)
	}
	glog.V(8).Infof("wrote admin kubeconfig for cluster %s to %s", cluster.Name, kubeconfigFilename)

	return kubeconfigFilename, getFileDeleteFinalizer(kubeconfigFilename), nil
}

func (c *Controller) setupManifestInteraction(addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) (string, string, fileHandlingDone, error) {
	manifests, err := c.getAddonManifests(addon, cluster)
	if err != nil {
		return "", "", nil, err
	}

	manifests, err = c.ensureAddonLabelOnManifests(addon, manifests)
	if err != nil {
		return "", "", nil, err
	}

	manifest := c.combineManifests(manifests)
	manifestFilename, manifestDone, err := c.writeCombinedManifest(manifest, addon, cluster)
	if err != nil {
		return "", "", nil, err
	}

	kubeconfigFilename, kubeconfigDone, err := c.writeAdminKubeconfig(addon, cluster)
	if err != nil {
		return "", "", nil, err
	}

	done := func() {
		kubeconfigDone()
		manifestDone()
	}
	return kubeconfigFilename, manifestFilename, done, nil
}

func (c *Controller) getDeleteCommand(kubeconfigFilename, manifestFilename string) *exec.Cmd {
	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfigFilename, "delete", "-f", manifestFilename)
	return cmd
}

func (c *Controller) getApplyCommand(kubeconfigFilename, manifestFilename string, selector labels.Selector) *exec.Cmd {
	//kubectl apply --prune -f manifest.yaml -l app=nginx
	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfigFilename, "apply", "--prune", "-f", manifestFilename, "-l", selector.String())
	return cmd
}

func (c *Controller) ensureFinalizerIsSet(addon *kubermaticv1.Addon) error {
	finalizers := sets.NewString(addon.Finalizers...)
	if finalizers.Has(cleanupFinalizerName) {
		return nil
	}

	var err error
	addon.Finalizers = append(addon.Finalizers, cleanupFinalizerName)
	addon, err = c.client.KubermaticV1().Addons(addon.Namespace).Update(addon)
	return err
}

func (c *Controller) ensureIsInstalled(addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) error {
	kubeconfigFilename, manifestFilename, done, err := c.setupManifestInteraction(addon, cluster)
	if err != nil {
		return err
	}
	defer done()

	// We delete all resources with this label which are not in the combined manifest
	selector := labels.SelectorFromSet(c.getAddonLabel(addon))
	cmd := c.getApplyCommand(kubeconfigFilename, manifestFilename, selector)

	glog.V(6).Infof("applying addon %s to cluster %s: %s ...", addon.Name, cluster.Name, strings.Join(cmd.Args, " "))
	out, err := cmd.CombinedOutput()
	glog.V(6).Infof("executed '%s' for addon %s of cluster %s: \n%s", strings.Join(cmd.Args, " "), addon.Name, cluster.Name, string(out))
	return err
}

func (c *Controller) cleanupManifests(addon *kubermaticv1.Addon, cluster *kubermaticv1.Cluster) error {
	if cluster != nil && cluster.DeletionTimestamp == nil {
		kubeconfigFilename, manifestFilename, done, err := c.setupManifestInteraction(addon, cluster)
		if err != nil {
			return err
		}
		defer done()

		cmd := c.getDeleteCommand(kubeconfigFilename, manifestFilename)
		glog.V(6).Infof("deleting addon (%s) manifests from cluster %s: %s ...", addon.Name, cluster.Name, strings.Join(cmd.Args, " "))
		out, err := cmd.CombinedOutput()
		glog.V(6).Infof("executed '%s' for addon %s of cluster %s: \n%s", strings.Join(cmd.Args, " "), addon.Name, cluster.Name, string(out))
		return err
	}

	finalizers := sets.NewString(addon.Finalizers...)
	if !finalizers.Has(cleanupFinalizerName) {
		return nil
	}

	finalizers.Delete(cleanupFinalizerName)
	addon.Finalizers = finalizers.List()
	addon, err := c.client.KubermaticV1().Addons(addon.Namespace).Update(addon)
	return err
}
