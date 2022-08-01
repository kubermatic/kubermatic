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
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for mutating Kubermatic Addon CRD.
type AdmissionHandler struct {
	log              logr.Logger
	decoder          *admission.Decoder
	seedGetter       provider.SeedGetter
	seedClientGetter provider.SeedClientGetter
}

// NewAdmissionHandler returns a new Addon AdmissionHandler.
func NewAdmissionHandler(seedGetter provider.SeedGetter, seedClientGetter provider.SeedClientGetter) *AdmissionHandler {
	return &AdmissionHandler{
		seedGetter:       seedGetter,
		seedClientGetter: seedClientGetter,
	}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/mutate-kubermatic-k8c-io-v1-addon", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) InjectLogger(l logr.Logger) error {
	h.log = l.WithName("addon-mutation-handler")
	return nil
}

func (h *AdmissionHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

func (h *AdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	addon := &kubermaticv1.Addon{}

	switch req.Operation {
	case admissionv1.Create:
		if err := h.decoder.Decode(req, addon); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		// apply defaults to the existing addon
		err := h.ensureClusterReference(ctx, addon)
		if err != nil {
			h.log.Error(err, "addon mutation failed")
			return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("addon mutation request %s failed: %w", req.UID, err))
		}

	case admissionv1.Update:
		oldAddon := &kubermaticv1.Addon{}

		if err := h.decoder.Decode(req, addon); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		if err := h.decoder.DecodeRaw(req.OldObject, oldAddon); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		err := h.validateUpdate(ctx, oldAddon, addon)
		if err != nil {
			h.log.Error(err, "addon mutation failed")
			return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("addon mutation request %s failed: %w", req.UID, err))
		}

	case admissionv1.Delete:
		return webhook.Allowed(fmt.Sprintf("no mutation done for request %s", req.UID))

	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on addon resources", req.Operation))
	}

	mutatedAddon, err := json.Marshal(addon)
	if err != nil {
		return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("marshaling addon object failed: %w", err))
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, mutatedAddon)
}

func (h *AdmissionHandler) ensureClusterReference(ctx context.Context, addon *kubermaticv1.Addon) error {
	seed, err := h.seedGetter()
	if err != nil {
		return fmt.Errorf("failed to get current Seed: %w", err)
	}
	if seed == nil {
		return errors.New("webhook not configured for a Seed cluster, cannot validate Addon resources")
	}

	client, err := h.seedClientGetter(seed)
	if err != nil {
		return fmt.Errorf("failed to get Seed client: %w", err)
	}

	cluster, err := kubernetes.ClusterFromNamespace(ctx, client, addon.Namespace)
	if err != nil {
		return fmt.Errorf("failed to list Cluster objects: %w", err)
	}

	if cluster == nil {
		return errors.New("Addons can only be created in cluster namespaces, but no matching Cluster was found")
	}

	if cluster.DeletionTimestamp != nil {
		return fmt.Errorf("Cluster %s is in deletion already, cannot create a new addon", cluster.Name)
	}

	addon.Spec.Cluster = corev1.ObjectReference{
		Name:       cluster.Name,
		Namespace:  "",
		UID:        cluster.UID,
		APIVersion: cluster.APIVersion,
		Kind:       "Cluster",
	}

	return nil
}

func (h *AdmissionHandler) validateUpdate(ctx context.Context, oldAddon *kubermaticv1.Addon, newAddon *kubermaticv1.Addon) error {
	if !equality.Semantic.DeepEqual(oldAddon.Spec.Cluster, newAddon.Spec.Cluster) {
		return errors.New("Cluster reference cannot be changed")
	}

	return nil
}
