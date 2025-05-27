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

package mutation

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for mutating Kubermatic Cluster CRD.
type AdmissionHandler struct {
	log     *zap.SugaredLogger
	decoder admission.Decoder

	client       ctrlruntimeclient.Client
	seedGetter   provider.SeedGetter
	configGetter provider.KubermaticConfigurationGetter
	caBundle     *x509.CertPool
}

// NewAdmissionHandler returns a new cluster AdmissionHandler.
func NewAdmissionHandler(log *zap.SugaredLogger, scheme *runtime.Scheme, client ctrlruntimeclient.Client, configGetter provider.KubermaticConfigurationGetter, seedGetter provider.SeedGetter, caBundle *x509.CertPool) *AdmissionHandler {
	return &AdmissionHandler{
		log:          log,
		decoder:      admission.NewDecoder(scheme),
		client:       client,
		configGetter: configGetter,
		seedGetter:   seedGetter,
		caBundle:     caBundle,
	}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/mutate-kubermatic-k8c-io-v1-cluster", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	cluster := &kubermaticv1.Cluster{}
	var oldCluster *kubermaticv1.Cluster

	switch req.Operation {
	case admissionv1.Create:
		if err := h.decoder.Decode(req, cluster); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

	case admissionv1.Update:
		if err := h.decoder.Decode(req, cluster); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		oldCluster = &kubermaticv1.Cluster{}
		if err := h.decoder.DecodeRaw(req.OldObject, oldCluster); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

	case admissionv1.Delete:
		return webhook.Allowed(fmt.Sprintf("no mutation done for request %s", req.UID))

	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on cluster resources", req.Operation))
	}

	mutator := NewMutator(h.client, h.configGetter, h.seedGetter, h.caBundle)

	mutated, mutateErr := mutator.Mutate(ctx, oldCluster, cluster)
	if mutateErr != nil {
		h.log.Error(mutateErr, "cluster mutation failed")

		status := http.StatusBadRequest
		if mutateErr.Type == field.ErrorTypeInternal {
			status = http.StatusInternalServerError
		}

		return webhook.Errored(int32(status), mutateErr)
	}

	mutatedCluster, err := json.Marshal(mutated)
	if err != nil {
		return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("marshaling cluster object failed: %w", err))
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, mutatedCluster)
}
