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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/utils/pointer"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for mutating Kubermatic Cluster CRD.
type AdmissionHandler struct {
	log     logr.Logger
	decoder *admission.Decoder

	defaultComponentSettings kubermaticv1.ComponentSettings
}

// NewAdmissionHandler returns a new cluster mutation AdmissionHandler.
func NewAdmissionHandler(defaults kubermaticv1.ComponentSettings) *AdmissionHandler {
	return &AdmissionHandler{
		defaultComponentSettings: defaults,
	}
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
		h.applyDefaults(cluster)
	case admissionv1.Update:
		if err := h.decoder.Decode(req, cluster); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		if err := h.decoder.DecodeRaw(req.OldObject, oldCluster); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		if err := h.mutateUpdate(ctx, oldCluster, cluster); err != nil {
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

func (h *AdmissionHandler) applyDefaults(c *kubermaticv1.Cluster) {
	// Add default CNI plugin settings if not present.
	if c.Spec.CNIPlugin == nil {
		c.Spec.CNIPlugin = &kubermaticv1.CNIPluginSettings{
			Type:    kubermaticv1.CNIPluginTypeCanal,
			Version: "v3.19",
		}
	}

	if len(c.Spec.ClusterNetwork.Services.CIDRBlocks) == 0 {
		if c.Spec.Cloud.Kubevirt != nil {
			// KubeVirt cluster can be provisioned on top of k8s cluster created by KKP
			// thus we have to avoid network collision
			c.Spec.ClusterNetwork.Services.CIDRBlocks = []string{"10.241.0.0/20"}
		} else {
			c.Spec.ClusterNetwork.Services.CIDRBlocks = []string{"10.240.16.0/20"}
		}
	}

	if len(c.Spec.ClusterNetwork.Pods.CIDRBlocks) == 0 {
		if c.Spec.Cloud.Kubevirt != nil {
			c.Spec.ClusterNetwork.Pods.CIDRBlocks = []string{"172.26.0.0/16"}
		} else {
			c.Spec.ClusterNetwork.Pods.CIDRBlocks = []string{"172.25.0.0/16"}
		}
	}

	if c.Spec.ClusterNetwork.DNSDomain == "" {
		c.Spec.ClusterNetwork.DNSDomain = "cluster.local"
	}

	if c.Spec.ClusterNetwork.ProxyMode == "" {
		// IPVS causes issues with Hetzner's LoadBalancers, which should
		// be addressed via https://github.com/kubernetes/enhancements/pull/1392
		if c.Spec.Cloud.Hetzner != nil {
			c.Spec.ClusterNetwork.ProxyMode = resources.IPTablesProxyMode
		} else {
			c.Spec.ClusterNetwork.ProxyMode = resources.IPVSProxyMode
		}
	}

	if c.Spec.ClusterNetwork.IPVS != nil {
		if c.Spec.ClusterNetwork.IPVS.StrictArp == nil {
			c.Spec.ClusterNetwork.IPVS.StrictArp = pointer.BoolPtr(resources.IPVSStrictArp)
		}
	}

	if c.Spec.ClusterNetwork.NodeLocalDNSCacheEnabled == nil {
		c.Spec.ClusterNetwork.NodeLocalDNSCacheEnabled = pointer.BoolPtr(true)
	}

	// Default component settings
	h.defaultClusterComponentSettings(c)
}

func (h *AdmissionHandler) defaultClusterComponentSettings(c *kubermaticv1.Cluster) {
	if c.Spec.ComponentsOverride.Apiserver.Replicas == nil {
		c.Spec.ComponentsOverride.Apiserver.Replicas = h.defaultComponentSettings.Apiserver.Replicas
	}
	if c.Spec.ComponentsOverride.Apiserver.Resources == nil {
		c.Spec.ComponentsOverride.Apiserver.Resources = h.defaultComponentSettings.Apiserver.Resources
	}
	if c.Spec.ComponentsOverride.Apiserver.EndpointReconcilingDisabled == nil {
		c.Spec.ComponentsOverride.Apiserver.EndpointReconcilingDisabled = h.defaultComponentSettings.Apiserver.EndpointReconcilingDisabled
	}
	if c.Spec.ComponentsOverride.Apiserver.NodePortRange == "" {
		c.Spec.ComponentsOverride.Apiserver.NodePortRange = h.defaultComponentSettings.Apiserver.NodePortRange
	}
	if c.Spec.ComponentsOverride.ControllerManager.Replicas == nil {
		c.Spec.ComponentsOverride.ControllerManager.Replicas = h.defaultComponentSettings.ControllerManager.Replicas
	}
	if c.Spec.ComponentsOverride.ControllerManager.Resources == nil {
		c.Spec.ComponentsOverride.ControllerManager.Resources = h.defaultComponentSettings.ControllerManager.Resources
	}
	if c.Spec.ComponentsOverride.ControllerManager.Tolerations == nil {
		c.Spec.ComponentsOverride.ControllerManager.Tolerations = h.defaultComponentSettings.ControllerManager.Tolerations
	}
	if c.Spec.ComponentsOverride.ControllerManager.LeaseDurationSeconds == nil {
		c.Spec.ComponentsOverride.ControllerManager.LeaseDurationSeconds = h.defaultComponentSettings.ControllerManager.LeaseDurationSeconds
	}
	if c.Spec.ComponentsOverride.ControllerManager.RenewDeadlineSeconds == nil {
		c.Spec.ComponentsOverride.ControllerManager.RenewDeadlineSeconds = h.defaultComponentSettings.ControllerManager.RenewDeadlineSeconds
	}
	if c.Spec.ComponentsOverride.ControllerManager.RetryPeriodSeconds == nil {
		c.Spec.ComponentsOverride.ControllerManager.RetryPeriodSeconds = h.defaultComponentSettings.ControllerManager.RetryPeriodSeconds
	}
	if c.Spec.ComponentsOverride.Scheduler.Replicas == nil {
		c.Spec.ComponentsOverride.Scheduler.Replicas = h.defaultComponentSettings.Scheduler.Replicas
	}
	if c.Spec.ComponentsOverride.Scheduler.Resources == nil {
		c.Spec.ComponentsOverride.Scheduler.Resources = h.defaultComponentSettings.Scheduler.Resources
	}
	if c.Spec.ComponentsOverride.Scheduler.Tolerations == nil {
		c.Spec.ComponentsOverride.Scheduler.Tolerations = h.defaultComponentSettings.Scheduler.Tolerations
	}
	if c.Spec.ComponentsOverride.Scheduler.LeaseDurationSeconds == nil {
		c.Spec.ComponentsOverride.Scheduler.LeaseDurationSeconds = h.defaultComponentSettings.Scheduler.LeaseDurationSeconds
	}
	if c.Spec.ComponentsOverride.Scheduler.RenewDeadlineSeconds == nil {
		c.Spec.ComponentsOverride.Scheduler.RenewDeadlineSeconds = h.defaultComponentSettings.Scheduler.RenewDeadlineSeconds
	}
	if c.Spec.ComponentsOverride.Scheduler.RetryPeriodSeconds == nil {
		c.Spec.ComponentsOverride.Scheduler.RetryPeriodSeconds = h.defaultComponentSettings.Scheduler.RetryPeriodSeconds
	}
	if c.Spec.ComponentsOverride.Etcd.ClusterSize == nil {
		c.Spec.ComponentsOverride.Etcd.ClusterSize = h.defaultComponentSettings.Etcd.ClusterSize
	}
	if c.Spec.ComponentsOverride.Etcd.Resources == nil {
		c.Spec.ComponentsOverride.Etcd.Resources = h.defaultComponentSettings.Etcd.Resources
	}
	if c.Spec.ComponentsOverride.Etcd.Tolerations == nil {
		c.Spec.ComponentsOverride.Etcd.Tolerations = h.defaultComponentSettings.Etcd.Tolerations
	}
	if c.Spec.ComponentsOverride.Etcd.DiskSize == nil {
		c.Spec.ComponentsOverride.Etcd.DiskSize = h.defaultComponentSettings.Etcd.DiskSize
	}
	if c.Spec.ComponentsOverride.Etcd.StorageClass == "" {
		c.Spec.ComponentsOverride.Etcd.StorageClass = h.defaultComponentSettings.Etcd.StorageClass
	}
	if c.Spec.ComponentsOverride.Prometheus.Resources == nil {
		c.Spec.ComponentsOverride.Prometheus.Resources = h.defaultComponentSettings.Prometheus.Resources
	}
}

func (h *AdmissionHandler) mutateUpdate(ctx context.Context, oldCluster, newCluster *kubermaticv1.Cluster) error {
	// This part of the code handles the CCM/CSI migration. It currently works
	// only for OpenStack clusters, in the following way:
	//   * Add the CCM/CSI migration annotations
	//   * Enable the UseOctaiva flag
	switch {
	case newCluster.Spec.Cloud.Openstack != nil:
		if v, oldV := newCluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider],
			oldCluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider]; v && !oldV {
			if newCluster.ObjectMeta.Annotations == nil {
				newCluster.ObjectMeta.Annotations = map[string]string{}
			}

			newCluster.ObjectMeta.Annotations[kubermaticv1.CCMMigrationNeededAnnotation] = ""
			newCluster.ObjectMeta.Annotations[kubermaticv1.CSIMigrationNeededAnnotation] = ""
			newCluster.Spec.Cloud.Openstack.UseOctavia = pointer.BoolPtr(true)
		}
	}

	return nil
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/mutate-kubermatic-k8s-io-cluster", &webhook.Admission{Handler: h})
}
