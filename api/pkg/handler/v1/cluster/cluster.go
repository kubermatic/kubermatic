package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-kit/kit/endpoint"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	prometheusapi "github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"k8s.io/client-go/tools/record"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources/cluster"
	machineresource "github.com/kubermatic/kubermatic/api/pkg/resources/machine"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"github.com/kubermatic/kubermatic/api/pkg/validation"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/rand"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
)

// NodeDeploymentEvent represents type of events related to Node Deployment
type NodeDeploymentEvent string

const (
	nodeDeploymentCreationStart   NodeDeploymentEvent = "NodeDeploymentCreationStart"
	nodeDeploymentCreationSuccess NodeDeploymentEvent = "NodeDeploymentCreationSuccess"
	nodeDeploymentCreationFail    NodeDeploymentEvent = "NodeDeploymentCreationFail"
	nodeDeploymentCreationRetry   NodeDeploymentEvent = "NodeDeploymentCreationRetry"
)

// clusterTypes holds a list of supported cluster types
var clusterTypes = []string{
	apiv1.OpenShiftClusterType,
	apiv1.KubernetesClusterType,
}

func CreateEndpoint(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, credentialsProvider provider.CredentialsProvider,
	seeds map[string]*kubermaticv1.Seed, initNodeDeploymentFailures *prometheus.CounterVec, eventRecorderProvider provider.EventRecorderProvider, credentialManager common.PresetsManager, exposeStrategy corev1.ServiceType) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateReq)
		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		k8sClient := privilegedClusterProvider.GetSeedClusterAdminClient()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		dc, err := provider.DatacenterFromSeedMap(seeds, req.Body.Cluster.Spec.Cloud.DatacenterName)
		if err != nil {
			return nil, fmt.Errorf("error getting dc: %v", err)
		}

		credentialName := req.Body.Cluster.Credential
		if len(credentialName) > 0 {
			cloudSpec, err := credentialManager.SetCloudCredentials(credentialName, req.Body.Cluster.Spec.Cloud, dc)
			if err != nil {
				return nil, errors.NewBadRequest("invalid credentials: %v", err)
			}
			req.Body.Cluster.Spec.Cloud = *cloudSpec
		}

		// Create the cluster.
		spec, err := cluster.Spec(req.Body.Cluster, dc)
		if err != nil {
			return nil, errors.NewBadRequest("invalid cluster: %v", err)
		}
		spec.ExposeStrategy = exposeStrategy

		existingClusters, err := clusterProvider.List(project, &provider.ClusterListOptions{ClusterSpecName: spec.HumanReadableName})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if len(existingClusters) > 0 {
			return nil, errors.NewAlreadyExists("cluster", spec.HumanReadableName)
		}

		partialCluster := &kubermaticv1.Cluster{}
		partialCluster.Spec = *spec
		if req.Body.Cluster.Type == "openshift" {
			partialCluster.Annotations = map[string]string{
				"kubermatic.io/openshift": "true",
			}
		}
		// generate the name here so that it can be used in the secretName below
		partialCluster.Name = rand.String(10)

		if _, err := credentialsProvider.Create(partialCluster, req.ProjectID); err != nil {
			return nil, err
		}

		newCluster, err := clusterProvider.New(project, userInfo, partialCluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// Create the initial node deployment in the background.
		if req.Body.NodeDeployment != nil {
			// for BringYourOwn provider we don't create ND
			isBYO, err := common.IsBringYourOwnProvider(spec.Cloud)
			if err != nil {
				return nil, errors.NewBadRequest("failed to create an initial node deployment due to an invalid spec: %v", err)
			}
			if isBYO {
				glog.V(5).Infof("KubeAdm provider detected an initial node deployment won't be created for cluster %s", newCluster.Name)
				return convertInternalClusterToExternal(newCluster), nil
			}

			go func() {
				defer utilruntime.HandleCrash()
				ndName := getNodeDeploymentDisplayName(req.Body.NodeDeployment)
				eventRecorderProvider.ClusterRecorderFor(k8sClient).Eventf(newCluster, corev1.EventTypeNormal, string(nodeDeploymentCreationStart), "started creation of initial node deployment%s", ndName)
				err := createInitialNodeDeploymentWithRetries(req.Body.NodeDeployment, newCluster, project, sshKeyProvider, seeds, clusterProvider, userInfo, eventRecorderProvider.ClusterRecorderFor(k8sClient))
				if err != nil {
					eventRecorderProvider.ClusterRecorderFor(k8sClient).Eventf(newCluster, corev1.EventTypeWarning, string(nodeDeploymentCreationFail), "failed to create initial node deployment%s: %v", ndName, err)
					glog.Errorf("failed to create initial node deployment for cluster %s: %v", newCluster.Name, err)
					initNodeDeploymentFailures.With(prometheus.Labels{"cluster": newCluster.Name, "node_dc": req.Body.Cluster.Spec.Cloud.DatacenterName}).Add(1)
				} else {
					eventRecorderProvider.ClusterRecorderFor(k8sClient).Eventf(newCluster, corev1.EventTypeNormal, string(nodeDeploymentCreationSuccess), "created initial node deployment%s", ndName)
					glog.V(5).Infof("created initial node deployment for cluster %s", newCluster.Name)
				}
			}()
		}

		return convertInternalClusterToExternal(newCluster), nil
	}
}

