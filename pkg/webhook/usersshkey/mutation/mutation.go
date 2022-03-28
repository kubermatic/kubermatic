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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/kubernetes"

	admissionv1 "k8s.io/api/admission/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for mutating Kubermatic Cluster CRD.
type AdmissionHandler struct {
	client  ctrlruntimeclient.Client
	log     logr.Logger
	decoder *admission.Decoder
}

// NewAdmissionHandler returns a new UserSSHKey AdmissionHandler.
func NewAdmissionHandler(client ctrlruntimeclient.Client) *AdmissionHandler {
	return &AdmissionHandler{
		client: client,
	}
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

		if err := h.applyDefaults(ctx, sshKey, nil); err != nil {
			h.log.Info("usersshkey mutation failed", "error", err)
			return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("usersshkey mutation request %s failed: %w", req.UID, err))
		}

		if err := h.ensureProjectRelation(ctx, sshKey, nil); err != nil {
			return webhook.Errored(http.StatusBadRequest, err)
		}

	case admissionv1.Update:
		if err := h.decoder.Decode(req, sshKey); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		if err := h.decoder.DecodeRaw(req.OldObject, oldKey); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		if err := h.applyDefaults(ctx, sshKey, oldKey); err != nil {
			h.log.Info("usersshkey mutation failed", "error", err)
			return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("usersshkey mutation request %s failed: %w", req.UID, err))
		}

		if err := h.ensureProjectRelation(ctx, sshKey, oldKey); err != nil {
			return webhook.Errored(http.StatusBadRequest, err)
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

func (h *AdmissionHandler) ensureProjectRelation(ctx context.Context, key *kubermaticv1.UserSSHKey, oldKey *kubermaticv1.UserSSHKey) error {
	isUpdate := oldKey != nil

	if isUpdate && key.Spec.Project != oldKey.Spec.Project {
		return errors.New("cannot change the project for an UserSSHKey object")
	}

	if key.Spec.Project == "" {
		return errors.New("project name must be configured")
	}

	project := &kubermaticv1.Project{}
	if err := h.client.Get(ctx, types.NamespacedName{Name: key.Spec.Project}, project); err != nil {
		if kerrors.IsNotFound(err) {
			// during key creation, we enforce the project association;
			// during updates we are more relaxed and only require that the association isn't changed,
			// so that if a project gets removed before the key (for whatever reason), then
			// the cluster cleanup can still progress and is not blocked by the webhook
			if isUpdate {
				return nil
			}

			return errors.New("no such project exists")
		}

		return fmt.Errorf("failed to get project: %w", err)
	}

	// Do not check the project phase, as projects only get Active after being successfully
	// reconciled. This requires the owner user to be setup properly as well, which in turn
	// requires owner references to be setup. All of this is super annoying when doing
	// GitOps. Instead we rely on _eventual_ consistency and only check that the project
	// exists and is not being deleted.
	if !isUpdate && project.DeletionTimestamp != nil {
		return errors.New("project is in deletion, cannot create new clusters in it")
	}

	// ensure the key has exactly 1 OwnerRef to a Project and that this ownerRef points to the
	// correct Project
	ownerRefs := []metav1.OwnerReference{}
	apiVersion := kubermaticv1.SchemeGroupVersion.String()
	kind := kubermaticv1.ProjectKindName

	for _, ref := range key.OwnerReferences {
		// skip all project owner refs
		if ref.Kind == kind && ref.APIVersion == apiVersion {
			continue
		}

		ownerRefs = append(ownerRefs, ref)
	}

	ownerRefs = append(ownerRefs, metav1.OwnerReference{
		APIVersion: apiVersion,
		Kind:       kind,
		UID:        project.UID,
		Name:       project.Name,
	})

	kubernetes.SortOwnerReferences(ownerRefs)
	key.SetOwnerReferences(ownerRefs)

	return nil
}
