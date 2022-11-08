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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const GlobalSettingsName = "globalsettings"

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// KubermaticSetting is the type representing a KubermaticSetting.
// These settings affect the KKP dashboard and are not relevant when
// using the Kube API on the master/seed clusters directly.
type KubermaticSetting struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SettingSpec `json:"spec,omitempty"`
}

type SettingSpec struct {
	// CustomLinks are additional links that can be shown the dashboard's footer.
	CustomLinks CustomLinks `json:"customLinks"`
	// DefaultNodeCount is the default number of replicas for the initial MachineDeployment.
	DefaultNodeCount int8 `json:"defaultNodeCount"`
	// DisplayDemoInfo controls whether a "Demo System" hint is shown in the footer.
	DisplayDemoInfo bool `json:"displayDemoInfo"`
	// DisplayDemoInfo controls whether a a link to the KKP API documentation is shown in the footer.
	DisplayAPIDocs bool `json:"displayAPIDocs"`
	// DisplayDemoInfo controls whether a a link to TOS is shown in the footer.
	DisplayTermsOfService bool `json:"displayTermsOfService"`
	// EnableDashboard enables the link to the Kubernetes dashboard for a user cluster.
	EnableDashboard bool `json:"enableDashboard"`

	// +kubebuilder:default=false

	// EnableWebTerminal enables the Web Terminal feature for the user clusters.
	EnableWebTerminal bool `json:"enableWebTerminal"`

	EnableOIDCKubeconfig bool `json:"enableOIDCKubeconfig"` //nolint:tagliatelle
	// UserProjectsLimit is the maximum number of projects a user can create.
	UserProjectsLimit           int64 `json:"userProjectsLimit"`
	RestrictProjectCreation     bool  `json:"restrictProjectCreation"`
	EnableExternalClusterImport bool  `json:"enableExternalClusterImport"`

	// CleanupOptions control what happens when a cluster is deleted via the dashboard.
	// +optional
	CleanupOptions CleanupOptions `json:"cleanupOptions,omitempty"`
	// +optional
	OpaOptions OpaOptions `json:"opaOptions,omitempty"`
	// +optional
	MlaOptions MlaOptions `json:"mlaOptions,omitempty"`

	MlaAlertmanagerPrefix string `json:"mlaAlertmanagerPrefix"`
	MlaGrafanaPrefix      string `json:"mlaGrafanaPrefix"`

	// Notifications are the configuration for notifications on dashboard.
	// +optional
	Notifications NotificationsOptions `json:"notifications,omitempty"`

	// MachineDeploymentVMResourceQuota is used to filter out allowed machine flavors based on the specified resource limits like CPU, Memory, and GPU etc.
	MachineDeploymentVMResourceQuota *MachineFlavorFilter `json:"machineDeploymentVMResourceQuota,omitempty"`

	// TODO: Datacenters, presets, user management, Google Analytics and default addons.
}

type CustomLinks []CustomLink

type CustomLink struct {
	Label    string `json:"label"`
	URL      string `json:"url"`
	Icon     string `json:"icon"`
	Location string `json:"location"`
}

type CleanupOptions struct {
	// Enable checkboxes that allow the user to ask for LoadBalancers and PVCs
	// to be deleted in order to not leave potentially expensive resources behind.
	Enabled bool `json:"enabled,omitempty"`
	// If enforced is set to true, the cleanup of LoadBalancers and PVCs is
	// enforced.
	Enforced bool `json:"enforced,omitempty"`
}

type OpaOptions struct {
	Enabled  bool `json:"enabled,omitempty"`
	Enforced bool `json:"enforced,omitempty"`
}

type MlaOptions struct {
	LoggingEnabled     bool `json:"loggingEnabled,omitempty"`
	LoggingEnforced    bool `json:"loggingEnforced,omitempty"`
	MonitoringEnabled  bool `json:"monitoringEnabled,omitempty"`
	MonitoringEnforced bool `json:"monitoringEnforced,omitempty"`
}

type NotificationsOptions struct {
	// HideErrors will silence error notifications for the dashboard.
	HideErrors bool `json:"hideErrors,omitempty"`
	// HideErrorEvents will silence error events for the dashboard.
	HideErrorEvents bool `json:"hideErrorEvents,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// KubermaticSettingList is a list of settings.
type KubermaticSettingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KubermaticSetting `json:"items"`
}
