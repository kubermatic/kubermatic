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
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	kubernetesprovider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/resources/cluster"
	machineresource "github.com/kubermatic/kubermatic/api/pkg/resources/machine"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	kubermaticerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"github.com/kubermatic/kubermatic/api/pkg/validation"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/rand"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
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
var clusterTypes = []string{
	apiv1.OpenShiftClusterType,
	apiv1.KubernetesClusterType,
}

func CreateEndpoint(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, seedsGetter provider.SeedsGetter,
	initNodeDeploymentFailures *prometheus.CounterVec, eventRecorderProvider provider.EventRecorderProvider, credentialManager common.PresetsManager, exposeStrategy corev1.ServiceType) endpoint.Endpoint {
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

		if req.Body.Cluster.ID != "" {
			return nil, errors.New(int(http.StatusBadRequest), "cluster.ID is read-only")
		}

		_, dc, err := provider.DatacenterFromSeedMap(seedsGetter, req.Body.Cluster.Spec.Cloud.DatacenterName)
		if err != nil {
			return nil, fmt.Errorf("error getting dc: %v", err)
		}

		credentialName := req.Body.Cluster.Credential
		if len(credentialName) > 0 {
			cloudSpec, err := credentialManager.SetCloudCredentials(userInfo, credentialName, req.Body.Cluster.Spec.Cloud, dc)
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
		spec.ExposeStrategy = exposeStrategy

		existingClusters, err := clusterProvider.List(project, &provider.ClusterListOptions{ClusterSpecName: spec.HumanReadableName})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if len(existingClusters.Items) > 0 {
			return nil, errors.NewAlreadyExists("cluster", spec.HumanReadableName)
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
		// generate the name here so that it can be used in the secretName below
		partialCluster.Name = rand.String(10)

		if err := kubernetesprovider.CreateCredentialSecretForCluster(ctx, privilegedClusterProvider.GetSeedClusterAdminRuntimeClient(), partialCluster, req.ProjectID); err != nil {
			return nil, err
		}
		kuberneteshelper.AddFinalizer(partialCluster, apiv1.CredentialsSecretsCleanupFinalizer)

		newCluster, err := clusterProvider.New(project, userInfo, partialCluster)
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
			if isBYO {
				glog.V(5).Infof("KubeAdm provider detected an initial node deployment won't be created for cluster %s", newCluster.Name)
				return convertInternalClusterToExternal(newCluster), nil
			}

			go func() {
				defer utilruntime.HandleCrash()
				ndName := getNodeDeploymentDisplayName(req.Body.NodeDeployment)
				eventRecorderProvider.ClusterRecorderFor(k8sClient).Eventf(newCluster, corev1.EventTypeNormal, string(nodeDeploymentCreationStart), "Started creation of initial node deployment %s", ndName)
				err := createInitialNodeDeploymentWithRetries(req.Body.NodeDeployment, newCluster, project, sshKeyProvider, seedsGetter, clusterProvider, userInfo)
				if err != nil {
					eventRecorderProvider.ClusterRecorderFor(k8sClient).Eventf(newCluster, corev1.EventTypeWarning, string(nodeDeploymentCreationFail), "Failed to create initial node deployment %s: %v", ndName, err)
					glog.Errorf("failed to create initial node deployment for cluster %s: %v", newCluster.Name, err)
					initNodeDeploymentFailures.With(prometheus.Labels{"cluster": newCluster.Name, "datacenter": req.Body.Cluster.Spec.Cloud.DatacenterName}).Add(1)
				} else {
					eventRecorderProvider.ClusterRecorderFor(k8sClient).Eventf(newCluster, corev1.EventTypeNormal, string(nodeDeploymentCreationSuccess), "Successfully created initial node deployment %s", ndName)
					glog.V(5).Infof("created initial node deployment for cluster %s", newCluster.Name)
				}
			}()
		}

		return convertInternalClusterToExternal(newCluster), nil
	}
}

func createInitialNodeDeploymentWithRetries(nodeDeployment *apiv1.NodeDeployment, cluster *kubermaticv1.Cluster,
	project *kubermaticv1.Project, sshKeyProvider provider.SSHKeyProvider,
	seedsGetter provider.SeedsGetter, clusterProvider provider.ClusterProvider, userInfo *provider.UserInfo) error {
	return wait.Poll(5*time.Second, 30*time.Minute, func() (bool, error) {
		err := createInitialNodeDeployment(nodeDeployment, cluster, project, sshKeyProvider, seedsGetter, clusterProvider, userInfo)
		if err != nil {
			// unrecoverable
			if strings.Contains(err.Error(), `admission webhook "machine-controller.kubermatic.io-machinedeployments" denied the request`) {
				glog.V(4).Infof("giving up creating initial Node Deployments for cluster %s (%s) due to an unrecoverabl err %#v", cluster.Name, cluster.Spec.HumanReadableName, err)
				return false, err
			}
			// Likely recoverable
			glog.V(4).Infof("retrying creating initial Node Deployments for cluster %s (%s) due to %v", cluster.Name, cluster.Spec.HumanReadableName, err)
			return false, nil
		}
		return true, nil
	})
}

func createInitialNodeDeployment(nodeDeployment *apiv1.NodeDeployment, cluster *kubermaticv1.Cluster,
	project *kubermaticv1.Project, sshKeyProvider provider.SSHKeyProvider,
	seedsGetter provider.SeedsGetter, clusterProvider provider.ClusterProvider, userInfo *provider.UserInfo) error {
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

	_, dc, err := provider.DatacenterFromSeedMap(seedsGetter, cluster.Spec.Cloud.DatacenterName)
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

func GetEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)

		cluster, err := GetCluster(ctx, req, projectProvider)
		if err != nil {
			return nil, err
		}

		return convertInternalClusterToExternal(cluster), nil
	}
}