func createInitialNodeDeploymentWithRetries(nodeDeployment *apiv1.NodeDeployment, cluster *kubermaticv1.Cluster,
	project *kubermaticv1.Project, sshKeyProvider provider.SSHKeyProvider, seeds map[string]*kubermaticv1.Seed,
	clusterProvider provider.ClusterProvider, userInfo *provider.UserInfo, eventRecorder record.EventRecorder) error {
	return wait.Poll(5*time.Second, 30*time.Minute, func() (bool, error) {
		err := createInitialNodeDeployment(nodeDeployment, cluster, project, sshKeyProvider, seeds, clusterProvider, userInfo)
		switch {
		case err == nil:
			return true, nil
		case kerrors.IsInternalError(err):
			// An internal error can be:
			//
			//  Internal error occurred: failed calling webhook "machine-controller.kubermatic.io-machinedeployments"
			//  Post REDACTED: net/http: request canceled while waiting for connection (Client.Timeout exceeded while awaiting headers)
			fallthrough
		case kerrors.IsServiceUnavailable(err):
			fallthrough
		case kerrors.IsServerTimeout(err):
			eventRecorder.Eventf(cluster, corev1.EventTypeNormal, string(nodeDeploymentCreationRetry), "retrying creating initial Node Deployments%s for cluster %s (%s) due to %v", getNodeDeploymentDisplayName(nodeDeployment), cluster.Name, cluster.Spec.HumanReadableName, err)
			glog.V(4).Infof("retrying creating initial Node Deployments for cluster %s (%s) due to %v", cluster.Name, cluster.Spec.HumanReadableName, err)
			return false, nil
		}
		glog.V(4).Infof("giving up creating initial Node Deployments for cluster %s (%s) due to an unknown err %#v", cluster.Name, cluster.Spec.HumanReadableName, err)
		return false, err
	})
}

func createInitialNodeDeployment(nodeDeployment *apiv1.NodeDeployment, cluster *kubermaticv1.Cluster,
	project *kubermaticv1.Project, sshKeyProvider provider.SSHKeyProvider, seeds map[string]*kubermaticv1.Seed,
	clusterProvider provider.ClusterProvider, userInfo *provider.UserInfo) error {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	nd, err := machineresource.Validate(nodeDeployment, cluster.Spec.Version.Semver())
	if err != nil {
		return fmt.Errorf("node deployment is not valid: %v", err)
	}

	cluster, err = clusterProvider.Get(userInfo, cluster.Name, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return err
	}

	keys, err := sshKeyProvider.List(project, &provider.SSHKeyListOptions{ClusterName: cluster.Name})
	if err != nil {
		return err
	}

	client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
	if err != nil {
		return err
	}

	dc, err := provider.DatacenterFromSeedMap(seeds, cluster.Spec.Cloud.DatacenterName)
	if err != nil {
		return fmt.Errorf("error getting dc: %v", err)
	}

	md, err := machineresource.Deployment(cluster, nd, dc, keys)
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

func GetEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {

			// Request came from the specified user. Instead `Not found` error status the `Forbidden` is returned.
			// Next request with privileged user checks if the cluster doesn't exist or some other error occurred.
			if !isStatus(err, http.StatusForbidden) {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			// Check if cluster really doesn't exist or some other error occurred.
			if _, errGetUnsecured := privilegedClusterProvider.GetUnsecured(project, req.ClusterID); errGetUnsecured != nil {
				return nil, common.KubernetesErrorToHTTPError(errGetUnsecured)
			}
			// Cluster is not ready yet, return original error
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalClusterToExternal(cluster), nil
	}
}

func isStatus(err error, status int32) bool {
	kubernetesError, ok := err.(*kerrors.StatusError)
	return ok && status == kubernetesError.Status().Code
}

func PatchEndpoint(projectProvider provider.ProjectProvider, seeds map[string]*kubermaticv1.Seed) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(PatchReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		existingCluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		existingClusterJSON, err := json.Marshal(existingCluster)
		if err != nil {
			return nil, errors.NewBadRequest("cannot decode existing cluster: %v", err)
		}

		patchedClusterJSON, err := jsonpatch.MergePatch(existingClusterJSON, req.Patch)
		if err != nil {
			return nil, errors.NewBadRequest("cannot patch cluster: %v", err)
		}

		var patchedCluster *kubermaticv1.Cluster
		err = json.Unmarshal(patchedClusterJSON, &patchedCluster)
		if err != nil {
			return nil, errors.NewBadRequest("cannot decode patched cluster: %v", err)
		}

		incompatibleKubelets, err := common.CheckClusterVersionSkew(ctx, userInfo, clusterProvider, patchedCluster)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing nodes' version skew: %v", err)
		}
		if len(incompatibleKubelets) > 0 {
			return nil, errors.NewBadRequest("Cluster contains nodes running the following incompatible kubelet versions: %v. Upgrade your nodes before you upgrade the cluster.", incompatibleKubelets)
		}

		dc, err := provider.DatacenterFromSeedMap(seeds, patchedCluster.Spec.Cloud.DatacenterName)
		if err != nil {
			return nil, fmt.Errorf("error getting dc: %v", err)
		}

		if err := validation.ValidateUpdateCluster(patchedCluster, existingCluster, dc); err != nil {
			return nil, errors.NewBadRequest("invalid cluster: %v", err)
		}

		updatedCluster, err := clusterProvider.Update(userInfo, patchedCluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalClusterToExternal(updatedCluster), nil
	}
}

