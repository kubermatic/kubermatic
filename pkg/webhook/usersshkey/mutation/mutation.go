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
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for mutating Kubermatic Cluster CRD.
type AdmissionHandler struct {
	log     *zap.SugaredLogger
	decoder admission.Decoder
	client  ctrlruntimeclient.Client
}

// NewAdmissionHandler returns a new UserSSHKey AdmissionHandler.
func NewAdmissionHandler(log *zap.SugaredLogger, scheme *runtime.Scheme, client ctrlruntimeclient.Client) *AdmissionHandler {
	return &AdmissionHandler{
		log:     log,
		decoder: admission.NewDecoder(scheme),
		client:  client,
	}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/mutate-kubermatic-k8c-io-v1-usersshkey", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	sshKey := &kubermaticv1.UserSSHKey{}
	oldKey := &kubermaticv1.UserSSHKey{}

	switch req.Operation {
	case admissionv1.Create:
		if err := h.decoder.Decode(req, sshKey); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		if err := h.applyDefaults(ctx, sshKey, nil); err != nil {
			h.log.Error(err, "usersshkey mutation failed")
			return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("usersshkey mutation request %s failed: %w", req.UID, err))
		}

	case admissionv1.Update:
		if err := h.decoder.Decode(req, sshKey); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		if err := h.decoder.DecodeRaw(req.OldObject, oldKey); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		if err := h.applyDefaults(ctx, sshKey, oldKey); err != nil {
			h.log.Error(err, "usersshkey mutation failed")
			return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("usersshkey mutation request %s failed: %w", req.UID, err))
		}

	case admissionv1.Delete:
		return webhook.Allowed(fmt.Sprintf("no mutation done for request %s", req.UID))

	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on usersshkey resources", req.Operation))
	}

	mutatedKey, err := json.Marshal(sshKey)
	if err != nil {
		return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("marshaling usersshkey object failed: %w", err))
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, mutatedKey)
}

func (h *AdmissionHandler) applyDefaults(ctx context.Context, key *kubermaticv1.UserSSHKey, oldKey *kubermaticv1.UserSSHKey) error {
	_, err := defaulting.DefaultUserSSHKey(key, oldKey)

	return err
}
