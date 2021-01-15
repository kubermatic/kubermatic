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

package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/label"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	kubernetesprovider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/resources/cloudcontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/cluster"
	machineresource "github.com/kubermatic/kubermatic/api/pkg/resources/machine"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	kubermaticerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"github.com/kubermatic/kubermatic/api/pkg/validation"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/rand"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NodeDeploymentEvent represents type of events related to Node Deployment
type NodeDeploymentEvent string

const (
	nodeDeploymentCreationStart   NodeDeploymentEvent = "NodeDeploymentCreationStart"
	nodeDeploymentCreationSuccess NodeDeploymentEvent = "NodeDeploymentCreationSuccess"
	nodeDeploymentCreationFail    NodeDeploymentEvent = "NodeDeploymentCreationFail"
)

// clusterTypes holds a list of supported cluster types
var clusterTypes = sets.NewString(apiv1.OpenShiftClusterType, apiv1.KubernetesClusterType)

func CreateEndpoint(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter,
	initNodeDeploymentFailures *prometheus.CounterVec, eventRecorderProvider provider.EventRecorderProvider, credentialManager provider.PresetProvider,
	exposeStrategy corev1.ServiceType, userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider, updateManager common.UpdateManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateReq)
		globalSettings, err := settingsProvider.GetGlobalSettings()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		err = req.Validate(globalSettings.Spec.ClusterTypeOptions, updateManager)
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
		adminUserInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: false})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		k8sClient := privilegedClusterProvider.GetSeedClusterAdminClient()

		seed, dc, err := provider.DatacenterFromSeedMap(adminUserInfo, seedsGetter, req.Body.Cluster.Spec.Cloud.DatacenterName)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		credentialName := req.Body.Cluster.Credential
		if len(credentialName) > 0 {
			cloudSpec, err := credentialManager.SetCloudCredentials(adminUserInfo, credentialName, req.Body.Cluster.Spec.Cloud, dc)
			if err != nil {
				return nil, errors.NewBadRequest("invalid credentials: %v", err)
			}
			req.Body.Cluster.Spec.Cloud = *cloudSpec
		}

		// Create the cluster.
		secretKeyGetter := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetSeedClusterAdminRuntimeClient())
		spec, err := cluster.Spec(req.Body.Cluster, dc, secretKeyGetter)
		if err != nil {
			return nil, errors.NewBadRequest("invalid cluster: %v", err)
		}

		// master level ExposeStrategy is the default
		spec.ExposeStrategy = exposeStrategy
		if seed.Spec.ExposeStrategy != "" {
			spec.ExposeStrategy = seed.Spec.ExposeStrategy
		}

		existingClusters, err := clusterProvider.List(project, &provider.ClusterListOptions{ClusterSpecName: spec.HumanReadableName})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if len(existingClusters.Items) > 0 {
			return nil, errors.NewAlreadyExists("cluster", spec.HumanReadableName)
		}

		if err = validation.ValidateUpdateWindow(spec.UpdateWindow); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		partialCluster := &kubermaticv1.Cluster{}
		partialCluster.Labels = req.Body.Cluster.Labels
		partialCluster.Spec = *spec
		if req.Body.Cluster.Type == "openshift" {
			if req.Body.Cluster.Spec.Openshift == nil || req.Body.Cluster.Spec.Openshift.ImagePullSecret == "" {
				return nil, errors.NewBadRequest("openshift clusters must be configured with an imagePullSecret")
			}
			partialCluster.Annotations = map[string]string{
				"kubermatic.io/openshift": "true",
			}
		}

		// Enforce audit logging
		if dc.Spec.EnforceAuditLogging {
			partialCluster.Spec.AuditLogging = &kubermaticv1.AuditLoggingSettings{
				Enabled: true,
			}
		}

		// generate the name here so that it can be used in the secretName below
		partialCluster.Name = rand.String(10)

		if cloudcontroller.ExternalCloudControllerFeatureSupported(dc, partialCluster) {
			partialCluster.Spec.Features = map[string]bool{kubermaticv1.ClusterFeatureExternalCloudProvider: true}
		}

		if err := kubernetesprovider.CreateOrUpdateCredentialSecretForCluster(ctx, privilegedClusterProvider.GetSeedClusterAdminRuntimeClient(), partialCluster); err != nil {
			return nil, err
		}
		kuberneteshelper.AddFinalizer(partialCluster, apiv1.CredentialsSecretsCleanupFinalizer)

		newCluster, err := createNewCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, partialCluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// Create the initial node deployment in the background.
		if req.Body.NodeDeployment != nil && req.Body.NodeDeployment.Spec.Replicas > 0 {
			// for BringYourOwn provider we don't create ND
			isBYO, err := common.IsBringYourOwnProvider(spec.Cloud)
			if err != nil {
				return nil, errors.NewBadRequest("failed to create an initial node deployment due to an invalid spec: %v", err)
			}
			if !isBYO {
				go func() {
					defer utilruntime.HandleCrash()
					ndName := getNodeDeploymentDisplayName(req.Body.NodeDeployment)
					eventRecorderProvider.ClusterRecorderFor(k8sClient).Eventf(newCluster, corev1.EventTypeNormal, string(nodeDeploymentCreationStart), "Started creation of initial node deployment %s", ndName)
					err := createInitialNodeDeploymentWithRetries(ctx, req.Body.NodeDeployment, newCluster, project, sshKeyProvider, seedsGetter, clusterProvider, privilegedClusterProvider, userInfoGetter)
					if err != nil {
						eventRecorderProvider.ClusterRecorderFor(k8sClient).Eventf(newCluster, corev1.EventTypeWarning, string(nodeDeploymentCreationFail), "Failed to create initial node deployment %s: %v", ndName, err)
						klog.Errorf("failed to create initial node deployment for cluster %s: %v", newCluster.Name, err)
						initNodeDeploymentFailures.With(prometheus.Labels{"cluster": newCluster.Name, "datacenter": req.Body.Cluster.Spec.Cloud.DatacenterName}).Add(1)
					} else {
						eventRecorderProvider.ClusterRecorderFor(k8sClient).Eventf(newCluster, corev1.EventTypeNormal, string(nodeDeploymentCreationSuccess), "Successfully created initial node deployment %s", ndName)
						klog.V(5).Infof("created initial node deployment for cluster %s", newCluster.Name)
					}
				}()
			} else {
				klog.V(5).Infof("KubeAdm provider detected an initial node deployment won't be created for cluster %s", newCluster.Name)
			}
		}

		log := kubermaticlog.Logger.With("cluster", newCluster.Name)

		// Block for up to 10 seconds to give the rbac controller time to create the bindings.
		// During that time we swallow all errors
		if err := wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
			_, err := getInternalCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, req.ProjectID, newCluster.Name, &provider.ClusterGetOptions{})
			if err != nil {
				log.Debugw("Error when waiting for cluster to become ready after creation", zap.Error(err))
				return false, nil
			}
			return true, nil
		}); err != nil {
			log.Error("Timed out waiting for cluster to become ready")
			return convertInternalClusterToExternal(newCluster, true), errors.New(http.StatusInternalServerError, "timed out waiting for cluster to become ready")
		}

		return convertInternalClusterToExternal(newCluster, true), nil
	}
}

