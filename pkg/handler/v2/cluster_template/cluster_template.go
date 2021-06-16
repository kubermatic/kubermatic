/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package clustertemplate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"
)

// scopeList holds a list of user cluster template access levels.
var scopeList = []string{
	kubermaticv1.UserClusterTemplateScope,
	kubermaticv1.ProjectClusterTemplateScope,
	kubermaticv1.GlobalClusterTemplateScope,
}

func CreateEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	userInfoGetter provider.UserInfoGetter, clusterTemplateProvider provider.ClusterTemplateProvider, settingsProvider provider.SettingsProvider, updateManager common.UpdateManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createClusterTemplateReq)

		globalSettings, err := settingsProvider.GetGlobalSettings()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		err = req.Validate(globalSettings.Spec.ClusterTypeOptions, updateManager)
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: false})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		spec, err := genSpec(req.Body.Cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		newClusterTemplate := &kubermaticv1.ClusterTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:        rand.String(10),
				Labels:      map[string]string{},
				Annotations: map[string]string{},
			},
			Credential:             req.Body.Cluster.Credential,
			ClusterLabels:          req.Body.Cluster.Labels,
			InheritedClusterLabels: req.Body.Cluster.InheritedLabels,
			Spec:                   *spec,
		}

		if req.Body.NodeDeployment != nil {
			isBYO, err := common.IsBringYourOwnProvider(spec.Cloud)
			if err != nil {
				return nil, errors.NewBadRequest(fmt.Sprintf("cannot verify the provider due to an invalid spec: %v", err))
			}
			if !isBYO {
				data, err := json.Marshal(req.Body.NodeDeployment)
				if err != nil {
					return "", fmt.Errorf("cannot marshal initial machine deployment: %v", err)
				}
				newClusterTemplate.Annotations[apiv1.InitialMachineDeploymentRequestAnnotation] = string(data)
			}
		}
		newClusterTemplate.Annotations[kubermaticv1.ClusterTemplateUserAnnotationKey] = userInfo.Email
		newClusterTemplate.Labels[kubermaticv1.ClusterTemplateProjectLabelKey] = project.Name
		newClusterTemplate.Labels[kubermaticv1.ClusterTemplateScopeLabelKey] = req.Body.Scope
		newClusterTemplate.Labels[kubermaticv1.ClusterTemplateHumanReadableNameLabelKey] = req.Body.Name

		clusterTemplate, err := clusterTemplateProvider.New(userInfo, newClusterTemplate, req.Body.Scope, project.Name)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalClusterTemplatetoExternal(clusterTemplate)
	}
}

// Validate validates addReq request
func (req createClusterTemplateReq) Validate(clusterType kubermaticv1.ClusterType, updateManager common.UpdateManager) error {
	if len(req.ProjectID) == 0 || len(req.Body.Name) == 0 || len(req.Body.Scope) == 0 {
		return fmt.Errorf("the name, project ID and scope cannot be empty")
	}

	if err := handlercommon.ValidateClusterSpec(clusterType, updateManager, req.Body.CreateClusterSpec); err != nil {
		return err
	}

	for _, scope := range scopeList {
		if scope == req.Body.Scope {
			return nil
		}
	}
	return fmt.Errorf("invalid scope name %s", req.Body.Scope)
}

// createClusterTemplateReq defines HTTP request for createClusterTemplate
// swagger:parameters createClusterTemplate
type createClusterTemplateReq struct {
	common.ProjectReq
	// in: body
	Body struct {
		Name  string `json:"name"`
		Scope string `json:"scope"`
		apiv1.CreateClusterSpec
	}
}

func DecodeCreateReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createClusterTemplateReq

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	if len(req.Body.Cluster.Type) == 0 {
		req.Body.Cluster.Type = apiv1.KubernetesClusterType
	}

	return req, nil
}

func ListEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	userInfoGetter provider.UserInfoGetter, clusterTemplateProvider provider.ClusterTemplateProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listClusterTemplatesReq)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: false})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		result := apiv2.ClusterTemplateList{}

		templates, err := clusterTemplateProvider.List(userInfo, project.Name)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		for _, template := range templates {
			result = append(result, apiv2.ClusterTemplate{
				Name:      template.Spec.HumanReadableName,
				ID:        template.Name,
				ProjectID: template.Labels[kubermaticv1.ProjectIDLabelKey],
				User:      template.Annotations[kubermaticv1.ClusterTemplateUserAnnotationKey],
				Scope:     template.Labels[kubermaticv1.ClusterTemplateScopeLabelKey],
			})
		}

		return result, nil
	}
}

// listClusterTemplateReq defines HTTP request for listClusterTemplates
// swagger:parameters listClusterTemplates
type listClusterTemplatesReq struct {
	common.ProjectReq
}

func DecodeListReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listClusterTemplatesReq

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	return req, nil
}

// Validate validates listClusterTemplatesReq request
func (req listClusterTemplatesReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("project ID cannot be empty")
	}
	return nil
}

func GetEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	userInfoGetter provider.UserInfoGetter, clusterTemplateProvider provider.ClusterTemplateProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getClusterTemplatesReq)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: false})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		template, err := clusterTemplateProvider.Get(userInfo, project.Name, req.ClusterTemplateID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalClusterTemplatetoExternal(template)
	}
}

func DeleteEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	userInfoGetter provider.UserInfoGetter, clusterTemplateProvider provider.ClusterTemplateProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getClusterTemplatesReq)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: false})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if err := clusterTemplateProvider.Delete(userInfo, project.Name, req.ClusterTemplateID); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return nil, nil
	}
}

