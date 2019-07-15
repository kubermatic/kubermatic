package seedproxy

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// ControllerName is the name of this very controller.
	ControllerName = "seed-proxy-controller"

	// MasterTargetNamespace is the namespace inside the
	// master where the components will be created in.
	MasterTargetNamespace = "kubermatic"

	// MasterDeploymentName is the name used for deployments'
	// NameLabel value.
	MasterDeploymentName = "seed-proxy"

	// MasterServiceName is the name used for services' NameLabel value.
	MasterServiceName = "seed-proxy"

	// MasterGrafanaNamespace is the namespace inside the master
	// cluster where Grafana is installed and where the ConfigMap
	// should be created in.
	MasterGrafanaNamespace = "monitoring-master"

	// MasterGrafanaConfigMapName is the name used for the newly
	// created Grafana ConfigMap.
	MasterGrafanaConfigMapName = "grafana-seed-proxies"

	// SeedServiceAccountName is the name used for service accounts
	// inside the seed cluster.
	SeedServiceAccountName = "seed-proxy"

	// SeedServiceAccountNamespace is the namespace inside the seed
	// cluster where the service account will be created.
	SeedServiceAccountNamespace = metav1.NamespaceSystem

	// SeedPrometheusNamespace is the namespace inside the seed
	// cluster where Prometheus is installed.
	SeedPrometheusNamespace = "monitoring"

	// SeedPrometheusServiceName is the service that is provided by
	// Prometheus inside the seed cluster.
	SeedPrometheusServiceName = "prometheus-kubermatic"

	// SeedPrometheusServicePort is the port name that is provided by
	// Prometheus inside the seed cluster.
	SeedPrometheusServicePort = "web"

	// SeedPrometheusRoleName is the name inside the seed
	// used for the new role used for proxying to Prometheus.
	SeedPrometheusRoleName = "seed-proxy-prometheus"

	// SeedPrometheusRoleBindingName is the name inside the seed
	// used for the new role binding used for proxying to Prometheus.
	SeedPrometheusRoleBindingName = "seed-proxy-prometheus"

	// KubectlProxyPort is the port used by kubectl to provide the
	// proxy connection on. This is not the port on which any of the
	// target applications inside the seed (Prometheus, Grafana)
	// listen on.
	KubectlProxyPort = 8001

	// NameLabel is the recommended name for an identifying label.
	NameLabel = "app.kubernetes.io/name"

	// InstanceLabel is the recommended label for distinguishing
	// multiple elements of the same name. The label is used to store
	// the seed cluster name.
	InstanceLabel = "app.kubernetes.io/instance"

	// ManagedByLabel is the label used to identify the resources
	// created by this controller.
	ManagedByLabel = "app.kubernetes.io/managed-by"
)

// Add creates a new Seed-Proxy controller that is responsible for
// establishing ServiceAccounts in all seeds and setting up proxy
// pods to allow access to monitoring applications inside the seed
// clusters, like Prometheus and Grafana.
func Add(
	mgr manager.Manager,
	numWorkers int,
	kubeconfig *clientcmdapi.Config,
	datacenters map[string]*kubermaticv1.SeedDatacenter,
) error {
	reconciler := &Reconciler{
		Client:      mgr.GetClient(),
		recorder:    mgr.GetRecorder(ControllerName),
		kubeconfig:  kubeconfig,
		datacenters: datacenters,
	}

	ctrlOptions := controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	eventHandler := &handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
		requests := make([]reconcile.Request, 0)

		for name := range kubeconfig.Contexts {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: name,
				},
			})
		}

		return requests
	})}

	ownedByPred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return managedByController(e.Meta)
		},

		UpdateFunc: func(e event.UpdateEvent) bool {
			return managedByController(e.MetaOld) || managedByController(e.MetaNew)
		},

		DeleteFunc: func(e event.DeleteEvent) bool {
			return managedByController(e.Meta)
		},

		GenericFunc: func(e event.GenericEvent) bool {
			return managedByController(e.Meta)
		},
	}

	typesToWatch := []runtime.Object{
		&appsv1.Deployment{},
		&corev1.Service{},
		&corev1.Secret{},
		&corev1.ConfigMap{},
	}

	for _, t := range typesToWatch {
		if err := c.Watch(&source.Kind{Type: t}, eventHandler, ownedByPred); err != nil {
			return fmt.Errorf("failed to create watcher for %T: %v", t, err)
		}
	}

	return nil
}

func managedByController(meta metav1.Object) bool {
	labels := meta.GetLabels()
	return labels[ManagedByLabel] == ControllerName
}
