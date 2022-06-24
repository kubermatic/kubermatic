package resourcequota

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func Handle(ctx context.Context, req webhook.AdmissionRequest, decoder *admission.Decoder, logger logr.Logger,
	client ctrlruntimeclient.Client) webhook.AdmissionResponse {
	resourceQuota := &kubermaticv1.ResourceQuota{}

	switch req.Operation {
	case admissionv1.Create:
		if err := decoder.Decode(req, resourceQuota); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		if resourceQuota.Spec.Subject.Kind != "project" {
			return webhook.Allowed(fmt.Sprintf("no mutation done for request %s", req.UID))
		}

		err := ensureProjectOwnershipRef(ctx, client, resourceQuota)
		if err != nil {
			logger.Info("ResourceQuota mutation failed", "error", err)
			return admission.Errored(http.StatusBadRequest, err)
		}

	case admissionv1.Update:
		if err := decoder.Decode(req, resourceQuota); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		oldResourceQuota := &kubermaticv1.ResourceQuota{}
		if err := decoder.DecodeRaw(req.OldObject, oldResourceQuota); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		err := validateUpdate(oldResourceQuota, resourceQuota)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

	case admissionv1.Delete:
		return webhook.Allowed(fmt.Sprintf("no mutation done for request %s", req.UID))

	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on addon resources", req.Operation))
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, nil)
}

func ensureProjectOwnershipRef(ctx context.Context, client ctrlruntimeclient.Client, resourceQuota *kubermaticv1.ResourceQuota) error {
	subjectName := resourceQuota.Spec.Subject.Name
	existingRefs := resourceQuota.OwnerReferences

	// check if reference already exists
	for _, owners := range existingRefs {
		if owners.Kind == kubermaticv1.ProjectKindName && owners.Name == subjectName {
			return nil
		}
	}

	// set project reference
	project := &kubermaticv1.Project{}
	key := types.NamespacedName{Name: subjectName}
	if err := client.Get(ctx, key, project); err != nil {
		return err
	}

	projectRef := resources.GetProjectRef(project)
	refs := append(existingRefs, projectRef)
	resourceQuota.SetOwnerReferences(refs)

	return nil
}

func validateUpdate(oldResourceQuota *kubermaticv1.ResourceQuota, newResourceQuora *kubermaticv1.ResourceQuota) error {
	if !equality.Semantic.DeepEqual(oldResourceQuota.OwnerReferences, newResourceQuora.OwnerReferences) {
		return errors.New("ResourceQuota reference cannot be changed")
	}

	return nil
}
