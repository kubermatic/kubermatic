/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resources

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"sync"

	"github.com/Masterminds/semver/v3"
	"go.uber.org/zap"

	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "user-cluster-controller"
)

type UserClusterMLA struct {
	Logging                      bool
	Monitoring                   bool
	MLAGatewayURL                string
	PrometheusScrapeConfigPrefix string
}

// Add creates a new user cluster controller.
func Add(
	mgr manager.Manager,
	seedMgr manager.Manager,
	version string,
	namespace string,
	cloudProviderName string,
	clusterURL *url.URL,
	clusterIsPaused userclustercontrollermanager.IsPausedChecker,
	overwriteRegistry string,
	openvpnServerPort uint32,
	kasSecurePort uint32,
	tunnelingAgentIP net.IP,
	registerReconciledCheck func(name string, check healthz.Checker) error,
	dnsClusterIP string,
	nodeLocalDNSCache bool,
	opaIntegration bool,
	opaEnableMutation bool,
	versions kubermatic.Versions,
	userSSHKeyAgent bool,
	opaWebhookTimeout int,
	caBundle resources.CABundle,
	userClusterMLA UserClusterMLA,
	clusterName string,
	konnectivity bool,
	ccmMigration bool,
	ccmMigrationCompleted bool,
	log *zap.SugaredLogger) error {
	r := &reconciler{
		version:               version,
		rLock:                 &sync.Mutex{},
		namespace:             namespace,
		clusterURL:            clusterURL,
		clusterIsPaused:       clusterIsPaused,
		overwriteRegistryFunc: registry.GetOverwriteFunc(overwriteRegistry),
		openvpnServerPort:     openvpnServerPort,
		kasSecurePort:         kasSecurePort,
		tunnelingAgentIP:      tunnelingAgentIP,
		log:                   log,
		dnsClusterIP:          dnsClusterIP,
		nodeLocalDNSCache:     nodeLocalDNSCache,
		opaIntegration:        opaIntegration,
		opaEnableMutation:     opaEnableMutation,
		opaWebhookTimeout:     opaWebhookTimeout,
		userSSHKeyAgent:       userSSHKeyAgent,
		versions:              versions,
		caBundle:              caBundle,
		userClusterMLA:        userClusterMLA,
		cloudProvider:         kubermaticv1.ProviderType(cloudProviderName),
		clusterName:           clusterName,
		isKonnectivityEnabled: konnectivity,
		ccmMigration:          ccmMigration,
		ccmMigrationCompleted: ccmMigrationCompleted,
	}

	var err error
	if r.clusterSemVer, err = semver.NewVersion(r.version); err != nil {
		return err
	}
	r.Client = mgr.GetClient()
	r.seedClient = seedMgr.GetClient()
	r.cache = mgr.GetCache()

	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	mapFn := handler.EnqueueRequestsFromMapFunc(func(o ctrlruntimeclient.Object) []reconcile.Request {
		log.Debugw("Controller got triggered",
			"type", fmt.Sprintf("%T", o),
			"name", o.GetName(),
			"namespace", o.GetNamespace())

		return []reconcile.Request{
			{NamespacedName: types.NamespacedName{
				// There is no "parent object" like e.G. a cluster that can be used to reconcile, we just have a random set of resources
				// we reconcile one after another. To ensure we always have only one reconcile running at a time, we
				// use a static string as identifier
				Name:      "identifier",
				Namespace: "",
			}}}
	})

	typesToWatch := []ctrlruntimeclient.Object{
		&apiregistrationv1.APIService{},
		&corev1.ServiceAccount{},
		&corev1.Service{},
		&corev1.ConfigMap{},
		&corev1.Secret{},
		&rbacv1.Role{},
		&rbacv1.RoleBinding{},
		&rbacv1.ClusterRole{},
		&rbacv1.ClusterRoleBinding{},
		&admissionregistrationv1.MutatingWebhookConfiguration{},
		&admissionregistrationv1.ValidatingWebhookConfiguration{},
		&apiextensionsv1.CustomResourceDefinition{},
		&appsv1.Deployment{},
		&v1beta1.PodDisruptionBudget{},
	}

	// Avoid getting triggered by the leader lease AKA: If the annotation exists AND changed on
	// UPDATE, do not reconcile
	predicateIgnoreLeaderLeaseRenew := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectOld.GetAnnotations()[resourcelock.LeaderElectionRecordAnnotationKey] == "" {
				return true
			}
			if e.ObjectOld.GetAnnotations()[resourcelock.LeaderElectionRecordAnnotationKey] ==
				e.ObjectNew.GetAnnotations()[resourcelock.LeaderElectionRecordAnnotationKey] {
				return true
			}
			return false
		},
	}
	for _, t := range typesToWatch {
		if err := c.Watch(&source.Kind{Type: t}, mapFn, predicateIgnoreLeaderLeaseRenew); err != nil {
			return fmt.Errorf("failed to create watch for %T: %v", t, err)
		}
	}

	seedTypesToWatch := []ctrlruntimeclient.Object{
		&corev1.Secret{},
		&corev1.ConfigMap{},
	}
	for _, t := range seedTypesToWatch {
		seedWatch := &source.Kind{Type: t}
		if err := seedWatch.InjectCache(seedMgr.GetCache()); err != nil {
			return fmt.Errorf("failed to inject cache in seed cluster watch for %T: %v", t, err)
		}
		if err := c.Watch(seedWatch, mapFn); err != nil {
			return fmt.Errorf("failed to watch %T in seed: %v", t, err)
		}
	}

	// Watch cluster if user cluster MLA is enabled so that controller can get resource requirements for user cluster MLA components.
	if r.userClusterMLA.Monitoring || r.userClusterMLA.Logging {
		seedWatch := &source.Kind{Type: &kubermaticv1.Cluster{}}
		if err := seedWatch.InjectCache(seedMgr.GetCache()); err != nil {
			return fmt.Errorf("failed to inject cache in seed cluster watch for cluster: %w", err)
		}
		clusterPredicate := predicate.Funcs{
			// For Update event, only trigger reconciliation when Resource Requirements change.
			UpdateFunc: func(event event.UpdateEvent) bool {
				oldCluster := event.ObjectOld.(*kubermaticv1.Cluster)
				newCluster := event.ObjectNew.(*kubermaticv1.Cluster)
				oldResourceRequirements := oldCluster.GetUserClusterMLAResourceRequirements()
				newResourceRequirements := newCluster.GetUserClusterMLAResourceRequirements()
				return !reflect.DeepEqual(oldResourceRequirements, newResourceRequirements)
			},
		}
		if err := c.Watch(seedWatch, mapFn, clusterPredicate); err != nil {
			return fmt.Errorf("failed to watch cluster in seed: %w", err)
		}
	}

	// A very simple but limited way to express the first successful reconciling to the seed cluster
	return registerReconciledCheck(fmt.Sprintf("%s-%s", controllerName, "reconciled_successfully_once"), func(_ *http.Request) error {
		r.rLock.Lock()
		defer r.rLock.Unlock()

		if !r.reconciledSuccessfullyOnce {
			return errors.New("no successful reconciliation so far")
		}
		return nil
	})
}