func createNewCluster(ctx context.Context, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ClusterProvider, privilegedClusterProvider provider.PrivilegedClusterProvider, project *kubermaticv1.Project, cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	if adminUserInfo.IsAdmin {
		return privilegedClusterProvider.NewUnsecured(project, cluster, adminUserInfo.Email)
	}
	userInfo, err := userInfoGetter(ctx, project.Name)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	return clusterProvider.New(project, userInfo, cluster)
}

func createInitialNodeDeploymentWithRetries(endpointContext context.Context, nodeDeployment *apiv1.NodeDeployment, cluster *kubermaticv1.Cluster,
	project *kubermaticv1.Project, sshKeyProvider provider.SSHKeyProvider,
	seedsGetter provider.SeedsGetter, clusterProvider provider.ClusterProvider, privilegedClusterProvider provider.PrivilegedClusterProvider, userInfoGetter provider.UserInfoGetter) error {
	return wait.Poll(5*time.Second, 30*time.Minute, func() (bool, error) {
		err := createInitialNodeDeployment(endpointContext, nodeDeployment, cluster, project, sshKeyProvider, seedsGetter, clusterProvider, privilegedClusterProvider, userInfoGetter)
		if err != nil {
			// unrecoverable
			if strings.Contains(err.Error(), `admission webhook "machine-controller.kubermatic.io-machinedeployments" denied the request`) {
				klog.V(4).Infof("giving up creating initial Node Deployments for cluster %s (%s) due to an unrecoverabl err %#v", cluster.Name, cluster.Spec.HumanReadableName, err)
				return false, err
			}
			// Likely recoverable
			klog.V(4).Infof("retrying creating initial Node Deployments for cluster %s (%s) due to %v", cluster.Name, cluster.Spec.HumanReadableName, err)
			return false, nil
		}
		return true, nil
	})
}

func createInitialNodeDeployment(endpointContext context.Context, nodeDeployment *apiv1.NodeDeployment, cluster *kubermaticv1.Cluster,
	project *kubermaticv1.Project, sshKeyProvider provider.SSHKeyProvider,
	seedsGetter provider.SeedsGetter, clusterProvider provider.ClusterProvider, privilegedClusterProvider provider.PrivilegedClusterProvider, userInfoGetter provider.UserInfoGetter) error {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	nd, err := machineresource.Validate(nodeDeployment, cluster.Spec.Version.Semver())
	if err != nil {
		return fmt.Errorf("node deployment is not valid: %v", err)
	}

	cluster, err = getInternalCluster(endpointContext, userInfoGetter, clusterProvider, privilegedClusterProvider, project, project.Name, cluster.Name, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return err
	}

	keys, err := sshKeyProvider.List(project, &provider.SSHKeyListOptions{ClusterName: cluster.Name})
	if err != nil {
		return err
	}

	client, err := common.GetClusterClient(endpointContext, userInfoGetter, clusterProvider, cluster, project.Name)
	if err != nil {
		return err
	}

	adminUserInfo, err := userInfoGetter(endpointContext, "")
	if err != nil {
		return err
	}
	_, dc, err := provider.DatacenterFromSeedMap(adminUserInfo, seedsGetter, cluster.Spec.Cloud.DatacenterName)
	if err != nil {
		return fmt.Errorf("error getting dc: %v", err)
	}

	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return errors.New(http.StatusInternalServerError, "clusterprovider is not a kubernetesprovider.Clusterprovider, can not create secret")
	}
	data := common.CredentialsData{
		Ctx:               ctx,
		KubermaticCluster: cluster,
		Client:            assertedClusterProvider.GetSeedClusterAdminRuntimeClient(),
	}
	md, err := machineresource.Deployment(cluster, nd, dc, keys, data)
	if err != nil {
		return err
	}

	return client.Create(ctx, md)
}

