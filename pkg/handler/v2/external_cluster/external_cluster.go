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

package externalcluster

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-kit/kit/endpoint"

	kubeonev1beta2 "k8c.io/kubeone/pkg/apis/kubeone/v1beta2"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/aks"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/eks"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gke"
	"k8c.io/kubermatic/v2/pkg/resources"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	warningType      = "warning"
	normalType       = "normal"
	DeleteAction     = "delete"
	DisconnectAction = "disconnect"
)

// createClusterReq defines HTTP request for createExternalCluster
// swagger:parameters createExternalCluster
type createClusterReq struct {
	common.ProjectReq
	// The credential name used in the preset for the provider
	// in: header
	// name: Credential
	Credential string
	// in: body
	Body body
}

type body struct {
	// Name is human readable name for the external cluster
	Name string `json:"name"`
	// Kubeconfig Base64 encoded kubeconfig
	Kubeconfig string                          `json:"kubeconfig,omitempty"`
	Cloud      *apiv2.ExternalClusterCloudSpec `json:"cloud,omitempty"`
	Spec       *apiv2.ExternalClusterSpec      `json:"spec,omitempty"`
}

func DecodeCreateReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createClusterReq

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)
	req.Credential = r.Header.Get("Credential")
	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// Validate validates CreateEndpoint request.
func (req createClusterReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	return nil
}

func DecodeManifestFromKubeOneReq(encodedManifest string) (*kubeonev1beta2.KubeOneCluster, error) {
	kubeOneCluster := &kubeonev1beta2.KubeOneCluster{}

	manifest, err := base64.StdEncoding.DecodeString(encodedManifest)
	if err != nil {
		return nil, utilerrors.NewBadRequest(err.Error())
	}
	if err := yaml.UnmarshalStrict(manifest, kubeOneCluster); err != nil {
		return nil, err
	}

	return kubeOneCluster, nil
}

func validatKubeOneReq(kubeOne *apiv2.KubeOneSpec) error {
	// validate manifest
	if len(kubeOne.Manifest) == 0 {
		return fmt.Errorf("the KubeOne Cluster manifest cannot be empty")
	} else {
		manifest, err := DecodeManifestFromKubeOneReq(kubeOne.Manifest)
		if err != nil {
			return fmt.Errorf("invalid KubeOne manifest yaml")
		}

		if manifest.APIVersion == "" || manifest.Kind == "" {
			return utilerrors.NewBadRequest("apiVersion and kind must be present in the manifest")
		}
	}
	// validate sshKey
	if len(kubeOne.SSHKey.PrivateKey) == 0 {
		return fmt.Errorf("the KubeOne SSH Key cannot be empty")
	}
	if kubeOne.CloudSpec == nil {
		return fmt.Errorf("the KubeOne Cluster Provider Credentials cannot be empty")
	}

	return nil
}

