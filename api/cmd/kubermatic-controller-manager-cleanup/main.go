package main

import (
	"flag"
	"strings"
	"sync"

	"github.com/golang/glog"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticKubernetesProvider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/util/hash"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type cleanupContext struct {
	kubeClient       kubernetes.Interface
	kubermaticClient kubermaticclientset.Interface
	config           *rest.Config
}

// ClusterTask represents a cleanup action, taking the current cluster for which the cleanup should be executed and the current context.
// In case of an error, the correspondent error will be returned, else nil.
type ClusterTask func(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error

// KeyTask represents a cleanup action, taking the current key for which the cleanup should be executed and the current context.
// In case of an error, the correspondent error will be returned, else nil.
type KeyTask func(key *kubermaticv1.UserSSHKey, ctx *cleanupContext) error

func main() {
	var kubeconfig, masterURL string

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.Parse()

	var err error
	ctx := cleanupContext{}
	ctx.config, err = clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}

	ctx.config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	ctx.config.APIPath = "/apis"
	ctx.kubeClient = kubernetes.NewForConfigOrDie(ctx.config)
	ctx.kubermaticClient = kubermaticclientset.NewForConfigOrDie(ctx.config)

	clusters, err := ctx.kubermaticClient.KubermaticV1().Clusters().List(metav1.ListOptions{})
	if err != nil {
		glog.Fatal(err)
	}

	w := sync.WaitGroup{}
	w.Add(len(clusters.Items))

	for i := range clusters.Items {
		go func(c *kubermaticv1.Cluster) {
			defer w.Done()
			cleanupCluster(c, &ctx)
		}(&clusters.Items[i])
	}
	w.Wait()

	keys, err := ctx.kubermaticClient.KubermaticV1().UserSSHKeies().List(metav1.ListOptions{})
	if err != nil {
		glog.Fatal(err)
	}

	w = sync.WaitGroup{}
	w.Add(len(keys.Items))

	for i := range keys.Items {
		go func(key *kubermaticv1.UserSSHKey) {
			defer w.Done()
			cleanupKey(key, &ctx)
		}(&keys.Items[i])
	}
	w.Wait()
}

func cleanupCluster(cluster *kubermaticv1.Cluster, ctx *cleanupContext) {
	glog.Infof("Cleaning up cluster %s", cluster.Name)

	tasks := []ClusterTask{
		cleanupPrometheus,
		cleanupAPIServer,
		cleanupControllerManager,
		cleanupETCD,
		cleanupKubeStateMetrics,
		cleanupMachineController,
		cleanupScheduler,
		removeDeprecatedFinalizers,
		migrateVersion,
		cleanupAddonManager,
		setVSphereInfraManagementUser,
		combineCACertAndKey,
		cleanupHeapsterAddon,
		migrateClusterUserLabel,
	}

	w := sync.WaitGroup{}
	w.Add(len(tasks))

	for _, task := range tasks {
		go func(t ClusterTask) {
			defer w.Done()
			err := t(cluster, ctx)

			if err != nil {
				glog.Error(err)
			}
		}(task)
	}

	w.Wait()
}

func cleanupKey(key *kubermaticv1.UserSSHKey, ctx *cleanupContext) {
	glog.Infof("Cleaning up SSHKey %s", key.Name)

	tasks := []KeyTask{
		migrateSSHKeyOwner,
	}

	w := sync.WaitGroup{}
	w.Add(len(tasks))

	for _, task := range tasks {
		go func(t KeyTask) {
			defer w.Done()
			err := t(key, ctx)

			if err != nil {
				glog.Error(err)
			}
		}(task)
	}

	w.Wait()
}

func deleteResourceIgnoreNonExistent(namespace string, group string, version string, kind string, name string, ctx *cleanupContext) error {
	client, err := rest.UnversionedRESTClientFor(ctx.config)
	if err != nil {
		return err
	}

	url := []string{"apis", group, version, "namespaces", namespace, strings.ToLower(kind), name}

	err = client.
		Delete().
		AbsPath(url...).
		Do().
		Error()

	if err != nil && k8serrors.IsNotFound(err) {
		glog.Infof("Skipping %s of kind %s in %s because it doesn't exist.", name, kind, namespace)
		return nil
	} else if err == nil {
		glog.Infof("Deleted %s of kind %s in %s.", name, kind, namespace)
	}

	return err
}

func cleanupPrometheus(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	ns := cluster.Status.NamespaceName
	return deleteResourceIgnoreNonExistent(ns, "monitoring.coreos.com", "v1", "prometheus", "prometheus", ctx)
}

func cleanupAPIServer(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	ns := cluster.Status.NamespaceName
	return deleteResourceIgnoreNonExistent(ns, "monitoring.coreos.com", "v1", "servicemonitors", "apiserver", ctx)
}

func cleanupControllerManager(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	ns := cluster.Status.NamespaceName
	return deleteResourceIgnoreNonExistent(ns, "monitoring.coreos.com", "v1", "servicemonitors", "controller-manager", ctx)
}

func cleanupETCD(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	ns := cluster.Status.NamespaceName
	return deleteResourceIgnoreNonExistent(ns, "monitoring.coreos.com", "v1", "servicemonitors", "etcd", ctx)
}

func cleanupKubeStateMetrics(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	ns := cluster.Status.NamespaceName
	return deleteResourceIgnoreNonExistent(ns, "monitoring.coreos.com", "v1", "servicemonitors", "kube-state-metrics", ctx)
}

func cleanupMachineController(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	ns := cluster.Status.NamespaceName
	return deleteResourceIgnoreNonExistent(ns, "monitoring.coreos.com", "v1", "servicemonitors", "machine-controller", ctx)
}

