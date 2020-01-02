package ownerbindingcreator

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	predicateutil "github.com/kubermatic/kubermatic/api/pkg/controller/util/predicate"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/cluster"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// This controller creates cluster role bindings for the API roles.
	controllerName = "owner_binding_controller"
)

type reconciler struct {
	ctx        context.Context
	log        *zap.SugaredLogger
	client     ctrlruntimeclient.Client
	recorder   record.EventRecorder
	ownerEmail string
}

func Add(ctx context.Context, log *zap.SugaredLogger, mgr manager.Manager, ownerEmail string) error {
	log = log.Named(controllerName)

	r := &reconciler{
		ctx:        ctx,
		log:        log,
		client:     mgr.GetClient(),
		recorder:   mgr.GetEventRecorderFor(controllerName),
		ownerEmail: ownerEmail,
	}
	c, err := controller.New(controllerName, mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %v", err)
	}

	// Watch for changes to ClusterRoles
	if err = c.Watch(&source.Kind{Type: &rbacv1.ClusterRole{}}, &handler.EnqueueRequestForObject{}, predicateutil.ByLabel(cluster.UserClusterComponentKey, cluster.UserClusterRoleComponentValue)); err != nil {
		return fmt.Errorf("failed to establish watch for the ClusterRoles %v", err)
	}
	// Watch for changes to ClusterRoleBindings
	if err = c.Watch(&source.Kind{Type: &rbacv1.ClusterRoleBinding{}}, enqueueAPIBindings(mgr.GetClient()), predicateutil.ByLabel(cluster.UserClusterComponentKey, cluster.UserClusterBindingComponentValue)); err != nil {
		return fmt.Errorf("failed to establish watch for the ClusterRoles %v", err)
	}

	return nil
}

func (r *reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("ClusterRole", request.Name)
	log.Debug("Reconciling")

	err := r.reconcile(log, request.Name)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(&rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: request.Name}}, corev1.EventTypeWarning, "AddingBindingFailed", err.Error())
	}
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(log *zap.SugaredLogger, clusterRoleName string) error {

	clusterRoleBindingList := &rbacv1.ClusterRoleBindingList{}
	if err := r.client.List(r.ctx, clusterRoleBindingList, ctrlruntimeclient.MatchingLabels{cluster.UserClusterComponentKey: cluster.UserClusterBindingComponentValue}); err != nil {
		return fmt.Errorf("failed get cluster role binding list %v", err)
	}

	var existingClusterRoleBinding *rbacv1.ClusterRoleBinding
	for _, clusterRoleBinding := range clusterRoleBindingList.Items {
		if clusterRoleBinding.RoleRef.Name == clusterRoleName {
			existingClusterRoleBinding = clusterRoleBinding.DeepCopy()
			break
		}
	}

	// Create Cluster Role Binding if doesn't exist
	if existingClusterRoleBinding == nil {
		log.Debug("creating cluster role binding for cluster role ", clusterRoleName)
		crb := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:   fmt.Sprintf("%s:%s", rand.String(10), clusterRoleName),
				Labels: map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterBindingComponentValue},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     clusterRoleName,
			},
		}
		// Bind user who created the cluster to the `admin` ClusterRole.
		// Add cluster owner only once when binding doesn't exist yet.
		// Later the user can remove/add subjects from the binding using the API.
		if clusterRoleName == "admin" {
			crb.Subjects = []rbacv1.Subject{
				{
					Kind:     rbacv1.UserKind,
					APIGroup: rbacv1.GroupName,
					Name:     r.ownerEmail,
				},
			}
		}
		if err := r.client.Create(r.ctx, crb); err != nil {
			return fmt.Errorf("failed to create cluster role binding %v", err)
		}
	}
	return nil
}

// enqueueAPIBindings enqueues the ClusterRoleBindings with a special label component=userClusterRole
func enqueueAPIBindings(client ctrlruntimeclient.Client) *handler.EnqueueRequestsFromMapFunc {
	return &handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
		clusterRoleList := &rbacv1.ClusterRoleList{}
		if err := client.List(context.Background(), clusterRoleList, ctrlruntimeclient.MatchingLabels{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue}); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list Cluster Roles: %v", err))
			return []reconcile.Request{}
		}

		var requests []reconcile.Request
		for _, clusterRole := range clusterRoleList.Items {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: clusterRole.Name}})
		}
		return requests
	})}
}