func CreateEndpoint(
	userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	clusterProvider provider.ExternalClusterProvider,
	privilegedClusterProvider provider.PrivilegedExternalClusterProvider,
	settingsProvider provider.SettingsProvider,
	presetProvider provider.PresetProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(ctx, settingsProvider) {
			return nil, utilerrors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(createClusterReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: false})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		var preset *kubermaticv1.Preset
		if len(req.Credential) > 0 {
			preset, err = presetProvider.GetPreset(ctx, userInfo, req.Credential)
			if err != nil {
				return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
		}

		cloud := req.Body.Cloud
		spec := req.Body.Spec

		// connect cluster by kubeconfig
		if cloud == nil {
			isImported := resources.ExternalClusterIsImportedTrue
			newCluster := genExternalCluster(req.Body.Name, project.Name, isImported)
			config, err := base64.StdEncoding.DecodeString(req.Body.Kubeconfig)
			if err != nil {
				return nil, utilerrors.NewBadRequest(err.Error())
			}
			if err := clusterProvider.ValidateKubeconfig(ctx, config); err != nil {
				return nil, err
			}
			if err := clusterProvider.CreateOrUpdateKubeconfigSecretForCluster(ctx, newCluster, config); err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}

			createdCluster, err := createNewCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, newCluster, project)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			apiCluster := convertClusterToAPI(createdCluster)
			apiCluster.Status = apiv2.ExternalClusterStatus{State: apiv2.PROVISIONING}
			return apiCluster, nil
		}
		// import GKE cluster
		if cloud.GKE != nil {
			if preset != nil {
				if credentials := preset.Spec.GKE; credentials != nil {
					req.Body.Cloud.GKE.ServiceAccount = credentials.ServiceAccount
				}
			}
			createdCluster, err := createOrImportGKECluster(ctx, req.Body.Name, userInfoGetter, project, spec, cloud, clusterProvider, privilegedClusterProvider)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			apiCluster := convertClusterToAPI(createdCluster)
			apiCluster.Status = apiv2.ExternalClusterStatus{State: apiv2.PROVISIONING}
			return apiCluster, nil
		}
		// import EKS cluster
		if cloud.EKS != nil {
			if preset != nil {
				if credentials := preset.Spec.EKS; credentials != nil {
					cloud.EKS.AccessKeyID = credentials.AccessKeyID
					cloud.EKS.SecretAccessKey = credentials.SecretAccessKey
					cloud.EKS.Region = credentials.Region
				}
			}

			createdCluster, err := createOrImportEKSCluster(ctx, req.Body.Name, userInfoGetter, project, spec, cloud, clusterProvider, privilegedClusterProvider)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}

			apiCluster := convertClusterToAPI(createdCluster)
			apiCluster.Status = apiv2.ExternalClusterStatus{State: apiv2.PROVISIONING}
			return apiCluster, nil
		}
		// import AKS cluster
		if cloud.AKS != nil {
			if preset != nil {
				if credentials := preset.Spec.AKS; credentials != nil {
					cloud.AKS.TenantID = credentials.TenantID
					cloud.AKS.SubscriptionID = credentials.SubscriptionID
					cloud.AKS.ClientID = credentials.ClientID
					cloud.AKS.ClientSecret = credentials.ClientSecret
				}
			}

			createdCluster, err := createOrImportAKSCluster(ctx, req.Body.Name, userInfoGetter, project, spec, cloud, clusterProvider, privilegedClusterProvider)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}

			apiCluster := convertClusterToAPI(createdCluster)
			apiCluster.Status = apiv2.ExternalClusterStatus{State: apiv2.PROVISIONING}
			return apiCluster, nil
		}
		// import KubeOne cluster
		if cloud.KubeOne != nil {
			if err := validatKubeOneReq(cloud.KubeOne); err != nil {
				return nil, utilerrors.NewBadRequest(err.Error())
			}
			createdCluster, err := importKubeOneCluster(ctx, req.Body.Name, userInfoGetter, project, cloud, clusterProvider, privilegedClusterProvider)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}

			apiCluster := convertClusterToAPI(createdCluster)
			apiCluster.Status = apiv2.ExternalClusterStatus{State: apiv2.PROVISIONING}
			return apiCluster, nil
		}
		return nil, utilerrors.NewBadRequest("kubeconfig or cloud provider structure missing")
	}
}

