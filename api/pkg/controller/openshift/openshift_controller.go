package openshift

import (
	"context"
	"fmt"
	"time"

	openshiftresources "github.com/kubermatic/kubermatic/api/pkg/controller/openshift/resources"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"

	"github.com/golang/glog"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const ControllerName = "kubermatic_openshift_controller"

// Check if the Reconciler fullfills the interface
// at compile time
var _ reconcile.Reconciler = &Reconciler{}

type Reconciler struct {
	client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

func Add(mgr manager.Manager, numWorkers int, workerName string) error {
	clusterPredicates := workerlabel.Predicates(workerName)

	dynamicClient := mgr.GetClient()
	reconciler := &Reconciler{Client: dynamicClient,
		scheme:   mgr.GetScheme(),
		recorder: mgr.GetRecorder(ControllerName)}

	c, err := controller.New(ControllerName, mgr,
		controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return err
	}

	enqueueClusterForNamespacedObject := &handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
		clusterList := &kubermaticv1.ClusterList{}
		if err := dynamicClient.List(context.Background(), &client.ListOptions{}, clusterList); err != nil {
			// TODO: Is there a better way to handle errors that occur here?
			glog.Errorf("failed to list Clusters: %v", err)
		}
		for _, cluster := range clusterList.Items {
			// Predicates are used on the watched object itself, hence we can not
			// use them for anything other than the Cluster CR itself
			if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != workerName {
				continue
			}
			if cluster.Status.NamespaceName == a.Meta.GetNamespace() {
				return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: cluster.Name}}}
			}
		}
		return []reconcile.Request{}
	})}
	if err := c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, enqueueClusterForNamespacedObject); err != nil {
		return fmt.Errorf("failed to create watch for ConfigMaps: %v", err)
	}
	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}}, enqueueClusterForNamespacedObject); err != nil {
		return fmt.Errorf("failed to create watch for Secrets: %v", err)
	}
	if err := c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, enqueueClusterForNamespacedObject); err != nil {
		return fmt.Errorf("failed to create watch for Deployments: %v", err)
	}

	//TODO: Ensure only openshift clusters are handled via a predicate
	return c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}, clusterPredicates)
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Add a wrapping here so we can emit an event on error
	result, err := r.reconcile(ctx, cluster)
	if err != nil {
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError", "%v", err)
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	if cluster.Spec.Pause {
		glog.V(6).Infof("skipping paused cluster %s", cluster.Name)
		return nil, nil
	}

	if cluster.Annotations["kubermatic.io/openshift"] == "" {
		return nil, nil
	}

	glog.V(4).Infof("Reconciling cluster %s", cluster.Name)

	// Wait for namespace
	if cluster.Status.NamespaceName == "" {
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}
	ns := &corev1.Namespace{}
	if err := r.Get(ctx, types.NamespacedName{Name: cluster.Status.NamespaceName}, ns); err != nil {
		if !kerrors.IsNotFound(err) {
			return nil, err
		}
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	osData := &openshiftData{cluster: cluster, client: r.Client}

	if err := r.secrets(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to reconcile Secrets: %v", err)
	}

	if err := r.configMaps(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to reconcile ConfigMaps: %v", err)
	}
	if err := r.deployments(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to reconcile Deployments: %v", err)
	}

	return nil, nil
}

func (r *Reconciler) getAllSecretCreators(ctx context.Context, osData *openshiftData) []resources.NamedSecretCreatorGetter {
	return []resources.NamedSecretCreatorGetter{openshiftresources.ServiceSignerCA(),
		openshiftresources.GetLoopbackKubeconfigCreator(ctx, osData)}
}

func (r *Reconciler) secrets(ctx context.Context, osData *openshiftData) error {
	for _, namedSecretCreator := range r.getAllSecretCreators(ctx, osData) {
		secretName, secretCreator := namedSecretCreator()
		if err := resources.EnsureNamedObjectV2(ctx,
			nn(osData.Cluster().Status.NamespaceName, secretName), resources.SecretObjectWrapper(secretCreator), r.Client, &corev1.Secret{}); err != nil {
			return fmt.Errorf("failed to ensure Secret %s: %v", secretName, err)
		}
	}

	return nil
}

func (r *Reconciler) getAllConfigmapCreators() []openshiftresources.NamedConfigMapCreator {
	return []openshiftresources.NamedConfigMapCreator{openshiftresources.OpenshiftControlPlaneConfigMapCreator}
}

func (r *Reconciler) configMaps(ctx context.Context, osData *openshiftData) error {
	for _, namedConfigmapCreator := range r.getAllConfigmapCreators() {
		configMapName, configMapCreator := namedConfigmapCreator(ctx, osData)
		if err := resources.EnsureNamedObjectV2(ctx,
			nn(osData.Cluster().Status.NamespaceName, configMapName), resources.ConfigMapObjectWrapper(configMapCreator), r.Client, &corev1.ConfigMap{}); err != nil {
			return fmt.Errorf("failed to ensure ConfigMap %s: %v", configMapName, err)
		}
	}
	return nil
}

func (r *Reconciler) getAllDeploymentCreators() []openshiftresources.NamedDeploymentCreator {
	return []openshiftresources.NamedDeploymentCreator{openshiftresources.APIDeploymentCreator}
}

func (r *Reconciler) deployments(ctx context.Context, osData *openshiftData) error {
	for _, namedDeploymentCreator := range r.getAllDeploymentCreators() {
		deploymentName, deploymentCreator := namedDeploymentCreator(ctx, osData)
		if err := resources.EnsureNamedObjectV2(ctx,
			nn(osData.Cluster().Status.NamespaceName, deploymentName), resources.DeploymentObjectWrapper(deploymentCreator), r.Client, &appsv1.Deployment{}); err != nil {
			return fmt.Errorf("failed to ensure Deployment %s: %v", deploymentName, err)
		}
	}
	return nil
}

// A cheap helper because I am too lazy to type this everytime
func nn(namespace, name string) types.NamespacedName {
	return types.NamespacedName{Namespace: namespace, Name: name}
}
