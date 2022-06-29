//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package resourcequota

import (
	"context"
	"encoding/json"
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

		ensureResourceQuotaLabels(resourceQuota)

		if resourceQuota.Spec.Subject.Kind != kubermaticv1.ProjectSubjectKind {
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
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on resource quota resources", req.Operation))
	}

	mutatedResourceQuota, err := json.Marshal(resourceQuota)
	if err != nil {
		return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("marshaling ResourceQuota object failed: %w", err))
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, mutatedResourceQuota)
}

func ensureProjectOwnershipRef(ctx context.Context, client ctrlruntimeclient.Client, resourceQuota *kubermaticv1.ResourceQuota) error {
	subjectName := resourceQuota.Spec.Subject.Name
	ownRefs := resourceQuota.OwnerReferences

	// check if reference already exists
	for _, owners := range ownRefs {
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
	ownRefs = append(ownRefs, projectRef)
	resourceQuota.SetOwnerReferences(ownRefs)

	return nil
}

func ensureResourceQuotaLabels(resourceQuota *kubermaticv1.ResourceQuota) {
	labels := resourceQuota.GetLabels()

	if labels == nil {
		labels = make(map[string]string)
	}

	labels[kubermaticv1.ResourceQuotaSubjectKindLabelKey] = resourceQuota.Spec.Subject.Kind
	labels[kubermaticv1.ResourceQuotaSubjectNameLabelKey] = resourceQuota.Spec.Subject.Name

	resourceQuota.SetLabels(labels)
}

func validateUpdate(oldResourceQuota *kubermaticv1.ResourceQuota, newResourceQuota *kubermaticv1.ResourceQuota) error {
	if !equality.Semantic.DeepEqual(oldResourceQuota.OwnerReferences, newResourceQuota.OwnerReferences) {
		return errors.New("ResourceQuota reference cannot be changed")
	}

	oldLabels := oldResourceQuota.GetLabels()
	newLabels := newResourceQuota.GetLabels()

	if oldLabels[kubermaticv1.ResourceQuotaSubjectKindLabelKey] != newLabels[kubermaticv1.ResourceQuotaSubjectKindLabelKey] {
		return fmt.Errorf("ResourceQuota %s label cannot be changed", kubermaticv1.ResourceQuotaSubjectKindLabelKey)
	}

	if oldLabels[kubermaticv1.ResourceQuotaSubjectNameLabelKey] != newLabels[kubermaticv1.ResourceQuotaSubjectNameLabelKey] {
		return fmt.Errorf("ResourceQuota %s label cannot be changed", kubermaticv1.ResourceQuotaSubjectNameLabelKey)
	}

	return nil
}
