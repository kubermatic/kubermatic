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

package common

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/label"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/cloudcontroller"
	"k8c.io/kubermatic/v2/pkg/resources/cluster"
	machineresource "k8c.io/kubermatic/v2/pkg/resources/machine"
	"k8c.io/kubermatic/v2/pkg/util/errors"
	"k8c.io/kubermatic/v2/pkg/validation"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/rand"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
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

// ClusterTypes holds a list of supported cluster types
var ClusterTypes = sets.NewString(apiv1.OpenShiftClusterType, apiv1.KubernetesClusterType)

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

func CreateEndpoint(ctx context.Context, projectID string, body apiv1.CreateClusterSpec, sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter,
	initNodeDeploymentFailures *prometheus.CounterVec, eventRecorderProvider provider.EventRecorderProvider, credentialManager provider.PresetProvider,
	exposeStrategy corev1.ServiceType, userInfoGetter provider.UserInfoGetter) (interface{}, error) {

	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, projectID, &provider.ProjectGetOptions{IncludeUninitialized: false})
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	k8sClient := privilegedClusterProvider.GetSeedClusterAdminClient()

	seed, dc, err := provider.DatacenterFromSeedMap(adminUserInfo, seedsGetter, body.Cluster.Spec.Cloud.DatacenterName)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	credentialName := body.Cluster.Credential
	if len(credentialName) > 0 {
		cloudSpec, err := credentialManager.SetCloudCredentials(adminUserInfo, credentialName, body.Cluster.Spec.Cloud, dc)
		if err != nil {
			return nil, errors.NewBadRequest("invalid credentials: %v", err)
		}
		body.Cluster.Spec.Cloud = *cloudSpec
	}

	// Create the cluster.
	secretKeyGetter := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetSeedClusterAdminRuntimeClient())
	spec, err := cluster.Spec(body.Cluster, dc, secretKeyGetter)
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
	partialCluster.Labels = body.Cluster.Labels
	if partialCluster.Labels == nil {
		partialCluster.Labels = make(map[string]string)
	}
	// Owning project ID must be set early, because it will be inherited by some child objects,
	// for example the credentials secret.
	partialCluster.Labels[kubermaticv1.ProjectIDLabelKey] = projectID
	partialCluster.Spec = *spec
	if body.Cluster.Type == "openshift" {
		if body.Cluster.Spec.Openshift == nil || body.Cluster.Spec.Openshift.ImagePullSecret == "" {
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

	// Enforce PodSecurityPolicy
	if dc.Spec.EnforcePodSecurityPolicy {
		partialCluster.Spec.UsePodSecurityPolicyAdmissionPlugin = true
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
	if body.NodeDeployment != nil && body.NodeDeployment.Spec.Replicas > 0 {
		// for BringYourOwn provider we don't create ND
		isBYO, err := common.IsBringYourOwnProvider(spec.Cloud)
		if err != nil {
			return nil, errors.NewBadRequest("failed to create an initial node deployment due to an invalid spec: %v", err)
		}
		if !isBYO {
			go func() {
				defer utilruntime.HandleCrash()
				ndName := getNodeDeploymentDisplayName(body.NodeDeployment)
				eventRecorderProvider.ClusterRecorderFor(k8sClient).Eventf(newCluster, corev1.EventTypeNormal, string(nodeDeploymentCreationStart), "Started creation of initial node deployment %s", ndName)
				err := createInitialNodeDeploymentWithRetries(ctx, body.NodeDeployment, newCluster, project, sshKeyProvider, seedsGetter, clusterProvider, privilegedClusterProvider, userInfoGetter)
				if err != nil {
					eventRecorderProvider.ClusterRecorderFor(k8sClient).Eventf(newCluster, corev1.EventTypeWarning, string(nodeDeploymentCreationFail), "Failed to create initial node deployment %s: %v", ndName, err)
					klog.Errorf("failed to create initial node deployment for cluster %s: %v", newCluster.Name, err)
					initNodeDeploymentFailures.With(prometheus.Labels{"cluster": newCluster.Name, "datacenter": body.Cluster.Spec.Cloud.DatacenterName}).Add(1)
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
		_, err := GetInternalCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, projectID, newCluster.Name, &provider.ClusterGetOptions{})
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

func GetExternalClusters(ctx context.Context, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ClusterProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID string) ([]*apiv1.Cluster, error) {
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

	return GetInternalCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, projectID, clusterID, options)
}

func GetEndpoint(ctx context.Context, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter, projectID, clusterID string) (interface{}, error) {
	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	return convertInternalClusterToExternal(cluster, true), nil
}

func DeleteEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID, clusterID string, deleteVolumes, deleteLoadBalancers bool, sshKeyProvider provider.SSHKeyProvider, privilegedSSHKeyProvider provider.PrivilegedSSHKeyProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)

	project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, projectID, nil)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	clusterSSHKeys, err := sshKeyProvider.List(project, &provider.SSHKeyListOptions{ClusterName: clusterID})
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	for _, clusterSSHKey := range clusterSSHKeys {
		clusterSSHKey.RemoveFromCluster(clusterID)
		if err := UpdateClusterSSHKey(ctx, userInfoGetter, sshKeyProvider, privilegedSSHKeyProvider, clusterSSHKey, projectID); err != nil {
			return nil, err
		}
	}

	existingCluster, err := GetInternalCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, projectID, clusterID, &provider.ClusterGetOptions{})
	if err != nil {
		return nil, err
	}

	// Use the NodeDeletionFinalizer to determine if the cluster was ever up, the LB and PV finalizers
	// will prevent cluster deletion if the APIserver was never created
	wasUpOnce := kuberneteshelper.HasFinalizer(existingCluster, apiv1.NodeDeletionFinalizer)
	if wasUpOnce && (deleteVolumes || deleteLoadBalancers) {
		if deleteLoadBalancers {
			kuberneteshelper.AddFinalizer(existingCluster, apiv1.InClusterLBCleanupFinalizer)
		}
		if deleteVolumes {
			kuberneteshelper.AddFinalizer(existingCluster, apiv1.InClusterPVCleanupFinalizer)
		}
	}

	return nil, updateAndDeleteCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, existingCluster)
}

func PatchEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID, clusterID string, patch json.RawMessage, seedsGetter provider.SeedsGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)

	project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, projectID, nil)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	oldInternalCluster, err := GetInternalCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, projectID, clusterID, &provider.ClusterGetOptions{})
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

	patchedClusterJSON, err := jsonpatch.MergePatch(existingClusterJSON, patch)
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
	newInternalCluster.Spec.OPAIntegration = patchedCluster.Spec.OPAIntegration

	incompatibleKubelets, err := common.CheckClusterVersionSkew(ctx, userInfoGetter, clusterProvider, newInternalCluster, projectID)
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

	// Enforce PodSecurityPolicy
	if dc.Spec.EnforcePodSecurityPolicy {
		newInternalCluster.Spec.UsePodSecurityPolicyAdmissionPlugin = true
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

func GetClusterEventsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID, clusterID, eventType string, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
	client := privilegedClusterProvider.GetSeedClusterAdminRuntimeClient()

	project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, projectID, nil)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	cluster, err := GetInternalCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, projectID, clusterID, &provider.ClusterGetOptions{})
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	eventTypeAPI := ""
	switch eventType {
	case "warning":
		eventTypeAPI = corev1.EventTypeWarning
	case "normal":
		eventTypeAPI = corev1.EventTypeNormal
	}

	events, err := common.GetEvents(ctx, client, cluster, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	if len(eventTypeAPI) > 0 {
		events = common.FilterEventsByType(events, eventTypeAPI)
	}

	return events, nil
}

func HealthEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID, clusterID string, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
	project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, projectID, nil)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	existingCluster, err := GetInternalCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, projectID, clusterID, &provider.ClusterGetOptions{})
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

func GetMetricsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID, clusterID string, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider) (interface{}, error) {
	privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, projectID)
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
		// Happens during cluster creation when the CRD is not setup yet
		if _, ok := err.(*meta.NoKindMatchError); !ok {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
	}

	seedAdminClient := privilegedClusterProvider.GetSeedClusterAdminRuntimeClient()
	podMetricsList := &v1beta1.PodMetricsList{}
	if err := seedAdminClient.List(ctx, podMetricsList, &ctrlruntimeclient.ListOptions{Namespace: fmt.Sprintf("cluster-%s", cluster.Name)}); err != nil {
		// Happens during cluster creation when the CRD is not setup yet
		if _, ok := err.(*meta.NoKindMatchError); !ok {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
	}
	return ConvertClusterMetrics(podMetricsList, allNodeMetricsList.Items, availableResources, cluster.Name)
}

func ListNamespaceEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID, clusterID string, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, projectID)
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

