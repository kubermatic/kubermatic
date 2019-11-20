package certificate

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/certificate/resources"
	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler (re)stores all components required for running a Kubermatic
// master cluster.
type Reconciler struct {
	ctrlruntimeclient.Client

	log      *zap.SugaredLogger
	recorder record.EventRecorder
	scheme   *runtime.Scheme
	ctx      context.Context
}

// Reconcile acts upon requests and will restore the state of resources
// for the given namespace. Will return an error if any API operation
// failed, otherwise will return an empty dummy Result struct.
func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// find the requested configuration
	config := &operatorv1alpha1.KubermaticConfiguration{}
	if err := r.Get(r.ctx, request.NamespacedName, config); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("could not get KubermaticConfiguration %q: %v", request, err)
	}

	identifier, err := cache.MetaNamespaceKeyFunc(config)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to determine string key for KubermaticConfiguration: %v", err)
	}

	return reconcile.Result{}, r.reconcile(config, r.log.With("config", identifier))
}

func (r *Reconciler) reconcile(config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling Certificates")

	creators := []reconciling.NamedCertificateCreatorGetter{
		resources.CertificateCreator(config),
	}

	if err := reconciling.ReconcileCertificates(r.ctx, creators, config.Namespace, r.Client, common.OwnershipModifierFactory(config, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Certificates: %v", err)
	}

	return nil
}