func DeleteEndpoint(userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	clusterProvider provider.ExternalClusterProvider,
	privilegedClusterProvider provider.PrivilegedExternalClusterProvider,
	settingsProvider provider.SettingsProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(ctx, settingsProvider) {
			return nil, utilerrors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(deleteClusterReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if req.Action == DeleteAction {
			err := deleteProviderCluster(ctx, cluster, privilegedClusterProvider)
			if err != nil {
				return nil, err
			}
		}
		return nil, deleteExternalCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, cluster)
	}
}
func deleteProviderCluster(ctx context.Context,
	cluster *kubermaticv1.ExternalCluster,
	privilegedClusterProvider provider.PrivilegedExternalClusterProvider,
) error {
	cloud := cluster.Spec.CloudSpec
	if cloud != nil && cloud.ProviderName != "" {
		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())
		if cloud.AKS != nil {
			err := deleteAKSCluster(ctx, secretKeySelector, cloud)
			if err != nil {
				return err
			}
		}
		if cloud.EKS != nil {
			err := deleteEKSCluster(ctx, secretKeySelector, cloud)
			if err != nil {
				return err
			}
		}
		if cloud.GKE != nil {
			err := deletGKECluster(ctx, secretKeySelector, cloud)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// deleteClusterReq defines HTTP request for deleteExternalCluster
// swagger:parameters deleteExternalCluster
type deleteClusterReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
	// The Action is used to check if to `Delete` the cluster:
	// both the actual cluter from the provider
	// and the respective KKP cluster object
	// By default the cluster will `Disconnect` which means the KKP cluster object will be deleted,
	// cluster still exists on the provider, but is no longer connected/imported in KKP
	// in: header
	// name: Action
	// Possible values: Delete, Disconnect
	Action string `json:"action"`
}

func DecodeDeleteReq(c context.Context, r *http.Request) (interface{}, error) {
	var req deleteClusterReq

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	req.Action = r.Header.Get("Action")
	return req, nil
}

// Validate validates DeleteEndpoint request.
func (req deleteClusterReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the \"ProjectID\" cannot be empty")
	}
	if len(req.ClusterID) == 0 {
		return fmt.Errorf("the \"ClusterID\" cannot be empty")
	}
	if len(req.Action) > 0 {
		if req.Action == DeleteAction || req.Action == DisconnectAction {
			return nil
		}
		return fmt.Errorf("wrong action parameter, unsupported action: %s", req.Action)
	}
	return nil
}

func ListEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(ctx, settingsProvider) {
			return nil, utilerrors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(listClusterReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		clusterList, err := clusterProvider.List(ctx, project)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		apiClusters := make([]*apiv2.ExternalCluster, 0)

		for _, cluster := range clusterList.Items {
			apiClusters = append(apiClusters, convertClusterToAPIWithStatus(ctx, clusterProvider, privilegedClusterProvider, &cluster))
		}

		return apiClusters, nil
	}
}

// listClusterReq defines HTTP request for listExternalClusters
// swagger:parameters listExternalClusters
type listClusterReq struct {
	common.ProjectReq
}

func DecodeListReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listClusterReq

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	return req, nil
}

// Validate validates ListEndpoint request.
func (req listClusterReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	return nil
}

func GetEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(ctx, settingsProvider) {
			return nil, utilerrors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(GetClusterReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		apiCluster := convertClusterToAPIWithStatus(ctx, clusterProvider, privilegedClusterProvider, cluster)

		if apiCluster.Status.State != apiv2.RUNNING {
			return apiCluster, nil
		}
		cloud := cluster.Spec.CloudSpec
		if cloud != nil {
			secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())
			if cloud.AKS != nil {
				apiCluster, err = getAKSClusterDetails(ctx, apiCluster, secretKeySelector, cloud)
				if err != nil {
					return nil, err
				}
			}
			if cloud.EKS != nil {
				apiCluster, err = getEKSClusterDetails(ctx, apiCluster, secretKeySelector, cloud)
				if err != nil {
					return nil, err
				}
			}
			if cloud.GKE != nil {
				apiCluster, err = getGKEClusterDetails(ctx, apiCluster, secretKeySelector, cloud)
				if err != nil {
					return nil, err
				}
			}
		}
		// get version for running cluster
		version, err := clusterProvider.GetVersion(ctx, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		apiCluster.Spec.Version = *version

		return apiCluster, nil
	}
}

// GetClusterReq defines HTTP request for getExternalCluster
// swagger:parameters getExternalCluster getExternalClusterMetrics getExternalClusterUpgrades getExternalClusterKubeconfig listGKEClusterDiskTypes listGKEClusterSizes listGKEClusterZones listGKEClusterImages listAKSNodeVersionsNoCredentials
type GetClusterReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
}

func DecodeGetReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GetClusterReq

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	return req, nil
}

