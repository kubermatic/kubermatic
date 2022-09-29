package mla

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8s.io/utils/strings/slices"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// TODO remove this in v2.22
// userProjectBindingReconciler just cleans up the unnecessary mla finalizer from UserProjectBindings, which were made obsolete
// in 2.21.2, through https://github.com/kubermatic/kubermatic/pull/11076
type userProjectBindingReconciler struct {
	ctrlruntimeclient.Client

	log                          *zap.SugaredLogger
	workerName                   string
	versions                     kubermatic.Versions
	userProjectBindingController *userGrafanaController
}

func newUserProjectBindingReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
) error {
	log = log.Named(ControllerName)
	client := mgr.GetClient()

	reconciler := &userProjectBindingReconciler{
		Client: client,

		log: log.Named("userprojectbinding-finalizer-cleaner"),
	}

	ctrlOptions := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	finalizerPredicate := predicate.NewPredicateFuncs(func(object ctrlruntimeclient.Object) bool {
		// We don't trigger reconciliation for already cleaned up upb.
		upb := object.(*kubermaticv1.UserProjectBinding)
		return slices.Contains(upb.Finalizers, mlaFinalizer)
	})

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.UserProjectBinding{}}, &handler.EnqueueRequestForObject{}, finalizerPredicate); err != nil {
		return fmt.Errorf("failed to watch UserProjectBindings: %w", err)
	}

	return err
}

func (r *userProjectBindingReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	upb := &kubermaticv1.UserProjectBinding{}
	if err := r.Get(ctx, request.NamespacedName, upb); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	return reconcile.Result{}, kubernetes.TryRemoveFinalizer(ctx, r.Client, upb, mlaFinalizer)
}
