package common

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	apiv1 "github.com/kubermatic/kubermatic/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/pkg/handler/v1/label"
	kuberneteshelper "github.com/kubermatic/kubermatic/pkg/kubernetes"
	kubermaticlog "github.com/kubermatic/kubermatic/pkg/log"
	"github.com/kubermatic/kubermatic/pkg/provider"
	kubernetesprovider "github.com/kubermatic/kubermatic/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/pkg/resources/cloudcontroller"
	"github.com/kubermatic/kubermatic/pkg/resources/cluster"
	machineresource "github.com/kubermatic/kubermatic/pkg/resources/machine"
	"github.com/kubermatic/kubermatic/pkg/util/errors"
	"github.com/kubermatic/kubermatic/pkg/validation"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/rand"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
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

	if cloudcontroller.ExternalCloudControllerFeatureSupported(partialCluster) {
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
		_, err := getInternalCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project, projectID, newCluster.Name, &provider.ClusterGetOptions{})
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
