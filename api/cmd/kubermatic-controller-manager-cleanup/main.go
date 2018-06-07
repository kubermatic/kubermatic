package main

import (
	"flag"
	"sync"

	"github.com/golang/glog"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type runOptions struct {
	kubeconfig string
	masterURL  string
}

type cleanupContext struct {
	kubeClient       kubernetes.Interface
	kubermaticClient kubermaticclientset.Interface
	config           *rest.Config
}

type resource struct {
	kind string
	name string
}

func main() {
	runOps := runOptions{}

	flag.StringVar(&runOps.kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&runOps.masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.Parse()

	ctx := cleanupContext{}

	var err error
	ctx.config, err = clientcmd.BuildConfigFromFlags(runOps.masterURL, runOps.kubeconfig)

	if err != nil {
		glog.Fatal(err)
	}

	ctx.config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	ctx.kubeClient = kubernetes.NewForConfigOrDie(ctx.config)
	ctx.kubermaticClient = kubermaticclientset.NewForConfigOrDie(ctx.config)

	namespaces, err := getClusterNamespaces(&ctx)

	if err != nil {
		glog.Fatal(err)
	}

	w := sync.WaitGroup{}
	w.Add(len(namespaces))

	for _, n := range namespaces {
		go func(ns string) {
			cleanupNamespace(ns, &ctx)
			w.Done()
		}(n)
	}

	w.Wait()
}

func cleanupNamespace(namespace string, ctx *cleanupContext) {
	glog.Infof("Cleaning up namespace %s", namespace)

	todos := [...]resource{
		resource{kind: "prometheus", name: "prometheus"},
		resource{kind: "servicemonitor", name: "apiserver"},
		resource{kind: "servicemonitor", name: "controller-manager"},
		resource{kind: "servicemonitor", name: "etcd"},
		resource{kind: "servicemonitor", name: "kube-state-metrics"},
		resource{kind: "servicemonitor", name: "machine-controller"},
		resource{kind: "servicemonitor", name: "scheduler"},
	}

	w := sync.WaitGroup{}
	w.Add(len(todos))

	for _, todo := range todos {
		go func(t resource) {
			client, err := rest.UnversionedRESTClientFor(ctx.config)

			if err != nil {
				glog.Fatal(err)
			}

			err = client.
				Delete().
				Namespace(namespace).
				Resource(t.kind).
				Name(t.name).
				Do().
				Error()

			if err != nil && k8serrors.IsNotFound(err) {
				glog.Infof("Skipping %s of kind %s in %s because it doesn't exist", t.name, t.kind, namespace)
			} else if err != nil {
				glog.Error(err)
			} else {
				glog.Infof("Deleted %s of kind %s in %s", t.name, t.kind, namespace)
			}

			w.Done()
		}(todo)
	}

	w.Wait()
}

func getClusterNamespaces(ctx *cleanupContext) ([]string, error) {
	clusters, err := ctx.kubermaticClient.KubermaticV1().Clusters().List(metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	namespaces := make([]string, len(clusters.Items), len(clusters.Items))

	for i, cluster := range clusters.Items {
		namespaces[i] = "cluster-" + cluster.Name
	}

	return namespaces, nil
}
