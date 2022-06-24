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
	log     logr.Logger
	decoder *admission.Decoder
	client  ctrlruntimeclient.Client
}

func NewAdmissionHandler(client ctrlruntimeclient.Client) *AdmissionHandler {
	return &AdmissionHandler{
		client: client,
	}
}

func (h *AdmissionHandler) SetupWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/mutate-kubermatic-k8c-io-v1-resource-quota", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) InjectLogger(l logr.Logger) error {
	h.log = l.WithName("resource-quota-mutation-handler")
	return nil
}

func (h *AdmissionHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

func (h *AdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	return handle(ctx, req, h.decoder, h.log, h.client)
}
