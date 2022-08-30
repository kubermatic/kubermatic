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
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud"
	"k8c.io/kubermatic/v2/pkg/version/cni"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for mutating Kubermatic Cluster CRD.
type AdmissionHandler struct {
	log     logr.Logger
	decoder *admission.Decoder

	client       ctrlruntimeclient.Client
	seedGetter   provider.SeedGetter
	configGetter provider.KubermaticConfigurationGetter
	caBundle     *x509.CertPool

	// disableProviderMutation is only for unit tests, to ensure no
	// provider would phone home to validate dummy test credentials
	disableProviderMutation bool
}

// NewAdmissionHandler returns a new cluster AdmissionHandler.
func NewAdmissionHandler(client ctrlruntimeclient.Client, configGetter provider.KubermaticConfigurationGetter, seedGetter provider.SeedGetter, caBundle *x509.CertPool) *AdmissionHandler {
	return &AdmissionHandler{
		client:       client,
		configGetter: configGetter,
		seedGetter:   seedGetter,
		caBundle:     caBundle,
	}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/mutate-kubermatic-k8c-io-v1-cluster", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) InjectLogger(l logr.Logger) error {
	h.log = l.WithName("cluster-mutation-handler")
	return nil
}

func (h *AdmissionHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

func (h *AdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	cluster := &kubermaticv1.Cluster{}
	oldCluster := &kubermaticv1.Cluster{}

	switch req.Operation {
	case admissionv1.Create:
		if err := h.decoder.Decode(req, cluster); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		err := h.applyDefaults(ctx, cluster)
		if err != nil {
			h.log.Error(err, "cluster mutation failed")
			return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("cluster mutation request %s failed: %w", req.UID, err))
		}

		if err := h.mutateCreate(cluster); err != nil {
			h.log.Error(err, "cluster mutation failed")
			return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("cluster mutation request %s failed: %w", req.UID, err))
		}

	case admissionv1.Update:
		if err := h.decoder.Decode(req, cluster); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		if err := h.decoder.DecodeRaw(req.OldObject, oldCluster); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		if cluster.DeletionTimestamp == nil {
			// apply defaults to the existing clusters
			err := h.applyDefaults(ctx, cluster)
			if err != nil {
				h.log.Error(err, "cluster mutation failed")
				return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("cluster mutation request %s failed: %w", req.UID, err))
			}

			if err := h.mutateUpdate(oldCluster, cluster); err != nil {
				h.log.Error(err, "cluster mutation failed")
				return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("cluster mutation request %s failed: %w", req.UID, err))
			}
		}

	case admissionv1.Delete:
		return webhook.Allowed(fmt.Sprintf("no mutation done for request %s", req.UID))

	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on cluster resources", req.Operation))
	}

	mutatedCluster, err := json.Marshal(cluster)
	if err != nil {
		return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("marshaling cluster object failed: %w", err))
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, mutatedCluster)
}

func (h *AdmissionHandler) applyDefaults(ctx context.Context, c *kubermaticv1.Cluster) error {
	seed, provider, fieldErr := h.buildDefaultingDependencies(ctx, c)
	if fieldErr != nil {
		return fieldErr
	}

	config, err := h.configGetter(ctx)
	if err != nil {
		return err
	}

	defaultTemplate, err := defaulting.GetDefaultingClusterTemplate(ctx, h.client, seed)
	if err != nil {
		return err
	}

	return defaulting.DefaultClusterSpec(ctx, &c.Spec, defaultTemplate, seed, config, provider)
}

// mutateCreate is an addition to regular defaulting for new clusters.
func (h *AdmissionHandler) mutateCreate(newCluster *kubermaticv1.Cluster) error {
	if newCluster.Spec.Features == nil {
		newCluster.Spec.Features = map[string]bool{}
	}

	// Network policies for Apiserver are deployed by default
	if _, ok := newCluster.Spec.Features[kubermaticv1.ApiserverNetworkPolicy]; !ok {
		newCluster.Spec.Features[kubermaticv1.ApiserverNetworkPolicy] = true
	}

	return nil
}

