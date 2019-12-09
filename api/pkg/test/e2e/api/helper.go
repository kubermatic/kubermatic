package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/Masterminds/semver"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	oidc "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils"
	apiclient "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/client"
	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/client/admin"
	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/client/credentials"
	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/client/gcp"
	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/client/project"
	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/client/serviceaccounts"
	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/client/tokens"
	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/client/users"
	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	defaultIssuerURL       = "http://dex.oauth:5556"
	defaultHost            = "localhost:8080"
	defaultScheme          = "http"
	defaultIssuerURLPrefix = ""
	maxAttempts            = 8
	timeout                = time.Second * 4
)

type APIRunner struct {
	client      *apiclient.Kubermatic
	bearerToken runtime.ClientAuthInfoWriter
	test        *testing.T
}

func GetMasterToken() (string, error) {

	var hClient = &http.Client{
		Timeout: time.Second * 10,
	}

	u, err := getIssuerURL()
	if err != nil {
		return "", err
	}

	issuerURLPrefix := getIssuerURLPrefix()

	var requestToken string
	if err := wait.PollImmediate(10*time.Second, time.Minute, func() (bool, error) {
		requestToken, err = oidc.GetOIDCReqToken(hClient, u, issuerURLPrefix, "http://localhost:8000")
		if err != nil {
			fmt.Printf("Failed to get OIDC request token: %v\n", err)
			return false, nil
		}
		return true, nil
	}); err != nil {
		return "", fmt.Errorf("failed to get OIDC request token: %v", err)
	}

	login, password := oidc.GetOIDCClient()

	var oidcAuthToken string
	if err := wait.PollImmediate(10*time.Second, time.Minute, func() (bool, error) {
		oidcAuthToken, err = oidc.GetOIDCAuthToken(hClient, requestToken, u, issuerURLPrefix, login, password)
		if err != nil {
			fmt.Printf("Failed to get OIDC auth token: %v\n", err)
			return false, nil
		}
		return true, nil
	}); err != nil {
		return "", fmt.Errorf("failed to get OIDC quath token: %v", err)
	}

	return oidcAuthToken, nil
}

func GetAdminMasterToken() (string, error) {
	var hClient = &http.Client{
		Timeout: time.Second * 10,
	}

	u, err := getIssuerURL()
	if err != nil {
		return "", err
	}

	issuerURLPrefix := getIssuerURLPrefix()

	requestToken, err := oidc.GetOIDCReqToken(hClient, u, issuerURLPrefix, "http://localhost:8000")
	if err != nil {
		return "", err
	}

	return oidc.GetOIDCAuthToken(hClient, requestToken, u, issuerURLPrefix, "roxy2@loodse.com", "password")
}

func getHost() string {
	host := os.Getenv("KUBERMATIC_HOST")
	if len(host) > 0 {
		return host
	}
	return defaultHost
}

func getScheme() string {
	scheme := os.Getenv("KUBERMATIC_SCHEME")
	if len(scheme) > 0 {
		return scheme
	}
	return defaultScheme
}

func getIssuerURL() (url.URL, error) {
	issuerURL := os.Getenv("KUBERMATIC_OIDC_ISSUER")
	if len(issuerURL) == 0 {
		issuerURL = defaultIssuerURL
	}
	u, err := url.Parse(issuerURL)
	if err != nil {
		return url.URL{}, err
	}
	return *u, nil
}

func getIssuerURLPrefix() string {
	prefix := os.Getenv("KUBERMATIC_OIDC_ISSUER_URL_PREFIX")
	if len(prefix) > 0 {
		return prefix
	}
	return defaultIssuerURLPrefix
}

// CreateAPIRunner util method to create APIRunner
func CreateAPIRunner(token string, t *testing.T) *APIRunner {
	client := apiclient.New(httptransport.New(getHost(), "", []string{getScheme()}), strfmt.Default)

	bearerTokenAuth := httptransport.BearerToken(token)
	return &APIRunner{
		client:      client,
		bearerToken: bearerTokenAuth,
		test:        t,
	}
}

