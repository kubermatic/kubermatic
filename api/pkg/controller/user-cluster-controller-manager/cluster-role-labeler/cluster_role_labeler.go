package clusterrolelabeler

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	predicateutil "github.com/kubermatic/kubermatic/api/pkg/controller/util/predicate"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/cluster"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// This controller adds special label for build-it cluster roles to make them visible in the API
	controllerName = "cluster_role_label_controller"
)

type reconciler struct {
	ctx      context.Context
	log      *zap.SugaredLogger
	client   ctrlruntimeclient.Client
	recorder record.EventRecorder
}

func Add(ctx context.Context, log *zap.SugaredLogger, mgr manager.Manager) error {
	log = log.Named(controllerName)

	r := &reconciler{
		ctx:      ctx,
		log:      log,
		client:   mgr.GetClient(),
		recorder: mgr.GetEventRecorderFor(controllerName),
	}
	c, err := controller.New(controllerName, mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %v", err)
	}

	// Watch for changes to ClusterRoles
	if err = c.Watch(&source.Kind{Type: &rbacv1.ClusterRole{}}, &handler.EnqueueRequestForObject{}, predicateutil.ByName("admin", "view", "edit")); err != nil {
		return fmt.Errorf("failed to establish watch for the ClusterRoles %v", err)
	}

	return nil
}

func (r *reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("ClusterRole", request.Name)
	log.Debug("Reconciling")

	clusterRole := &rbacv1.ClusterRole{}
	if err := r.client.Get(r.ctx, request.NamespacedName, clusterRole); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("cluster role not found, returning")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get cluster role: %v", err)
	}

	err := r.reconcile(log, clusterRole)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(clusterRole, corev1.EventTypeWarning, "AddingLabelFailed", err.Error())
	}
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(log *zap.SugaredLogger, clusterRole *rbacv1.ClusterRole) error {
	oldClusterRole := clusterRole.DeepCopy()
	if clusterRole.Labels == nil {
		clusterRole.Labels = map[string]string{}
	}

	if value, ok := clusterRole.Labels[cluster.UserClusterComponentKey]; ok {
		if value == cluster.UserClusterRoleComponentValue {
			log.Debug("label ", cluster.UserClusterRoleLabelSelector, " exists, not updating cluster role: ", clusterRole.Name)
			return nil
		}
	}

	clusterRole.Labels[cluster.UserClusterComponentKey] = cluster.UserClusterRoleComponentValue

	if err := r.client.Patch(r.ctx, clusterRole, ctrlruntimeclient.MergeFrom(oldClusterRole)); err != nil {
		return fmt.Errorf("failed to update cluster role: %v", err)
	}

	return nil
}