func getNodeDeploymentDisplayName(nd *apiv1.NodeDeployment) string {
	if len(nd.Name) != 0 {
		return " " + nd.Name
	}

	return ""
}

func GetEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)

		cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		return convertInternalClusterToExternal(cluster, true), nil
	}
}

// GetCluster returns the cluster for a given request
func GetCluster(ctx context.Context, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter, projectID, clusterID string, options *provider.ClusterGetOptions) (*kubermaticv1.Cluster, error) {
	clusterProvider, ok := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	if !ok {
		return nil, errors.New(http.StatusInternalServerError, "no cluster in request")
	}
	privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
	project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, projectID, nil)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return getInternalCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, projectID, clusterID, options)
}

func getInternalCluster(ctx context.Context, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ClusterProvider, privilegedClusterProvider provider.PrivilegedClusterProvider, project *kubermaticv1.Project, projectID, clusterID string, options *provider.ClusterGetOptions) (*kubermaticv1.Cluster, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	if adminUserInfo.IsAdmin {
		cluster, err := privilegedClusterProvider.GetUnsecured(project, clusterID, options)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return cluster, nil
	}

	return getClusterForRegularUser(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, projectID, clusterID, options)
}

func getClusterForRegularUser(ctx context.Context, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ClusterProvider, privilegedClusterProvider provider.PrivilegedClusterProvider, project *kubermaticv1.Project, projectID, clusterID string, options *provider.ClusterGetOptions) (*kubermaticv1.Cluster, error) {
	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	cluster, err := clusterProvider.Get(userInfo, clusterID, options)
	if err != nil {

		// Request came from the specified user. Instead `Not found` error status the `Forbidden` is returned.
		// Next request with privileged user checks if the cluster doesn't exist or some other error occurred.
		if !isStatus(err, http.StatusForbidden) {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		// Check if cluster really doesn't exist or some other error occurred.
		if _, errGetUnsecured := privilegedClusterProvider.GetUnsecured(project, clusterID, options); errGetUnsecured != nil {
			return nil, common.KubernetesErrorToHTTPError(errGetUnsecured)
		}
		// Cluster is not ready yet, return original error
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	return cluster, nil
}

func isStatus(err error, status int32) bool {
	kubernetesError, ok := err.(*kerrors.StatusError)
	return ok && status == kubernetesError.Status().Code
}

// patchClusterSpec is equivalent of ClusterSpec but it uses default JSON marshalling method instead of custom
// MarshalJSON defined for ClusterSpec type. This means it should be only used internally as it may contain
// sensitive cloud provider authentication data.
type patchClusterSpec apiv1.ClusterSpec

// patchCluster is equivalent of Cluster but it uses patchClusterSpec instead of original ClusterSpec.
// This means it should be only used internally as it may contain sensitive cloud provider authentication data.
type patchCluster struct {
	apiv1.Cluster `json:",inline"`
	Spec          patchClusterSpec `json:"spec"`
}

func PatchEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(PatchReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		oldInternalCluster, err := getInternalCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// Converting to API type as it is the type exposed externally.
		externalCluster := convertInternalClusterToExternal(oldInternalCluster, false)

		// Changing the type to patchCluster as during marshalling it doesn't remove the cloud provider authentication
		// data that is required here for validation.
		externalClusterSpec := (patchClusterSpec)(externalCluster.Spec)
		clusterToPatch := patchCluster{
			Cluster: *externalCluster,
			Spec:    externalClusterSpec,
		}

		existingClusterJSON, err := json.Marshal(clusterToPatch)
		if err != nil {
			return nil, errors.NewBadRequest("cannot decode existing cluster: %v", err)
		}

		patchedClusterJSON, err := jsonpatch.MergePatch(existingClusterJSON, req.Patch)
		if err != nil {
			return nil, errors.NewBadRequest("cannot patch cluster: %v", err)
		}

		var patchedCluster *apiv1.Cluster
		err = json.Unmarshal(patchedClusterJSON, &patchedCluster)
		if err != nil {
			return nil, errors.NewBadRequest("cannot decode patched cluster: %v", err)
		}

		// Only specific fields from old internal cluster will be updated by a patch.
		// It prevents user from changing other fields like resource ID or version that should not be modified.
		newInternalCluster := oldInternalCluster.DeepCopy()
		newInternalCluster.Spec.HumanReadableName = patchedCluster.Name
		newInternalCluster.Labels = patchedCluster.Labels
		newInternalCluster.Spec.Cloud = patchedCluster.Spec.Cloud
		newInternalCluster.Spec.MachineNetworks = patchedCluster.Spec.MachineNetworks
		newInternalCluster.Spec.Version = patchedCluster.Spec.Version
		newInternalCluster.Spec.OIDC = patchedCluster.Spec.OIDC
		newInternalCluster.Spec.UsePodSecurityPolicyAdmissionPlugin = patchedCluster.Spec.UsePodSecurityPolicyAdmissionPlugin
		newInternalCluster.Spec.UsePodNodeSelectorAdmissionPlugin = patchedCluster.Spec.UsePodNodeSelectorAdmissionPlugin
		newInternalCluster.Spec.AdmissionPlugins = patchedCluster.Spec.AdmissionPlugins
		newInternalCluster.Spec.AuditLogging = patchedCluster.Spec.AuditLogging
		newInternalCluster.Spec.Openshift = patchedCluster.Spec.Openshift
		newInternalCluster.Spec.UpdateWindow = patchedCluster.Spec.UpdateWindow
		newInternalCluster.Spec.PodNodeSelectorAdmissionPluginConfig = patchedCluster.Spec.PodNodeSelectorAdmissionPluginConfig

		incompatibleKubelets, err := common.CheckClusterVersionSkew(ctx, userInfoGetter, clusterProvider, newInternalCluster, req.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing nodes' version skew: %v", err)
		}
		if len(incompatibleKubelets) > 0 {
			return nil, errors.NewBadRequest("Cluster contains nodes running the following incompatible kubelet versions: %v. Upgrade your nodes before you upgrade the cluster.", incompatibleKubelets)
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, err.Error())
		}
		_, dc, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, newInternalCluster.Spec.Cloud.DatacenterName)
		if err != nil {
			return nil, fmt.Errorf("error getting dc: %v", err)
		}

		if err := kubernetesprovider.CreateOrUpdateCredentialSecretForCluster(ctx, privilegedClusterProvider.GetSeedClusterAdminRuntimeClient(), newInternalCluster); err != nil {
			return nil, err
		}

		// Enforce audit logging
		if dc.Spec.EnforceAuditLogging {
			newInternalCluster.Spec.AuditLogging = &kubermaticv1.AuditLoggingSettings{
				Enabled: true,
			}
		}

		assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
		}
		if err := validation.ValidateUpdateCluster(ctx, newInternalCluster, oldInternalCluster, dc, assertedClusterProvider); err != nil {
			return nil, errors.NewBadRequest("invalid cluster: %v", err)
		}
		if err = validation.ValidateUpdateWindow(newInternalCluster.Spec.UpdateWindow); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		updatedCluster, err := updateCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, newInternalCluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalClusterToExternal(updatedCluster, true), nil
	}
}