// CreateProject creates a new project
func (r *APIRunner) CreateProject(name string) (*apiv1.Project, error) {
	params := &project.CreateProjectParams{Body: project.CreateProjectBody{Name: name}}
	params.WithTimeout(timeout)
	project, err := r.client.Project.CreateProject(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	var apiProject *apiv1.Project
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		apiProject, err = r.GetProject(project.Payload.ID, maxAttempts)
		if err != nil {
			return nil, err
		}

		if apiProject.Status == kubermaticv1.ProjectActive {
			break
		}
		time.Sleep(time.Second)
	}

	if apiProject.Status != kubermaticv1.ProjectActive {
		return nil, fmt.Errorf("project is not redy after %d attempts", maxAttempts)
	}

	return apiProject, nil
}

// GetProject gets the project with the given ID
func (r *APIRunner) GetProject(id string, attempts int) (*apiv1.Project, error) {
	params := &project.GetProjectParams{ProjectID: id}
	params.WithTimeout(timeout)

	var err error
	var project *project.GetProjectOK
	for attempt := 0; attempt <= attempts; attempt++ {
		project, err = r.client.Project.GetProject(params, r.bearerToken)
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		return nil, err
	}

	return convertProject(project.Payload)
}

// UpdateProject updates the given project
func (r *APIRunner) UpdateProject(projectToUpdate *apiv1.Project) (*apiv1.Project, error) {
	params := &project.UpdateProjectParams{ProjectID: projectToUpdate.ID, Body: &models.Project{Name: projectToUpdate.Name}}
	params.WithTimeout(timeout)
	project, err := r.client.Project.UpdateProject(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return convertProject(project.Payload)
}

func convertProject(project *models.Project) (*apiv1.Project, error) {
	apiProject := &apiv1.Project{}
	apiProject.Name = project.Name
	apiProject.ID = project.ID
	apiProject.Status = project.Status

	creationTime, err := time.Parse(time.RFC3339, project.CreationTimestamp.String())
	if err != nil {
		return nil, err
	}
	apiProject.CreationTimestamp = apiv1.NewTime(creationTime)

	return apiProject, nil
}

// DeleteProject deletes given project
func (r *APIRunner) DeleteProject(id string) error {
	params := &project.DeleteProjectParams{ProjectID: id}
	params.WithTimeout(timeout)
	if _, err := r.client.Project.DeleteProject(params, r.bearerToken); err != nil {
		return err
	}
	return nil
}

// CreateServiceAccount method creates a new service account
func (r *APIRunner) CreateServiceAccount(name, group, projectID string) (*apiv1.ServiceAccount, error) {
	params := &serviceaccounts.AddServiceAccountToProjectParams{ProjectID: projectID, Body: &models.ServiceAccount{Name: name, Group: group}}
	params.WithTimeout(timeout)
	params.SetTimeout(timeout)
	sa, err := r.client.Serviceaccounts.AddServiceAccountToProject(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	var apiServiceAccount *apiv1.ServiceAccount
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		apiServiceAccount, err = r.GetServiceAccount(sa.Payload.ID, projectID)
		if err != nil {
			return nil, err
		}

		if apiServiceAccount.Status == apiv1.ServiceAccountActive {
			break
		}
		time.Sleep(time.Second)
	}
	if apiServiceAccount.Status != apiv1.ServiceAccountActive {
		return nil, fmt.Errorf("service account is not redy after %d attempts", maxAttempts)
	}

	return apiServiceAccount, nil
}

// GetServiceAccount returns service account for given ID and project
func (r *APIRunner) GetServiceAccount(saID, projectID string) (*apiv1.ServiceAccount, error) {
	params := &serviceaccounts.ListServiceAccountsParams{ProjectID: projectID}
	params.WithTimeout(timeout)

	var err error
	var saList *serviceaccounts.ListServiceAccountsOK
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		saList, err = r.client.Serviceaccounts.ListServiceAccounts(params, r.bearerToken)
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		return nil, err
	}

	for _, sa := range saList.Payload {
		if sa.ID == saID {
			return convertServiceAccount(sa)
		}
	}

	return nil, fmt.Errorf("service account %s not found", saID)
}

func convertServiceAccount(sa *models.ServiceAccount) (*apiv1.ServiceAccount, error) {
	apiServiceAccount := &apiv1.ServiceAccount{}
	apiServiceAccount.ID = sa.ID
	apiServiceAccount.Group = sa.Group
	apiServiceAccount.Name = sa.Name
	apiServiceAccount.Status = sa.Status

	creationTime, err := time.Parse(time.RFC3339, sa.CreationTimestamp.String())
	if err != nil {
		return nil, err
	}
	apiServiceAccount.CreationTimestamp = apiv1.NewTime(creationTime)

	return apiServiceAccount, nil
}

