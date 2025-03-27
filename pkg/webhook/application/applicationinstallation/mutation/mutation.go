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
	"k8s.io/apimachinery/pkg/types"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for mutating Kubermatic ApplicationInstallation CRD.
type AdmissionHandler struct {
	log     *zap.SugaredLogger
	decoder admission.Decoder
	client  ctrlruntimeclient.Client
}

// NewAdmissionHandler returns a new ApplicationInstallation AdmissionHandler.
func NewAdmissionHandler(log *zap.SugaredLogger, scheme *runtime.Scheme, client ctrlruntimeclient.Client) *AdmissionHandler {
	return &AdmissionHandler{
		log:     log,
		decoder: admission.NewDecoder(scheme),
		client:  client,
	}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/mutate-application-installation", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	appInstall := &appskubermaticv1.ApplicationInstallation{}

	switch req.Operation {
	case admissionv1.Create:
		if err := h.decoder.Decode(req, appInstall); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		if err := defaulting.DefaultApplicationInstallation(appInstall); err != nil {
			h.log.Error(err, "ApplicationInstallation mutation failed")
			return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("ApplicationInstallation mutation request %s failed: %w", req.UID, err))
		}

		if err := mutateAppNamespace(ctx, h.client, appInstall); err != nil {
			h.log.Error(err, "ApplicationInstallation mutation failed")
			return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("ApplicationInstallation mutation request %s failed: %w", req.UID, err))
		}

	case admissionv1.Update:
		if err := h.decoder.Decode(req, appInstall); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		if err := defaulting.DefaultApplicationInstallation(appInstall); err != nil {
			h.log.Error(err, "ApplicationInstallation mutation failed")
			return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("ApplicationInstallation mutation request %s failed: %w", req.UID, err))
		}

	case admissionv1.Delete:
		return webhook.Allowed(fmt.Sprintf("no mutation done for request %s", req.UID))

	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on ApplicationInstallation resources", req.Operation))
	}

	mutatedAppInstall, err := json.Marshal(appInstall)
	if err != nil {
		return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("marshaling ApplicationInstallation object failed: %w", err))
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, mutatedAppInstall)
}

func mutateAppNamespace(ctx context.Context, seedClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) error {
	if applicationInstallation.Spec.Namespace != nil && applicationInstallation.Spec.Namespace.Name != "" {
		return nil
	}

	applicationDefinition := appskubermaticv1.ApplicationDefinition{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: applicationInstallation.Spec.ApplicationRef.Name}, &applicationDefinition); err != nil {
		return fmt.Errorf("error on fetching application definition for mutating appinstallation namespace. %w", err)
	}

	// if there is a default namespace specified in the related application definition we will use that
	if applicationDefinition.Spec.DefaultNamespace != nil && applicationDefinition.Spec.DefaultNamespace.Name != "" {
		applicationInstallation.Spec.Namespace = applicationDefinition.Spec.DefaultNamespace
		return nil
	}
	// if there is no default we will use the application installation name
	applicationInstallation.Spec.Namespace = &appskubermaticv1.AppNamespaceSpec{
		Name:   applicationInstallation.Namespace,
		Create: true,
	}
	return nil
}
