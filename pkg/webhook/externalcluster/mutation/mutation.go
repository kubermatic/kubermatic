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
	"fmt"
	"net/http"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for mutating Kubermatic Cluster CRD.
type AdmissionHandler struct {
	log     *zap.SugaredLogger
	decoder admission.Decoder
}

// NewAdmissionHandler returns a new cluster AdmissionHandler.
func NewAdmissionHandler(log *zap.SugaredLogger, scheme *runtime.Scheme) *AdmissionHandler {
	return &AdmissionHandler{
		log:     log,
		decoder: admission.NewDecoder(scheme),
	}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/mutate-kubermatic-k8c-io-v1-externalcluster", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	cluster := &kubermaticv1.ExternalCluster{}
	oldCluster := &kubermaticv1.ExternalCluster{}

	switch req.Operation {
	case admissionv1.Create:
		if err := h.decoder.Decode(req, cluster); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		err := h.applyDefaults(ctx, cluster)
		if err != nil {
			h.log.Info("externalcluster mutation failed", "error", err)
			return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("externalcluster mutation request %s failed: %w", req.UID, err))
		}

		if err := h.mutateCreate(cluster); err != nil {
			h.log.Info("externalcluster mutation failed", "error", err)
			return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("externalcluster mutation request %s failed: %w", req.UID, err))
		}

	case admissionv1.Update:
		if err := h.decoder.Decode(req, cluster); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		if err := h.decoder.DecodeRaw(req.OldObject, oldCluster); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		if cluster.DeletionTimestamp == nil {
			// apply defaults to the existing clusters
			err := h.applyDefaults(ctx, cluster)
			if err != nil {
				h.log.Info("externalcluster mutation failed", "error", err)
				return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("externalcluster mutation request %s failed: %w", req.UID, err))
			}

			if err := h.mutateUpdate(oldCluster, cluster); err != nil {
				h.log.Info("externalcluster mutation failed", "error", err)
				return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("externalcluster mutation request %s failed: %w", req.UID, err))
			}
		}

	case admissionv1.Delete:
		return webhook.Allowed(fmt.Sprintf("no mutation done for request %s", req.UID))

	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on externalcluster resources", req.Operation))
	}

	mutatedCluster, err := json.Marshal(cluster)
	if err != nil {
		return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("marshaling externalcluster object failed: %w", err))
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, mutatedCluster)
}

func (h *AdmissionHandler) applyDefaults(ctx context.Context, c *kubermaticv1.ExternalCluster) error {
	return defaulting.DefaultExternalClusterSpec(ctx, &c.Spec)
}

// mutateCreate is an addition to regular defaulting for new external clusters.
func (h *AdmissionHandler) mutateCreate(newCluster *kubermaticv1.ExternalCluster) error {
	return nil
}

// mutateCreate is an addition to regular defaulting for updated external clusters.
func (h *AdmissionHandler) mutateUpdate(oldCluster, newCluster *kubermaticv1.ExternalCluster) error {
	return nil
}