// AddTokenToServiceAccount creates a new token for service account
func (r *APIRunner) AddTokenToServiceAccount(name, saID, projectID string) (*apiv1.ServiceAccountToken, error) {
	params := &tokens.AddTokenToServiceAccountParams{ProjectID: projectID, ServiceaccountID: saID, Body: &models.ServiceAccountToken{Name: name}}
	params.WithTimeout(timeout)
	token, err := r.client.Tokens.AddTokenToServiceAccount(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return convertServiceAccountToken(token.Payload)
}

func convertServiceAccountToken(saToken *models.ServiceAccountToken) (*apiv1.ServiceAccountToken, error) {
	apiServiceAccountToken := &apiv1.ServiceAccountToken{}
	apiServiceAccountToken.ID = saToken.ID
	apiServiceAccountToken.Name = saToken.Name
	apiServiceAccountToken.Token = saToken.Token

	expiry, err := time.Parse(time.RFC3339, saToken.Expiry.String())
	if err != nil {
		return nil, err
	}
	apiServiceAccountToken.Expiry = apiv1.NewTime(expiry)

	return apiServiceAccountToken, nil
}

// ListCredentials returns list of credential names for the provider
func (r *APIRunner) ListCredentials(providerName string) ([]string, error) {
	params := &credentials.ListCredentialsParams{ProviderName: providerName}
	params.WithTimeout(timeout)
	credentialsResponse, err := r.client.Credentials.ListCredentials(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0)
	names = append(names, credentialsResponse.Payload.Names...)

	return names, nil
}

// CreateAWSCluster creates cluster for AWS provider
func (r *APIRunner) CreateAWSCluster(projectID, dc, name, secretAccessKey, accessKeyID, version, location, availabilityZone string, replicas int32) (*apiv1.Cluster, error) {

	vr, err := semver.NewVersion(version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version %s: %v", version, err)
	}

	instanceType := "t3.small"
	volumeSize := int64(25)
	volumeType := "standard"
	clusterSpec := &models.CreateClusterSpec{}
	clusterSpec.Cluster = &models.Cluster{
		Type: "kubernetes",
		Name: name,
		Spec: &models.ClusterSpec{
			Cloud: &models.CloudSpec{
				DatacenterName: location,
				Aws: &models.AWSCloudSpec{
					SecretAccessKey: secretAccessKey,
					AccessKeyID:     accessKeyID,
				},
			},
			Version: vr,
		},
	}
	clusterSpec.NodeDeployment = &models.NodeDeployment{
		Spec: &models.NodeDeploymentSpec{
			Replicas: &replicas,
			Template: &models.NodeSpec{
				Cloud: &models.NodeCloudSpec{
					Aws: &models.AWSNodeSpec{
						AvailabilityZone: availabilityZone,
						InstanceType:     &instanceType,
						VolumeSize:       &volumeSize,
						VolumeType:       &volumeType,
					},
				},
				OperatingSystem: &models.OperatingSystemSpec{
					Ubuntu: &models.UbuntuSpec{
						DistUpgradeOnBoot: false,
					},
				},
			},
		},
	}

	params := &project.CreateClusterParams{ProjectID: projectID, Dc: dc, Body: clusterSpec}
	params.WithTimeout(timeout)
	clusterResponse, err := r.client.Project.CreateCluster(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return convertCluster(clusterResponse.Payload)
}

// CreateDOCluster creates cluster for DigitalOcean provider
func (r *APIRunner) CreateDOCluster(projectID, dc, name, credential, version, location string, replicas int32) (*apiv1.Cluster, error) {

	vr, err := semver.NewVersion(version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version %s: %v", version, err)
	}

	instanceSize := "s-1vcpu-1gb"

	clusterSpec := &models.CreateClusterSpec{}
	clusterSpec.Cluster = &models.Cluster{
		Type:       "kubernetes",
		Name:       name,
		Credential: credential,
		Spec: &models.ClusterSpec{
			Cloud: &models.CloudSpec{
				DatacenterName: location,
				Digitalocean:   &models.DigitaloceanCloudSpec{},
			},
			Version: vr,
		},
	}
	clusterSpec.NodeDeployment = &models.NodeDeployment{
		Spec: &models.NodeDeploymentSpec{
			Replicas: &replicas,
			Template: &models.NodeSpec{
				Cloud: &models.NodeCloudSpec{
					Digitalocean: &models.DigitaloceanNodeSpec{
						Size:       &instanceSize,
						Backups:    false,
						IPV6:       false,
						Monitoring: false,
					},
				},
				OperatingSystem: &models.OperatingSystemSpec{
					Ubuntu: &models.UbuntuSpec{
						DistUpgradeOnBoot: false,
					},
				},
			},
		},
	}

	params := &project.CreateClusterParams{ProjectID: projectID, Dc: dc, Body: clusterSpec}
	params.WithTimeout(timeout * 2)
	clusterResponse, err := r.client.Project.CreateCluster(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return convertCluster(clusterResponse.Payload)
}

// DeleteCluster delete cluster method
func (r *APIRunner) DeleteCluster(projectID, dc, clusterID string) error {

	params := &project.DeleteClusterParams{ProjectID: projectID, Dc: dc, ClusterID: clusterID}
	params.WithTimeout(timeout)

	if _, err := r.client.Project.DeleteCluster(params, r.bearerToken); err != nil {
		return err
	}
	return nil
}

// GetCluster cluster getter
func (r *APIRunner) GetCluster(projectID, dc, clusterID string) (*apiv1.Cluster, error) {

	params := &project.GetClusterParams{ProjectID: projectID, Dc: dc, ClusterID: clusterID}
	params.WithTimeout(timeout)

	cluster, err := r.client.Project.GetCluster(params, r.bearerToken)
	if err != nil {
		return nil, err
	}
	return convertCluster(cluster.Payload)
}

// GetClusterEvents returns the cluster events
func (r *APIRunner) GetClusterEvents(projectID, dc, clusterID string) ([]*models.Event, error) {
	params := &project.GetClusterEventsParams{ProjectID: projectID, Dc: dc, ClusterID: clusterID}
	params.WithTimeout(timeout)

	events, err := r.client.Project.GetClusterEvents(params, r.bearerToken)
	if err != nil {
		return nil, err
	}
	return events.Payload, nil
}

// PrintClusterEvents prints all cluster events using its test.Logf
func (r *APIRunner) PrintClusterEvents(projectID, dc, clusterID string) error {
	events, err := r.GetClusterEvents(projectID, dc, clusterID)
	if err != nil {
		return fmt.Errorf("failed to get cluster events: %v", err)
	}
	encodedEvents, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("failed to serialize events: %v", err)
	}
	r.test.Logf("Cluster events:\n%s", string(encodedEvents))
	return nil
}

// GetClusterHealthStatus gets the cluster status
func (r *APIRunner) GetClusterHealthStatus(projectID, dc, clusterID string) (*apiv1.ClusterHealth, error) {
	params := &project.GetClusterHealthParams{Dc: dc, ProjectID: projectID, ClusterID: clusterID}
	params.WithTimeout(timeout)

	var err error
	var response *project.GetClusterHealthOK
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		response, err = r.client.Project.GetClusterHealth(params, r.bearerToken)
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		return nil, err
	}

	apiClusterHealth := &apiv1.ClusterHealth{}
	apiClusterHealth.Apiserver = convertHealthStatus(response.Payload.Apiserver)
	apiClusterHealth.Controller = convertHealthStatus(response.Payload.Controller)
	apiClusterHealth.Etcd = convertHealthStatus(response.Payload.Etcd)
	apiClusterHealth.MachineController = convertHealthStatus(response.Payload.MachineController)
	apiClusterHealth.Scheduler = convertHealthStatus(response.Payload.Scheduler)
	apiClusterHealth.UserClusterControllerManager = convertHealthStatus(response.Payload.UserClusterControllerManager)

	return apiClusterHealth, nil
}

func convertHealthStatus(status models.HealthStatus) kubermaticv1.HealthStatus {
	switch int64(status) {
	case int64(kubermaticv1.HealthStatusProvisioning):
		return kubermaticv1.HealthStatusProvisioning
	case int64(kubermaticv1.HealthStatusUp):
		return kubermaticv1.HealthStatusUp
	default:
		return kubermaticv1.HealthStatusDown
	}
}

// GetClusterNodeDeployment returns the cluster node deployments
func (r *APIRunner) GetClusterNodeDeployment(projectID, dc, clusterID string) ([]apiv1.NodeDeployment, error) {
	params := &project.ListNodeDeploymentsParams{ClusterID: clusterID, ProjectID: projectID, Dc: dc}
	params.WithTimeout(timeout * 2)

	var err error
	var response *project.ListNodeDeploymentsOK
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		response, err = r.client.Project.ListNodeDeployments(params, r.bearerToken)
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	if err != nil {
		return nil, err
	}
	list := make([]apiv1.NodeDeployment, 0)
	for _, nd := range response.Payload {
		apiNd := apiv1.NodeDeployment{}
		apiNd.Name = nd.Name
		apiNd.ID = nd.ID
		apiNd.Status = v1alpha1.MachineDeploymentStatus{
			Replicas:          nd.Status.Replicas,
			AvailableReplicas: nd.Status.AvailableReplicas,
		}
		list = append(list, apiNd)
	}

	return list, nil
}

func convertCluster(cluster *models.Cluster) (*apiv1.Cluster, error) {
	apiCluster := &apiv1.Cluster{}
	apiCluster.ID = cluster.ID
	apiCluster.Name = cluster.Name
	apiCluster.Type = cluster.Type

	creationTime, err := time.Parse(time.RFC3339, cluster.CreationTimestamp.String())
	if err != nil {
		return nil, err
	}
	apiCluster.CreationTimestamp = apiv1.NewTime(creationTime)

	return apiCluster, nil
}

// ListGCPZones returns list of GCP zones
func (r *APIRunner) ListGCPZones(credential, dc string) ([]string, error) {
	params := &gcp.ListGCPZonesParams{Credential: &credential, Dc: dc}
	params.WithTimeout(timeout)
	zonesResponse, err := r.client.Gcp.ListGCPZones(params, r.bearerToken)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0)
	for _, name := range zonesResponse.Payload {
		names = append(names, name.Name)
	}

	return names, nil
}