func UpdateClusterSSHKey(ctx context.Context, userInfoGetter provider.UserInfoGetter, sshKeyProvider provider.SSHKeyProvider, privilegedSSHKeyProvider provider.PrivilegedSSHKeyProvider, clusterSSHKey *kubermaticv1.UserSSHKey, projectID string) error {
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

func convertInternalClustersToExternal(internalClusters []kubermaticv1.Cluster) []*apiv1.Cluster {
	apiClusters := make([]*apiv1.Cluster, len(internalClusters))
	for index, cluster := range internalClusters {
		apiClusters[index] = convertInternalClusterToExternal(cluster.DeepCopy(), true)
	}
	return apiClusters
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

	cluster, err = GetInternalCluster(endpointContext, userInfoGetter, clusterProvider, privilegedClusterProvider, project, project.Name, cluster.Name, &provider.ClusterGetOptions{CheckInitStatus: true})
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

func GetInternalCluster(ctx context.Context, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ClusterProvider, privilegedClusterProvider provider.PrivilegedClusterProvider, project *kubermaticv1.Project, projectID, clusterID string, options *provider.ClusterGetOptions) (*kubermaticv1.Cluster, error) {
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
			Cloud:                               internalCluster.Spec.Cloud,
			Version:                             internalCluster.Spec.Version,
			MachineNetworks:                     internalCluster.Spec.MachineNetworks,
			OIDC:                                internalCluster.Spec.OIDC,
			UpdateWindow:                        internalCluster.Spec.UpdateWindow,
			AuditLogging:                        internalCluster.Spec.AuditLogging,
			UsePodSecurityPolicyAdmissionPlugin: internalCluster.Spec.UsePodSecurityPolicyAdmissionPlugin,
			UsePodNodeSelectorAdmissionPlugin:   internalCluster.Spec.UsePodNodeSelectorAdmissionPlugin,
			AdmissionPlugins:                    internalCluster.Spec.AdmissionPlugins,
			OPAIntegration:                      internalCluster.Spec.OPAIntegration,
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

func ValidateClusterSpec(clusterType kubermaticv1.ClusterType, updateManager common.UpdateManager, body apiv1.CreateClusterSpec) error {
	if body.Cluster.Spec.Cloud.DatacenterName == "" {
		return fmt.Errorf("cluster datacenter name is empty")
	}
	if body.Cluster.ID != "" {
		return fmt.Errorf("cluster.ID is read-only")
	}
	if !ClusterTypes.Has(body.Cluster.Type) {
		return fmt.Errorf("invalid cluster type %s", body.Cluster.Type)
	}
	if clusterType != kubermaticv1.ClusterTypeAll && clusterType != apiv1.ToInternalClusterType(body.Cluster.Type) {
		return fmt.Errorf("disabled cluster type %s", body.Cluster.Type)
	}
	if body.Cluster.Spec.Version.Version == nil {
		return fmt.Errorf("invalid cluster: invalid cloud spec \"Version\" is required but was not specified")
	}

	versions, err := updateManager.GetVersions(body.Cluster.Type)
	if err != nil {
		return fmt.Errorf("failed to get available cluster versions: %v", err)
	}
	for _, availableVersion := range versions {
		if body.Cluster.Spec.Version.Version.Equal(availableVersion.Version) {
			return nil
		}
	}

	return fmt.Errorf("invalid cluster: invalid cloud spec: unsupported version %v", body.Cluster.Spec.Version.Version)
}

func ConvertClusterMetrics(podMetrics *v1beta1.PodMetricsList, nodeMetrics []v1beta1.NodeMetrics, availableNodesResources map[string]corev1.ResourceList, clusterName string) (*apiv1.ClusterMetrics, error) {
	if podMetrics == nil {
		return nil, fmt.Errorf("metric list can not be nil")
	}

	clusterMetrics := &apiv1.ClusterMetrics{
		Name:                clusterName,
		ControlPlaneMetrics: apiv1.ControlPlaneMetrics{},
		NodesMetrics:        apiv1.NodesMetric{},
	}

	for _, m := range nodeMetrics {
		resourceMetricsInfo := common.ResourceMetricsInfo{
			Name:      m.Name,
			Metrics:   m.Usage.DeepCopy(),
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
			usage := container.Usage.DeepCopy()
			quantityCPU := usage[corev1.ResourceCPU]
			clusterMetrics.ControlPlaneMetrics.CPUTotalMillicores += quantityCPU.MilliValue()
			quantityM := usage[corev1.ResourceMemory]
			clusterMetrics.ControlPlaneMetrics.MemoryTotalBytes += quantityM.Value() / (1024 * 1024)
		}
	}

	return clusterMetrics, nil
}
