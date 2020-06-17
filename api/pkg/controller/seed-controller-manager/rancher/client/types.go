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

package client

import "net/http"

type Filters map[string]string

type Client struct {
	options Options
	client  *http.Client
}

type Options struct {
	Endpoint  string
	Token     string
	AccessKey string
	SecretKey string
	Insecure  bool
}

type Resource struct {
	ID      string            `json:"id,omitempty"`
	Type    string            `json:"type,omitempty"`
	Links   map[string]string `json:"links"`
	Actions map[string]string `json:"actions"`
}

type ClusterList struct {
	Data []Cluster `json:"data,omitempty"`
}

type Cluster struct {
	Resource
	APIEndpoint                          string             `json:"apiEndpoint,omitempty" yaml:"apiEndpoint,omitempty"`
	AgentImage                           string             `json:"agentImage,omitempty" yaml:"agentImage,omitempty"`
	Allocatable                          map[string]string  `json:"allocatable,omitempty" yaml:"allocatable,omitempty"`
	Annotations                          map[string]string  `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	AppliedEnableNetworkPolicy           bool               `json:"appliedEnableNetworkPolicy,omitempty" yaml:"appliedEnableNetworkPolicy,omitempty"`
	AppliedPodSecurityPolicyTemplateName string             `json:"appliedPodSecurityPolicyTemplateId,omitempty" yaml:"appliedPodSecurityPolicyTemplateId,omitempty"`
	AppliedSpec                          *ClusterSpec       `json:"appliedSpec,omitempty" yaml:"appliedSpec,omitempty"`
	AuthImage                            string             `json:"authImage,omitempty" yaml:"authImage,omitempty"`
	CACert                               string             `json:"caCert,omitempty" yaml:"caCert,omitempty"`
	Capacity                             map[string]string  `json:"capacity,omitempty" yaml:"capacity,omitempty"`
	ClusterTemplateID                    string             `json:"clusterTemplateId,omitempty" yaml:"clusterTemplateId,omitempty"`
	ClusterTemplateRevisionID            string             `json:"clusterTemplateRevisionId,omitempty" yaml:"clusterTemplateRevisionId,omitempty"`
	Conditions                           []ClusterCondition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	Created                              string             `json:"created,omitempty" yaml:"created,omitempty"`
	CreatorID                            string             `json:"creatorId,omitempty" yaml:"creatorId,omitempty"`
	DefaultClusterRoleForProjectMembers  string             `json:"defaultClusterRoleForProjectMembers,omitempty" yaml:"defaultClusterRoleForProjectMembers,omitempty"`
	DefaultPodSecurityPolicyTemplateID   string             `json:"defaultPodSecurityPolicyTemplateId,omitempty" yaml:"defaultPodSecurityPolicyTemplateId,omitempty"`
	Description                          string             `json:"description,omitempty" yaml:"description,omitempty"`
	DesiredAgentImage                    string             `json:"desiredAgentImage,omitempty" yaml:"desiredAgentImage,omitempty"`
	DesiredAuthImage                     string             `json:"desiredAuthImage,omitempty" yaml:"desiredAuthImage,omitempty"`
	DockerRootDir                        string             `json:"dockerRootDir,omitempty" yaml:"dockerRootDir,omitempty"`
	Driver                               string             `json:"driver,omitempty" yaml:"driver,omitempty"`
	EnableClusterAlerting                bool               `json:"enableClusterAlerting,omitempty" yaml:"enableClusterAlerting,omitempty"`
	EnableClusterMonitoring              bool               `json:"enableClusterMonitoring,omitempty" yaml:"enableClusterMonitoring,omitempty"`
	EnableNetworkPolicy                  *bool              `json:"enableNetworkPolicy,omitempty" yaml:"enableNetworkPolicy,omitempty"`
	FailedSpec                           *ClusterSpec       `json:"failedSpec,omitempty" yaml:"failedSpec,omitempty"`
	ImportedConfig                       *ImportedConfig    `json:"importedConfig,omitempty" yaml:"importedConfig,omitempty"`
	Internal                             bool               `json:"internal,omitempty" yaml:"internal,omitempty"`
	IstioEnabled                         bool               `json:"istioEnabled,omitempty" yaml:"istioEnabled,omitempty"`
	Labels                               map[string]string  `json:"labels,omitempty" yaml:"labels,omitempty"`
	Limits                               map[string]string  `json:"limits,omitempty" yaml:"limits,omitempty"`
	Name                                 string             `json:"name,omitempty" yaml:"name,omitempty"`
	Removed                              string             `json:"removed,omitempty" yaml:"removed,omitempty"`
	Requested                            map[string]string  `json:"requested,omitempty" yaml:"requested,omitempty"`
	State                                string             `json:"state,omitempty" yaml:"state,omitempty"`
	Transitioning                        string             `json:"transitioning,omitempty" yaml:"transitioning,omitempty"`
	TransitioningMessage                 string             `json:"transitioningMessage,omitempty" yaml:"transitioningMessage,omitempty"`
	UUID                                 string             `json:"uuid,omitempty" yaml:"uuid,omitempty"`
	WindowsPreferedCluster               bool               `json:"windowsPreferedCluster,omitempty" yaml:"windowsPreferedCluster,omitempty"`
}

type ClusterSpec struct {
	AmazonElasticContainerServiceConfig map[string]interface{} `json:"amazonElasticContainerServiceConfig,omitempty" yaml:"amazonElasticContainerServiceConfig,omitempty"`
	AzureKubernetesServiceConfig        map[string]interface{} `json:"azureKubernetesServiceConfig,omitempty" yaml:"azureKubernetesServiceConfig,omitempty"`
	ClusterTemplateID                   string                 `json:"clusterTemplateId,omitempty" yaml:"clusterTemplateId,omitempty"`
	ClusterTemplateRevisionID           string                 `json:"clusterTemplateRevisionId,omitempty" yaml:"clusterTemplateRevisionId,omitempty"`
	DefaultClusterRoleForProjectMembers string                 `json:"defaultClusterRoleForProjectMembers,omitempty" yaml:"defaultClusterRoleForProjectMembers,omitempty"`
	DefaultPodSecurityPolicyTemplateID  string                 `json:"defaultPodSecurityPolicyTemplateId,omitempty" yaml:"defaultPodSecurityPolicyTemplateId,omitempty"`
	Description                         string                 `json:"description,omitempty" yaml:"description,omitempty"`
	DesiredAgentImage                   string                 `json:"desiredAgentImage,omitempty" yaml:"desiredAgentImage,omitempty"`
	DesiredAuthImage                    string                 `json:"desiredAuthImage,omitempty" yaml:"desiredAuthImage,omitempty"`
	DisplayName                         string                 `json:"displayName,omitempty" yaml:"displayName,omitempty"`
	DockerRootDir                       string                 `json:"dockerRootDir,omitempty" yaml:"dockerRootDir,omitempty"`
	EnableClusterAlerting               bool                   `json:"enableClusterAlerting,omitempty" yaml:"enableClusterAlerting,omitempty"`
	EnableClusterMonitoring             bool                   `json:"enableClusterMonitoring,omitempty" yaml:"enableClusterMonitoring,omitempty"`
	EnableNetworkPolicy                 *bool                  `json:"enableNetworkPolicy,omitempty" yaml:"enableNetworkPolicy,omitempty"`
	GenericEngineConfig                 map[string]interface{} `json:"genericEngineConfig,omitempty" yaml:"genericEngineConfig,omitempty"`
	GoogleKubernetesEngineConfig        map[string]interface{} `json:"googleKubernetesEngineConfig,omitempty" yaml:"googleKubernetesEngineConfig,omitempty"`
	ImportedConfig                      *ImportedConfig        `json:"importedConfig,omitempty" yaml:"importedConfig,omitempty"`
	Internal                            bool                   `json:"internal,omitempty" yaml:"internal,omitempty"`
	WindowsPreferedCluster              bool                   `json:"windowsPreferedCluster,omitempty" yaml:"windowsPreferedCluster,omitempty"`
}

type ClusterCondition struct {
	LastTransitionTime string `json:"lastTransitionTime,omitempty" yaml:"lastTransitionTime,omitempty"`
	LastUpdateTime     string `json:"lastUpdateTime,omitempty" yaml:"lastUpdateTime,omitempty"`
	Message            string `json:"message,omitempty" yaml:"message,omitempty"`
	Reason             string `json:"reason,omitempty" yaml:"reason,omitempty"`
	Status             string `json:"status,omitempty" yaml:"status,omitempty"`
	Type               string `json:"type,omitempty" yaml:"type,omitempty"`
}

type ImportedConfig struct {
	KubeConfig string `json:"kubeConfig,omitempty" yaml:"kubeConfig,omitempty"`
}

type UserList struct {
	Data []User `json:"data,omitempty"`
}

type User struct {
	Resource
	Annotations          map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Created              string            `json:"created,omitempty" yaml:"created,omitempty"`
	CreatorID            string            `json:"creatorId,omitempty" yaml:"creatorId,omitempty"`
	Description          string            `json:"description,omitempty" yaml:"description,omitempty"`
	Enabled              *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Labels               map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Me                   bool              `json:"me,omitempty" yaml:"me,omitempty"`
	MustChangePassword   bool              `json:"mustChangePassword,omitempty" yaml:"mustChangePassword,omitempty"`
	Name                 string            `json:"name,omitempty" yaml:"name,omitempty"`
	Password             string            `json:"password,omitempty" yaml:"password,omitempty"`
	PrincipalIDs         []string          `json:"principalIds,omitempty" yaml:"principalIds,omitempty"`
	Removed              string            `json:"removed,omitempty" yaml:"removed,omitempty"`
	State                string            `json:"state,omitempty" yaml:"state,omitempty"`
	Transitioning        string            `json:"transitioning,omitempty" yaml:"transitioning,omitempty"`
	TransitioningMessage string            `json:"transitioningMessage,omitempty" yaml:"transitioningMessage,omitempty"`
	UUID                 string            `json:"uuid,omitempty" yaml:"uuid,omitempty"`
	Username             string            `json:"username,omitempty" yaml:"username,omitempty"`
}

type ClusterRegistrationToken struct {
	Resource
	Annotations          map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	ClusterID            string            `json:"clusterId,omitempty" yaml:"clusterId,omitempty"`
	Command              string            `json:"command,omitempty" yaml:"command,omitempty"`
	Created              string            `json:"created,omitempty" yaml:"created,omitempty"`
	CreatorID            string            `json:"creatorId,omitempty" yaml:"creatorId,omitempty"`
	InsecureCommand      string            `json:"insecureCommand,omitempty" yaml:"insecureCommand,omitempty"`
	Labels               map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	ManifestURL          string            `json:"manifestUrl,omitempty" yaml:"manifestUrl,omitempty"`
	Name                 string            `json:"name,omitempty" yaml:"name,omitempty"`
	NamespaceID          string            `json:"namespaceId,omitempty" yaml:"namespaceId,omitempty"`
	NodeCommand          string            `json:"nodeCommand,omitempty" yaml:"nodeCommand,omitempty"`
	Removed              string            `json:"removed,omitempty" yaml:"removed,omitempty"`
	State                string            `json:"state,omitempty" yaml:"state,omitempty"`
	Token                string            `json:"token,omitempty" yaml:"token,omitempty"`
	Transitioning        string            `json:"transitioning,omitempty" yaml:"transitioning,omitempty"`
	TransitioningMessage string            `json:"transitioningMessage,omitempty" yaml:"transitioningMessage,omitempty"`
	UUID                 string            `json:"uuid,omitempty" yaml:"uuid,omitempty"`
	WindowsNodeCommand   string            `json:"windowsNodeCommand,omitempty" yaml:"windowsNodeCommand,omitempty"`
}

type SetPasswordInput struct {
	NewPassword string `json:"newPassword,omitempty" yaml:"newPassword,omitempty"`
}