// ListGCPDiskTypes returns list of GCP disk types
func (r *APIRunner) ListGCPDiskTypes(credential, zone string) ([]string, error) {
	params := &gcp.ListGCPDiskTypesParams{Credential: &credential, Zone: &zone}
	params.WithTimeout(timeout)
	typesResponse, err := r.client.Gcp.ListGCPDiskTypes(params, r.bearerToken)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0)
	for _, name := range typesResponse.Payload {
		names = append(names, name.Name)
	}

	return names, nil
}

// ListGCPSizes returns list of GCP sizes
func (r *APIRunner) ListGCPSizes(credential, zone string) ([]apiv1.GCPMachineSize, error) {
	params := &gcp.ListGCPSizesParams{Credential: &credential, Zone: &zone}
	params.WithTimeout(timeout)
	sizesResponse, err := r.client.Gcp.ListGCPSizes(params, r.bearerToken)
	if err != nil {
		return nil, err
	}
	sizes := make([]apiv1.GCPMachineSize, 0)
	for _, machineType := range sizesResponse.Payload {
		mt := apiv1.GCPMachineSize{
			Name:        machineType.Name,
			Description: machineType.Description,
			Memory:      machineType.Memory,
			VCPUs:       machineType.VCpus,
		}
		sizes = append(sizes, mt)
	}

	return sizes, nil
}

