package operatormaster

import (
	"context"
	"fmt"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/cache"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler (re)stores all components required for running a Kubermatic
// master cluster.
type Reconciler struct {
	ctrlruntimeclient.Client

	clientConfig *clientcmdapi.Config
	log          *zap.SugaredLogger
	recorder     record.EventRecorder
	workerName   string
	ctx          context.Context
}

// Reconcile acts upon requests and will restore the state of resources
// for the given namespace. Will return an error if any API operation
// failed, otherwise will return an empty dummy Result struct.
func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	var cancel context.CancelFunc

	r.ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	// find the requested configuration
	config := &operatorv1alpha1.KubermaticConfiguration{}
	if err := r.Get(r.ctx, request.NamespacedName, config); err != nil {
		return reconcile.Result{}, fmt.Errorf("could not get KubermaticConfiguration %q: %v", request, err)
	}

	// silently ignore other worker names
	if config.Labels[WorkerNameLabel] != r.workerName {
		return reconcile.Result{}, nil
	}

	identifier, err := cache.MetaNamespaceKeyFunc(config)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to determine string key for KubermaticConfiguration: %v", err)
	}

	logger := r.log.With("config", identifier)

	return reconcile.Result{}, r.reconcile(config, logger)
}

func (r *Reconciler) reconcile(config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling Kubermatic configuration now.")
	return nil
}
