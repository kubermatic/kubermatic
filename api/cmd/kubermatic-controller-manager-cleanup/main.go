package main

import (
	"flag"
	"strings"
	"sync"

	"github.com/golang/glog"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
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

type Task func(cluster *kubermaticv1.Cluster, ctx *cleanupContext) error

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

	for _, cluster := range clusters.Items {
		go func(c *kubermaticv1.Cluster) {
			defer w.Done()
			cleanupCluster(c, &ctx)
		}(&cluster)
	}

	w.Wait()
}

func cleanupCluster(cluster *kubermaticv1.Cluster, ctx *cleanupContext) {
	glog.Infof("Cleaning up cluster %s", cluster.Name)

	tasks := []Task{
		cleanupPrometheus,
		cleanupAPIServer,
		cleanupControllerManager,
		cleanupETCD,
		cleanupKubeStateMetrics,
		cleanupMachineController,
		cleanupScheduler,
	}

	w := sync.WaitGroup{}
	w.Add(len(tasks))

	for _, task := range tasks {
		go func(t Task) {
			defer w.Done()
			err := t(cluster, ctx)

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
