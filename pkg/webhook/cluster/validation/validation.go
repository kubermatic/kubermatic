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
	"crypto/x509"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud"
	"k8c.io/kubermatic/v2/pkg/validation"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for validating Kubermatic Cluster CRD.
type AdmissionHandler struct {
	log        logr.Logger
	decoder    *admission.Decoder
	features   features.FeatureGate
	client     ctrlruntimeclient.Client
	seedGetter provider.SeedGetter
	caBundle   *x509.CertPool

	// disableProviderValidation is only for unit tests, to ensure no
	// provide would phone home to validate dummy test credentials
	disableProviderValidation bool
}

// NewAdmissionHandler returns a new cluster validation AdmissionHandler.
func NewAdmissionHandler(client ctrlruntimeclient.Client, seedGetter provider.SeedGetter, features features.FeatureGate, caBundle *x509.CertPool) *AdmissionHandler {
	return &AdmissionHandler{
		client:     client,
		features:   features,
		seedGetter: seedGetter,
		caBundle:   caBundle,
	}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/validate-kubermatic-k8s-io-cluster", &webhook.Admission{Handler: h})
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
	allErrs := field.ErrorList{}
	cluster := &kubermaticv1.Cluster{}
	oldCluster := &kubermaticv1.Cluster{}

	switch req.Operation {
	case admissionv1.Create:
		if err := h.decoder.Decode(req, cluster); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		allErrs = append(allErrs, h.validateCreate(ctx, cluster)...)

	case admissionv1.Update:
		if err := h.decoder.Decode(req, cluster); err != nil {
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("error occurred while decoding cluster: %w", err))
		}
		if err := h.decoder.DecodeRaw(req.OldObject, oldCluster); err != nil {
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("error occurred while decoding old cluster: %w", err))
		}
		allErrs = append(allErrs, h.validateUpdate(ctx, cluster, oldCluster)...)

	case admissionv1.Delete:
		// NOP we always allow delete operarions at the moment

	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on cluster resources", req.Operation))
	}

	if len(allErrs) > 0 {
		return webhook.Denied(fmt.Sprintf("cluster validation request %s denied: %v", req.UID, allErrs))
	}

	return webhook.Allowed(fmt.Sprintf("cluster validation request %s allowed", req.UID))
}

func (h *AdmissionHandler) validateCreate(ctx context.Context, c *kubermaticv1.Cluster) field.ErrorList {
	datacenter, cloudProvider, err := h.buildValidationDependencies(ctx, c)
	if err != nil {
		return field.ErrorList{err}
	}

	return validation.ValidateNewClusterSpec(&c.Spec, datacenter, cloudProvider, h.features, nil)
}

func (h *AdmissionHandler) validateUpdate(ctx context.Context, c, oldC *kubermaticv1.Cluster) field.ErrorList {
	datacenter, cloudProvider, err := h.buildValidationDependencies(ctx, c)
	if err != nil {
		return field.ErrorList{err}
	}

	return validation.ValidateClusterUpdate(ctx, c, oldC, datacenter, cloudProvider, h.features)
}

func (h *AdmissionHandler) buildValidationDependencies(ctx context.Context, c *kubermaticv1.Cluster) (*kubermaticv1.Datacenter, provider.CloudProvider, *field.Error) {
	seed, err := h.seedGetter()
	if err != nil {
		return nil, nil, field.InternalError(nil, err)
	}

	datacenter, fieldErr := defaulting.DatacenterForClusterSpec(&c.Spec, seed)
	if fieldErr != nil {
		return nil, nil, fieldErr
	}

	if h.disableProviderValidation {
		return datacenter, nil, nil
	}

	secretKeySelectorFunc := provider.SecretKeySelectorValueFuncFactory(ctx, h.client)
	cloudProvider, err := cloud.Provider(datacenter, secretKeySelectorFunc, h.caBundle)
	if err != nil {
		return nil, nil, field.InternalError(nil, err)
	}

	return datacenter, cloudProvider, nil
}
