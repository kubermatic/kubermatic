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

	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	"github.com/imdario/mergo"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud"
	"k8c.io/kubermatic/v2/pkg/resources"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/types"
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

	namespace    string
	client       ctrlruntimeclient.Client
	seedsGetter  provider.SeedsGetter
	configGetter provider.KubermaticConfigurationGetter
	caBundle     *x509.CertPool

	// disableProviderMutation is only for unit tests, to ensure no
	// provide would phone home to validate dummy test credentials
	disableProviderMutation bool
}

// NewAdmissionHandler returns a new cluster mutation AdmissionHandler.
func NewAdmissionHandler(namespace string, client ctrlruntimeclient.Client, configGetter provider.KubermaticConfigurationGetter, seedsGetter provider.SeedsGetter, caBundle *x509.CertPool) *AdmissionHandler {
	return &AdmissionHandler{
		namespace:    namespace,
		client:       client,
		configGetter: configGetter,
		seedsGetter:  seedsGetter,
		caBundle:     caBundle,
	}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/mutate-kubermatic-k8s-io-cluster", &webhook.Admission{Handler: h})
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

		err := h.mutateCreate(ctx, cluster)
		if err != nil {
			h.log.Info("cluster mutation failed", "error", err)
			return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("cluster mutation request %s failed: %v", req.UID, err))
		}

	case admissionv1.Update:
		if err := h.decoder.Decode(req, cluster); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		if err := h.decoder.DecodeRaw(req.OldObject, oldCluster); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		if err := h.mutateUpdate(oldCluster, cluster); err != nil {
			h.log.Info("cluster mutation failed", "error", err)
			return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("cluster mutation request %s failed: %v", req.UID, err))
		}

	case admissionv1.Delete:
		return webhook.Allowed(fmt.Sprintf("no mutation done for request %s", req.UID))

	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on cluster resources", req.Operation))
	}

	mutatedCluster, err := json.Marshal(cluster)
	if err != nil {
		return webhook.Errored(http.StatusInternalServerError, fmt.Errorf("marshaling cluster object failed: %v", err))
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, mutatedCluster)
}

func (h *AdmissionHandler) mutateCreate(ctx context.Context, c *kubermaticv1.Cluster) error {
	seed, provider, err := h.buildDefaultingDependencies(ctx, c)
	if err != nil {
		return err
	}

	config, err := h.configGetter(ctx)
	if err != nil {
		return err
	}

	// merge provided defaults into newly created cluster
	defaultTemplate, err := h.getDefaultingTemplate(ctx, seed, config)
	if err := mergo.Merge(&c.Spec, defaultTemplate.Spec); err != nil {
		return err
	}

	return defaulting.DefaultCreateClusterSpec(&c.Spec, seed, config, provider)
}

func (h *AdmissionHandler) getDefaultingTemplate(ctx context.Context, seed *kubermaticv1.Seed, config *operatorv1alpha1.KubermaticConfiguration) (*kubermaticv1.ClusterTemplate, error) {
	var defaultingTemplate kubermaticv1.ClusterTemplate

	if seed.Spec.DefaultClusterTemplate != "" {
		key := types.NamespacedName{Namespace: h.namespace, Name: seed.Spec.DefaultClusterTemplate}
		err := h.client.Get(ctx, key, &defaultingTemplate)
		if err != nil {
			return nil, fmt.Errorf("the configured default cluster template could not be fetched: %w", err)
		}

		if scope := defaultingTemplate.Labels["scope"]; scope != kubermaticv1.SeedTemplateScope {
			return nil, fmt.Errorf("invalid scope of default cluster template, is %q but must be %q", scope, kubermaticv1.SeedTemplateScope)
		}
	}

	defaultingTemplate.Spec.ComponentsOverride = defaultComponentSettings(seed.Spec.DefaultComponentSettings, config)

	return &defaultingTemplate, nil
}