// GetErrorResponse converts the client error response to string
func GetErrorResponse(err error) string {
	rawData, newErr := json.Marshal(err)
	if newErr != nil {
		return err.Error()
	}
	return string(rawData)
}

// IsHealthyCluster check if all cluster components are up
func IsHealthyCluster(healthStatus *apiv1.ClusterHealth) bool {
	if healthStatus.UserClusterControllerManager == kubermaticv1.HealthStatusUp && healthStatus.Scheduler == kubermaticv1.HealthStatusUp &&
		healthStatus.MachineController == kubermaticv1.HealthStatusUp && healthStatus.Etcd == kubermaticv1.HealthStatusUp &&
		healthStatus.Controller == kubermaticv1.HealthStatusUp && healthStatus.Apiserver == kubermaticv1.HealthStatusUp {
		return true
	}
	return false
}

func cleanUpProject(id string, attempts int) func(t *testing.T) {
	return func(t *testing.T) {
		masterToken, err := GetMasterToken()
		if err != nil {
			t.Fatalf("can not get master token due error: %v", err)
		}
		apiRunner := CreateAPIRunner(masterToken, t)

		if err := apiRunner.DeleteProject(id); err != nil {
			t.Fatalf("can not delete project due error: %v", err)
		}
		t.Log("project deleting ...")
		for attempt := 1; attempt <= attempts; attempt++ {
			_, err := apiRunner.GetProject(id, 5)
			if err != nil {
				break
			}
			time.Sleep(3 * time.Second)
		}
		_, err = apiRunner.GetProject(id, 5)
		if err == nil {
			t.Fatalf("can not delete the project")
		}
		t.Log("project deleted successfully")
	}
}