func updateCluster(ctx context.Context, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ClusterProvider, privilegedClusterProvider provider.PrivilegedClusterProvider, project *kubermaticv1.Project, cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get user information: %v", err)
	}
	if adminUserInfo.IsAdmin {
		return privilegedClusterProvider.UpdateUnsecured(project, cluster)
	}
	userInfo, err := userInfoGetter(ctx, project.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get user information: %v", err)
	}
	return clusterProvider.Update(project, userInfo, cluster)
}

// ListEndpoint list clusters within the given datacenter
func ListEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		apiClusters, err := getClusters(ctx, userInfoGetter, clusterProvider, projectProvider, privilegedProjectProvider, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return apiClusters, nil
	}
}

// ListAllEndpoint list clusters for the given project in all datacenters
func ListAllEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, clusterProviderGetter provider.ClusterProviderGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetProjectRq)
		allClusters := make([]*apiv1.Cluster, 0)

		seeds, err := seedsGetter()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		for _, seed := range seeds {
			// if a Seed is bad, do not forward that error to the user, but only log
			clusterProvider, err := clusterProviderGetter(seed)
			if err != nil {
				klog.Errorf("failed to create cluster provider for seed %s: %v", seed.Name, err)
				continue
			}
			apiClusters, err := getClusters(ctx, userInfoGetter, clusterProvider, projectProvider, privilegedProjectProvider, req.ProjectID)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			allClusters = append(allClusters, apiClusters...)
		}

		return allClusters, nil
	}
}

func DeleteEndpoint(sshKeyProvider provider.SSHKeyProvider, privilegedSSHKeyProvider provider.PrivilegedSSHKeyProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DeleteReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		clusterSSHKeys, err := sshKeyProvider.List(project, &provider.SSHKeyListOptions{ClusterName: req.ClusterID})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		for _, clusterSSHKey := range clusterSSHKeys {
			clusterSSHKey.RemoveFromCluster(req.ClusterID)
			if err := updateClusterSSHKey(ctx, userInfoGetter, sshKeyProvider, privilegedSSHKeyProvider, clusterSSHKey, req.ProjectID); err != nil {
				return nil, err
			}
		}

		existingCluster, err := getInternalCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, err
		}

		// Use the NodeDeletionFinalizer to determine if the cluster was ever up, the LB and PV finalizers
		// will prevent cluster deletion if the APIserver was never created
		wasUpOnce := kuberneteshelper.HasFinalizer(existingCluster, apiv1.NodeDeletionFinalizer)
		if wasUpOnce && (req.DeleteVolumes || req.DeleteLoadBalancers) {
			if req.DeleteLoadBalancers {
				kuberneteshelper.AddFinalizer(existingCluster, apiv1.InClusterLBCleanupFinalizer)
			}
			if req.DeleteVolumes {
				kuberneteshelper.AddFinalizer(existingCluster, apiv1.InClusterPVCleanupFinalizer)
			}
		}

		return nil, updateAndDeleteCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, existingCluster)
	}
}

func updateClusterSSHKey(ctx context.Context, userInfoGetter provider.UserInfoGetter, sshKeyProvider provider.SSHKeyProvider, privilegedSSHKeyProvider provider.PrivilegedSSHKeyProvider, clusterSSHKey *kubermaticv1.UserSSHKey, projectID string) error {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return errors.New(http.StatusInternalServerError, err.Error())
	}
	if adminUserInfo.IsAdmin {
		if _, err := privilegedSSHKeyProvider.UpdateUnsecured(clusterSSHKey); err != nil {
			return common.KubernetesErrorToHTTPError(err)
		}
		return nil
	}
	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return errors.New(http.StatusInternalServerError, err.Error())
	}
	if _, err = sshKeyProvider.Update(userInfo, clusterSSHKey); err != nil {
		return common.KubernetesErrorToHTTPError(err)
	}
	return nil
}

