/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package validation

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"

	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	admissionv1 "k8s.io/api/admission/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for validating OperatingSystemConfig CRD.
type AdmissionHandler struct {
	log     logr.Logger
	decoder *admission.Decoder
}

// NewAdmissionHandler returns a new validation AdmissionHandler.
func NewAdmissionHandler() *AdmissionHandler {
	return &AdmissionHandler{}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/validate-operating-system-config", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) InjectLogger(l logr.Logger) error {
	h.log = l.WithName("operating-system-config-validation-handler")
	return nil
}

func (h *AdmissionHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

func (h *AdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	allErrs := field.ErrorList{}
	osc := &osmv1alpha1.OperatingSystemConfig{}
	oldOSC := &osmv1alpha1.OperatingSystemConfig{}

	switch req.Operation {
	case admissionv1.Update:
		if err := h.decoder.Decode(req, osc); err != nil {
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("error occurred while decoding osc: %w", err))
		}
		if err := h.decoder.DecodeRaw(req.OldObject, oldOSC); err != nil {
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("error occurred while decoding old osc: %w", err))
		}
		allErrs = append(allErrs, h.validateUpdate(ctx, osc, oldOSC)...)

	case admissionv1.Create, admissionv1.Delete:
		// NOP we always allow create, delete operarions at the moment

	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on osc resources", req.Operation))
	}

	if len(allErrs) > 0 {
		return webhook.Denied(fmt.Sprintf("operatingSystemConfig validation request %s denied: %v", req.UID, allErrs))
	}

	return webhook.Allowed(fmt.Sprintf("operatingSystemConfig validation request %s allowed", req.UID))
}

func (h *AdmissionHandler) validateUpdate(ctx context.Context, osc, oldOSC *osmv1alpha1.OperatingSystemConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	// Updates for OperatingSystemConfig Spec are not allowed
	if equal := apiequality.Semantic.DeepEqual(oldOSC.Spec, osc.Spec); !equal {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), "", "OperatingSystemConfig is immutable and updates are not allowed"))
	}
	return allErrs
}