// reconcileUserCluster reconciles objects in the user cluster
type reconciler struct {
	ctrlruntimeclient.Client
	seedClient            ctrlruntimeclient.Client
	version               string
	clusterSemVer         *semver.Version
	cache                 cache.Cache
	namespace             string
	clusterURL            *url.URL
	clusterIsPaused       userclustercontrollermanager.IsPausedChecker
	overwriteRegistryFunc registry.WithOverwriteFunc
	openvpnServerPort     uint32
	kasSecurePort         uint32
	tunnelingAgentIP      net.IP
	dnsClusterIP          string
	nodeLocalDNSCache     bool
	opaIntegration        bool
	opaEnableMutation     bool
	opaWebhookTimeout     int
	userSSHKeyAgent       bool
	versions              kubermatic.Versions
	caBundle              resources.CABundle
	userClusterMLA        UserClusterMLA
	cloudProvider         kubermaticv1.ProviderType
	clusterName           string
	isKonnectivityEnabled bool
	ccmMigration          bool
	ccmMigrationCompleted bool

	rLock                      *sync.Mutex
	reconciledSuccessfullyOnce bool

	log *zap.SugaredLogger
}

// Reconcile makes changes in response to objects in the user cluster.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		r.log.Debug("Cluster is paused, not reconciling.")
		return reconcile.Result{}, nil
	}

	if err := r.reconcile(ctx); err != nil {
		r.log.Errorw("Reconciling failed", zap.Error(err))
		return reconcile.Result{}, err
	}

	r.rLock.Lock()
	defer r.rLock.Unlock()
	r.reconciledSuccessfullyOnce = true
	return reconcile.Result{}, nil
}

func (r *reconciler) caCert(ctx context.Context) (*triple.KeyPair, error) {
	return resources.GetClusterRootCA(ctx, r.namespace, r.seedClient)
}

func (r *reconciler) openVPNCA(ctx context.Context) (*resources.ECDSAKeyPair, error) {
	return resources.GetOpenVPNCA(ctx, r.namespace, r.seedClient)
}

func (r *reconciler) mlaGatewayCA(ctx context.Context) (*resources.ECDSAKeyPair, error) {
	return resources.GetMLAGatewayCA(ctx, r.namespace, r.seedClient)
}

func (r *reconciler) userSSHKeys(ctx context.Context) (map[string][]byte, error) {
	secret := &corev1.Secret{}
	if err := r.seedClient.Get(
		ctx,
		types.NamespacedName{Namespace: r.namespace, Name: resources.UserSSHKeys},
		secret,
	); err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return secret.Data, nil
}

func (r *reconciler) cloudConfig(ctx context.Context, cloudConfigConfigmapName string) ([]byte, error) {
	configmap := &corev1.ConfigMap{}
	name := types.NamespacedName{Namespace: r.namespace, Name: cloudConfigConfigmapName}
	if err := r.seedClient.Get(ctx, name, configmap); err != nil {
		return nil, fmt.Errorf("failed to get cloud-config: %v", err)
	}
	value, exists := configmap.Data[resources.CloudConfigConfigMapKey]
	if !exists {
		return nil, fmt.Errorf("cloud-config configmap contains no data for key %s", resources.CloudConfigConfigMapKey)
	}
	return []byte(value), nil
}

func (r *reconciler) mlaReconcileData(ctx context.Context) (monitoring, logging *corev1.ResourceRequirements, monitoringReplicas *int32, err error) {
	cluster := &kubermaticv1.Cluster{}
	if err = r.seedClient.Get(ctx, types.NamespacedName{
		Name: r.clusterName,
	}, cluster); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get cluster: %w", err)
	}
	return cluster.Spec.MLA.MonitoringResources, cluster.Spec.MLA.LoggingResources, cluster.Spec.MLA.MonitoringReplicas, nil
}