// getClusterTemplatesReq defines HTTP request for getClusterTemplate
// swagger:parameters getClusterTemplate deleteClusterTemplate
type getClusterTemplatesReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterTemplateID string `json:"template_id"`
}

// Validate validates getClusterTemplatesReq request
func (req getClusterTemplatesReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("project ID cannot be empty")
	}
	if len(req.ClusterTemplateID) == 0 {
		return fmt.Errorf("cluster template ID cannot be empty")
	}
	return nil
}

func DecodeGetReq(c context.Context, r *http.Request) (interface{}, error) {
	var req getClusterTemplatesReq

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	req.ClusterTemplateID = mux.Vars(r)["template_id"]

	return req, nil
}

// genSpec builds ClusterSpec kubermatic Custom Resource from API Cluster
func genSpec(apiCluster apiv1.Cluster) (*kubermaticv1.ClusterSpec, error) {
	var userSSHKeysAgentEnabled = pointer.BoolPtr(true)

	if apiCluster.Spec.EnableUserSSHKeyAgent != nil {
		userSSHKeysAgentEnabled = apiCluster.Spec.EnableUserSSHKeyAgent
	}

	spec := &kubermaticv1.ClusterSpec{
		HumanReadableName:                    apiCluster.Name,
		Cloud:                                apiCluster.Spec.Cloud,
		MachineNetworks:                      apiCluster.Spec.MachineNetworks,
		OIDC:                                 apiCluster.Spec.OIDC,
		UpdateWindow:                         apiCluster.Spec.UpdateWindow,
		Version:                              apiCluster.Spec.Version,
		UsePodSecurityPolicyAdmissionPlugin:  apiCluster.Spec.UsePodSecurityPolicyAdmissionPlugin,
		UsePodNodeSelectorAdmissionPlugin:    apiCluster.Spec.UsePodNodeSelectorAdmissionPlugin,
		EnableUserSSHKeyAgent:                userSSHKeysAgentEnabled,
		AuditLogging:                         apiCluster.Spec.AuditLogging,
		AdmissionPlugins:                     apiCluster.Spec.AdmissionPlugins,
		OPAIntegration:                       apiCluster.Spec.OPAIntegration,
		PodNodeSelectorAdmissionPluginConfig: apiCluster.Spec.PodNodeSelectorAdmissionPluginConfig,
		ServiceAccount:                       apiCluster.Spec.ServiceAccount,
		MLA:                                  apiCluster.Spec.MLA,
		ContainerRuntime:                     apiCluster.Spec.ContainerRuntime,
	}

	if apiCluster.Spec.ClusterNetwork != nil {
		spec.ClusterNetwork = *apiCluster.Spec.ClusterNetwork
	}

	providerName, err := provider.ClusterCloudProviderName(spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("invalid cloud spec: %v", err)
	}
	if providerName == "" {
		return nil, fmt.Errorf("cluster has no cloudprovider")
	}

	if spec.ComponentsOverride.Etcd.ClusterSize == 0 {
		spec.ComponentsOverride.Etcd.ClusterSize = kubermaticv1.DefaultEtcdClusterSize
	}

	return spec, nil
}

func convertInternalClusterTemplatetoExternal(template *kubermaticv1.ClusterTemplate) (*apiv2.ClusterTemplate, error) {
	md := &apiv1.NodeDeployment{}
	rawMachineDeployment, ok := template.Annotations[apiv1.InitialMachineDeploymentRequestAnnotation]
	if ok {
		err := json.Unmarshal([]byte(rawMachineDeployment), md)
		if err != nil {
			return nil, err
		}
	}

	return &apiv2.ClusterTemplate{
		Name:      template.Labels[kubermaticv1.ClusterTemplateHumanReadableNameLabelKey],
		ID:        template.Name,
		ProjectID: template.Labels[kubermaticv1.ClusterTemplateProjectLabelKey],
		User:      template.Annotations[kubermaticv1.ClusterTemplateUserAnnotationKey],
		Scope:     template.Labels[kubermaticv1.ClusterTemplateScopeLabelKey],
		Cluster: &apiv1.Cluster{
			Labels:          template.ClusterLabels,
			InheritedLabels: template.InheritedClusterLabels,
			Credential:      template.Credential,
			Spec: apiv1.ClusterSpec{
				Cloud:                                template.Spec.Cloud,
				Version:                              template.Spec.Version,
				MachineNetworks:                      template.Spec.MachineNetworks,
				OIDC:                                 template.Spec.OIDC,
				UpdateWindow:                         template.Spec.UpdateWindow,
				AuditLogging:                         template.Spec.AuditLogging,
				UsePodSecurityPolicyAdmissionPlugin:  template.Spec.UsePodSecurityPolicyAdmissionPlugin,
				UsePodNodeSelectorAdmissionPlugin:    template.Spec.UsePodNodeSelectorAdmissionPlugin,
				EnableUserSSHKeyAgent:                template.Spec.EnableUserSSHKeyAgent,
				AdmissionPlugins:                     template.Spec.AdmissionPlugins,
				OPAIntegration:                       template.Spec.OPAIntegration,
				PodNodeSelectorAdmissionPluginConfig: template.Spec.PodNodeSelectorAdmissionPluginConfig,
				ServiceAccount:                       template.Spec.ServiceAccount,
				MLA:                                  template.Spec.MLA,
				ContainerRuntime:                     template.Spec.ContainerRuntime,
			},
		},
		NodeDeployment: md,
	}, nil
}
