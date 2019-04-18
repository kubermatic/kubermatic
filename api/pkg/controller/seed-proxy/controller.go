package seedproxy

import (
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	healthCheckPeriod             = 5 * time.Second
	ControllerName                = "seed-proxy-controller"
	KubermaticNamespace           = "kubermatic"
	KubeconfigSecret              = "kubeconfig"
	DatacentersSecret             = "datacenters"
	OwnerLabel                    = "kubermatic.io/controller"
	OwnerLabelValue               = ControllerName
	ServiceAccountName            = "seed-proxy"
	ServiceAccountNamespace       = metav1.NamespaceSystem
	SeedPrometheusNamespace       = "monitoring"
	SeedPrometheusRoleName        = "seed-proxy-prometheus"
	SeedPrometheusRoleBindingName = "seed-proxy-prometheus"
)

// seedClusterConnectionProvider offers functions to retrieve clients for the given seed clusters
type seedClusterConnectionProvider interface {
	GetClient(*kubermaticv1.Cluster) (kubernetes.Interface, error)
}

// Add creates a new Monitoring controller that is responsible for
// operating the monitoring components for all managed user clusters
func Add(
	mgr manager.Manager,
	numWorkers int,
	// userClusterConnProvider seedClusterConnectionProvider,
	kubeconfig *clientcmdapi.Config,
	datacenters map[string]provider.DatacenterMeta,
) error {
	reconciler := &Reconciler{
		Client: mgr.GetClient(),
		// userClusterConnProvider: userClusterConnProvider,
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
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: "",
					Name:      "seed-proxy-reconcile",
				},
			},
		}
	})}

	type watcher struct {
		obj  runtime.Object
		pred predicate.Funcs
	}

	typesToWatch := []watcher{
		{obj: &appsv1.Deployment{}, pred: deploymentPredicate()},
		{obj: &corev1.Service{}, pred: servicePredicate()},
		{obj: &corev1.Secret{}, pred: secretsPredicate()},
	}

	for _, t := range typesToWatch {
		if err := c.Watch(&source.Kind{Type: t.obj}, eventHandler, t.pred); err != nil {
			return fmt.Errorf("failed to create watcher for %T: %v", t, err)
		}
	}

	return nil
}

func secretsPredicate() predicate.Funcs {
	return makePredicateFuncs(func(meta metav1.Object) bool {
		return meta.GetNamespace() == KubermaticNamespace && (meta.GetName() == KubeconfigSecret || meta.GetName() == DatacentersSecret)
	})
}

func deploymentPredicate() predicate.Funcs {
	return makePredicateFuncs(func(meta metav1.Object) bool {
		labels := meta.GetLabels()
		return meta.GetNamespace() == KubermaticNamespace && labels[OwnerLabel] == OwnerLabelValue
	})
}

func servicePredicate() predicate.Funcs {
	return makePredicateFuncs(func(meta metav1.Object) bool {
		labels := meta.GetLabels()
		return meta.GetNamespace() == KubermaticNamespace && labels[OwnerLabel] == OwnerLabelValue
	})
}

func makePredicateFuncs(pred func(metav1.Object) bool) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return pred(e.Meta)
		},

		UpdateFunc: func(e event.UpdateEvent) bool {
			return pred(e.MetaOld) || pred(e.MetaNew)
		},

		DeleteFunc: func(e event.DeleteEvent) bool {
			return pred(e.Meta)
		},

		GenericFunc: func(e event.GenericEvent) bool {
			return pred(e.Meta)
		},
	}
}