func updateAndDeleteCluster(ctx context.Context, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ClusterProvider, privilegedClusterProvider provider.PrivilegedClusterProvider, project *kubermaticv1.Project, cluster *kubermaticv1.Cluster) error {

	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return errors.New(http.StatusInternalServerError, err.Error())
	}
	if adminUserInfo.IsAdmin {
		cluster, err := privilegedClusterProvider.UpdateUnsecured(project, cluster)
		if err != nil {
			return common.KubernetesErrorToHTTPError(err)
		}
		err = privilegedClusterProvider.DeleteUnsecured(cluster)
		if err != nil {
			return common.KubernetesErrorToHTTPError(err)
		}
		return nil
	}

	return updateAndDeleteClusterForRegularUser(ctx, userInfoGetter, clusterProvider, project, cluster)
}

func updateAndDeleteClusterForRegularUser(ctx context.Context, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ClusterProvider, project *kubermaticv1.Project, cluster *kubermaticv1.Cluster) error {

	userInfo, err := userInfoGetter(ctx, project.Name)
	if err != nil {
		return errors.New(http.StatusInternalServerError, err.Error())
	}
	if _, err = clusterProvider.Update(project, userInfo, cluster); err != nil {
		return common.KubernetesErrorToHTTPError(err)
	}

	err = clusterProvider.Delete(userInfo, cluster.Name)
	if err != nil {
		return common.KubernetesErrorToHTTPError(err)
	}
	return nil
}

func GetClusterEventsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(EventsReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
		client := privilegedClusterProvider.GetSeedClusterAdminRuntimeClient()

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := getInternalCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		eventType := ""
		switch req.Type {
		case "warning":
			eventType = corev1.EventTypeWarning
		case "normal":
			eventType = corev1.EventTypeNormal
		}

		events, err := common.GetEvents(ctx, client, cluster, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if len(eventType) > 0 {
			events = common.FilterEventsByType(events, eventType)
		}

		return events, nil
	}
}

func HealthEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		existingCluster, err := getInternalCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return apiv1.ClusterHealth{
			Apiserver:                    existingCluster.Status.ExtendedHealth.Apiserver,
			Scheduler:                    existingCluster.Status.ExtendedHealth.Scheduler,
			Controller:                   existingCluster.Status.ExtendedHealth.Controller,
			MachineController:            existingCluster.Status.ExtendedHealth.MachineController,
			Etcd:                         existingCluster.Status.ExtendedHealth.Etcd,
			CloudProviderInfrastructure:  existingCluster.Status.ExtendedHealth.CloudProviderInfrastructure,
			UserClusterControllerManager: existingCluster.Status.ExtendedHealth.UserClusterControllerManager,
		}, nil
	}
}

func AssignSSHKeyEndpoint(sshKeyProvider provider.SSHKeyProvider, privilegedSSHKeyProvider provider.PrivilegedSSHKeyProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AssignSSHKeysReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
		if len(req.KeyID) == 0 {
			return nil, errors.NewBadRequest("please provide an SSH key")
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		_, err = getInternalCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// sanity check, make sure that the key belongs to the project
		// alternatively we could examine the owner references
		{
			projectSSHKeys, err := sshKeyProvider.List(project, nil)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			found := false
			for _, projectSSHKey := range projectSSHKeys {
				if projectSSHKey.Name == req.KeyID {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("the given ssh key %s does not belong to the given project %s (%s)", req.KeyID, project.Spec.Name, project.Name)
			}
		}

		sshKey, err := getSSHKey(ctx, userInfoGetter, sshKeyProvider, privilegedSSHKeyProvider, req.ProjectID, req.KeyID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		apiKey := apiv1.SSHKey{
			ObjectMeta: apiv1.ObjectMeta{
				ID:                sshKey.Name,
				Name:              sshKey.Spec.Name,
				CreationTimestamp: apiv1.NewTime(sshKey.CreationTimestamp.Time),
			},
		}

		if sshKey.IsUsedByCluster(req.ClusterID) {
			return apiKey, nil
		}
		sshKey.AddToCluster(req.ClusterID)
		if err := updateClusterSSHKey(ctx, userInfoGetter, sshKeyProvider, privilegedSSHKeyProvider, sshKey, req.ProjectID); err != nil {
			return nil, err
		}

		return apiKey, nil
	}
}

func getSSHKey(ctx context.Context, userInfoGetter provider.UserInfoGetter, sshKeyProvider provider.SSHKeyProvider, privilegedSSHKeyProvider provider.PrivilegedSSHKeyProvider, projectID, keyName string) (*kubermaticv1.UserSSHKey, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, err.Error())
	}
	if adminUserInfo.IsAdmin {
		return privilegedSSHKeyProvider.GetUnsecured(keyName)
	}
	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, err.Error())
	}
	return sshKeyProvider.Get(userInfo, keyName)
}

