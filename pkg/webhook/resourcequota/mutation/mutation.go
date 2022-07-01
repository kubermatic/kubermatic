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

	"github.com/go-logr/logr"

	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type AdmissionHandler struct {
	Log     logr.Logger
	Decoder *admission.Decoder
	Client  ctrlruntimeclient.Client
}

func NewAdmissionHandler(client ctrlruntimeclient.Client) *AdmissionHandler {
	return &AdmissionHandler{
		Client: client,
	}
}

func (h *AdmissionHandler) SetupWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/mutate-kubermatic-k8c-io-v1-resourcequota", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) InjectLogger(l logr.Logger) error {
	h.Log = l.WithName("resource-quota-mutation-handler")
	return nil
}

func (h *AdmissionHandler) InjectDecoder(d *admission.Decoder) error {
	h.Decoder = d
	return nil
}

func (h *AdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	return handle(ctx, req, h.Decoder, h.Log, h.Client)
}
