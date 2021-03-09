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

package validation

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/validation"

	admissionv1 "k8s.io/api/admission/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for validating Kubermatic Cluster CRD.
type AdmissionHandler struct {
	log      logr.Logger
	decoder  *admission.Decoder
	features features.FeatureGate
	client   ctrlruntimeclient.Client
}

// NewAdmissionHandler returns a new cluster validation AdmissionHandler.
func NewAdmissionHandler(client ctrlruntimeclient.Client, features features.FeatureGate) *AdmissionHandler {
	return &AdmissionHandler{
		features: features,
		client:   client,
	}
}

func (h *AdmissionHandler) InjectLogger(l logr.Logger) error {
	h.log = l.WithName("cluster-validation-handler")
	return nil
}

func (h *AdmissionHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

func (h *AdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	cluster := &kubermaticv1.Cluster{}

	switch req.Operation {
	case admissionv1.Create:
		fallthrough
	case admissionv1.Update:
		err := h.decoder.Decode(req, cluster)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		validationErr := h.validateCreateOrUpdate(ctx, cluster)
		if validationErr != nil {
			h.log.Info("cluster admission failed", "error", validationErr)
			return webhook.Denied(fmt.Sprintf("cluster validation request %s rejected: %v", req.UID, validationErr))
		}

		immutabilityErr := h.validateUpdateImmutability(req, cluster)
		if immutabilityErr != nil {
			h.log.Info("cluster admission failed", "error", immutabilityErr)
			return webhook.Denied(fmt.Sprintf("cluster validation request %s rejected: %v", req.UID, immutabilityErr))
		}
	case admissionv1.Delete:
		// NOP we always allow delete operarions at the moment
	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on cluster resources", req.Operation))
	}
	return webhook.Allowed(fmt.Sprintf("cluster validation request %s allowed", req.UID))
}

func (h *AdmissionHandler) validateCreateOrUpdate(ctx context.Context, c *kubermaticv1.Cluster) error {
	if !kubermaticv1.AllExposeStrategies.Has(c.Spec.ExposeStrategy) {
		return fmt.Errorf("unknown expose strategy %q, use one between: %s", c.Spec.ExposeStrategy, kubermaticv1.AllExposeStrategies)
	}
	if c.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyTunneling &&
		!h.features.Enabled(features.TunnelingExposeStrategy) {
		return errors.New("cannot create cluster with Tunneling expose strategy, the TunnelingExposeStrategy feature gate is not enabled")
	}
	if err := validation.ValidateLeaderElectionSettings(c.Spec.ComponentsOverride.ControllerManager.LeaderElectionSettings); err != nil {
		return fmt.Errorf("controller manager leader election settings are not valid: %w", err)
	}
	if err := validation.ValidateLeaderElectionSettings(c.Spec.ComponentsOverride.Scheduler.LeaderElectionSettings); err != nil {
		return fmt.Errorf("scheduler leader election settings are not valid: %w", err)
	}

	if err := h.rejectUserSSHKeyAgentChanges(ctx, c); err != nil {
		h.log.Info("cluster admission failed", "error", err)
		return err
	}

	return nil
}

func (h *AdmissionHandler) validateUpdateImmutability(req webhook.AdmissionRequest, c *kubermaticv1.Cluster) error {
	// Immutability should be validated only for update requests
	if req.Operation != admissionv1.Update {
		return nil
	}
	oldCluster := &kubermaticv1.Cluster{}
	if err := h.decoder.DecodeRaw(req.OldObject, oldCluster); err != nil {
		return fmt.Errorf("failed to decode old cluster object: %v", err)
	}

	// Validate ExternalCloudProvider feature flag immutability.
	// Once the feature flag is enabled, it must not be disabled.
	if vOld, v := oldCluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider],
		c.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider]; vOld && !v {
		return fmt.Errorf("feature gate %q cannot be disabled once it's enabled", kubermaticv1.ClusterFeatureExternalCloudProvider)
	}

	return nil
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/validate-kubermatic-k8s-io-cluster", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) rejectUserSSHKeyAgentChanges(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	var (
		oldCluster = &kubermaticv1.Cluster{}
		nName      = types.NamespacedName{Name: cluster.Name}
	)

	if h.client != nil {
		if err := h.client.Get(ctx, nName, oldCluster); err != nil {
			if kerrors.IsNotFound(err) {
				return nil
			}

			return fmt.Errorf("failed to fetch cluster name=%s: %v", cluster.Name, err)
		}

		if oldCluster.Spec.EnableUserSSHKeyAgent != cluster.Spec.EnableUserSSHKeyAgent {
			return errors.New("enableUserSSHKeyAgent field cannot be updated after cluster creation")
		}
	}

	return nil
}
