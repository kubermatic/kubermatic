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

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Masterminds/semver"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	apiv1 "github.com/kubermatic/kubermatic/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	apiclient "github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/client"
	"github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/client/admin"
	"github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/client/credentials"
	"github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/client/datacenter"
	"github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/client/gcp"
	"github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/client/project"
	"github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/client/serviceaccounts"
	"github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/client/tokens"
	"github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/client/users"
	"github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/models"
	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	maxAttempts = 8
	timeout     = time.Second * 4
)

type runner struct {
	client      *apiclient.Kubermatic
	bearerToken runtime.ClientAuthInfoWriter
	test        *testing.T
}

func getHost() string {
	host := os.Getenv("KUBERMATIC_HOST")
	if len(host) == 0 {
		fmt.Printf("No KUBERMATIC_HOST env variable set.")
		os.Exit(1)
	}
	return host
}

func getScheme() string {
	scheme := os.Getenv("KUBERMATIC_SCHEME")
	if len(scheme) == 0 {
		fmt.Printf("No KUBERMATIC_SCHEME env variable set.")
		os.Exit(1)
	}
	return scheme
}

func createRunner(token string, t *testing.T) *runner {
	client := apiclient.New(httptransport.New(getHost(), "", []string{getScheme()}), strfmt.Default)

	bearerTokenAuth := httptransport.BearerToken(token)
	return &runner{
		client:      client,
		bearerToken: bearerTokenAuth,
		test:        t,
	}
}

