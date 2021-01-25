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

package nodecsrapprover

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	certificatesv1beta1client "k8s.io/client-go/kubernetes/typed/certificates/v1beta1"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const ControllerName = "node_csr_autoapprover"

// Check if the Reconciler fulfills the interface
// at compile time
var _ reconcile.Reconciler = &reconciler{}

type reconciler struct {
	ctrlruntimeclient.Client
	// Have to use the typed client because csr approval is a subresource
	// the dynamic client does not approve
	certClient certificatesv1beta1client.CertificateSigningRequestInterface
	log        *zap.SugaredLogger
}

func Add(mgr manager.Manager, numWorkers int, cfg *rest.Config, log *zap.SugaredLogger) error {
	certClient, err := certificatesv1beta1client.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create certificate client: %v", err)
	}

	r := &reconciler{Client: mgr.GetClient(), certClient: certClient.CertificateSigningRequests(), log: log}
	c, err := controller.New(ControllerName, mgr,
		controller.Options{Reconciler: r, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}
	return c.Watch(&source.Kind{Type: &certificatesv1beta1.CertificateSigningRequest{}}, &handler.EnqueueRequestForObject{})
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	err := r.reconcile(ctx, request)
	if err != nil {
		r.log.Errorw("Reconciliation of request failed", "request", request.NamespacedName.String(), zap.Error(err))
	}
	return reconcile.Result{}, err
}

var allowedUsages = []certificatesv1beta1.KeyUsage{certificatesv1beta1.UsageDigitalSignature,
	certificatesv1beta1.UsageKeyEncipherment,
	certificatesv1beta1.UsageServerAuth}

func (r *reconciler) reconcile(ctx context.Context, request reconcile.Request) error {
	log := r.log.With("csr", request.NamespacedName.String())
	log.Debug("Reconciling")

	csr := &certificatesv1beta1.CertificateSigningRequest{}
	if err := r.Get(ctx, request.NamespacedName, csr); err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	for _, condition := range csr.Status.Conditions {
		if condition.Type == certificatesv1beta1.CertificateApproved {
			log.Debug("already approved, skipping reconciling")
			return nil
		}
	}

	if !sets.NewString(csr.Spec.Groups...).Has("system:nodes") {
		log.Debug("Skipping reconciling because 'system:nodes' is not in its groups")
		return nil
	}

	if len(csr.Spec.Usages) != 3 {
		log.Debug("Skipping reconciling because it has not exactly three usages defined")
		return nil
	}

	for _, usage := range csr.Spec.Usages {
		if !isUsageInUsageList(usage, allowedUsages) {
			r.log.Debugw("Skipping reconciling because its usage is not in the list of allowed usages",
				"usage", usage, "allowed-usages", allowedUsages)
			return nil
		}
	}

	log.Debug("Approving")
	approvalCondition := certificatesv1beta1.CertificateSigningRequestCondition{
		Type:   certificatesv1beta1.CertificateApproved,
		Reason: "Kubermatic NodeCSRApprover controller approved node serving cert",
	}
	csr.Status.Conditions = append(csr.Status.Conditions, approvalCondition)

	if _, err := r.certClient.UpdateApproval(ctx, csr, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update approval for CSR %q: %v", csr.Name, err)
	}

	log.Infof("Successfully approved")
	return nil
}

func isUsageInUsageList(usage certificatesv1beta1.KeyUsage, usageList []certificatesv1beta1.KeyUsage) bool {
	for _, usageListItem := range usageList {
		if usage == usageListItem {
			return true
		}
	}
	return false
}
