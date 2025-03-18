//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0")
                     Copyright © 2021 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package kyverno

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	admissionresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/admission-controller"
	backgroundresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/background-controller"
	cleanupresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/cleanup-controller"
	reportsresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/reports-controller"
	userclusterresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/user-cluster"
	userclusteradmissionresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/user-cluster/admission-controller"
	userclusterbackgroundresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/user-cluster/background-controller"
	userclustercleanupresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/user-cluster/cleanup-controller"
	userclusterreportsresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/user-cluster/reports-controller"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kkp-kyverno-controller"
)

// UserClusterClientProvider provides functionality to get a user cluster client.
type UserClusterClientProvider interface {
	GetClient(ctx context.Context, c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

type reconciler struct {
	ctrlruntimeclient.Client

	workerName                    string
	recorder                      record.EventRecorder
	log                           *zap.SugaredLogger
	userClusterConnectionProvider UserClusterClientProvider
	versions                      kubermatic.Versions
}

func Add(mgr manager.Manager, log *zap.SugaredLogger, numWorkers int, workerName string, userClusterConnectionProvider UserClusterClientProvider, versions kubermatic.Versions) error {
	reconciler := &reconciler{
		Client:                        mgr.GetClient(),
		workerName:                    workerName,
		recorder:                      mgr.GetEventRecorderFor(ControllerName),
		log:                           log.Named(ControllerName),
		userClusterConnectionProvider: userClusterConnectionProvider,
		versions:                      versions,
	}

	clusterIsAlive := predicateutil.Factory(func(o ctrlruntimeclient.Object) bool {
		cluster := o.(*kubermaticv1.Cluster)
		// Only watch clusters that are in a state where they can be reconciled.
		return !cluster.Spec.Pause && cluster.Status.NamespaceName != ""
	})

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{}, builder.WithPredicates(workerlabel.Predicate(workerName), clusterIsAlive)).
		Build(reconciler)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.Name)
	log.Debug("Reconciling")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionKyvernoControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, cluster)
		},
	)

	if result == nil || err != nil {
		result = &reconcile.Result{}
	}

	if err != nil {
		r.log.Errorw("Reconciling failed", zap.Error(err))
	}

	return *result, err
}

func (r *reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// Create/update required resources in seed cluster namespace
	if err := r.ensureKyvernoResources(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to ensure Kyverno resources: %w", err)
	}

	// Create/update required resources in user cluster
	if err := r.ensureUserClusterResources(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to ensure Kyverno user cluster resources: %w", err)
	}

	return nil, nil
}

func (r *reconciler) ensureKyvernoResources(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	// Create/update ServiceAccounts for all controllers
	serviceAccountCreators := []reconciling.NamedServiceAccountReconcilerFactory{
		admissionresources.ServiceAccountReconciler(cluster),
		backgroundresources.ServiceAccountReconciler(cluster),
		reportsresources.ServiceAccountReconciler(cluster),
		cleanupresources.ServiceAccountReconciler(cluster),
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, serviceAccountCreators, cluster.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAccounts: %w", err)
	}

	// // Create/update ClusterRoles and ClusterRoleBindings for all controllers
	// clusterRoleCreators := []reconciling.NamedClusterRoleReconcilerFactory{
	// 	admissionresources.ClusterRoleReconciler(cluster),
	// 	backgroundresources.ClusterRoleReconciler(cluster),
	// 	reportsresources.ClusterRoleReconciler(cluster),
	// 	cleanupresources.ClusterRoleReconciler(cluster),
	// 	cleanupresources.CoreClusterRoleReconciler(cluster),
	// }

	// if err := reconciling.ReconcileClusterRoles(ctx, clusterRoleCreators, cluster.Status.NamespaceName, r.Client); err != nil {
	// 	return fmt.Errorf("failed to reconcile ClusterRoles: %w", err)
	// }

	// clusterRoleBindingCreators := []reconciling.NamedClusterRoleBindingReconcilerFactory{
	// 	admissionresources.ClusterRoleBindingReconciler(cluster),
	// 	admissionresources.ViewClusterRoleBindingReconciler(cluster),
	// 	backgroundresources.ClusterRoleBindingReconciler(cluster),
	// 	backgroundresources.ViewClusterRoleBindingReconciler(cluster),
	// 	reportsresources.ClusterRoleBindingReconciler(cluster),
	// 	reportsresources.ViewClusterRoleBindingReconciler(cluster),
	// 	cleanupresources.ClusterRoleBindingReconciler(cluster),
	// }

	// if err := reconciling.ReconcileClusterRoleBindings(ctx, clusterRoleBindingCreators, cluster.Status.NamespaceName, r.Client); err != nil {
	// 	return fmt.Errorf("failed to reconcile ClusterRoleBindings: %w", err)
	// }

	// Create/update Roles and RoleBindings for all controllers
	roleCreators := []reconciling.NamedRoleReconcilerFactory{
		admissionresources.RoleReconciler(cluster),
		backgroundresources.RoleReconciler(cluster),
		reportsresources.RoleReconciler(cluster),
		cleanupresources.RoleReconciler(cluster),
	}

	if err := reconciling.ReconcileRoles(ctx, roleCreators, cluster.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Roles: %w", err)
	}

	roleBindingCreators := []reconciling.NamedRoleBindingReconcilerFactory{
		admissionresources.RoleBindingReconciler(cluster),
		backgroundresources.RoleBindingReconciler(cluster),
		reportsresources.RoleBindingReconciler(cluster),
		cleanupresources.RoleBindingReconciler(cluster),
	}

	if err := reconciling.ReconcileRoleBindings(ctx, roleBindingCreators, cluster.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings: %w", err)
	}

	// Create/update Deployments for all controllers
	deploymentCreators := []reconciling.NamedDeploymentReconcilerFactory{
		admissionresources.DeploymentReconciler(cluster),
		backgroundresources.DeploymentReconciler(cluster),
		reportsresources.DeploymentReconciler(cluster),
		cleanupresources.DeploymentReconciler(cluster),
	}

	if err := reconciling.ReconcileDeployments(ctx, deploymentCreators, cluster.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Deployments: %w", err)
	}

	// Create/update Services for all controllers
	serviceCreators := []reconciling.NamedServiceReconcilerFactory{
		admissionresources.ServiceReconciler(cluster),
		admissionresources.MetricsServiceReconciler(cluster),
		backgroundresources.ServiceReconciler(cluster),
		backgroundresources.MetricsServiceReconciler(cluster),
		reportsresources.ServiceReconciler(cluster),
		reportsresources.MetricsServiceReconciler(cluster),
		cleanupresources.ServiceReconciler(cluster),
		cleanupresources.MetricsServiceReconciler(cluster),
	}

	if err := reconciling.ReconcileServices(ctx, serviceCreators, cluster.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Services: %w", err)
	}
	return nil
}