func ListSSHKeysEndpoint(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListSSHKeysReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		_, err = getInternalCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		keys, err := sshKeyProvider.List(project, &provider.SSHKeyListOptions{ClusterName: req.ClusterID})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		apiKeys := common.ConvertInternalSSHKeysToExternal(keys)
		return apiKeys, nil
	}
}

func convertInternalClusterToExternal(internalCluster *kubermaticv1.Cluster, filterSystemLabels bool) *apiv1.Cluster {
	cluster := &apiv1.Cluster{
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
		Labels:          internalCluster.Labels,
		InheritedLabels: internalCluster.Status.InheritedLabels,
		Spec: apiv1.ClusterSpec{
			Cloud:                                internalCluster.Spec.Cloud,
			Version:                              internalCluster.Spec.Version,
			MachineNetworks:                      internalCluster.Spec.MachineNetworks,
			OIDC:                                 internalCluster.Spec.OIDC,
			UpdateWindow:                         internalCluster.Spec.UpdateWindow,
			AuditLogging:                         internalCluster.Spec.AuditLogging,
			UsePodSecurityPolicyAdmissionPlugin:  internalCluster.Spec.UsePodSecurityPolicyAdmissionPlugin,
			UsePodNodeSelectorAdmissionPlugin:    internalCluster.Spec.UsePodNodeSelectorAdmissionPlugin,
			AdmissionPlugins:                     internalCluster.Spec.AdmissionPlugins,
			PodNodeSelectorAdmissionPluginConfig: internalCluster.Spec.PodNodeSelectorAdmissionPluginConfig,
		},
		Status: apiv1.ClusterStatus{
			Version: internalCluster.Spec.Version,
			URL:     internalCluster.Address.URL,
		},
		Type: apiv1.KubernetesClusterType,
	}

	if filterSystemLabels {
		cluster.Labels = label.FilterLabels(label.ClusterResourceType, internalCluster.Labels)
	}
	if internalCluster.IsOpenshift() {
		cluster.Type = apiv1.OpenShiftClusterType
	}

	return cluster
}

func convertInternalClustersToExternal(internalClusters []kubermaticv1.Cluster) []*apiv1.Cluster {
	apiClusters := make([]*apiv1.Cluster, len(internalClusters))
	for index, cluster := range internalClusters {
		apiClusters[index] = convertInternalClusterToExternal(cluster.DeepCopy(), true)
	}
	return apiClusters
}

func DetachSSHKeyEndpoint(sshKeyProvider provider.SSHKeyProvider, privilegedSSHKeyProvider provider.PrivilegedSSHKeyProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DetachSSHKeysReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		_, err = getInternalCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// sanity check, make sure that the key belongs to the project
		// alternatively we could examine the owner references
		{
			projectSSHKeys, err := sshKeyProvider.List(project, nil)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}

			found := false
			for _, projectSSHKey := range projectSSHKeys {
				if projectSSHKey.Name == req.KeyID {
					found = true
					break
				}
			}
			if !found {
				return nil, errors.NewNotFound("sshkey", req.KeyID)
			}
		}

		clusterSSHKey, err := getSSHKey(ctx, userInfoGetter, sshKeyProvider, privilegedSSHKeyProvider, req.ProjectID, req.KeyID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		clusterSSHKey.RemoveFromCluster(req.ClusterID)
		if err := updateClusterSSHKey(ctx, userInfoGetter, sshKeyProvider, privilegedSSHKeyProvider, clusterSSHKey, req.ProjectID); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return nil, nil
	}
}

func GetMetricsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, req.ProjectID)
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

		dynamicClient, err := clusterProvider.GetAdminClientForCustomerCluster(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		allNodeMetricsList := &v1beta1.NodeMetricsList{}
		if err := dynamicClient.List(ctx, allNodeMetricsList); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		seedAdminClient := privilegedClusterProvider.GetSeedClusterAdminRuntimeClient()
		podMetricsList := &v1beta1.PodMetricsList{}
		if err := seedAdminClient.List(ctx, podMetricsList, &ctrlruntimeclient.ListOptions{Namespace: fmt.Sprintf("cluster-%s", cluster.Name)}); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertClusterMetrics(podMetricsList, allNodeMetricsList.Items, availableResources, cluster)
	}
}

