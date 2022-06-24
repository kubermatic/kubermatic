package resourcequota

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	return admission.PatchResponseFromRaw(req.Object.Raw, nil)
}
