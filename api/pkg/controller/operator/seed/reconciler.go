package seed

import (
	"context"

	"go.uber.org/zap"

	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler (re)stores all components required for running a Kubermatic
// seed cluster.
type Reconciler struct {
	ctrlruntimeclient.Client

	ctx            context.Context
	log            *zap.SugaredLogger
	namespace      string
	masterClient   ctrlruntimeclient.Client
	seedClients    map[string]ctrlruntimeclient.Client
	masterRecorder record.EventRecorder
	workerName     string
}

// Reconcile acts upon requests and will restore the state of resources
// for the given namespace. Will return an error if any API operation
// failed, otherwise will return an empty dummy Result struct.
func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	r.log.Debugw("reconciling", "request", request.NamespacedName)

	// find the requested seed

	return reconcile.Result{}, nil
}
