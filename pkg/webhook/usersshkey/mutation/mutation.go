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

	"github.com/go-logr/logr"
	"golang.org/x/crypto/ssh"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	admissionv1 "k8s.io/api/admission/v1"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for mutating Kubermatic Cluster CRD.
type AdmissionHandler struct {
	log     logr.Logger
	decoder *admission.Decoder
}

// NewAdmissionHandler returns a new UserSSHKey AdmissionHandler.
func NewAdmissionHandler() *AdmissionHandler {
	return &AdmissionHandler{}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/mutate-kubermatic-k8c-io-v1-usersshkey", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) InjectLogger(l logr.Logger) error {
	h.log = l.WithName("usersshkey-mutation-handler")
	return nil
}

func (h *AdmissionHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

func (h *AdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	sshKey := &kubermaticv1.UserSSHKey{}
	oldKey := &kubermaticv1.UserSSHKey{}

	switch req.Operation {
	case admissionv1.Create:
		if err := h.decoder.Decode(req, sshKey); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		err := h.applyDefaults(ctx, sshKey, true)
		if err != nil {
			h.log.Info("usersshkey mutation failed", "error", err)
			return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("usersshkey mutation request %s failed: %w", req.UID, err))
		}

	case admissionv1.Update:
		if err := h.decoder.Decode(req, sshKey); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		if err := h.decoder.DecodeRaw(req.OldObject, oldKey); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		// apply defaults to the existing sshKey
		err := h.applyDefaults(ctx, sshKey, oldKey.Spec.PublicKey != sshKey.Spec.PublicKey)
		if err != nil {
			h.log.Info("usersshkey mutation failed", "error", err)
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

func (h *AdmissionHandler) applyDefaults(ctx context.Context, key *kubermaticv1.UserSSHKey, pubkeyChanged bool) error {
	if pubkeyChanged {
		if key.Spec.PublicKey == "" {
			return errors.New("spec.publicKey cannot be empty")
		}

		// parse the key
		pubKeyParsed, _, _, _, err := ssh.ParseAuthorizedKey([]byte(key.Spec.PublicKey))
		if err != nil {
			return fmt.Errorf("the provided SSH key is invalid: %w", err)
		}

		// calculate the fingerprint
		key.Spec.Fingerprint = ssh.FingerprintLegacyMD5(pubKeyParsed)
	}

	return nil
}