// ListEndpoint list clusters within the given datacenter
func ListEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		apiClusters, err := getClusters(clusterProvider, userInfo, projectProvider, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return apiClusters, nil
	}
}

// ListAllEndpoint list clusters for the given project in all datacenters
func ListAllEndpoint(projectProvider provider.ProjectProvider, clusterProviders map[string]provider.ClusterProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetProjectRq)
		allClusters := make([]*apiv1.Cluster, 0)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		for _, clusterProvider := range clusterProviders {
			apiClusters, err := getClusters(clusterProvider, userInfo, projectProvider, req.ProjectID)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			allClusters = append(allClusters, apiClusters...)
		}
		return allClusters, nil
	}
}

func DeleteEndpoint(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DeleteReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		clusterSSHKeys, err := sshKeyProvider.List(project, &provider.SSHKeyListOptions{ClusterName: req.ClusterID})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		for _, clusterSSHKey := range clusterSSHKeys {
			clusterSSHKey.RemoveFromCluster(req.ClusterID)
			if _, err = sshKeyProvider.Update(userInfo, clusterSSHKey); err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
		}

		existingCluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		kuberneteshelper.AddFinalizer(existingCluster, apiv1.CredentialsSecretsCleanupFinalizer)

		if req.DeleteVolumes || req.DeleteLoadBalancers {
			if req.DeleteLoadBalancers {
				kuberneteshelper.AddFinalizer(existingCluster, apiv1.InClusterLBCleanupFinalizer)
			}
			if req.DeleteVolumes {
				kuberneteshelper.AddFinalizer(existingCluster, apiv1.InClusterPVCleanupFinalizer)
			}
		}

		if _, err = clusterProvider.Update(userInfo, existingCluster); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		err = clusterProvider.Delete(userInfo, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return nil, nil
	}
}

func GetClusterEventsEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(EventsReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		client := privilegedClusterProvider.GetSeedClusterAdminRuntimeClient()

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
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

func HealthEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		existingCluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
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

func AssignSSHKeyEndpoint(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AssignSSHKeysReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		if len(req.KeyID) == 0 {
			return nil, errors.NewBadRequest("please provide an SSH key")
		}

		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		_, err = clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: false})
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

		sshKey, err := sshKeyProvider.Get(userInfo, req.KeyID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if sshKey.IsUsedByCluster(req.ClusterID) {
			return nil, nil
		}
		sshKey.AddToCluster(req.ClusterID)
		_, err = sshKeyProvider.Update(userInfo, sshKey)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return nil, nil
	}
}

func ListSSHKeysEndpoint(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListSSHKeysReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		_, err = clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
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

func convertInternalClusterToExternal(internalCluster *kubermaticv1.Cluster) *apiv1.Cluster {
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
		Spec: apiv1.ClusterSpec{
			Cloud:           internalCluster.Spec.Cloud,
			Version:         internalCluster.Spec.Version,
			MachineNetworks: internalCluster.Spec.MachineNetworks,
			OIDC:            internalCluster.Spec.OIDC,
		},
		Status: apiv1.ClusterStatus{
			Version: internalCluster.Spec.Version,
			URL:     internalCluster.Address.URL,
		},
		Type: apiv1.KubernetesClusterType,
	}

	isOpenShift, ok := internalCluster.Annotations["kubermatic.io/openshift"]
	if ok {
		if isOpenShift == "true" {
			cluster.Type = apiv1.OpenShiftClusterType
		}
	}

	return cluster
}