func (h *AdmissionHandler) mutateUpdate(oldCluster, newCluster *kubermaticv1.Cluster) error {
	// This part of the code handles the CCM/CSI migration. It currently works
	// for OpenStack, vSphere and Azure clusters, in the following way:
	//   * Add the CCM/CSI migration annotations
	//   * Enable the UseOctavia flag (for OpenStack only)
	if v, oldV := newCluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider],
		oldCluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider]; v && !oldV {
		switch {
		case newCluster.Spec.Cloud.Openstack != nil:
			addCCMCSIMigrationAnnotations(newCluster)
			newCluster.Spec.Cloud.Openstack.UseOctavia = pointer.BoolPtr(true)

		case newCluster.Spec.Cloud.VSphere != nil:
			addCCMCSIMigrationAnnotations(newCluster)

		case newCluster.Spec.Cloud.Azure != nil:
			addCCMCSIMigrationAnnotations(newCluster)
		}
	}

	// just because spec.Version might say 1.23 doesn't say that the cluster is already on 1.23,
	// so for all feature toggles and migrations we should base this on the actual, current apiserver
	curVersion := newCluster.Status.Versions.ControlPlane
	if curVersion == "" {
		curVersion = newCluster.Spec.Version
	}

	if newCluster.Spec.CNIPlugin.Type == kubermaticv1.CNIPluginTypeCanal {
		// This part handles CNI upgrade from unsupported CNI version to the default Canal version.
		// This upgrade is necessary for k8s versions >= 1.22, where v1beta1 CRDs used in old Canal version (v3.8)
		// are not supported anymore.
		if newCluster.Spec.CNIPlugin.Version == cni.CanalCNILastUnspecifiedVersion {
			upgradeConstraint, err := semverlib.NewConstraint(">= 1.22")
			if err != nil {
				return fmt.Errorf("parsing CNI upgrade constraint failed: %w", err)
			}
			if curVersion.String() != "" && upgradeConstraint.Check(curVersion.Semver()) {
				newCluster.Spec.CNIPlugin = &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: cni.GetDefaultCNIPluginVersion(kubermaticv1.CNIPluginTypeCanal),
				}
			}
		}

		// This part handles Canal version upgrade for clusters with Kubernetes version 1.23 and higher,
		// where the minimal Canal version is v3.22.
		cniVersion, err := semverlib.NewVersion(newCluster.Spec.CNIPlugin.Version)
		if err != nil {
			return fmt.Errorf("CNI plugin version parsing failed: %w", err)
		}
		lowerThan322, err := semverlib.NewConstraint("< 3.22")
		if err != nil {
			return fmt.Errorf("semver constraint parsing failed: %w", err)
		}
		equalOrHigherThan123, err := semverlib.NewConstraint(">= 1.23")
		if err != nil {
			return fmt.Errorf("semver constraint parsing failed: %w", err)
		}
		if lowerThan322.Check(cniVersion) && curVersion.String() != "" && equalOrHigherThan123.Check(curVersion.Semver()) {
			newCluster.Spec.CNIPlugin = &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCanal,
				Version: "v3.22",
			}
		}
	}
	return nil
}

func addCCMCSIMigrationAnnotations(cluster *kubermaticv1.Cluster) {
	if cluster.ObjectMeta.Annotations == nil {
		cluster.ObjectMeta.Annotations = map[string]string{}
	}

	cluster.ObjectMeta.Annotations[kubermaticv1.CCMMigrationNeededAnnotation] = ""
	cluster.ObjectMeta.Annotations[kubermaticv1.CSIMigrationNeededAnnotation] = ""
}

func (h *AdmissionHandler) buildDefaultingDependencies(ctx context.Context, c *kubermaticv1.Cluster) (*kubermaticv1.Seed, provider.CloudProvider, *field.Error) {
	seed, err := h.seedGetter()
	if err != nil {
		return nil, nil, field.InternalError(nil, err)
	}
	if seed == nil {
		return nil, nil, field.InternalError(nil, errors.New("webhook is not configured with -seed-name, cannot validate Clusters"))
	}

	if h.disableProviderMutation {
		return seed, nil, nil
	}

	datacenter, fieldErr := defaulting.DatacenterForClusterSpec(&c.Spec, seed)
	if fieldErr != nil {
		return nil, nil, fieldErr
	}

	secretKeySelectorFunc := provider.SecretKeySelectorValueFuncFactory(ctx, h.client)
	cloudProvider, err := cloud.Provider(datacenter, secretKeySelectorFunc, h.caBundle)
	if err != nil {
		return nil, nil, field.InternalError(nil, err)
	}

	return seed, cloudProvider, nil
}
