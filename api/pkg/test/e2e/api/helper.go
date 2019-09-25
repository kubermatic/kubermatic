package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	oidc "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils"
	apiclient "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/client"
	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/client/credentials"
	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/client/project"
	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/client/serviceaccounts"
	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/client/tokens"
	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/client/users"
	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"

	"github.com/Masterminds/semver"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
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

func CleanUpProject(id string, attempts int) func(t *testing.T) {
	return func(t *testing.T) {
		masterToken, err := GetMasterToken()
		if err != nil {
			t.Fatalf("can not get master token due error: %v", err)
		}
		apiRunner := CreateAPIRunner(masterToken, t)

		if err := apiRunner.DeleteProject(id); err != nil {
			t.Fatalf("can not delete project due error: %v", err)
		}
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
	}
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

	requestToken, err := oidc.GetOIDCReqToken(hClient, u, issuerURLPrefix, "http://localhost:8000")
	if err != nil {
		return "", err
	}

	login, password := oidc.GetOIDCClient()

	return oidc.GetOIDCAuthToken(hClient, requestToken, u, issuerURLPrefix, login, password)
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

		if apiProject.Status == "Active" {
			break
		}
		time.Sleep(time.Second)
	}

	if apiProject.Status != "Active" {
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

		if apiServiceAccount.Status == "Active" {
			break
		}
		time.Sleep(time.Second)
	}
	if apiServiceAccount.Status != "Active" {
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

// CreateAWSCluster creates cluster for Vsphere provider
func (r *APIRunner) CreateAWSCluster(projectID, dc, name, secretAccessKey, accessKeyID, version, location string, replicas int32) (*apiv1.Cluster, error) {

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
						InstanceType: &instanceType,
						VolumeSize:   &volumeSize,
						VolumeType:   &volumeType,
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
	apiClusterHealth.Apiserver = response.Payload.Apiserver
	apiClusterHealth.Controller = response.Payload.Controller
	apiClusterHealth.Etcd = response.Payload.Etcd
	apiClusterHealth.MachineController = response.Payload.MachineController
	apiClusterHealth.Scheduler = response.Payload.Scheduler
	apiClusterHealth.UserClusterControllerManager = response.Payload.UserClusterControllerManager

	return apiClusterHealth, nil
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

// GetErrorResponse converts the client error response to string
func GetErrorResponse(err error) string {
	rawData, newErr := json.Marshal(err)
	if newErr != nil {
		return err.Error()
	}
	return string(rawData)
}

func (r *APIRunner) DeleteUserFromProject(projectID, userID string) error {
	params := &users.DeleteUserFromProjectParams{ProjectID: projectID, UserID: userID}
	params.WithTimeout(timeout)
	if _, err := r.client.Users.DeleteUserFromProject(params, r.bearerToken); err != nil {
		return err
	}
	return nil
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