func convertInternalClustersToExternal(internalClusters []*kubermaticv1.Cluster) []*apiv1.Cluster {
	apiClusters := make([]*apiv1.Cluster, len(internalClusters))
	for index, cluster := range internalClusters {
		apiClusters[index] = convertInternalClusterToExternal(cluster)
	}
	return apiClusters
}

func DetachSSHKeyEndpoint(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DetachSSHKeysReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		_, err = clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
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

		clusterSSHKey, err := sshKeyProvider.Get(userInfo, req.KeyID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		clusterSSHKey.RemoveFromCluster(req.ClusterID)
		_, err = sshKeyProvider.Update(userInfo, clusterSSHKey)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return nil, nil
	}
}

func GetMetricsEndpoint(projectProvider provider.ProjectProvider, prometheusClient prometheusapi.Client) endpoint.Endpoint {
	if prometheusClient == nil {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			return nil, fmt.Errorf("metrics endpoint disabled")
		}
	}

	promAPI := prometheusv1.NewAPI(prometheusClient)

	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		c, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		var resp []apiv1.ClusterMetric

		val, err := prometheusQuery(ctx, promAPI, fmt.Sprintf(`sum(machine_controller_machines{cluster="%s"})`, c.Name))
		if err != nil {
			return nil, err
		}
		resp = append(resp, apiv1.ClusterMetric{
			Name:   "Machines",
			Values: []float64{val},
		})

		vals, err := prometheusQueryRange(ctx, promAPI, fmt.Sprintf(`sum(machine_controller_machines{cluster="%s"})`, c.Name))
		if err != nil {
			return nil, err
		}
		resp = append(resp, apiv1.ClusterMetric{
			Name:   "Machines (1h)",
			Values: vals,
		})

		return resp, nil
	}
}

func prometheusQuery(ctx context.Context, api prometheusv1.API, query string) (float64, error) {
	now := time.Now()
	val, err := api.Query(ctx, query, now)
	if err != nil {
		return 0, nil
	}
	if val.Type() != model.ValVector {
		return 0, fmt.Errorf("failed to retrieve correct value type")
	}

	vec := val.(model.Vector)
	for _, sample := range vec {
		return float64(sample.Value), nil
	}

	return 0, nil
}

func prometheusQueryRange(ctx context.Context, api prometheusv1.API, query string) ([]float64, error) {
	now := time.Now()
	val, err := api.QueryRange(ctx, query, prometheusv1.Range{
		Start: now.Add(-1 * time.Hour),
		End:   now,
		Step:  30 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	if val.Type() != model.ValMatrix {
		return nil, fmt.Errorf("failed to retrieve correct value type")
	}

	var vals []float64
	matrix := val.(model.Matrix)
	for _, sample := range matrix {
		for _, v := range sample.Values {
			vals = append(vals, float64(v.Value))
		}
	}

	return vals, nil
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
func (r CreateReq) Validate() error {
	if len(r.ProjectID) == 0 || len(r.DC) == 0 {
		return fmt.Errorf("the service account ID and datacenter cannot be empty")
	}

	for _, clusterType := range clusterTypes {
		if clusterType == r.Body.Cluster.Type {
			return nil
		}
	}
	return fmt.Errorf("invalid cluster type %s", r.Body.Cluster.Type)
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
	Patch []byte
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

// AdminTokenReq defines HTTP request data for revokeClusterAdminToken endpoints.
// swagger:parameters revokeClusterAdminToken
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

func RevokeAdminTokenEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AdminTokenReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster.Address.AdminToken = kubernetes.GenerateToken()

		_, err = clusterProvider.Update(userInfo, cluster)
		return nil, common.KubernetesErrorToHTTPError(err)
	}
}

type DeleteReq struct {
	common.GetClusterReq
	// DeleteVolumes if true all cluster PV's and PVC's will be deleted from cluster
	DeleteVolumes bool
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

func getClusters(clusterProvider provider.ClusterProvider, userInfo *provider.UserInfo, projectProvider provider.ProjectProvider, projectID string) ([]*apiv1.Cluster, error) {
	project, err := projectProvider.Get(userInfo, projectID, &provider.ProjectGetOptions{})
	if err != nil {
		return nil, err
	}

	clusters, err := clusterProvider.List(project, nil)
	if err != nil {
		return nil, err
	}

	apiClusters := convertInternalClustersToExternal(clusters)
	return apiClusters, nil
}

// EventsReq defines HTTP request for getClusterEvents endpoint
// swagger:parameters getClusterEvents
type EventsReq struct {
	common.GetClusterReq

	// in: query
	Type string
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