func cleanupScheduler(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	ns := cluster.Status.NamespaceName
	return deleteResourceIgnoreNonExistent(ns, "monitoring.coreos.com", "v1", "servicemonitors", "scheduler", ctx)
}

func cleanupAddonManager(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	ns := cluster.Status.NamespaceName

	policy := metav1.DeletePropagationForeground
	err := ctx.kubeClient.AppsV1().Deployments(ns).Delete("addon-manager", &metav1.DeleteOptions{PropagationPolicy: &policy})
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	selector := labels.SelectorFromSet(map[string]string{"app": "addon-manager"})
	err = ctx.kubeClient.AppsV1().ReplicaSets(ns).DeleteCollection(&metav1.DeleteOptions{PropagationPolicy: &policy}, metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	return nil
}

// We changed the finalizers in https://github.com/kubermatic/kubermatic/pull/1196
func removeDeprecatedFinalizers(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	finalizers := sets.NewString(cluster.Finalizers...)
	if finalizers.Has("kubermatic.io/delete-ns") || finalizers.Has("kubermatic.io/cleanup-cloud-provider") {
		finalizers.Delete("kubermatic.io/delete-ns")
		finalizers.Delete("kubermatic.io/cleanup-cloud-provider")
		cluster.Finalizers = finalizers.List()
		if _, err := ctx.kubermaticClient.KubermaticV1().Clusters().Update(cluster); err != nil {
			return err
		}
	}

	return nil
}

// We moved MasterVersion to Version
func migrateVersion(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	if cluster.Spec.Version == "" {
		cluster.Spec.Version = cluster.Spec.MasterVersion
		if _, err := ctx.kubermaticClient.KubermaticV1().Clusters().Update(cluster); err != nil {
			return err
		}
	}
	return nil
}

// We introduced the InfraManagementUser for Vsphere to allow using a dedicated user
// for everything except the cloud provider functionality
func setVSphereInfraManagementUser(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	if cluster.Spec.Cloud.VSphere != nil {
		if cluster.Spec.Cloud.VSphere.InfraManagementUser.Username == "" || cluster.Spec.Cloud.VSphere.InfraManagementUser.Password == "" {
			cluster.Spec.Cloud.VSphere.InfraManagementUser.Username = cluster.Spec.Cloud.VSphere.Username
			cluster.Spec.Cloud.VSphere.InfraManagementUser.Password = cluster.Spec.Cloud.VSphere.Password
			if _, err := ctx.kubermaticClient.KubermaticV1().Clusters().Update(cluster); err != nil {
				return err
			}
		}
	}
	return nil
}

// We combine the ca cert and key into one secret
func combineCACertAndKey(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	_, err := ctx.kubeClient.CoreV1().Secrets(cluster.Status.NamespaceName).Get(resources.CASecretName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			//Create the combined secret
			caKeySecret, err := ctx.kubeClient.CoreV1().Secrets(cluster.Status.NamespaceName).Get("ca-key", metav1.GetOptions{})
			if err != nil {
				// If no old secret can be found, we assume it does not need a migration
				if k8serrors.IsNotFound(err) {
					return nil
				}
				return err
			}
			//Create the combined secret
			caCertSecret, err := ctx.kubeClient.CoreV1().Secrets(cluster.Status.NamespaceName).Get("ca-cert", metav1.GetOptions{})
			if err != nil {
				// If no old secret can be found, we assume it does not need a migration
				if k8serrors.IsNotFound(err) {
					return nil
				}
				return err
			}
			caSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: resources.CASecretName,
				},
				Data: map[string][]byte{
					resources.CAKeySecretKey:  caKeySecret.Data[resources.CAKeySecretKey],
					resources.CACertSecretKey: caCertSecret.Data[resources.CACertSecretKey],
				},
			}
			_, err = ctx.kubeClient.CoreV1().Secrets(cluster.Status.NamespaceName).Create(caSecret)
			return err
		}
	}
	return nil
}

// We migrated from heapster to the metrics-server
func cleanupHeapsterAddon(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	ns := cluster.Status.NamespaceName
	return deleteResourceIgnoreNonExistent(ns, "kubermatic.k8s.io", "v1", "addons", "heapster", ctx)
}

// We now hash all user ID's to avoid breaking the label requirements
func migrateClusterUserLabel(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	oldID := cluster.Labels[kubermaticKubernetesProvider.UserLabelKey]
	if !strings.HasSuffix(oldID, hash.UserIDSuffix) {
		newID, err := hash.GetUserID(oldID)
		if err != nil {
			return err
		}

		// Set new ID
		cluster.Labels[kubermaticKubernetesProvider.UserLabelKey] = newID
		cluster.Labels[kubermaticKubernetesProvider.UserLabelKey+"_RAW"] = oldID
		if _, err := ctx.kubermaticClient.KubermaticV1().Clusters().Update(cluster); err != nil {
			return err
		}
	}
	return nil
}

// We now hash all user ID's to avoid breaking the label requirements
func migrateSSHKeyOwner(key *kubermaticv1.UserSSHKey, ctx *cleanupContext) error {
	oldID := key.Spec.Owner
	if !strings.HasSuffix(oldID, hash.UserIDSuffix) {
		newID, err := hash.GetUserID(oldID)
		if err != nil {
			return err
		}

		// Set new ID
		key.Spec.Owner = newID
		// Saving as label. Otherwise we would need to create a new field
		if key.Labels == nil {
			key.Labels = map[string]string{}
		}
		key.Labels[kubermaticKubernetesProvider.UserLabelKey+"_RAW"] = oldID
		if _, err := ctx.kubermaticClient.KubermaticV1().UserSSHKeies().Update(key); err != nil {
			return err
		}
	}
	return nil
}
