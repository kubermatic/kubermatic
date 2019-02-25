package openshift

import (
	"context"
	"fmt"
	"time"

	openshiftresources "github.com/kubermatic/kubermatic/api/pkg/controller/openshift/resources"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"github.com/golang/glog"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
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
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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

func Add(mgr manager.Manager, numWorkers int, clusterPredicates predicate.Predicate) error {
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
			if cluster.Status.NamespaceName == a.Meta.GetNamespace() {
				return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: cluster.Name}}}
			}
		}
		return []reconcile.Request{}
	})}
	if err := c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, enqueueClusterForNamespacedObject, clusterPredicates); err != nil {
		return fmt.Errorf("failed to create  watch for ConfigMaps: %v", err)
	}
	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}}, enqueueClusterForNamespacedObject, clusterPredicates); err != nil {
		return fmt.Errorf("failed to create  watch for Secrets: %v", err)
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

	if err := r.configMaps(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to reconcile ConfigMaps: %v", err)
	}
	if err := r.deployments(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to reconcile Deployments: %v", err)
	}

	return nil, nil
}

func setNamespace(object metav1.Object, namespace string) {
	object.SetNamespace(namespace)
}

func (r *Reconciler) getAllSecretCreators() []openshiftresources.NamedSecretCreator {
	return []openshiftresources.NamedSecretCreator{openshiftresources.ServiceSignerCA}
}

func (r *Reconciler) secrets(ctx context.Context, osData *openshiftData) error {
	for _, namedSecretCreator := range r.getAllSecretCreators() {
		secretName, secretCreator := namedSecretCreator(ctx, osData)
		secret := &corev1.Secret{}
		if err := r.Client.Get(ctx, nn(osData.Cluster().Status.NamespaceName, secretName), secret); err != nil {
			if !kerrors.IsNotFound(err) {
				return fmt.Errorf("failed to get secret %s: %v", secretName, err)
			}
			secret, err := secretCreator(&corev1.Secret{})
			if err != nil {
				return fmt.Errorf("failed to get initial secret %s from creator: %v", secretName, err)
			}
			setNamespace(secret, osData.Cluster().Status.NamespaceName)
			if err := r.Create(ctx, secret); err != nil {
				return fmt.Errorf("failed to create initial secret %s: %v", secretName, err)
			}
		}
		generatedSecret, err := secretCreator(secret.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to get secret %s from creator: %v", secret.Name, err)
		}
		setNamespace(generatedSecret, osData.Cluster().Status.NamespaceName)
		if equal := apiequality.Semantic.DeepEqual(secret, generatedSecret); equal {
			continue
		}
		if err := r.Update(ctx, generatedSecret); err != nil {
			return fmt.Errorf("failed to update secret %s: %v", secretName, err)
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
		configMap := &corev1.ConfigMap{}
		if err := r.Client.Get(ctx, nn(osData.Cluster().Status.NamespaceName, configMapName), configMap); err != nil {
			if !kerrors.IsNotFound(err) {
				return fmt.Errorf("failed to get configMap %s: %v", configMapName, err)
			}
			configMap, err := configMapCreator(&corev1.ConfigMap{})
			if err != nil {
				return fmt.Errorf("failed to get initial configMap %s from creator: %v", configMapName, err)
			}
			setNamespace(configMap, osData.Cluster().Status.NamespaceName)
			if err := r.Create(ctx, configMap); err != nil {
				return fmt.Errorf("failed to create initial configmap %s: %v", configMapName, err)
			}
			continue
		}
		generatedConfigMap, err := configMapCreator(configMap.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to get configMap %s: %v", configMapName, err)
		}
		setNamespace(generatedConfigMap, osData.Cluster().Status.NamespaceName)
		if equal := apiequality.Semantic.DeepEqual(configMap, generatedConfigMap); equal {
			continue
		}
		if err := r.Update(ctx, generatedConfigMap); err != nil {
			return fmt.Errorf("failed to update configMap %s: %v", configMapName, err)
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
		deployment := &appsv1.Deployment{}
		if err := r.Client.Get(ctx, nn(osData.Cluster().Status.NamespaceName, deploymentName), deployment); err != nil {
			if !kerrors.IsNotFound(err) {
				return fmt.Errorf("failed to get deployment %s: %v", deploymentName, err)
			}
			deployment, err := deploymentCreator(&appsv1.Deployment{})
			if err != nil {
				return fmt.Errorf("failed to get initial deployment %s from creator: %v", deploymentName, err)
			}
			setNamespace(deployment, osData.Cluster().Status.NamespaceName)
			if err := r.Create(ctx, deployment); err != nil {
				return fmt.Errorf("failed to create initial deployment %s: %v", deploymentName, err)
			}
			continue
		}
		generatedDeployment, err := deploymentCreator(deployment.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to get deployment %s: %v", deploymentName, err)
		}
		setNamespace(generatedDeployment, osData.Cluster().Status.NamespaceName)
		if equal := apiequality.Semantic.DeepEqual(generatedDeployment, deployment); equal {
			continue
		}
		if err := r.Update(ctx, generatedDeployment); err != nil {
			return fmt.Errorf("failed to update deployment %s: %v", deploymentName, err)
		}
	}
	return nil
}

func (r *Reconciler) updateCluster(name string, modify func(*kubermaticv1.Cluster)) (updatedCluster *kubermaticv1.Cluster, err error) {
	err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		cluster := &kubermaticv1.Cluster{}
		if err := r.Get(context.Background(), types.NamespacedName{Name: name}, cluster); err != nil {
			return err
		}
		modify(cluster)
		err := r.Update(context.Background(), cluster)
		if err == nil {
			updatedCluster = cluster
		}
		return err
	})

	return updatedCluster, err
}

// A cheap helper because I am too lazy to type this everytime
func nn(namespace, name string) types.NamespacedName {
	return types.NamespacedName{Namespace: namespace, Name: name}
}