func defaultComponentSettings(defaultComponentSettings kubermaticv1.ComponentSettings, config *operatorv1alpha1.KubermaticConfiguration) kubermaticv1.ComponentSettings {
	settings := defaultComponentSettings.DeepCopy()

	// This function uses the values from the KubermaticConfiguration, which at this point have already been defaulted
	// in case the user didn't set them explicitly, so we do not have to check and reach for the Default* constants
	// here again.

	if settings.Apiserver.Replicas == nil {
		settings.Apiserver.Replicas = config.Spec.UserCluster.APIServerReplicas
	}

	if settings.Apiserver.NodePortRange == "" {
		settings.Apiserver.NodePortRange = config.Spec.UserCluster.NodePortRange
	}

	if settings.Apiserver.EndpointReconcilingDisabled == nil && config.Spec.UserCluster.DisableAPIServerEndpointReconciling {
		settings.Apiserver.EndpointReconcilingDisabled = &config.Spec.UserCluster.DisableAPIServerEndpointReconciling
	}

	if settings.ControllerManager.Replicas == nil {
		settings.ControllerManager.Replicas = pointer.Int32Ptr(resources.DefaultControllerManagerReplicas)
	}

	if settings.Scheduler.Replicas == nil {
		settings.Scheduler.Replicas = pointer.Int32Ptr(resources.DefaultSchedulerReplicas)
	}

	return *settings
}

func (h *AdmissionHandler) mutateUpdate(oldCluster, newCluster *kubermaticv1.Cluster) error {
	// This part of the code handles the CCM/CSI migration. It currently works
	// only for OpenStack clusters, in the following way:
	//   * Add the CCM/CSI migration annotations
	//   * Enable the UseOctaiva flag
	if v, oldV := newCluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider],
		oldCluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider]; v && !oldV {

		switch {
		case newCluster.Spec.Cloud.Openstack != nil:
			addCCMCSIMigrationAnnotations(newCluster)
			newCluster.Spec.Cloud.Openstack.UseOctavia = pointer.BoolPtr(true)

		case newCluster.Spec.Cloud.VSphere != nil:
			addCCMCSIMigrationAnnotations(newCluster)
		}
	}

	// This part handles CNI upgrade from unspecified (= very old) CNI version to the default Canal version.
	// This upgrade is necessary for k8s versions >= 1.22, where v1beta1 CRDs are not supported anymore.
	if newCluster.Spec.CNIPlugin == nil {
		upgradeConstraint, err := semver.NewConstraint(">= 1.22")
		if err != nil {
			return fmt.Errorf("parsing CNI upgrade constraint failed: %v", err)
		}
		if newCluster.Spec.Version.String() != "" && upgradeConstraint.Check(newCluster.Spec.Version.Semver()) {
			newCluster.Spec.CNIPlugin = &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCanal,
				Version: defaultCNIPluginVersion[kubermaticv1.CNIPluginTypeCanal],
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

func (h *AdmissionHandler) buildDefaultingDependencies(ctx context.Context, c *kubermaticv1.Cluster) (*kubermaticv1.Seed, provider.CloudProvider, error) {
	if h.disableProviderMutation {
		return nil, nil, nil
	}

	var (
		seed          *kubermaticv1.Seed
		datacenter    *kubermaticv1.Datacenter
		cloudProvider provider.CloudProvider
	)

	datacenterName := c.Spec.Cloud.DatacenterName
	if datacenterName == "" {
		// silently ignore it here, the validation later in the validation webhook will properly report the broken spec
		return nil, nil, errors.New("no datacenter name set in spec.cloud.dc")
	}

	seeds, err := h.seedsGetter()
	if err != nil {
		return nil, nil, err
	}

	for i, seed := range seeds {
		for dcName, dc := range seed.Spec.Datacenters {
			if dcName == datacenterName {
				seed = seeds[i]
				datacenter = &dc
				break
			}
		}
	}

	if seed == nil {
		return nil, nil, fmt.Errorf("invalid datacenter %q", datacenterName)
	}

	secretKeySelectorFunc := provider.SecretKeySelectorValueFuncFactory(ctx, h.client)
	cloudProvider, err = cloud.Provider(datacenter, secretKeySelectorFunc, h.caBundle)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cloud provider: %w", err)
	}

	return seed, cloudProvider, nil
}
