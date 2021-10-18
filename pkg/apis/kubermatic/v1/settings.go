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

type ClusterType int8

const GlobalSettingsName = "globalsettings"

const (
	ClusterTypeAll ClusterType = iota
	ClusterTypeKubernetes
)

//+genclient
//+genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubermaticSetting is the type representing a KubermaticSetting
type KubermaticSetting struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SettingSpec `json:"spec"`
}

type SettingSpec struct {
	CustomLinks                 CustomLinks    `json:"customLinks"`
	CleanupOptions              CleanupOptions `json:"cleanupOptions"`
	DefaultNodeCount            int8           `json:"defaultNodeCount"`
	ClusterTypeOptions          ClusterType    `json:"clusterTypeOptions"`
	DisplayDemoInfo             bool           `json:"displayDemoInfo"`
	DisplayAPIDocs              bool           `json:"displayAPIDocs"`
	DisplayTermsOfService       bool           `json:"displayTermsOfService"`
	EnableDashboard             bool           `json:"enableDashboard"`
	EnableOIDCKubeconfig        bool           `json:"enableOIDCKubeconfig"` //nolint:tagliatelle
	UserProjectsLimit           int64          `json:"userProjectsLimit"`
	RestrictProjectCreation     bool           `json:"restrictProjectCreation"`
	EnableExternalClusterImport bool           `json:"enableExternalClusterImport"`
	OpaOptions                  OpaOptions     `json:"opaOptions"`
	MlaOptions                  MlaOptions     `json:"mlaOptions"`
	MlaAlertmanagerPrefix       string         `json:"mlaAlertmanagerPrefix"`
	MlaGrafanaPrefix            string         `json:"mlaGrafanaPrefix"`

	MachineDeploymentVMResourceQuota MachineDeploymentVMResourceQuota `json:"machineDeploymentVMResourceQuota"`

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
	Enabled  bool `json:"enabled"`
	Enforced bool `json:"enforced"`
}

type MachineDeploymentVMResourceQuota struct {
	// Minimal number of vCPU
	MinCPU int `json:"minCPU"`
	// Maximal number of vCPU
	MaxCPU int `json:"maxCPU"`
	// Minimal RAM size in GB
	MinRAM int `json:"minRAM"`
	// Maximum RAM size in GB
	MaxRAM int `json:"maxRAM"`

	EnableGPU bool `json:"enableGPU"` //nolint:tagliatelle
}

type OpaOptions struct {
	Enabled  bool `json:"enabled"`
	Enforced bool `json:"enforced"`
}

type MlaOptions struct {
	LoggingEnabled     bool `json:"loggingEnabled"`
	LoggingEnforced    bool `json:"loggingEnforced"`
	MonitoringEnabled  bool `json:"monitoringEnabled"`
	MonitoringEnforced bool `json:"monitoringEnforced"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubermaticSettingList is a list of settings
type KubermaticSettingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KubermaticSetting `json:"items"`
}
