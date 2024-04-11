/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package policies

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	// PreventDeletionAnnotation is an annotation that will block any attempts to delete the object
	// that has it. The value does not matter.
	PreventDeletionAnnotation = "policy.k8c.io/prevent-deletion"
)

// AdmissionHandler for validating ApplicationDefinition CRD.
type AdmissionHandler struct {
	log     *zap.SugaredLogger
	decoder *admission.Decoder
}

// NewAdmissionHandler returns a new validation AdmissionHandler.
func NewAdmissionHandler(log *zap.SugaredLogger, scheme *runtime.Scheme) *AdmissionHandler {
	return &AdmissionHandler{
		log:     log,
		decoder: admission.NewDecoder(scheme),
	}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/validate-policies", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	allErrs := field.ErrorList{}
	obj := &unstructured.Unstructured{}

	switch req.Operation {
	case admissionv1.Delete:
		if err := h.decoder.DecodeRaw(req.OldObject, obj); err != nil {
			return webhook.Errored(http.StatusBadRequest, err)
		}

		if _, exists := obj.GetAnnotations()[PreventDeletionAnnotation]; exists {
			allErrs = append(allErrs, field.Forbidden(nil, "object is annotated to prevent deletions"))
		}

	default:
		return webhook.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported", req.Operation))
	}

	if len(allErrs) > 0 {
		return webhook.Denied(fmt.Sprintf("request %s denied: %v", req.UID, allErrs))
	}

	return webhook.Allowed(fmt.Sprintf("request %s allowed", req.UID))
}
