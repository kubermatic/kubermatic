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
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for validating OperatingSystemProfile CRD.
type AdmissionHandler struct {
	log     logr.Logger
	decoder *admission.Decoder
}

// NewAdmissionHandler returns a new validation AdmissionHandler.
func NewAdmissionHandler() *AdmissionHandler {
	return &AdmissionHandler{}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/validate-operating-system-profile", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) InjectLogger(l logr.Logger) error {
	h.log = l.WithName("operating-system-profile-validation-handler")
	return nil
}

func (h *AdmissionHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

func (h *AdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	osp := &osmv1alpha1.OperatingSystemProfile{}
	oldOSP := &osmv1alpha1.OperatingSystemProfile{}

	switch req.Operation {
	case admissionv1.Update:
		if err := h.decoder.Decode(req, osp); err != nil {
			return webhook.Errored(http.StatusBadRequest, fmt.Errorf("error occurred while decoding osp: %w", err))
		}
		if err := h.decoder.DecodeRaw(req.OldObject, oldOSP); err != nil {
			return webhook.Errored(http.StatusBadRequest, fmt.Errorf("error occurred while decoding old osp: %w", err))
		}
		err := h.validateUpdate(osp, oldOSP)
		if err != nil {
			return webhook.Denied(fmt.Sprintf("operatingSystemProfile validation request %s denied: %v", req.UID, err))
		}

	case admissionv1.Create, admissionv1.Delete:
		// NOP we always allow create, delete operations

	default:
		return webhook.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on osp resources", req.Operation))
	}

	return webhook.Allowed(fmt.Sprintf("operatingSystemProfile validation request %s allowed", req.UID))
}

func (h *AdmissionHandler) validateUpdate(osp, oldOSP *osmv1alpha1.OperatingSystemProfile) error {
	if equal := apiequality.Semantic.DeepEqual(oldOSP.Spec, osp.Spec); equal {
		// There is no change in spec so no validation is required
		return nil
	}

	// OSP is immutable by nature and to make modifications a version bump is mandatory
	if osp.Spec.Version == oldOSP.Spec.Version {
		return fmt.Errorf("OperatingSystemProfile is immutable. For updates .spec.version needs to be updated")
	}

	return nil
}
