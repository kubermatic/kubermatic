package seed

import (
	"fmt"
	"sync"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

type cleanupContext struct {
	kubeClient       kubernetes.Interface
	kubermaticClient kubermaticclientset.Interface
}

// ClusterTask represents a cleanup action, taking the current cluster for which the cleanup should be executed and the current context.
// In case of an error, the correspondent error will be returned, else nil.
type ClusterTask func(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error

// RunAll runs all migrations
func RunAll(config *rest.Config, workerName string) error {
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubeClient: %v", err)
	}
	kubermaticClient, err := kubermaticclientset.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubermatic client: %v", err)
	}

	ctx := &cleanupContext{
		kubeClient:       kubeClient,
		kubermaticClient: kubermaticClient,
	}

	if err := cleanupClusters(workerName, ctx); err != nil {
		return fmt.Errorf("failed to cleanup clusters: %v", err)
	}

	return nil
}

func cleanupClusters(workerName string, ctx *cleanupContext) error {
	selector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return err
	}
	options := metav1.ListOptions{
		LabelSelector: selector.String(),
	}
	clusters, err := ctx.kubermaticClient.KubermaticV1().Clusters().List(options)
	if err != nil {
		return fmt.Errorf("failed to list clusters: %v", err)
	}

	var errs []error
	errLock := &sync.Mutex{}
	w := sync.WaitGroup{}
	w.Add(len(clusters.Items))
	for i := range clusters.Items {
		go func(c *kubermaticv1.Cluster) {
			defer w.Done()

			if err := cleanupCluster(c, ctx); err != nil {
				err = fmt.Errorf("failed to cleanup cluster %q: %v", c.Name, err)
				errLock.Lock()
				defer errLock.Unlock()
				errs = append(errs, err)
			}
		}(&clusters.Items[i])
	}
	w.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("%v", errs)
	}

	return nil
}

func cleanupCluster(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	if cluster.Status.NamespaceName == "" {
		klog.Infof("Skipping cleanup of cluster %q because its namespace is unset", cluster.Name)
		return nil
	}

	klog.Infof("Cleaning up cluster %s", cluster.Name)

	tasks := []ClusterTask{
		setExposeStrategyIfEmpty,
		setProxyModeIfEmpty,
		cleanupDashboardAddon,
		migrateClusterUserLabel,
	}

	w := sync.WaitGroup{}
	w.Add(len(tasks))
	var errs []error
	errLock := &sync.Mutex{}

	for _, task := range tasks {
		go func(t ClusterTask) {
			defer w.Done()
			err := t(cluster, ctx)

			if err != nil {
				klog.Error(err)
				errLock.Lock()
				defer errLock.Unlock()
				errs = append(errs, err)
			}
		}(task)
	}

	w.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("%v", errs)
	}

	return nil
}

// We started to offer the option to expose the cluster via a LoadBalancer. We need to track
// the expose strategy that is being used for a cluster. If there is none set, we default to NodePort
// as that was initially the only option
func setExposeStrategyIfEmpty(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	if cluster.Spec.ExposeStrategy == "" {
		cluster.Spec.ExposeStrategy = corev1.ServiceTypeNodePort
		updatedCluster, err := ctx.kubermaticClient.KubermaticV1().Clusters().Update(cluster)
		if err != nil {
			return fmt.Errorf("failed to default exposeStrategy to NodePort for cluster %q: %v", cluster.Name, err)
		}
		*cluster = *updatedCluster
	}
	return nil
}

// We started to offer a config option for setting the kube-proxy mode.
// If there is none set, we default to iptables as that was initially the
// one being used.
func setProxyModeIfEmpty(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	if cluster.Spec.ClusterNetwork.ProxyMode == "" {
		cluster.Spec.ClusterNetwork.ProxyMode = resources.IPTablesProxyMode
		updatedCluster, err := ctx.kubermaticClient.KubermaticV1().Clusters().Update(cluster)
		if err != nil {
			return fmt.Errorf("failed to default proxyMode to iptables for cluster %q: %v", cluster.Name, err)
		}
		*cluster = *updatedCluster
	}
	return nil
}

func cleanupDashboardAddon(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	if err := ctx.kubermaticClient.KubermaticV1().Addons(cluster.Status.NamespaceName).Delete("dashboard", &metav1.DeleteOptions{}); err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func migrateClusterUserLabel(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error {
	// If there is not label - nothing to migrate
	if cluster.Labels == nil {
		return nil
	}
	newLabels := map[string]string{}
	userLabelSet := sets.NewString("user", "user_RAW")
	for key, value := range cluster.Labels {
		if userLabelSet.Has(key) {
			continue
		}
		newLabels[key] = value
	}
	if len(newLabels) != len(cluster.Labels) {
		cluster.Labels = newLabels
		updatedCluster, err := ctx.kubermaticClient.KubermaticV1().Clusters().Update(cluster)
		if err != nil {
			return err
		}
		*cluster = *updatedCluster
	}

	return nil
}