// GetCluster returns the cluster for a given request
func GetCluster(ctx context.Context, req common.GetClusterReq, projectProvider provider.ProjectProvider) (*kubermaticv1.Cluster, error) {
	clusterProvider, ok := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	if !ok {
		return nil, errors.New(http.StatusInternalServerError, "no cluster in request")
	}
	privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
	userInfo, ok := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
	if !ok {
		return nil, errors.New(http.StatusInternalServerError, "no userInfo in request")
	}
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

func PatchEndpoint(projectProvider provider.ProjectProvider, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(PatchReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		oldInternalCluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// Converting to API type as it is the type exposed externally.
		externalCluster := convertInternalClusterToExternal(oldInternalCluster)

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
		newInternalCluster.Spec.AuditLogging = patchedCluster.Spec.AuditLogging
		newInternalCluster.Spec.Openshift = patchedCluster.Spec.Openshift

		incompatibleKubelets, err := common.CheckClusterVersionSkew(ctx, userInfo, clusterProvider, newInternalCluster)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing nodes' version skew: %v", err)
		}
		if len(incompatibleKubelets) > 0 {
			return nil, errors.NewBadRequest("Cluster contains nodes running the following incompatible kubelet versions: %v. Upgrade your nodes before you upgrade the cluster.", incompatibleKubelets)
		}

		_, dc, err := provider.DatacenterFromSeedMap(seedsGetter, newInternalCluster.Spec.Cloud.DatacenterName)
		if err != nil {
			return nil, fmt.Errorf("error getting dc: %v", err)
		}

		assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
		}
		if err := validation.ValidateUpdateCluster(ctx, newInternalCluster, oldInternalCluster, dc, assertedClusterProvider); err != nil {
			return nil, errors.NewBadRequest("invalid cluster: %v", err)
		}

		updatedCluster, err := clusterProvider.Update(project, userInfo, newInternalCluster)
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
func ListAllEndpoint(projectProvider provider.ProjectProvider, seedsGetter provider.SeedsGetter, clusterProviderGetter provider.ClusterProviderGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetProjectRq)
		allClusters := make([]*apiv1.Cluster, 0)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		seeds, err := seedsGetter()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		for _, seed := range seeds {
			clusterProvider, err := clusterProviderGetter(seed)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
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

		if _, err = clusterProvider.Update(project, userInfo, existingCluster); err != nil {
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
		_, err = sshKeyProvider.Update(userInfo, sshKey)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return apiKey, nil
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
		Labels: internalCluster.Labels,
		Spec: apiv1.ClusterSpec{
			Cloud:                               internalCluster.Spec.Cloud,
			Version:                             internalCluster.Spec.Version,
			MachineNetworks:                     internalCluster.Spec.MachineNetworks,
			OIDC:                                internalCluster.Spec.OIDC,
			UsePodSecurityPolicyAdmissionPlugin: internalCluster.Spec.UsePodSecurityPolicyAdmissionPlugin,
		},
		Status: apiv1.ClusterStatus{
			Version: internalCluster.Spec.Version,
			URL:     internalCluster.Address.URL,
		},
		Type: apiv1.KubernetesClusterType,
	}

	isOpenShift, ok := internalCluster.Annotations["kubermatic.io/openshift"]
	if ok && isOpenShift == "true" {
		cluster.Type = apiv1.OpenShiftClusterType
	}

	return cluster
}

func convertInternalClustersToExternal(internalClusters []kubermaticv1.Cluster) []*apiv1.Cluster {
	apiClusters := make([]*apiv1.Cluster, len(internalClusters))
	for index, cluster := range internalClusters {
		apiClusters[index] = convertInternalClusterToExternal(cluster.DeepCopy())
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

func GetMetricsEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		cluster, err := GetCluster(ctx, req, projectProvider)
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		nodeList := &corev1.NodeList{}
		if err := client.List(ctx, &ctrlruntimeclient.ListOptions{}, nodeList); err != nil {
			return nil, err
		}
		availableResources := make(map[string]corev1.ResourceList)
		for _, n := range nodeList.Items {
			availableResources[n.Name] = n.Status.Allocatable
		}

		dynamicCLient, err := clusterProvider.GetAdminClientForCustomerCluster(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		allNodeMetricsList := &v1beta1.NodeMetricsList{}
		if err := dynamicCLient.List(ctx, &ctrlruntimeclient.ListOptions{}, allNodeMetricsList); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		seedAdminClient := privilegedClusterProvider.GetSeedClusterAdminRuntimeClient()
		podMetricsList := &v1beta1.PodMetricsList{}
		if err := seedAdminClient.List(ctx, &ctrlruntimeclient.ListOptions{Namespace: fmt.Sprintf("cluster-%s", cluster.Name)}, podMetricsList); err != nil {
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
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster.Address.AdminToken = kuberneteshelper.GenerateToken()

		_, err = clusterProvider.Update(project, userInfo, cluster)
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

	apiClusters := convertInternalClustersToExternal(clusters.Items)
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

func ListNamespaceEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		namespaceList := &corev1.NamespaceList{}
		if err := client.List(ctx, &ctrlruntimeclient.ListOptions{}, namespaceList); err != nil {
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
	projectProvider provider.ProjectProvider) (*kubermaticv1.Cluster, *kubernetesprovider.ClusterProvider, error) {

	req, ok := request.(common.GetClusterReq)
	if !ok {
		return nil, nil, kubermaticerrors.New(http.StatusBadRequest, "invalid request")
	}
	cluster, err := GetCluster(ctx, req, projectProvider)
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
