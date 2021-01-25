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

package seed

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	admissionv1 "k8s.io/api/admission/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for Kubermatic Seed CRD.
type AdmissionHandler interface {
	webhook.AdmissionHandler
	inject.Logger
	admission.DecoderInjector
	SetupWebhookWithManager(mgr ctrl.Manager)
}

type seedAdmissionHandler struct {
	log          logr.Logger
	validateFunc validateFunc
	decoder      *admission.Decoder
}

var _ AdmissionHandler = &seedAdmissionHandler{}

func (h *seedAdmissionHandler) InjectLogger(l logr.Logger) error {
	h.log = l.WithName("seed-validation-handler")
	return nil
}

func (h *seedAdmissionHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

func (h *seedAdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	seed := &kubermaticv1.Seed{}

	switch req.Operation {
	case admissionv1.Create:
		fallthrough
	case admissionv1.Update:
		err := h.decoder.Decode(req, seed)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
	// On DELETE, the req.Object is unset
	// Ref: https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#webhook-request-and-response
	case admissionv1.Delete:
		seed.Name = req.Name
		seed.Namespace = req.Namespace
	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on seed resources", req.Operation))
	}
	validationErr := h.validateFunc(ctx, seed, req.Operation)
	if validationErr != nil {
		h.log.Info("seed admission failed", "error", validationErr)
		return webhook.Denied(fmt.Sprintf("seed validation request %s rejected: %v", req.UID, validationErr))
	}
	return webhook.Allowed(fmt.Sprintf("seed validation request %s allowed", req.UID))
}

func (h *seedAdmissionHandler) SetupWebhookWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register("/validate-kubermatic-k8s-io-seed", &webhook.Admission{Handler: h})
}
