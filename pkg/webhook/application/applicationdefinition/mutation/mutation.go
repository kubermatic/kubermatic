/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package mutation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for mutating Kubermatic ApplicationDefinition CRD.
type AdmissionHandler struct {
	log     *zap.SugaredLogger
	decoder admission.Decoder
}

// NewAdmissionHandler returns a new ApplicationDefinition AdmissionHandler.
func NewAdmissionHandler(log *zap.SugaredLogger, scheme *runtime.Scheme) *AdmissionHandler {
	return &AdmissionHandler{
		log:     log,
		decoder: admission.NewDecoder(scheme),
	}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/mutate-application-definition", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	appDef := &appskubermaticv1.ApplicationDefinition{}

	switch req.Operation {
	case admissionv1.Create:
		if err := h.decoder.Decode(req, appDef); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		if err := defaulting.DefaultApplicationDefinition(appDef); err != nil {
			h.log.Errorw("ApplicationDefinition mutation failed", zap.Error(err))
			return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("ApplicationDefinition mutation request %s failed: %w", req.UID, err))
		}

	case admissionv1.Update:
		if err := h.decoder.Decode(req, appDef); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		if err := defaulting.DefaultApplicationDefinition(appDef); err != nil {
			h.log.Errorw("ApplicationDefinition mutation failed", zap.Error(err))
			return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("ApplicationDefinition mutation request %s failed: %w", req.UID, err))
		}

	case admissionv1.Delete:
		return webhook.Allowed(fmt.Sprintf("no mutation done for request %s", req.UID))

	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on ApplicationDefinition resources", req.Operation))
	}

	mutatedAppInstall, err := json.Marshal(appDef)
	if err != nil {
		return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("marshaling ApplicationDefinition object failed: %w", err))
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, mutatedAppInstall)
}
