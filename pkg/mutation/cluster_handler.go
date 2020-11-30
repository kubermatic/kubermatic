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
	"fmt"
	"net/http"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	"github.com/go-logr/logr"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// ClusterAdmissionHandler for Kubermatic Cluster CRD.
type ClusterAdmissionHandler struct {
	log                      logr.Logger
	decoder                  *admission.Decoder
	defaultComponentSettings *kubermaticv1.ComponentSettings
}

// NewClusterAdmissionHandler return new ClusterAdmissionHandler instance.
func NewClusterAdmissionHandler(log logr.Logger, decoder *admission.Decoder, settings *kubermaticv1.ComponentSettings) *ClusterAdmissionHandler {
	return &ClusterAdmissionHandler{log, decoder, settings}
}

// Handle handles cluster creation requests and sets defaults componentsOverride field which are not set.
func (h *ClusterAdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	cluster := &kubermaticv1.Cluster{}

	switch req.Operation {
	case admissionv1beta1.Create:
		if err := h.decoder.Decode(req, cluster); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("'%s' is not supported on cluster resources", req.Operation))
	}
	if overrides, hasUpdates := h.updates(cluster.Spec.ComponentsOverride); hasUpdates {
		fieldPath := "/spec/componentsOverride"
		return webhook.Patched("UpdatedComponentsOvverride", jsonpatch.NewOperation("remove", fieldPath, nil), jsonpatch.NewOperation("add", fieldPath, overrides))
	}

	return webhook.Allowed("CompanentsOverrideAlreadySet")
}

func (h *ClusterAdmissionHandler) updates(overrides kubermaticv1.ComponentSettings) (kubermaticv1.ComponentSettings, bool) {
	hasUpdates := false
	if overrides.Apiserver.Replicas == nil {
		hasUpdates = true
		overrides.Apiserver.Replicas = h.defaultComponentSettings.Apiserver.Replicas
	}
	if overrides.Apiserver.Resources == nil {
		hasUpdates = true
		overrides.Apiserver.Resources = h.defaultComponentSettings.Apiserver.Resources
	}
	if overrides.Apiserver.EndpointReconcilingDisabled == nil {
		hasUpdates = true
		overrides.Apiserver.EndpointReconcilingDisabled = h.defaultComponentSettings.Apiserver.EndpointReconcilingDisabled
	}
	if overrides.ControllerManager.Replicas == nil {
		hasUpdates = true
		overrides.ControllerManager.Replicas = h.defaultComponentSettings.ControllerManager.Replicas
	}
	if overrides.ControllerManager.Resources == nil {
		hasUpdates = true
		overrides.ControllerManager.Resources = h.defaultComponentSettings.ControllerManager.Resources
	}
	if overrides.Scheduler.Replicas == nil {
		hasUpdates = true
		overrides.Scheduler.Replicas = h.defaultComponentSettings.Scheduler.Replicas
	}
	if overrides.Scheduler.Resources == nil {
		hasUpdates = true
		overrides.Scheduler.Resources = h.defaultComponentSettings.Scheduler.Resources
	}
	if overrides.Etcd.Replicas == nil {
		hasUpdates = true
		overrides.Etcd.Replicas = h.defaultComponentSettings.Etcd.Replicas
	}
	if overrides.Etcd.Resources == nil {
		hasUpdates = true
		overrides.Etcd.Resources = h.defaultComponentSettings.Etcd.Resources
	}
	if overrides.Prometheus.Resources == nil {
		hasUpdates = true
		overrides.Prometheus.Resources = h.defaultComponentSettings.Prometheus.Resources
	}
	return overrides, hasUpdates
}

// SetupWebhookWithManager sets webhook into webhook server.
func (h *ClusterAdmissionHandler) SetupWebhookWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register("/default-components-settings", &webhook.Admission{Handler: h})
}