func (r *reconciler) ensureUserClusterResources(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	// Get the user cluster client
	userClusterClient, err := r.getClusterClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get user cluster client: %w", err)
	}

	// Load and reconcile CRDs
	crds, err := userclusterresources.CRDs()
	if err != nil {
		return fmt.Errorf("failed to load Kyverno CRDs: %w", err)
	}

	creators := []kkpreconciling.NamedCustomResourceDefinitionReconcilerFactory{}
	for i := range crds {
		creators = append(creators, userclusterresources.CRDReconciler(crds[i]))
	}

	if err := kkpreconciling.ReconcileCustomResourceDefinitions(ctx, creators, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile Kyverno CRDs: %w", err)
	}

	// Create/update ClusterRoles for all controllers
	clusterRoleCreators := []reconciling.NamedClusterRoleReconcilerFactory{
		userclusterreportsresources.ClusterRoleReconciler(),
		userclusterreportsresources.CoreClusterRoleReconciler(),
		userclustercleanupresources.ClusterRoleReconciler(),
		userclustercleanupresources.CoreClusterRoleReconciler(),
		userclusterbackgroundresources.ClusterRoleReconciler(),
		userclusterbackgroundresources.CoreClusterRoleReconciler(),
		userclusteradmissionresources.ClusterRoleReconciler(),
		userclusteradmissionresources.CoreClusterRoleReconciler(),
	}

	if err := reconciling.ReconcileClusterRoles(ctx, clusterRoleCreators, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoles: %w", err)
	}

	// Create/update ClusterRoleBindings for all controllers
	clusterRoleBindingCreators := []reconciling.NamedClusterRoleBindingReconcilerFactory{
		userclusterreportsresources.ClusterRoleBindingReconciler(cluster.Status.NamespaceName),
		userclusterreportsresources.ViewClusterRoleBindingReconciler(cluster.Status.NamespaceName),
		userclustercleanupresources.ClusterRoleBindingReconciler(cluster.Status.NamespaceName),
		userclusterbackgroundresources.ClusterRoleBindingReconciler(cluster.Status.NamespaceName),
		userclusterbackgroundresources.ViewClusterRoleBindingReconciler(cluster.Status.NamespaceName),
		userclusteradmissionresources.ClusterRoleBindingReconciler(cluster.Status.NamespaceName),
		userclusteradmissionresources.ViewClusterRoleBindingReconciler(cluster.Status.NamespaceName),
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, clusterRoleBindingCreators, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBindings: %w", err)
	}

	return nil
}

func (r *reconciler) getClusterClient(ctx context.Context, cluster *kubermaticv1.Cluster) (ctrlruntimeclient.Client, error) {
	return r.userClusterConnectionProvider.GetClient(ctx, cluster)
}
