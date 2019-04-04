package rbac

import (
	"strings"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
)

// Metrics contains metrics that this controller will collect and expose
type Metrics struct {
	Workers prometheus.Gauge
}

// NewMetrics creates RBACGeneratorControllerMetrics
// with default values initialized, so metrics always show up.
func NewMetrics() *Metrics {
	subsystem := "rbac_generator_controller"
	cm := &Metrics{
		Workers: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "workers",
			Help:      "The number of running RBACGenerator controller workers",
		}),
	}

	cm.Workers.Set(0)
	return cm
}

// ControllerAggregator type holds controllers for managing RBAC for projects and theirs resources
type ControllerAggregator struct {
	rbacProjectController  *projectController
	rbacResourceController *resourcesController

	metrics *Metrics
}

type projectResource struct {
	gvr         schema.GroupVersionResource
	kind        string
	destination string
	namespace   string

	// shouldEnqueue is a convenience function that is called right before
	// the object is added to the queue. This is your last chance to say "no"
	shouldEnqueue func(obj metav1.Object) bool
}

// New creates a new controller aggregator for managing RBAC for resources
func New(metrics *Metrics, allClusterProviders []*ClusterProvider) (*ControllerAggregator, error) {
	projectResources := []projectResource{
		{
			gvr: schema.GroupVersionResource{
				Group:    kubermaticv1.GroupName,
				Version:  kubermaticv1.GroupVersion,
				Resource: kubermaticv1.ClusterResourceName,
			},
			kind:        kubermaticv1.ClusterKindName,
			destination: destinationSeed,
		},

		{
			gvr: schema.GroupVersionResource{
				Group:    kubermaticv1.GroupName,
				Version:  kubermaticv1.GroupVersion,
				Resource: kubermaticv1.SSHKeyResourceName,
			},
			kind: kubermaticv1.SSHKeyKind,
		},

		{
			gvr: schema.GroupVersionResource{
				Group:    kubermaticv1.GroupName,
				Version:  kubermaticv1.GroupVersion,
				Resource: kubermaticv1.UserProjectBindingResourceName,
			},
			kind: kubermaticv1.UserProjectBindingKind,
		},

		{
			gvr: schema.GroupVersionResource{
				Group:    k8scorev1.GroupName,
				Version:  k8scorev1.SchemeGroupVersion.Version,
				Resource: "secrets",
			},
			kind:      "Secret",
			namespace: "kubermatic",
			shouldEnqueue: func(obj metav1.Object) bool {
				// do not reconcile secrets without "sa-token" prefix
				return strings.HasPrefix(obj.GetName(), "sa-token")
			},
		},

		{
			gvr: schema.GroupVersionResource{
				Group:    kubermaticv1.GroupName,
				Version:  kubermaticv1.GroupVersion,
				Resource: kubermaticv1.UserResourceName,
			},
			kind: kubermaticv1.UserKindName,
			shouldEnqueue: func(obj metav1.Object) bool {
				// do not reconcile resources without "serviceaccount" prefix
				return strings.HasPrefix(obj.GetName(), "serviceaccount")
			},
		},
	}

	prometheus.MustRegister(metrics.Workers)

	projectRBACCtrl, err := newProjectRBACController(metrics, allClusterProviders, projectResources)
	if err != nil {
		return nil, err
	}

	resourcesRBACCtrl, err := newResourcesController(metrics, allClusterProviders, projectResources)
	if err != nil {
		return nil, err
	}

	return &ControllerAggregator{
		rbacProjectController:  projectRBACCtrl,
		rbacResourceController: resourcesRBACCtrl,
		metrics:                metrics,
	}, nil
}

// Run starts the controller's worker routines. This method is blocking and ends when stopCh gets closed
func (a *ControllerAggregator) Run(workerCount int, stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	go a.rbacProjectController.run(workerCount, stopCh)
	go a.rbacResourceController.run(workerCount, stopCh)

	glog.Info("RBAC generator aggregator controller started")
	<-stopCh
}