// CreateProject creates a new project
func (r *runner) CreateProject(name string) (*apiv1.Project, error) {
	params := &project.CreateProjectParams{Body: project.CreateProjectBody{Name: name}}
	params.WithTimeout(timeout)
	project, err := r.client.Project.CreateProject(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	var apiProject *apiv1.Project
	if err := wait.PollImmediate(time.Second, maxAttempts*time.Second, func() (bool, error) {
		apiProject, err = r.GetProject(project.Payload.ID, maxAttempts)
		if err != nil {
			return false, nil
		}
		if apiProject.Status == kubermaticv1.ProjectActive {
			return true, nil
		}
		return false, nil

	}); err != nil {
		return nil, fmt.Errorf("project is not redy after %d attempts", maxAttempts)
	}

	return apiProject, nil
}

// GetProject gets the project with the given ID
func (r *runner) GetProject(id string, attempts int) (*apiv1.Project, error) {
	params := &project.GetProjectParams{ProjectID: id}
	params.WithTimeout(timeout)

	var errGetProject error
	var project *project.GetProjectOK
	duration := time.Duration(attempts) * time.Second
	if err := wait.PollImmediate(time.Second, duration, func() (bool, error) {
		project, errGetProject = r.client.Project.GetProject(params, r.bearerToken)
		if errGetProject != nil {
			return false, nil
		}
		return true, nil
	}); err != nil {
		// first check error from GetProject
		if errGetProject != nil {
			return nil, errGetProject
		}
		return nil, err
	}

	return convertProject(project.Payload)
}

// ListProjects gets projects
func (r *runner) ListProjects(displayAll bool, attempts int) ([]*apiv1.Project, error) {
	params := &project.ListProjectsParams{
		DisplayAll: &displayAll,
	}
	params.WithTimeout(timeout)

	var errListProjects error
	var projects *project.ListProjectsOK
	duration := time.Duration(attempts) * time.Second
	if err := wait.PollImmediate(time.Second, duration, func() (bool, error) {
		projects, errListProjects = r.client.Project.ListProjects(params, r.bearerToken)
		if errListProjects != nil {
			return false, nil
		}
		return true, nil
	}); err != nil {
		// first check error from ListProjects
		if errListProjects != nil {
			return nil, errListProjects
		}
		return nil, err
	}

	projectList := make([]*apiv1.Project, 0)
	for _, project := range projects.Payload {
		apiProject, err := convertProject(project)
		if err != nil {
			return nil, err
		}
		projectList = append(projectList, apiProject)
	}

	return projectList, nil
}

// UpdateProject updates the given project
func (r *runner) UpdateProject(projectToUpdate *apiv1.Project) (*apiv1.Project, error) {
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
func (r *runner) DeleteProject(id string) error {
	params := &project.DeleteProjectParams{ProjectID: id}
	params.WithTimeout(timeout)
	if _, err := r.client.Project.DeleteProject(params, r.bearerToken); err != nil {
		return err
	}
	return nil
}

// CreateServiceAccount method creates a new service account
func (r *runner) CreateServiceAccount(name, group, projectID string) (*apiv1.ServiceAccount, error) {
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
func (r *runner) GetServiceAccount(saID, projectID string) (*apiv1.ServiceAccount, error) {
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

// DeleteServiceAccount deletes service account for given ID and project
func (r *runner) DeleteServiceAccount(saID, projectID string) error {
	params := &serviceaccounts.DeleteServiceAccountParams{
		ProjectID:        projectID,
		ServiceAccountID: saID,
	}
	params.WithTimeout(timeout)

	if _, err := r.client.Serviceaccounts.DeleteServiceAccount(params, r.bearerToken); err != nil {
		return err
	}

	return nil
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
func (r *runner) AddTokenToServiceAccount(name, saID, projectID string) (*apiv1.ServiceAccountToken, error) {
	params := &tokens.AddTokenToServiceAccountParams{ProjectID: projectID, ServiceAccountID: saID, Body: &models.ServiceAccountToken{Name: name}}
	params.WithTimeout(timeout)
	token, err := r.client.Tokens.AddTokenToServiceAccount(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return convertServiceAccountToken(token.Payload)
}

// DeleteTokenForServiceAccount deletes a token for service account
func (r *runner) DeleteTokenForServiceAccount(tokenID, saID, projectID string) error {
	params := &tokens.DeleteServiceAccountTokenParams{ProjectID: projectID, ServiceAccountID: saID, TokenID: tokenID}
	params.WithTimeout(timeout)
	if _, err := r.client.Tokens.DeleteServiceAccountToken(params, r.bearerToken); err != nil {
		return err
	}

	return nil
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
func (r *runner) ListCredentials(providerName, datacenter string) ([]string, error) {
	params := &credentials.ListCredentialsParams{ProviderName: providerName, Datacenter: &datacenter}
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
func (r *runner) CreateAWSCluster(projectID, dc, name, secretAccessKey, accessKeyID, version, location, availabilityZone string, replicas int32) (*apiv1.Cluster, error) {

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

	params := &project.CreateClusterParams{ProjectID: projectID, DC: dc, Body: clusterSpec}
	params.WithTimeout(timeout)
	clusterResponse, err := r.client.Project.CreateCluster(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return convertCluster(clusterResponse.Payload)
}

// CreateDOCluster creates cluster for DigitalOcean provider
func (r *runner) CreateDOCluster(projectID, dc, name, credential, version, location string, replicas int32) (*apiv1.Cluster, error) {

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

	params := &project.CreateClusterParams{ProjectID: projectID, DC: dc, Body: clusterSpec}
	params.WithTimeout(timeout * 2)
	clusterResponse, err := r.client.Project.CreateCluster(params, r.bearerToken)
	if err != nil {
		return nil, errors.New(fmtSwaggerError(err))
	}

	return convertCluster(clusterResponse.Payload)
}

// DeleteCluster delete cluster method
func (r *runner) DeleteCluster(projectID, dc, clusterID string) error {

	params := &project.DeleteClusterParams{ProjectID: projectID, DC: dc, ClusterID: clusterID}
	params.WithTimeout(timeout)

	if _, err := r.client.Project.DeleteCluster(params, r.bearerToken); err != nil {
		return err
	}
	return nil
}

// GetCluster cluster getter
func (r *runner) GetCluster(projectID, dc, clusterID string) (*apiv1.Cluster, error) {

	params := &project.GetClusterParams{ProjectID: projectID, DC: dc, ClusterID: clusterID}
	params.WithTimeout(timeout)

	cluster, err := r.client.Project.GetCluster(params, r.bearerToken)
	if err != nil {
		return nil, err
	}
	return convertCluster(cluster.Payload)
}

// GetClusterEvents returns the cluster events
func (r *runner) GetClusterEvents(projectID, dc, clusterID string) ([]*models.Event, error) {
	params := &project.GetClusterEventsParams{ProjectID: projectID, DC: dc, ClusterID: clusterID}
	params.WithTimeout(timeout)

	events, err := r.client.Project.GetClusterEvents(params, r.bearerToken)
	if err != nil {
		return nil, err
	}
	return events.Payload, nil
}

// PrintClusterEvents prints all cluster events using its test.Logf
func (r *runner) PrintClusterEvents(projectID, dc, clusterID string) error {
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
func (r *runner) GetClusterHealthStatus(projectID, dc, clusterID string) (*apiv1.ClusterHealth, error) {
	params := &project.GetClusterHealthParams{DC: dc, ProjectID: projectID, ClusterID: clusterID}
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
func (r *runner) GetClusterNodeDeployment(projectID, dc, clusterID string) ([]apiv1.NodeDeployment, error) {
	params := &project.ListNodeDeploymentsParams{ClusterID: clusterID, ProjectID: projectID, DC: dc}
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
	apiCluster.Labels = cluster.Labels

	creationTime, err := time.Parse(time.RFC3339, cluster.CreationTimestamp.String())
	if err != nil {
		return nil, err
	}
	apiCluster.CreationTimestamp = apiv1.NewTime(creationTime)

	return apiCluster, nil
}

// ListGCPZones returns list of GCP zones
func (r *runner) ListGCPZones(credential, dc string) ([]string, error) {
	params := &gcp.ListGCPZonesParams{Credential: &credential, DC: dc}
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
func (r *runner) ListGCPDiskTypes(credential, zone string) ([]string, error) {
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
func (r *runner) ListGCPSizes(credential, zone string) ([]apiv1.GCPMachineSize, error) {
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
			VCPUs:       machineType.VCPUs,
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
		masterToken, err := retrieveMasterToken()
		if err != nil {
			t.Fatalf("can not get master token: %v", err)
		}
		runner := createRunner(masterToken, t)

		t.Log("deleting project...")
		if err := runner.DeleteProject(id); err != nil {
			t.Fatalf("can not delete project: %v", err)
		}

		for attempt := 1; attempt <= attempts; attempt++ {
			_, err := runner.GetProject(id, 5)
			if err != nil {
				break
			}
			time.Sleep(3 * time.Second)
		}

		_, err = runner.GetProject(id, 5)
		if err == nil {
			t.Fatalf("can not delete the project")
		}

		t.Log("project deleted successfully")
	}
}

func cleanUpCluster(t *testing.T, runner *runner, projectID, dc, clusterID string) {
	if err := runner.DeleteCluster(projectID, dc, clusterID); err != nil {
		t.Fatalf("can not delete the cluster %v", GetErrorResponse(err))
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_, err := runner.GetCluster(projectID, dc, clusterID)
		if err != nil {
			t.Logf("cluster deleted %v", GetErrorResponse(err))
			break
		}
		time.Sleep(60 * time.Second)
	}
	_, err := runner.GetCluster(projectID, dc, clusterID)
	if err == nil {
		t.Fatalf("can not delete the cluster after %d attempts", maxAttempts)
	}
}

func (r *runner) DeleteUserFromProject(projectID, userID string) error {
	params := &users.DeleteUserFromProjectParams{ProjectID: projectID, UserID: userID}
	params.WithTimeout(timeout)
	if _, err := r.client.Users.DeleteUserFromProject(params, r.bearerToken); err != nil {
		return err
	}
	return nil
}

func (r *runner) GetProjectUsers(projectID string) ([]apiv1.User, error) {
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

func (r *runner) AddProjectUser(projectID, email, name, group string) (*apiv1.User, error) {
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

func (r *runner) GetGlobalSettings() (*apiv1.GlobalSettings, error) {
	params := &admin.GetKubermaticSettingsParams{}
	params.WithTimeout(timeout)
	responseSettings, err := r.client.Admin.GetKubermaticSettings(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return convertGlobalSettings(responseSettings.Payload), nil
}

func (r *runner) UpdateGlobalSettings(s string) (*apiv1.GlobalSettings, error) {
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
		ClusterTypeOptions:    kubermaticv1.ClusterType(gSettings.ClusterTypeOptions),
		DisplayDemoInfo:       gSettings.DisplayDemoInfo,
		DisplayAPIDocs:        gSettings.DisplayAPIDocs,
		DisplayTermsOfService: gSettings.DisplayTermsOfService,
		EnableOIDCKubeconfig:  gSettings.EnableOIDCKubeconfig,
		EnableDashboard:       gSettings.EnableDashboard,
	}
}

func (r *runner) SetAdmin(email string, isAdmin bool) error {
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

// GetRoles
func (r *runner) GetRoles(projectID, dc, clusterID string) ([]apiv1.RoleName, error) {
	params := &project.ListRoleNamesParams{DC: dc, ProjectID: projectID, ClusterID: clusterID}
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
func (r *runner) BindUserToRole(projectID, dc, clusterID, roleName, namespace, user string) (*apiv1.RoleBinding, error) {
	params := &project.BindUserToRoleParams{
		Body:      &models.RoleUser{UserEmail: user},
		ClusterID: clusterID,
		DC:        dc,
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

func (r *runner) GetClusterRoles(projectID, dc, clusterID string) ([]apiv1.ClusterRoleName, error) {
	params := &project.ListClusterRoleNamesParams{DC: dc, ProjectID: projectID, ClusterID: clusterID}
	params.WithTimeout(timeout)

	var err error
	var response *project.ListClusterRoleNamesOK
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		response, err = r.client.Project.ListClusterRoleNames(params, r.bearerToken)
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		return nil, err
	}

	clusterRoleNames := []apiv1.ClusterRoleName{}

	for _, roleName := range response.Payload {
		clusterRoleNames = append(clusterRoleNames, apiv1.ClusterRoleName{
			Name: roleName.Name,
		})
	}

	return clusterRoleNames, nil
}

// BindUserToClusterRole
func (r *runner) BindUserToClusterRole(projectID, dc, clusterID, roleName, user string) (*apiv1.ClusterRoleBinding, error) {
	params := &project.BindUserToClusterRoleParams{
		Body:      &models.ClusterRoleUser{UserEmail: user},
		ClusterID: clusterID,
		DC:        dc,
		ProjectID: projectID,
		RoleID:    roleName,
	}
	params.WithTimeout(timeout)

	var err error
	var response *project.BindUserToClusterRoleOK
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		response, err = r.client.Project.BindUserToClusterRole(params, r.bearerToken)
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		return nil, err
	}

	return &apiv1.ClusterRoleBinding{
		RoleRefName: response.Payload.RoleRefName,
	}, nil
}

func (r *runner) GetClusterBindings(projectID, dc, clusterID string) ([]apiv1.ClusterRoleBinding, error) {
	params := &project.ListClusterRoleBindingParams{DC: dc, ProjectID: projectID, ClusterID: clusterID}
	params.WithTimeout(timeout)

	var err error
	var response *project.ListClusterRoleBindingOK
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		response, err = r.client.Project.ListClusterRoleBinding(params, r.bearerToken)
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		return nil, err
	}

	var clusterRoleBindings []apiv1.ClusterRoleBinding

	for _, roleBinding := range response.Payload {
		newBinding := apiv1.ClusterRoleBinding{
			RoleRefName: roleBinding.RoleRefName,
		}
		var subjects []rbacv1.Subject
		for _, subject := range roleBinding.Subjects {
			subjects = append(subjects, rbacv1.Subject{
				Kind:     subject.Kind,
				APIGroup: subject.APIGroup,
				Name:     subject.Name,
			})
		}
		newBinding.Subjects = subjects
		clusterRoleBindings = append(clusterRoleBindings, newBinding)
	}

	return clusterRoleBindings, nil
}

// fmtSwaggerError works around the Error() implementration generated by swagger
// which prints only a pointer to the body but we want to see the actual content of the body.
// to fix this we can either type assert for each request type or naively use json
func fmtSwaggerError(err error) string {
	rawData, newErr := json.Marshal(err)
	if newErr != nil {
		return fmt.Sprintf("failed to marshal response(%v): %v", err, newErr)
	}
	return string(rawData)
}

// UpdateCluster updates cluster
func (r *runner) UpdateCluster(projectID, dc, clusterID string, patch PatchCluster) (*apiv1.Cluster, error) {

	params := &project.PatchClusterParams{ProjectID: projectID, DC: dc, ClusterID: clusterID, Patch: patch}
	params.WithTimeout(timeout)

	cluster, err := r.client.Project.PatchCluster(params, r.bearerToken)
	if err != nil {
		return nil, err
	}
	return convertCluster(cluster.Payload)
}

type PatchCluster struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`
}

// CreateUserSSHKey creates a new user SSH key
func (r *runner) CreateUserSSHKey(projectID, keyName, publicKey string) (*apiv1.SSHKey, error) {
	params := &project.CreateSSHKeyParams{
		Key: &models.SSHKey{
			Name: keyName,
			Spec: &models.SSHKeySpec{
				PublicKey: publicKey,
			},
		},
		ProjectID: projectID,
	}
	params.WithTimeout(timeout)
	key, err := r.client.Project.CreateSSHKey(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return convertSSHKey(key.Payload), nil
}

// ListUserSSHKey list user SSH keys
func (r *runner) ListUserSSHKey(projectID string) ([]*apiv1.SSHKey, error) {
	params := &project.ListSSHKeysParams{
		ProjectID: projectID,
	}
	params.WithTimeout(timeout)
	keyList, err := r.client.Project.ListSSHKeys(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	resultList := make([]*apiv1.SSHKey, 0)
	for _, key := range keyList.Payload {
		resultList = append(resultList, convertSSHKey(key))
	}

	return resultList, nil
}

// DeleteUserSSHKey deletes user SSH keys
func (r *runner) DeleteUserSSHKey(projectID, keyID string) error {
	params := &project.DeleteSSHKeyParams{
		ProjectID: projectID,
		SSHKeyID:  keyID,
	}
	params.WithTimeout(timeout)

	var deleteError error
	if err := wait.PollImmediate(time.Second, maxAttempts*time.Second, func() (bool, error) {
		_, deleteError := r.client.Project.DeleteSSHKey(params, r.bearerToken)
		if deleteError != nil {
			return false, nil
		}
		return true, nil

	}); err != nil {
		return fmt.Errorf("the user SSH key can not be deleted after %d attempts %v", maxAttempts, deleteError)
	}

	return nil
}

// AssignSSHKeyToCluster adds user SSH key to the cluster
func (r *runner) AssignSSHKeyToCluster(projectID, clusterID, dc, keyID string) error {
	params := &project.AssignSSHKeyToClusterParams{
		ClusterID: clusterID,
		DC:        dc,
		KeyID:     keyID,
		ProjectID: projectID,
	}
	params.WithTimeout(timeout)
	if _, err := r.client.Project.AssignSSHKeyToCluster(params, r.bearerToken); err != nil {
		return err
	}
	return nil
}

// DetachSSHKeyFromClusterParams detaches user SSH key from the cluster
func (r *runner) DetachSSHKeyFromClusterParams(projectID, clusterID, dc, keyID string) error {
	params := &project.DetachSSHKeyFromClusterParams{
		ClusterID: clusterID,
		DC:        dc,
		KeyID:     keyID,
		ProjectID: projectID,
	}
	params.WithTimeout(timeout)
	if _, err := r.client.Project.DetachSSHKeyFromCluster(params, r.bearerToken); err != nil {
		return err
	}
	return nil
}

func convertSSHKey(key *models.SSHKey) *apiv1.SSHKey {
	return &apiv1.SSHKey{
		ObjectMeta: apiv1.ObjectMeta{
			ID:   key.ID,
			Name: key.Name,
		},
		Spec: apiv1.SSHKeySpec{
			Fingerprint: key.Spec.Fingerprint,
			PublicKey:   key.Spec.PublicKey,
		},
	}
}

// DC

func (r *runner) ListDCForProvider(provider string) ([]*models.Datacenter, error) {
	params := &datacenter.ListDCForProviderParams{
		Provider: provider,
	}
	params.WithTimeout(timeout)

	list, err := r.client.Datacenter.ListDCForProvider(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return list.GetPayload(), nil
}

func (r *runner) GetDCForProvider(provider, dc string) (*models.Datacenter, error) {
	params := &datacenter.GetDCForProviderParams{
		Provider:   provider,
		Datacenter: dc,
	}
	params.WithTimeout(timeout)

	receivedDC, err := r.client.Datacenter.GetDCForProvider(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return receivedDC.GetPayload(), nil
}

func (r *runner) CreateDC(seed string, dc *models.Datacenter) (*models.Datacenter, error) {
	params := &datacenter.CreateDCParams{
		Body: datacenter.CreateDCBody{
			Name: dc.Metadata.Name,
			Spec: dc.Spec,
		},
		Seed: seed,
	}
	params.WithTimeout(timeout)

	createdDC, err := r.client.Datacenter.CreateDC(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return createdDC.GetPayload(), nil
}

func (r *runner) DeleteDC(seed, dc string) error {
	params := &datacenter.DeleteDCParams{
		Seed: seed,
		DC:   dc,
	}
	params.WithTimeout(timeout)

	_, err := r.client.Datacenter.DeleteDC(params, r.bearerToken)
	return err
}

func (r *runner) UpdateDC(seed, dcToUpdate string, dc *models.Datacenter) (*models.Datacenter, error) {
	params := &datacenter.UpdateDCParams{
		Body: datacenter.UpdateDCBody{
			Name: dc.Metadata.Name,
			Spec: dc.Spec,
		},
		DCToUpdate: dcToUpdate,
		Seed:       seed,
	}
	params.WithTimeout(timeout)

	updatedDC, err := r.client.Datacenter.UpdateDC(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return updatedDC.GetPayload(), nil
}

func (r *runner) PatchDC(seed, dcToPatch, patch string) (*models.Datacenter, error) {
	params := &datacenter.PatchDCParams{
		Patch:     strings.NewReader(patch),
		DCToPatch: dcToPatch,
		Seed:      seed,
	}
	params.WithTimeout(timeout)

	updatedDC, err := r.client.Datacenter.PatchDC(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return updatedDC.GetPayload(), nil
}

func (r *runner) GetDCForSeed(seed, dc string) (*models.Datacenter, error) {
	params := &datacenter.GetDCForSeedParams{
		Seed: seed,
		DC:   dc,
	}
	params.WithTimeout(timeout)

	receivedDC, err := r.client.Datacenter.GetDCForSeed(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return receivedDC.GetPayload(), nil
}

func (r *runner) ListDCForSeed(seed string) ([]*models.Datacenter, error) {
	params := &datacenter.ListDCForSeedParams{
		Seed: seed,
	}
	params.WithTimeout(timeout)

	list, err := r.client.Datacenter.ListDCForSeed(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return list.GetPayload(), nil
}

func (r *runner) GetDC(dc string) (*models.Datacenter, error) {
	params := &datacenter.GetDatacenterParams{
		DC: dc,
	}
	params.WithTimeout(timeout)

	receivedDC, err := r.client.Datacenter.GetDatacenter(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return receivedDC.GetPayload(), nil
}

func (r *runner) ListDC() ([]*models.Datacenter, error) {
	params := &datacenter.ListDatacentersParams{}
	params.WithTimeout(timeout)

	list, err := r.client.Datacenter.ListDatacenters(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return list.GetPayload(), nil
}

func (r *runner) Logout() error {
	params := &users.LogoutCurrentUserParams{}
	params.WithTimeout(timeout)

	_, err := r.client.Users.LogoutCurrentUser(params, r.bearerToken)
	if err != nil {
		return err
	}
	return nil
}

func cleanUpDC(seed, dc string) func(t *testing.T) {
	return func(t *testing.T) {
		adminMasterToken, err := retrieveAdminMasterToken()
		if err != nil {
			t.Fatalf("can not get admin master token: %v", err)
		}
		runner := createRunner(adminMasterToken, t)

		t.Logf("deleting dc %s...", dc)
		_, err = runner.GetDC(dc)
		if err != nil {
			t.Logf("dc %s already deleted, skipping cleanup", dc)
			return
		}

		if err := runner.DeleteDC(seed, dc); err != nil {
			t.Fatalf("can not delete dc %s : %v", dc, err)
		}

		t.Logf("dc %s deleted successfully", dc)
	}
}