// Validate validates GetEndpoint request.
func (req GetClusterReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	if len(req.ClusterID) == 0 {
		return fmt.Errorf("the cluster ID cannot be empty")
	}
	return nil
}

func UpdateEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(ctx, settingsProvider) {
			return nil, utilerrors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(updateClusterReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: false})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if req.Body.Kubeconfig != "" {
			config, err := base64.StdEncoding.DecodeString(req.Body.Kubeconfig)
			if err != nil {
				return nil, utilerrors.NewBadRequest(err.Error())
			}
			if err := clusterProvider.ValidateKubeconfig(ctx, config); err != nil {
				return nil, err
			}
			if err := clusterProvider.CreateOrUpdateKubeconfigSecretForCluster(ctx, cluster, config); err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
		}

		if req.Body.Name != "" && req.Body.Name != cluster.Spec.HumanReadableName {
			cluster.Spec.HumanReadableName = req.Body.Name
			cluster, err = updateCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, cluster)
			if err != nil {
				return nil, utilerrors.NewBadRequest(err.Error())
			}
		}

		return convertClusterToAPI(cluster), nil
	}
}

func PatchEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	clusterProvider provider.ExternalClusterProvider,
	privilegedClusterProvider provider.PrivilegedExternalClusterProvider,
	settingsProvider provider.SettingsProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(ctx, settingsProvider) {
			return nil, utilerrors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(patchClusterReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: false})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		clusterToPatch := convertClusterToAPI(cluster)

		version, err := clusterProvider.GetVersion(ctx, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		clusterToPatch.Spec = apiv2.ExternalClusterSpec{
			Version: *version,
		}

		cloud := cluster.Spec.CloudSpec
		if cloud != nil {
			secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())
			patchedCluster := &apiv2.ExternalCluster{}

			if cloud.GKE != nil {
				if err := patchCluster(clusterToPatch, patchedCluster, req.Patch); err != nil {
					return nil, err
				}
				return patchGKECluster(ctx, clusterToPatch, patchedCluster, secretKeySelector, cloud.GKE.CredentialsReference)
			}
			if cloud.EKS != nil {
				if err := patchCluster(clusterToPatch, patchedCluster, req.Patch); err != nil {
					return nil, err
				}
				return patchEKSCluster(clusterToPatch, patchedCluster, secretKeySelector, cloud)
			}
			if cloud.AKS != nil {
				if err := patchCluster(clusterToPatch, patchedCluster, req.Patch); err != nil {
					return nil, err
				}
				return patchAKSCluster(ctx, clusterToPatch, patchedCluster, secretKeySelector, cloud)
			}
			if cloud.KubeOne != nil {
				containerRuntime, err := kuberneteshelper.CheckContainerRuntime(ctx, cluster, clusterProvider)
				if err != nil {
					return nil, err
				}
				clusterToPatch.Cloud.KubeOne.ContainerRuntime = containerRuntime
				if err := patchCluster(clusterToPatch, patchedCluster, req.Patch); err != nil {
					return nil, err
				}
				return patchKubeOneCluster(ctx, cluster, clusterToPatch, patchedCluster, secretKeySelector, clusterProvider, privilegedClusterProvider.GetMasterClient())
			}
		}
		return convertClusterToAPI(cluster), nil
	}
}

func patchCluster(clusterToPatch, patchedCluster *apiv2.ExternalCluster, patchJson json.RawMessage) error {
	existingClusterJSON, err := json.Marshal(clusterToPatch)
	if err != nil {
		return utilerrors.NewBadRequest("cannot decode existing cluster: %v", err)
	}
	patchedClusterJSON, err := jsonpatch.MergePatch(existingClusterJSON, patchJson)
	if err != nil {
		return utilerrors.NewBadRequest("cannot patch cluster: %v, %v", err, patchJson)
	}
	err = json.Unmarshal(patchedClusterJSON, &patchedCluster)
	if err != nil {
		return utilerrors.NewBadRequest("cannot decode patched cluster: %v", err)
	}
	return nil
}