func convertClusterMetrics(podMetrics *v1beta1.PodMetricsList, nodeMetrics []v1beta1.NodeMetrics, availableNodesResources map[string]corev1.ResourceList, cluster *kubermaticv1.Cluster) (*apiv1.ClusterMetrics, error) {

	if podMetrics == nil {
		return nil, fmt.Errorf("metric list can not be nil")
	}
	if cluster == nil {
		return nil, fmt.Errorf("cluster object can not be nil")
	}
	clusterMetrics := &apiv1.ClusterMetrics{
		Name:                cluster.Name,
		ControlPlaneMetrics: apiv1.ControlPlaneMetrics{},
		NodesMetrics:        apiv1.NodesMetric{},
	}

	for _, m := range nodeMetrics {
		usage := corev1.ResourceList{}
		err := scheme.Scheme.Convert(&m.Usage, &usage, nil)
		if err != nil {
			return nil, err
		}
		resourceMetricsInfo := common.ResourceMetricsInfo{
			Name:      m.Name,
			Metrics:   usage,
			Available: availableNodesResources[m.Name],
		}

		availableCPU, foundCPU := resourceMetricsInfo.Available[corev1.ResourceCPU]
		availableMemory, foundMemory := resourceMetricsInfo.Available[corev1.ResourceMemory]
		if foundCPU && foundMemory {
			quantityCPU := resourceMetricsInfo.Metrics[corev1.ResourceCPU]
			clusterMetrics.NodesMetrics.CPUTotalMillicores += quantityCPU.MilliValue()
			clusterMetrics.NodesMetrics.CPUAvailableMillicores += availableCPU.MilliValue()

			quantityM := resourceMetricsInfo.Metrics[corev1.ResourceMemory]
			clusterMetrics.NodesMetrics.MemoryTotalBytes += quantityM.Value() / (1024 * 1024)
			clusterMetrics.NodesMetrics.MemoryAvailableBytes += availableMemory.Value() / (1024 * 1024)
		}
	}
	fractionCPU := float64(clusterMetrics.NodesMetrics.CPUTotalMillicores) / float64(clusterMetrics.NodesMetrics.CPUAvailableMillicores) * 100
	clusterMetrics.NodesMetrics.CPUUsedPercentage += int64(fractionCPU)
	fractionMemory := float64(clusterMetrics.NodesMetrics.MemoryTotalBytes) / float64(clusterMetrics.NodesMetrics.MemoryAvailableBytes) * 100
	clusterMetrics.NodesMetrics.MemoryUsedPercentage += int64(fractionMemory)

	for _, podMetrics := range podMetrics.Items {
		for _, container := range podMetrics.Containers {
			usage := corev1.ResourceList{}
			err := scheme.Scheme.Convert(&container.Usage, &usage, nil)
			if err != nil {
				return nil, err
			}
			quantityCPU := usage[corev1.ResourceCPU]
			clusterMetrics.ControlPlaneMetrics.CPUTotalMillicores += quantityCPU.MilliValue()
			quantityM := usage[corev1.ResourceMemory]
			clusterMetrics.ControlPlaneMetrics.MemoryTotalBytes += quantityM.Value() / (1024 * 1024)
		}

	}

	return clusterMetrics, nil
}

// AssignSSHKeysReq defines HTTP request data for assignSSHKeyToCluster  endpoint
// swagger:parameters assignSSHKeyToCluster
type AssignSSHKeysReq struct {
	common.DCReq
	// in: path
	ClusterID string `json:"cluster_id"`
	// in: path
	KeyID string `json:"key_id"`
}

// ListSSHKeysReq defines HTTP request data for listSSHKeysAssignedToCluster endpoint
// swagger:parameters listSSHKeysAssignedToCluster
type ListSSHKeysReq struct {
	common.DCReq
	// in: path
	ClusterID string `json:"cluster_id"`
}

// DetachSSHKeysReq defines HTTP request for detachSSHKeyFromCluster endpoint
// swagger:parameters detachSSHKeyFromCluster
type DetachSSHKeysReq struct {
	common.DCReq
	// in: path
	KeyID string `json:"key_id"`
	// in: path
	ClusterID string `json:"cluster_id"`
}

// CreateReq defines HTTP request for createCluster endpoint
// swagger:parameters createCluster
type CreateReq struct {
	common.DCReq
	// in: body
	Body apiv1.CreateClusterSpec
}

// Validate validates DeleteEndpoint request
func (r CreateReq) Validate(clusterType kubermaticv1.ClusterType, updateManager common.UpdateManager) error {
	if len(r.ProjectID) == 0 || len(r.DC) == 0 {
		return fmt.Errorf("the service account ID and datacenter cannot be empty")
	}
	if r.Body.Cluster.ID != "" {
		return fmt.Errorf("cluster.ID is read-only")
	}

	if !clusterTypes.Has(r.Body.Cluster.Type) {
		return fmt.Errorf("invalid cluster type %s", r.Body.Cluster.Type)
	}

	if clusterType != kubermaticv1.ClusterTypeAll && clusterType != apiv1.ToInternalClusterType(r.Body.Cluster.Type) {
		return fmt.Errorf("disabled cluster type %s", r.Body.Cluster.Type)
	}

	if r.Body.Cluster.Spec.Version.Version == nil {
		return fmt.Errorf("invalid cluster: invalid cloud spec \"Version\" is required but was not specified")
	}

	versions, err := updateManager.GetVersions(r.Body.Cluster.Type)
	if err != nil {
		return fmt.Errorf("failed to get available cluster versions: %v", err)
	}
	for _, availableVersion := range versions {
		if r.Body.Cluster.Spec.Version.Version.Equal(availableVersion.Version) {
			return nil
		}
	}

	return fmt.Errorf("invalid cluster: invalid cloud spec: unsupported version %v", r.Body.Cluster.Spec.Version.Version)
}

func DecodeCreateReq(c context.Context, r *http.Request) (interface{}, error) {
	var req CreateReq

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	if len(req.Body.Cluster.Type) == 0 {
		req.Body.Cluster.Type = apiv1.KubernetesClusterType
	}

	return req, nil
}

// ListReq defines HTTP request for listClusters endpoint
// swagger:parameters listClusters
type ListReq struct {
	common.DCReq
}

func DecodeListReq(c context.Context, r *http.Request) (interface{}, error) {
	var req ListReq

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)

	return req, nil
}

func decodeSSHKeyID(c context.Context, r *http.Request) (string, error) {
	keyID := mux.Vars(r)["key_id"]
	if keyID == "" {
		return "", fmt.Errorf("'key_id' parameter is required but was not provided")
	}

	return keyID, nil
}

