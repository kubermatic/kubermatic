package nodecsrapprover

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/glog"

	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	certificatesv1beta1client "k8s.io/client-go/kubernetes/typed/certificates/v1beta1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const ControllerName = "node_csr_autoapprover"

// Check if the Reconciler fullfills the interface
// at compile time
var _ reconcile.Reconciler = &reconciler{}

type reconciler struct {
	client.Client
	// Have to use the typed client because csr approval is a subresource
	// the dynamic client does not approve
	certClient certificatesv1beta1client.CertificateSigningRequestInterface
}

func Add(mgr manager.Manager, numWorkers int, cfg *rest.Config) error {
	certClient, err := certificatesv1beta1client.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create certificate client: %v", err)
	}
	client := mgr.GetClient()

	r := &reconciler{Client: client, certClient: certClient.CertificateSigningRequests()}
	c, err := controller.New(ControllerName, mgr,
		controller.Options{Reconciler: r, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}
	return c.Watch(&source.Kind{Type: &certificatesv1beta1.CertificateSigningRequest{}}, &handler.EnqueueRequestForObject{})
}

func (r *reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := r.reconcile(ctx, request)
	if err != nil {
		glog.Errorf("Reconciliation of request %s failed: %v", request.NamespacedName.String(), err)
	}
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, request reconcile.Request) error {
	csr := &certificatesv1beta1.CertificateSigningRequest{}
	if err := r.Get(ctx, request.NamespacedName, csr); err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	for _, condition := range csr.Status.Conditions {
		if condition.Type == certificatesv1beta1.CertificateApproved {
			return nil
		}
	}

	if csr.Spec.Username == "system:serviceaccount:openshift-infra:node-bootstrapper" ||
		strings.HasPrefix(csr.Spec.Username, "system:node:") {
		glog.V(4).Infof("Approving csr %s", csr.Name)
		csr.Status.Conditions = append(csr.Status.Conditions, certificatesv1beta1.CertificateSigningRequestCondition{
			Type:   certificatesv1beta1.CertificateApproved,
			Reason: "Kubermatic nodecsrapprover approved",
		})
		_, err := r.certClient.UpdateApproval(csr)
		return err
	}
	return nil
}