func cleanUpCluster(t *testing.T, apiRunner *APIRunner, projectID, dc, clusterID string) {
	if err := apiRunner.DeleteCluster(projectID, dc, clusterID); err != nil {
		t.Fatalf("can not delete the cluster %v", GetErrorResponse(err))
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_, err := apiRunner.GetCluster(projectID, dc, clusterID)
		if err != nil {
			t.Logf("cluster deleted %v", GetErrorResponse(err))
			break
		}
		time.Sleep(60 * time.Second)
	}
	_, err := apiRunner.GetCluster(projectID, dc, clusterID)
	if err == nil {
		t.Fatalf("can not delete the cluster after %d attempts", maxAttempts)
	}
}

func (r *APIRunner) DeleteUserFromProject(projectID, userID string) error {
	params := &users.DeleteUserFromProjectParams{ProjectID: projectID, UserID: userID}
	params.WithTimeout(timeout)
	if _, err := r.client.Users.DeleteUserFromProject(params, r.bearerToken); err != nil {
		return err
	}
	return nil
}

func (r *APIRunner) GetProjectUsers(projectID string) ([]apiv1.User, error) {
	params := &users.GetUsersForProjectParams{ProjectID: projectID}
	params.WithTimeout(timeout)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_, err := r.client.Users.GetUsersForProject(params, r.bearerToken)
		if err != nil {
			break
		}
		time.Sleep(2 * time.Second)
	}

	responseUsers, err := r.client.Users.GetUsersForProject(params, r.bearerToken)
	if err != nil {
		return nil, err
	}
	users := make([]apiv1.User, 0)
	for _, user := range responseUsers.Payload {
		usr := apiv1.User{
			Email: user.Email,
			ObjectMeta: apiv1.ObjectMeta{
				ID:   user.ID,
				Name: user.Name,
			},
		}
		users = append(users, usr)
	}

	return users, nil
}

func (r *APIRunner) AddProjectUser(projectID, email, name, group string) (*apiv1.User, error) {
	params := &users.AddUserToProjectParams{ProjectID: projectID, Body: &models.User{
		Email: email,
		Name:  name,
		Projects: []*models.ProjectGroup{
			{ID: projectID,
				GroupPrefix: group,
			},
		},
	}}
	params.WithTimeout(timeout)
	responseUser, err := r.client.Users.AddUserToProject(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	usr := &apiv1.User{
		Email: responseUser.Payload.Email,
		ObjectMeta: apiv1.ObjectMeta{
			ID:   responseUser.Payload.ID,
			Name: responseUser.Payload.Name,
		},
	}
	return usr, nil
}

// RunOIDCProxy runs the OIDC proxy. It is non-blocking. It does
// so by shelling out which is not pretty, but better than the previous
// approach of forking in a script and having no way of making the test
// fail of the OIDC failed
func RunOIDCProxy(errChan chan error, cancel <-chan struct{}) error {
	gopathRaw, err := exec.Command("go", "env", "GOPATH").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get gopath: %v", err)
	}
	goPathSanitized := strings.Replace(string(gopathRaw), "\n", "", -1)
	oidcProxyDir := fmt.Sprintf("%s/src/github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/oidc-proxy-client", goPathSanitized)

	oidProxyCommand := exec.Command("make", "run")
	oidProxyCommand.Dir = oidcProxyDir
	var out bytes.Buffer
	oidProxyCommand.Stdout = &out
	oidProxyCommand.Stderr = &out
	if err := oidProxyCommand.Start(); err != nil {
		return fmt.Errorf("failed to run oidc proxy command: %v", err)
	}
	defer func() {
		fmt.Printf("Output of oidc proxy:\n---\n%s\n---\n", out.String())
	}()

	go func() {
		if err := oidProxyCommand.Wait(); err != nil {
			errChan <- fmt.Errorf("failed to run oidc proxy. Output:\n%s\nError: %v", out.String(), err)
		}
	}()

	return nil
}

