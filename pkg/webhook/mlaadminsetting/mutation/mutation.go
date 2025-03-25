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

package mutation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for mutating Kubermatic MLAAdminSetting CRD.
type AdmissionHandler struct {
	log              *zap.SugaredLogger
	decoder          admission.Decoder
	seedGetter       provider.SeedGetter
	seedClientGetter provider.SeedClientGetter
}

// NewAdmissionHandler returns a new MLAAdminSetting AdmissionHandler.
func NewAdmissionHandler(log *zap.SugaredLogger, scheme *runtime.Scheme, seedGetter provider.SeedGetter, seedClientGetter provider.SeedClientGetter) *AdmissionHandler {
	return &AdmissionHandler{
		log:              log,
		decoder:          admission.NewDecoder(scheme),
		seedGetter:       seedGetter,
		seedClientGetter: seedClientGetter,
	}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/mutate-kubermatic-k8c-io-v1-mlaadminsetting", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	adminSetting := &kubermaticv1.MLAAdminSetting{}

	switch req.Operation {
	case admissionv1.Create:
		if err := h.decoder.Decode(req, adminSetting); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		// apply defaults to the existing MLAAdminSetting
		err := h.ensureClusterReference(ctx, adminSetting)
		if err != nil {
			h.log.Error(err, "MLAAdminSetting mutation failed")
			return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("MLAAdminSetting mutation request %s failed: %w", req.UID, err))
		}

	case admissionv1.Update:
		oldSetting := &kubermaticv1.MLAAdminSetting{}

		if err := h.decoder.Decode(req, adminSetting); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		if err := h.decoder.DecodeRaw(req.OldObject, oldSetting); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		err := h.validateUpdate(ctx, oldSetting, adminSetting)
		if err != nil {
			h.log.Error(err, "MLAAdminSetting mutation failed")
			return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("MLAAdminSetting mutation request %s failed: %w", req.UID, err))
		}

	case admissionv1.Delete:
		return webhook.Allowed(fmt.Sprintf("no mutation done for request %s", req.UID))

	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on MLAAdminSetting resources", req.Operation))
	}

	mutatedSetting, err := json.Marshal(adminSetting)
	if err != nil {
		return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("marshaling MLAAdminSetting object failed: %w", err))
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, mutatedSetting)
}

func (h *AdmissionHandler) ensureClusterReference(ctx context.Context, adminSetting *kubermaticv1.MLAAdminSetting) error {
	seed, err := h.seedGetter()
	if err != nil {
		return fmt.Errorf("failed to get current Seed: %w", err)
	}
	if seed == nil {
		return errors.New("webhook not configured for a Seed cluster, cannot validate MLAAdminSetting resources")
	}

	client, err := h.seedClientGetter(seed)
	if err != nil {
		return fmt.Errorf("failed to get Seed client: %w", err)
	}

	cluster, err := kubernetes.ClusterFromNamespace(ctx, client, adminSetting.Namespace)
	if err != nil {
		return fmt.Errorf("failed to list Cluster objects: %w", err)
	}

	if cluster == nil {
		return errors.New("MLAAdminSetting resources can only be created in cluster namespaces, but no matching Cluster was found")
	}

	if cluster.DeletionTimestamp != nil {
		return fmt.Errorf("Cluster %s is in deletion already, cannot create a new MLAAdminSetting", cluster.Name)
	}

	adminSetting.Spec.ClusterName = cluster.Name

	return nil
}

func (h *AdmissionHandler) validateUpdate(ctx context.Context, oldSetting *kubermaticv1.MLAAdminSetting, newSetting *kubermaticv1.MLAAdminSetting) error {
	if !equality.Semantic.DeepEqual(oldSetting.Spec.ClusterName, newSetting.Spec.ClusterName) {
		return errors.New("Cluster reference cannot be changed")
	}

	return nil
}