// patchClusterReq defines HTTP request for patchExternalCluster
// swagger:parameters patchExternalCluster
type patchClusterReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
	// in: body
	Patch json.RawMessage
}

// Validate validates CreateEndpoint request.
func (req patchClusterReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	if len(req.ClusterID) == 0 {
		return fmt.Errorf("the cluster ID cannot be empty")
	}
	return nil
}

func DecodePatchReq(c context.Context, r *http.Request) (interface{}, error) {
	var req patchClusterReq

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	if req.Patch, err = io.ReadAll(r.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// updateClusterReq defines HTTP request for updateExternalCluster
// swagger:parameters updateExternalCluster
type updateClusterReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
	// in: body
	Body struct {
		// Name is human readable name for the external cluster
		Name string `json:"name"`
		// Kubeconfig Base64 encoded kubeconfig
		Kubeconfig string `json:"kubeconfig,omitempty"`
	}
}

func DecodeUpdateReq(c context.Context, r *http.Request) (interface{}, error) {
	var req updateClusterReq

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// Validate validates CreateEndpoint request.
func (req updateClusterReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	if len(req.ClusterID) == 0 {
		return fmt.Errorf("the cluster ID cannot be empty")
	}
	return nil
}

func GetMetricsEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(ctx, settingsProvider) {
			return nil, utilerrors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(GetClusterReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		apiCluster := convertClusterToAPIWithStatus(ctx, clusterProvider, privilegedClusterProvider, cluster)

		if apiCluster.Status.State == apiv2.RUNNING {
			isMetricServer, err := clusterProvider.IsMetricServerAvailable(ctx, cluster)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			if isMetricServer {
				client, err := clusterProvider.GetClient(ctx, cluster)
				if err != nil {
					return nil, common.KubernetesErrorToHTTPError(err)
				}
				nodeList := &corev1.NodeList{}
				if err := client.List(ctx, nodeList); err != nil {
					return nil, err
				}
				availableResources := make(map[string]corev1.ResourceList)
				for _, n := range nodeList.Items {
					availableResources[n.Name] = n.Status.Allocatable
				}
				allNodeMetricsList := &v1beta1.NodeMetricsList{}
				if err := client.List(ctx, allNodeMetricsList); err != nil {
					return nil, common.KubernetesErrorToHTTPError(err)
				}
				podMetricsList := &v1beta1.PodMetricsList{}
				if err := client.List(ctx, podMetricsList, &ctrlruntimeclient.ListOptions{Namespace: metav1.NamespaceSystem}); err != nil {
					return nil, common.KubernetesErrorToHTTPError(err)
				}
				return handlercommon.ConvertClusterMetrics(podMetricsList, allNodeMetricsList.Items, availableResources, cluster.Name)
			}
		}
		return &apiv1.ClusterMetrics{
			Name:                cluster.Name,
			ControlPlaneMetrics: apiv1.ControlPlaneMetrics{},
			NodesMetrics:        apiv1.NodesMetric{},
		}, nil
	}
}

func ListEventsEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(ctx, settingsProvider) {
			return nil, utilerrors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(listEventsReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: false})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		apiCluster := convertClusterToAPIWithStatus(ctx, clusterProvider, privilegedClusterProvider, cluster)

		eventType := ""
		events := make([]apiv1.Event, 0)

		switch req.Type {
		case warningType:
			eventType = corev1.EventTypeWarning
		case normalType:
			eventType = corev1.EventTypeNormal
		}

		if apiCluster.Status.State == apiv2.RUNNING {
			client, err := clusterProvider.GetClient(ctx, cluster)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			// get nodes events
			nodes := &corev1.NodeList{}
			if err := client.List(ctx, nodes); err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			for _, node := range nodes.Items {
				nodeEvents, err := common.GetEvents(ctx, client, &node, "")
				if err != nil {
					return nil, common.KubernetesErrorToHTTPError(err)
				}
				events = append(events, nodeEvents...)
			}

			// get pods events from kube-system namespace
			pods := &corev1.PodList{}
			if err := client.List(ctx, pods, ctrlruntimeclient.InNamespace(metav1.NamespaceSystem)); err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			for _, pod := range pods.Items {
				nodeEvents, err := common.GetEvents(ctx, client, &pod, metav1.NamespaceSystem)
				if err != nil {
					return nil, common.KubernetesErrorToHTTPError(err)
				}
				events = append(events, nodeEvents...)
			}
		}
		kubermaticEvents, err := common.GetEvents(ctx, privilegedClusterProvider.GetMasterClient(), cluster, metav1.NamespaceDefault)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		events = append(events, kubermaticEvents...)
		if len(eventType) > 0 {
			events = common.FilterEventsByType(events, eventType)
		}

		return events, nil
	}
}