func (r *APIRunner) GetGlobalSettings() (*apiv1.GlobalSettings, error) {
	params := &admin.GetKubermaticSettingsParams{}
	params.WithTimeout(timeout)
	responseSettings, err := r.client.Admin.GetKubermaticSettings(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return convertGlobalSettings(responseSettings.Payload), nil
}

func (r *APIRunner) UpdateGlobalSettings(s string) (*apiv1.GlobalSettings, error) {
	params := &admin.PatchKubermaticSettingsParams{
		Patch: []uint8(s),
	}
	params.WithTimeout(timeout)
	responseSettings, err := r.client.Admin.PatchKubermaticSettings(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return convertGlobalSettings(responseSettings.Payload), nil
}

func convertGlobalSettings(gSettings *models.GlobalSettings) *apiv1.GlobalSettings {
	var customLinks kubermaticv1.CustomLinks
	for _, customLink := range gSettings.CustomLinks {
		customLinks = append(customLinks, kubermaticv1.CustomLink{
			Label:    customLink.Label,
			URL:      customLink.URL,
			Icon:     customLink.Icon,
			Location: customLink.Location,
		})
	}

	return &apiv1.GlobalSettings{
		CustomLinks: customLinks,
		CleanupOptions: kubermaticv1.CleanupOptions{
			Enabled:  gSettings.CleanupOptions.Enabled,
			Enforced: gSettings.CleanupOptions.Enforced,
		},
		DefaultNodeCount:      gSettings.DefaultNodeCount,
		ClusterTypeOptions:    gSettings.ClusterTypeOptions,
		DisplayDemoInfo:       gSettings.DisplayDemoInfo,
		DisplayAPIDocs:        gSettings.DisplayAPIDocs,
		DisplayTermsOfService: gSettings.DisplayTermsOfService,
	}
}

func (r *APIRunner) SetAdmin(email string, isAdmin bool) error {
	params := &admin.SetAdminParams{
		Body: &models.Admin{
			Email:   email,
			IsAdmin: isAdmin,
		},
	}
	params.WithTimeout(timeout)
	_, err := r.client.Admin.SetAdmin(params, r.bearerToken)
	if err != nil {
		return err
	}

	return nil
}

// GetUserClusterRoles
func (r *APIRunner) GetUserClusterRoles(projectID, dc, clusterID string) ([]apiv1.RoleName, error) {
	params := &project.ListRoleNamesParams{Dc: dc, ProjectID: projectID, ClusterID: clusterID}
	params.WithTimeout(timeout)

	var err error
	var response *project.ListRoleNamesOK
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		response, err = r.client.Project.ListRoleNames(params, r.bearerToken)
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		return nil, err
	}

	roleNames := []apiv1.RoleName{}

	for _, roleName := range response.Payload {
		roleNames = append(roleNames, apiv1.RoleName{
			Name:      roleName.Name,
			Namespace: roleName.Namespace,
		})
	}

	return roleNames, nil
}

// BindUserToRole
func (r *APIRunner) BindUserToRole(projectID, dc, clusterID, roleName, namespace, user string) (*apiv1.RoleBinding, error) {
	params := &project.BindUserToRoleParams{
		Body:      &models.RoleUser{UserEmail: user},
		ClusterID: clusterID,
		Dc:        dc,
		Namespace: namespace,
		ProjectID: projectID,
		RoleID:    roleName,
	}
	params.WithTimeout(timeout)

	var err error
	var response *project.BindUserToRoleOK
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		response, err = r.client.Project.BindUserToRole(params, r.bearerToken)
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		return nil, err
	}

	return &apiv1.RoleBinding{
		Namespace:   response.Payload.Namespace,
		RoleRefName: response.Payload.RoleRefName,
	}, nil
}