// PatchReq defines HTTP request for patchCluster endpoint
// swagger:parameters patchCluster
type PatchReq struct {
	common.GetClusterReq

	// in: body
	Patch json.RawMessage
}

func DecodePatchReq(c context.Context, r *http.Request) (interface{}, error) {
	var req PatchReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)
	req.ClusterID = clusterID

	if req.Patch, err = ioutil.ReadAll(r.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func DecodeAssignSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AssignSSHKeysReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)

	keyID, err := decodeSSHKeyID(c, r)
	if err != nil {
		return nil, err
	}
	req.KeyID = keyID

	return req, nil
}

func DecodeListSSHKeysReq(c context.Context, r *http.Request) (interface{}, error) {
	var req ListSSHKeysReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)

	return req, nil
}

func DecodeDetachSSHKeysReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DetachSSHKeysReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)

	sshKeyID, ok := mux.Vars(r)["key_id"]
	if !ok {
		return nil, fmt.Errorf("'key_id' parameter is required in order to delete ssh key")
	}
	req.KeyID = sshKeyID

	return req, nil
}

// AdminTokenReq defines HTTP request data for revokeClusterAdminToken and revokeClusterViewerToken endpoints.
// swagger:parameters revokeClusterAdminToken revokeClusterViewerToken
type AdminTokenReq struct {
	common.DCReq
	// in: path
	ClusterID string `json:"cluster_id"`
}

func DecodeAdminTokenReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AdminTokenReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)

	return req, nil
}

func RevokeAdminTokenEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AdminTokenReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		return nil, common.KubernetesErrorToHTTPError(clusterProvider.RevokeAdminKubeconfig(cluster))
	}
}

func RevokeViewerTokenEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AdminTokenReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		return nil, common.KubernetesErrorToHTTPError(clusterProvider.RevokeViewerKubeconfig(cluster))
	}
}

// DeleteReq defines HTTP request for deleteCluster endpoints
// swagger:parameters deleteCluster
type DeleteReq struct {
	common.GetClusterReq
	// in: header
	// DeleteVolumes if true all cluster PV's and PVC's will be deleted from cluster
	DeleteVolumes bool
	// in: header
	// DeleteLoadBalancers if true all load balancers will be deleted from cluster
	DeleteLoadBalancers bool
}

func DecodeDeleteReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DeleteReq

	clusterReqRaw, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	clusterReq := clusterReqRaw.(common.GetClusterReq)
	req.GetClusterReq = clusterReq

	headerValue := r.Header.Get("DeleteVolumes")
	if len(headerValue) > 0 {
		deleteVolumes, err := strconv.ParseBool(headerValue)
		if err != nil {
			return nil, err
		}
		req.DeleteVolumes = deleteVolumes
	}

	headerValue = r.Header.Get("DeleteLoadBalancers")
	if len(headerValue) > 0 {
		deleteLB, err := strconv.ParseBool(headerValue)
		if err != nil {
			return nil, err
		}
		req.DeleteLoadBalancers = deleteLB
	}

	return req, nil
}

func getClusters(ctx context.Context, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ClusterProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID string) ([]*apiv1.Cluster, error) {
	project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, projectID, nil)
	if err != nil {
		return nil, err
	}

	clusters, err := clusterProvider.List(project, nil)
	if err != nil {
		return nil, err
	}

	apiClusters := convertInternalClustersToExternal(clusters.Items)
	return apiClusters, nil
}

// EventsReq defines HTTP request for getClusterEvents endpoint
// swagger:parameters getClusterEvents
type EventsReq struct {
	common.GetClusterReq

	// in: query
	Type string `json:"type,omitempty"`
}

func DecodeGetClusterEvents(c context.Context, r *http.Request) (interface{}, error) {
	var req EventsReq

	clusterReqRaw, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	clusterReq := clusterReqRaw.(common.GetClusterReq)
	req.GetClusterReq = clusterReq

	req.Type = r.URL.Query().Get("type")
	if len(req.Type) > 0 {
		if req.Type == "warning" || req.Type == "normal" {
			return req, nil
		}
		return nil, fmt.Errorf("wrong query paramater, unsupported type: %s", req.Type)
	}

	return req, nil
}

func ListNamespaceEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		namespaceList := &corev1.NamespaceList{}
		if err := client.List(ctx, namespaceList); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		var apiNamespaces []apiv1.Namespace

		for _, namespace := range namespaceList.Items {
			apiNamespace := apiv1.Namespace{Name: namespace.Name}
			apiNamespaces = append(apiNamespaces, apiNamespace)
		}

		return apiNamespaces, nil
	}
}

// GetClusterProviderFromRequest returns cluster and cluster provider based on the provided request.
func GetClusterProviderFromRequest(
	ctx context.Context,
	request interface{},
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	userInfoGetter provider.UserInfoGetter) (*kubermaticv1.Cluster, *kubernetesprovider.ClusterProvider, error) {

	req, ok := request.(common.GetClusterReq)
	if !ok {
		return nil, nil, kubermaticerrors.New(http.StatusBadRequest, "invalid request")
	}

	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
	if err != nil {
		return nil, nil, kubermaticerrors.New(http.StatusInternalServerError, err.Error())
	}

	rawClusterProvider, ok := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
	if !ok {
		return nil, nil, kubermaticerrors.New(http.StatusInternalServerError, "no clusterProvider in request")
	}
	clusterProvider, ok := rawClusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, nil, kubermaticerrors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}
	return cluster, clusterProvider, nil
}
