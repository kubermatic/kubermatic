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
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud"
	"k8c.io/kubermatic/v2/pkg/validation"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	supportedCNIPlugins        = sets.NewString(kubermaticv1.CNIPluginTypeCanal.String(), kubermaticv1.CNIPluginTypeCilium.String(), kubermaticv1.CNIPluginTypeNone.String())
	supportedCNIPluginVersions = map[kubermaticv1.CNIPluginType]sets.String{
		kubermaticv1.CNIPluginTypeCanal:  sets.NewString("v3.8", "v3.19", "v3.20"),
		kubermaticv1.CNIPluginTypeCilium: sets.NewString("v1.10"),
		kubermaticv1.CNIPluginTypeNone:   sets.NewString(""),
	}
)

// AdmissionHandler for validating Kubermatic Cluster CRD.
type AdmissionHandler struct {
	log         logr.Logger
	decoder     *admission.Decoder
	features    features.FeatureGate
	client      ctrlruntimeclient.Client
	seedsGetter provider.SeedsGetter
	caBundle    *x509.CertPool
}

// NewAdmissionHandler returns a new cluster validation AdmissionHandler.
func NewAdmissionHandler(client ctrlruntimeclient.Client, seedsGetter provider.SeedsGetter, features features.FeatureGate, caBundle *x509.CertPool) *AdmissionHandler {
	return &AdmissionHandler{
		client:      client,
		features:    features,
		seedsGetter: seedsGetter,
		caBundle:    caBundle,
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
	datacenter, cloudProvider, allErrs := h.buildValidationDependencies(ctx, c)

	return append(allErrs, validation.ValidateCreateClusterSpec(&c.Spec, datacenter, cloudProvider, h.features)...)
}

func (h *AdmissionHandler) validateUpdate(ctx context.Context, c, oldC *kubermaticv1.Cluster) field.ErrorList {
	datacenter, cloudProvider, allErrs := h.buildValidationDependencies(ctx, c)

	return append(allErrs, validation.ValidateUpdateCluster(ctx, c, oldC, datacenter, cloudProvider, h.features)...)
}

func (h *AdmissionHandler) buildValidationDependencies(ctx context.Context, c *kubermaticv1.Cluster) (*kubermaticv1.Datacenter, provider.CloudProvider, field.ErrorList) {
	allErrs := field.ErrorList{}

	// retrieve datacenter so we can prepare the dependencies for ValidateCreateClusterSpec()
	var (
		datacenter    *kubermaticv1.Datacenter
		cloudProvider provider.CloudProvider
	)

	datacenterName := c.Spec.Cloud.DatacenterName
	if datacenterName != "" {
		seeds, err := h.seedsGetter()
		if err != nil {
			return nil, nil, field.ErrorList{field.InternalError(nil, err)}
		}

		for _, seed := range seeds {
			for dcName, dc := range seed.Spec.Datacenters {
				if dcName == datacenterName {
					datacenter = &dc
				}
			}
		}

		if datacenter == nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "cloud", "dc"), datacenterName, "invalid datacenter name"))
		} else {
			secretKeySelectorFunc := provider.SecretKeySelectorValueFuncFactory(ctx, h.client)
			cloudProvider, err = cloud.Provider(datacenter, secretKeySelectorFunc, h.caBundle)
			if err != nil {
				allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "cloud", "dc"), datacenterName, fmt.Sprintf("failed to create cloud provider: %v", err)))
			}
		}
	}

	return datacenter, cloudProvider, allErrs
}
