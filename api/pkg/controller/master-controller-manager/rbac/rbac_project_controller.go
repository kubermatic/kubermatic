package rbac

import (
	"context"

	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/client-go/util/workqueue"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	metricNamespace = "kubermatic"
	destinationSeed = "seed"
)

type projectController struct {
	projectQueue workqueue.RateLimitingInterface
	metrics      *Metrics

	projectLister kubermaticv1lister.ProjectLister

	seedClusterProviders  []*ClusterProvider
	masterClusterProvider *ClusterProvider

	projectResources []projectResource
	client           client.Client
	seedClientMap    map[string]client.Client
	ctx              context.Context
}

// newProjectRBACController creates a new controller that is responsible for
// managing RBAC roles for project's

// The controller will also set proper ownership chain through OwnerReferences
// so that whenever a project is deleted dependants object will be garbage collected.
func newProjectRBACController(metrics *Metrics, mgr manager.Manager, seedManagerMap map[string]manager.Manager, masterClusterProvider *ClusterProvider, seedClusterProviders []*ClusterProvider, resources []projectResource, workerPredicate predicate.Predicate) error {
	seedClientMap := make(map[string]client.Client)
	for k, v := range seedManagerMap {
		seedClientMap[k] = v.GetClient()
	}

	c := &projectController{
		projectQueue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "rbac_generator_for_project"),
		metrics:               metrics,
		projectResources:      resources,
		masterClusterProvider: masterClusterProvider,
		seedClusterProviders:  seedClusterProviders,
		client:                mgr.GetClient(),
		seedClientMap:         seedClientMap,
		ctx:                   context.TODO(),
	}

	// Create a new controller
	cc, err := controller.New("rbac_generator_for_project", mgr, controller.Options{Reconciler: c})
	if err != nil {
		return err
	}

	// Watch for changes to UserProjectBinding
	err = cc.Watch(&source.Kind{Type: &kubermaticv1.Project{}}, &handler.EnqueueRequestForObject{}, workerPredicate)
	if err != nil {
		return err
	}

	return nil
}

func (c *projectController) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	err := c.sync(req.NamespacedName)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