// listEventsReq defines HTTP request for listExternalClusterEvents
// swagger:parameters listExternalClusterEvents
type listEventsReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`

	// in: query
	Type string `json:"type,omitempty"`
}

func DecodeListEventsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listEventsReq

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	req.Type = r.URL.Query().Get("type")
	if len(req.Type) > 0 {
		if req.Type == warningType || req.Type == normalType {
			return req, nil
		}
		return nil, fmt.Errorf("wrong query parameter, unsupported type: %s", req.Type)
	}

	return req, nil
}

// Validate validates ListNodesEventsEndpoint request.
func (req listEventsReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	if len(req.ClusterID) == 0 {
		return fmt.Errorf("the cluster ID cannot be empty")
	}
	return nil
}

func genExternalCluster(name, projectID, isImported string) *kubermaticv1.ExternalCluster {
	return &kubermaticv1.ExternalCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: rand.String(10),
			Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: projectID,
				resources.ExternalClusterIsImported: isImported},
		},
		Spec: kubermaticv1.ExternalClusterSpec{
			HumanReadableName: name,
			CloudSpec:         &kubermaticv1.ExternalClusterCloudSpec{},
		},
	}
}

func createNewCluster(ctx context.Context, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, cluster *kubermaticv1.ExternalCluster, project *kubermaticv1.Project) (*kubermaticv1.ExternalCluster, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		return privilegedClusterProvider.NewUnsecured(ctx, project, cluster)
	}
	userInfo, err := userInfoGetter(ctx, project.Name)
	if err != nil {
		return nil, err
	}
	return clusterProvider.New(ctx, userInfo, project, cluster)
}

func convertClusterToAPI(internalCluster *kubermaticv1.ExternalCluster) *apiv2.ExternalCluster {
	cluster := &apiv2.ExternalCluster{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                internalCluster.Name,
			Name:              internalCluster.Spec.HumanReadableName,
			CreationTimestamp: apiv1.NewTime(internalCluster.CreationTimestamp.Time),
			DeletionTimestamp: func() *apiv1.Time {
				if internalCluster.DeletionTimestamp != nil {
					deletionTimestamp := apiv1.NewTime(internalCluster.DeletionTimestamp.Time)
					return &deletionTimestamp
				}
				return nil
			}(),
		},
		Labels: internalCluster.Labels,
	}
	cloud := internalCluster.Spec.CloudSpec
	if cloud != nil {
		cluster.Cloud = &apiv2.ExternalClusterCloudSpec{}
		if cloud.EKS != nil {
			cluster.Cloud.EKS = &apiv2.EKSCloudSpec{
				Name:   cloud.EKS.Name,
				Region: cloud.EKS.Region,
			}
		}
		if cloud.GKE != nil {
			cluster.Cloud.GKE = &apiv2.GKECloudSpec{
				Name: cloud.GKE.Name,
				Zone: cloud.GKE.Zone,
			}
		}
		if cloud.AKS != nil {
			cluster.Cloud.AKS = &apiv2.AKSCloudSpec{
				Name:          cloud.AKS.Name,
				ResourceGroup: cloud.AKS.ResourceGroup,
			}
		}
		if cloud.KubeOne != nil {
			cluster.Cloud.KubeOne = &apiv2.KubeOneSpec{}
		}
	}

	return cluster
}

func convertClusterToAPIWithStatus(ctx context.Context, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, internalCluster *kubermaticv1.ExternalCluster) *apiv2.ExternalCluster {
	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())
	status := apiv2.ExternalClusterStatus{
		State: apiv2.UNKNOWN,
	}
	apiCluster := convertClusterToAPI(internalCluster)
	apiCluster.Status = status
	cloud := internalCluster.Spec.CloudSpec
	kubeOneCondtion := internalCluster.Status.Condition
	if cloud == nil || cloud.ProviderName == "" {
		apiCluster.Status.State = apiv2.RUNNING
	} else {
		if cloud.EKS != nil {
			eksStatus, err := eks.GetEKSClusterStatus(secretKeySelector, cloud)
			if err != nil {
				apiCluster.Status = apiv2.ExternalClusterStatus{
					State:         apiv2.ERROR,
					StatusMessage: err.Error(),
				}
				return apiCluster
			}
			apiCluster.Status = *eksStatus
		}
		if cloud.AKS != nil {
			aksStatus, err := aks.GetAKSClusterStatus(ctx, secretKeySelector, cloud)
			if err != nil {
				apiCluster.Status = apiv2.ExternalClusterStatus{
					State:         apiv2.ERROR,
					StatusMessage: err.Error(),
				}
				return apiCluster
			}
			apiCluster.Status = *aksStatus
		}
		if cloud.GKE != nil {
			gkeStatus, err := gke.GetGKEClusterStatus(ctx, secretKeySelector, cloud)
			if err != nil {
				apiCluster.Status = apiv2.ExternalClusterStatus{
					State:         apiv2.ERROR,
					StatusMessage: err.Error(),
				}
				return apiCluster
			}
			apiCluster.Status = *gkeStatus
		}
		if cloud.KubeOne != nil {
			kubeoneStatus := &apiv2.ExternalClusterStatus{
				State:         apiv2.ExternalClusterState(kubeOneCondtion.Phase),
				StatusMessage: kubeOneCondtion.Message,
			}
			apiCluster.Status = *kubeoneStatus
		}
	}

	// check kubeconfig access
	_, err := clusterProvider.GetVersion(ctx, internalCluster)
	if err != nil && apiCluster.Status.State == apiv2.RUNNING {
		apiCluster.Status = apiv2.ExternalClusterStatus{
			State:         apiv2.ERROR,
			StatusMessage: fmt.Sprintf("can't access cluster via kubeconfig, check the privilidges, %v", err),
		}
	}
	return apiCluster
}

func getCluster(ctx context.Context, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, projectID, clusterName string) (*kubermaticv1.ExternalCluster, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		return privilegedClusterProvider.GetUnsecured(ctx, clusterName)
	}

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return clusterProvider.Get(ctx, userInfo, clusterName)
}

func deleteExternalCluster(ctx context.Context, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, projectID string, cluster *kubermaticv1.ExternalCluster) error {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return err
	}
	if adminUserInfo.IsAdmin {
		return privilegedClusterProvider.DeleteUnsecured(ctx, cluster)
	}

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return err
	}
	return clusterProvider.Delete(ctx, userInfo, cluster)
}

func updateCluster(ctx context.Context, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, projectID string, cluster *kubermaticv1.ExternalCluster) (*kubermaticv1.ExternalCluster, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		return privilegedClusterProvider.UpdateUnsecured(ctx, cluster)
	}

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return clusterProvider.Update(ctx, userInfo, cluster)
}

func AreExternalClustersEnabled(ctx context.Context, provider provider.SettingsProvider) bool {
	settings, err := provider.GetGlobalSettings(ctx)
	if err != nil {
		return false
	}

	return settings.Spec.EnableExternalClusterImport
}

func GetKubeconfigEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(ctx, settingsProvider) {
			return nil, utilerrors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(GetClusterReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return handlercommon.GetKubeconfigEndpoint(ctx, cluster, privilegedClusterProvider)
	}
}
