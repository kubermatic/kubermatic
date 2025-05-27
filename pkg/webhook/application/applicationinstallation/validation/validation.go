/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/validation"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for validating ApplicationInstallation CRD.
type AdmissionHandler struct {
	log         *zap.SugaredLogger
	decoder     admission.Decoder
	client      ctrlruntimeclient.Client
	clusterName string
}

// NewAdmissionHandler returns a new validation AdmissionHandler.
func NewAdmissionHandler(log *zap.SugaredLogger, scheme *runtime.Scheme, client ctrlruntimeclient.Client, clusterName string) *AdmissionHandler {
	return &AdmissionHandler{
		log:         log,
		decoder:     admission.NewDecoder(scheme),
		client:      client,
		clusterName: clusterName,
	}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/validate-application-installation", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	allErrs := field.ErrorList{}
	ad := &appskubermaticv1.ApplicationInstallation{}
	oldAD := &appskubermaticv1.ApplicationInstallation{}

	switch req.Operation {
	case admissionv1.Create:
		if err := h.decoder.Decode(req, ad); err != nil {
			return webhook.Errored(http.StatusBadRequest, err)
		}
		allErrs = append(allErrs, validation.ValidateApplicationInstallationSpec(ctx, h.client, *ad)...)

	case admissionv1.Update:
		if err := h.decoder.Decode(req, ad); err != nil {
			return webhook.Errored(http.StatusBadRequest, err)
		}
		if err := h.decoder.DecodeRaw(req.OldObject, oldAD); err != nil {
			return webhook.Errored(http.StatusBadRequest, err)
		}
		allErrs = append(allErrs, validation.ValidateApplicationInstallationUpdate(ctx, h.client, *ad, *oldAD)...)

	case admissionv1.Delete:
		if err := h.decoder.DecodeRaw(req.OldObject, ad); err != nil {
			return webhook.Errored(http.StatusBadRequest, err)
		}
		allErrs = append(allErrs, validation.ValidateApplicationInstallationDelete(ctx, h.client, h.clusterName, *ad)...)

	default:
		return webhook.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on ApplicationInstallation resources", req.Operation))
	}

	if len(allErrs) > 0 {
		return webhook.Denied(fmt.Sprintf("ApplicationInstallation validation request %s denied for ApplicationInstallation %s/%s with error: %v", req.UID, ad.Namespace, ad.Name, allErrs))
	}

	return webhook.Allowed(fmt.Sprintf("ApplicationInstallation validation request %s allowed", req.UID))
}
