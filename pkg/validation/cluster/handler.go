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

package cluster

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/features"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for Kubermatic Cluster CRD.
type AdmissionHandler struct {
	log      logr.Logger
	decoder  *admission.Decoder
	features features.FeatureGate
	client   client.Client
}

// NewAdmissionHandler returns a new cluster.AdmissionHandler.
func NewAdmissionHandler(client client.Client, features features.FeatureGate) *AdmissionHandler {
	return &AdmissionHandler{
		features: features,
		client:   client,
	}
}

func (h *AdmissionHandler) InjectLogger(l logr.Logger) error {
	h.log = l.WithName("cluster-validation-handler")
	return nil
}

func (h *AdmissionHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

func (h *AdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	cluster := &kubermaticv1.Cluster{}

	switch req.Operation {
	case admissionv1beta1.Create:
		fallthrough
	case admissionv1beta1.Update:
		err := h.decoder.Decode(req, cluster)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		validationErr := h.validateCreateOrUpdate(cluster)
		if validationErr != nil {
			h.log.Info("cluster admission failed", "error", validationErr)
			return webhook.Denied(fmt.Sprintf("cluster validation request %s rejected: %v", req.UID, validationErr))
		}

		if err := h.rejectEnableUserSSHKeyAgentUpdate(ctx, cluster); err != nil {
			h.log.Info("cluster admission failed", "error", err)
			return webhook.Denied(fmt.Sprintf("cluster validation request %s rejected: %v", req.UID, err))
		}

	case admissionv1beta1.Delete:
		// NOP we always allow delete operarions at the moment
	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on cluster resources", req.Operation))
	}
	return webhook.Allowed(fmt.Sprintf("cluster validation request %s allowed", req.UID))
}

func (h *AdmissionHandler) validateCreateOrUpdate(c *kubermaticv1.Cluster) error {
	if !kubermaticv1.AllExposeStrategies.Has(c.Spec.ExposeStrategy) {
		return fmt.Errorf("unknown expose strategy %q, use one between: %s", c.Spec.ExposeStrategy, kubermaticv1.AllExposeStrategies)
	}
	if c.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyTunneling &&
		!h.features.Enabled(features.TunnelingExposeStrategy) {
		return errors.New("cannot create cluster with Tunneling expose strategy, the TunnelingExposeStrategy feature gate is not enabled")
	}
	return nil
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register("/validate-kubermatic-k8s-io-cluster", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) rejectEnableUserSSHKeyAgentUpdate(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	var (
		oldCluster = &kubermaticv1.Cluster{}
		nName      = types.NamespacedName{Name: cluster.Name}
	)

	if h.client != nil {
		if err := h.client.Get(ctx, nName, oldCluster); err != nil {
			if kerrors.IsNotFound(err) {
				return nil
			}

			return fmt.Errorf("failed to fetch cluster name=%s: %v", cluster.Name, err)
		}

		if oldCluster.Spec.EnableUserSSHKeyAgent != cluster.Spec.EnableUserSSHKeyAgent {
			return errors.New("enableUserSSHKeyAgent field cannot be updated after cluster creation")
		}
	}

	return nil
}
