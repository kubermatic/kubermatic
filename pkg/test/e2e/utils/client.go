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

package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	"k8s.io/client-go/kubernetes/scheme"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	apiclient "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/admin"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/credentials"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/datacenter"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/gcp"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/project"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/serviceaccounts"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/tokens"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/users"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// TestClient wraps the Swagger-generated API client with some more
// convenient functions, tailor-made for writing tests.
type TestClient struct {
	client      *apiclient.KubermaticAPI
	bearerToken runtime.ClientAuthInfoWriter
	test        *testing.T
}

func NewTestClient(token string, t *testing.T) *TestClient {
	endpoint, err := APIEndpoint()
	if err != nil {
		t.Fatalf("Failed to get API endpoint: %v", err)
	}

	client, err := NewKubermaticClient(endpoint)
	if err != nil {
		t.Fatalf("Failed to create API client: %v", err)
	}

	bearerTokenAuth := httptransport.BearerToken(token)
	return &TestClient{
		client:      client,
		bearerToken: bearerTokenAuth,
		test:        t,
	}
}

// CreateProject creates a new project and waits for it to become active (ready).
func (r *TestClient) CreateProject(name string) (*apiv1.Project, error) {
	before := time.Now()
	timeout := 30 * time.Second

	params := &project.CreateProjectParams{Body: project.CreateProjectBody{Name: name}}
	SetupParams(r.test, params, 1*time.Second, timeout)

	r.test.Logf("Creating project %s...", name)

	response, err := r.client.Project.CreateProject(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	var apiProject *apiv1.Project
	if !WaitFor(1*time.Second, timeout, func() bool {
		apiProject, _ = r.GetProject(response.Payload.ID)
		return apiProject != nil && apiProject.Status == kubermaticv1.ProjectActive
	}) {
		// best effort cleanup of a failed cluster
		_ = r.DeleteProject(name)

		return nil, fmt.Errorf("project is not ready after %s", timeout)
	}

	r.test.Logf("Created project and it became ready after %v", time.Since(before))

	return apiProject, nil
}

// GetProject gets the project with the given ID; it does not perform any
// retries if the API returns errors.
func (r *TestClient) GetProject(id string) (*apiv1.Project, error) {
	params := &project.GetProjectParams{ProjectID: id}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute, http.StatusUnauthorized)

	project, err := r.client.Project.GetProject(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return convertProject(project.Payload)
}

// ListProjects gets projects
func (r *TestClient) ListProjects(displayAll bool) ([]*apiv1.Project, error) {
	params := &project.ListProjectsParams{DisplayAll: &displayAll}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	projects, err := r.client.Project.ListProjects(params, r.bearerToken)
	if err != nil {
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
func (r *TestClient) UpdateProject(projectToUpdate *apiv1.Project) (*apiv1.Project, error) {
	params := &project.UpdateProjectParams{ProjectID: projectToUpdate.ID, Body: &models.Project{Name: projectToUpdate.Name}}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	r.test.Log("Updating project...")

	response, err := r.client.Project.UpdateProject(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	r.test.Log("Project updated successfully")

	return convertProject(response.Payload)
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
func (r *TestClient) DeleteProject(id string) error {
	r.test.Log("Deleting project...")

	params := &project.DeleteProjectParams{ProjectID: id}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute, http.StatusConflict)

	_, err := r.client.Project.DeleteProject(params, r.bearerToken)
	if err != nil {
		return err
	}

	r.test.Log("Project deleted successfully")
	return nil
}

func (r *TestClient) CleanupProject(t *testing.T, id string) {
	before := time.Now()

	t.Logf("Deleting project %s...", id)
	if err := r.DeleteProject(id); err != nil {
		t.Fatalf("Failed to delete project: %v", err)
	}

	timeout := 3 * time.Minute
	t.Logf("Waiting %v for project to be gone...", timeout)

	err := wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		_, err := r.GetProject(id)
		return err != nil, nil // return true if there *was* an error, i.e. project is gone
	})
	if err != nil {
		t.Fatalf("Failed to wait for project to be gone: %v", err)
	}

	t.Logf("Project deleted successfully after %v", time.Since(before))
}

func (r *TestClient) CleanupCluster(t *testing.T, projectID, dc, clusterID string) {
	before := time.Now()

	t.Logf("Deleting cluster %s...", clusterID)
	if err := r.DeleteCluster(projectID, dc, clusterID); err != nil {
		t.Fatalf("Failed to delete cluster: %v", err)
	}

	timeout := 3 * time.Minute
	t.Logf("Waiting %v for cluster to be gone...", timeout)

	err := wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		_, err := r.GetCluster(projectID, dc, clusterID)
		return err != nil, nil // return true if there *was* an error, i.e. project is gone
	})
	if err != nil {
		t.Fatalf("Failed to wait for cluster to be gone: %v", err)
	}

	t.Logf("Cluster deleted successfully after %v", time.Since(before))
}

// CreateServiceAccount method creates a new service account and waits a certain
// amount of time for it to become active.
func (r *TestClient) CreateServiceAccount(name, group, projectID string) (*apiv1.ServiceAccount, error) {
	params := &serviceaccounts.AddServiceAccountToProjectParams{ProjectID: projectID, Body: &models.ServiceAccount{Name: name, Group: group}}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	r.test.Logf("Creating ServiceAccount %q in group %q...", name, group)

	sa, err := r.client.Serviceaccounts.AddServiceAccountToProject(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	before := time.Now()

	var apiServiceAccount *apiv1.ServiceAccount
	if !WaitFor(1*time.Second, 60*time.Second, func() bool {
		apiServiceAccount, _ = r.GetServiceAccount(sa.Payload.ID, projectID)
		return apiServiceAccount != nil && apiServiceAccount.Status == apiv1.ServiceAccountActive
	}) {
		return nil, err
	}

	r.test.Logf("Created ServiceAccount and it became active after %v", time.Since(before))

	return apiServiceAccount, nil
}

// GetServiceAccount returns service account for given ID and project
func (r *TestClient) GetServiceAccount(saID, projectID string) (*apiv1.ServiceAccount, error) {
	params := &serviceaccounts.ListServiceAccountsParams{ProjectID: projectID}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	saList, err := r.client.Serviceaccounts.ListServiceAccounts(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	for _, sa := range saList.Payload {
		if sa.ID == saID {
			return convertServiceAccount(sa)
		}
	}

	return nil, fmt.Errorf("ServiceAccount %s not found", saID)
}

// DeleteServiceAccount deletes service account for given ID and project
func (r *TestClient) DeleteServiceAccount(saID, projectID string) error {
	r.test.Logf("Deleting ServiceAccount %s...", saID)

	params := &serviceaccounts.DeleteServiceAccountParams{
		ProjectID:        projectID,
		ServiceAccountID: saID,
	}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	_, err := r.client.Serviceaccounts.DeleteServiceAccount(params, r.bearerToken)
	if err != nil {
		return err
	}

	r.test.Log("ServiceAccount deleted successfully")
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
func (r *TestClient) AddTokenToServiceAccount(name, saID, projectID string) (*apiv1.ServiceAccountToken, error) {
	r.test.Logf("Adding token %s to ServiceAccount %s...", name, saID)

	params := &tokens.AddTokenToServiceAccountParams{ProjectID: projectID, ServiceAccountID: saID, Body: &models.ServiceAccountToken{Name: name}}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute, http.StatusForbidden)

	token, err := r.client.Tokens.AddTokenToServiceAccount(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	r.test.Log("ServiceAccount token added successfully")

	return convertServiceAccountToken(token.Payload)
}

// DeleteTokenForServiceAccount deletes a token for service account
func (r *TestClient) DeleteTokenForServiceAccount(tokenID, saID, projectID string) error {
	r.test.Logf("Deleting token %s from ServiceAccount %s...", tokenID, saID)

	params := &tokens.DeleteServiceAccountTokenParams{ProjectID: projectID, ServiceAccountID: saID, TokenID: tokenID}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	_, err := r.client.Tokens.DeleteServiceAccountToken(params, r.bearerToken)
	if err != nil {
		return err
	}

	r.test.Log("ServiceAccount token deleted successfully")
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
func (r *TestClient) ListCredentials(providerName, datacenter string) ([]string, error) {
	params := &credentials.ListCredentialsParams{ProviderName: providerName, Datacenter: &datacenter}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	credentialsResponse, err := r.client.Credentials.ListCredentials(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0)
	names = append(names, credentialsResponse.Payload.Names...)

	return names, nil
}

// CreateAWSCluster creates cluster for AWS provider
func (r *TestClient) CreateAWSCluster(projectID, dc, name, secretAccessKey, accessKeyID, version, location, availabilityZone string, replicas int32) (*apiv1.Cluster, error) {
	vr, err := semver.NewVersion(version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version %s: %v", version, err)
	}

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

	if replicas > 0 {
		instanceType := "t3.small"
		volumeSize := int64(25)
		volumeType := "standard"

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
	}

	r.test.Logf("Creating AWS cluster %q (%s, %d nodes)...", name, version, replicas)

	params := &project.CreateClusterParams{ProjectID: projectID, DC: dc, Body: clusterSpec}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	clusterResponse, err := r.client.Project.CreateCluster(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	r.test.Log("Cluster created successfully.")

	return convertCluster(clusterResponse.Payload)
}

// CreateDOCluster creates cluster for DigitalOcean provider
func (r *TestClient) CreateDOCluster(projectID, dc, name, credential, version, location string, replicas int32) (*apiv1.Cluster, error) {
	vr, err := semver.NewVersion(version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version %s: %v", version, err)
	}

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

	if replicas > 0 {
		instanceSize := "s-2vcpu-2gb"

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
	}

	params := &project.CreateClusterParams{ProjectID: projectID, DC: dc, Body: clusterSpec}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	r.test.Logf("Creating DigitalOcean cluster %q (%s, %d nodes)...", name, version, replicas)

	clusterResponse, err := r.client.Project.CreateCluster(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	r.test.Log("Cluster created successfully.")

	return convertCluster(clusterResponse.Payload)
}

// DeleteCluster delete cluster method
func (r *TestClient) DeleteCluster(projectID, dc, clusterID string) error {
	r.test.Logf("Deleting cluster %s...", clusterID)

	params := &project.DeleteClusterParams{ProjectID: projectID, DC: dc, ClusterID: clusterID}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute, http.StatusConflict)

	_, err := r.client.Project.DeleteCluster(params, r.bearerToken)
	if err != nil {
		return err
	}

	r.test.Log("Cluster deleted successfully")
	return nil
}

// GetCluster cluster getter
func (r *TestClient) GetCluster(projectID, dc, clusterID string) (*apiv1.Cluster, error) {
	params := &project.GetClusterParams{ProjectID: projectID, DC: dc, ClusterID: clusterID}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	cluster, err := r.client.Project.GetCluster(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return convertCluster(cluster.Payload)
}

// GetClusterEvents returns the cluster events
func (r *TestClient) GetClusterEvents(projectID, dc, clusterID string) ([]*models.Event, error) {
	params := &project.GetClusterEventsParams{ProjectID: projectID, DC: dc, ClusterID: clusterID}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	events, err := r.client.Project.GetClusterEvents(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return events.Payload, nil
}

// PrintClusterEvents prints all cluster events using its test.Logf
func (r *TestClient) PrintClusterEvents(projectID, dc, clusterID string) error {
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
func (r *TestClient) GetClusterHealthStatus(projectID, dc, clusterID string) (*apiv1.ClusterHealth, error) {
	params := &project.GetClusterHealthParams{DC: dc, ProjectID: projectID, ClusterID: clusterID}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	response, err := r.client.Project.GetClusterHealth(params, r.bearerToken)
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
	apiClusterHealth.GatekeeperController = convertHealthStatus(response.Payload.GatekeeperController)
	apiClusterHealth.GatekeeperAudit = convertHealthStatus(response.Payload.GatekeeperAudit)

	return apiClusterHealth, nil
}

func (r *TestClient) WaitForClusterHealthy(projectID, dc, clusterID string) error {
	timeout := 5 * time.Minute
	before := time.Now()

	r.test.Logf("Waiting %v for cluster %s to become healthy...", timeout, clusterID)

	if !WaitFor(5*time.Second, timeout, func() bool {
		healthStatus, _ := r.GetClusterHealthStatus(projectID, dc, clusterID)
		return IsHealthyCluster(healthStatus)
	}) {
		return errors.New("cluster did not become healthy")
	}

	r.test.Logf("Cluster became healthy after %v", time.Since(before))
	return nil
}

func (r *TestClient) WaitForOPAEnabledClusterHealthy(projectID, dc, clusterID string) error {
	timeout := 5 * time.Minute
	before := time.Now()

	r.test.Logf("Waiting %v for OPA enabled cluster %s to become healthy...", timeout, clusterID)

	if !WaitFor(5*time.Second, timeout, func() bool {
		healthStatus, _ := r.GetClusterHealthStatus(projectID, dc, clusterID)
		return IsHealthyCluster(healthStatus) &&
			healthStatus.GatekeeperController == kubermaticv1.HealthStatusUp &&
			healthStatus.GatekeeperAudit == kubermaticv1.HealthStatusUp

	}) {
		return errors.New("OPA enabled cluster did not become healthy")
	}

	r.test.Logf("OPA enabled cluster became healthy after %v", time.Since(before))
	return nil
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

// GetClusterNodeDeployments returns the cluster node deployments
func (r *TestClient) GetClusterNodeDeployments(projectID, dc, clusterID string) ([]apiv1.NodeDeployment, error) {
	params := &project.ListNodeDeploymentsParams{ClusterID: clusterID, ProjectID: projectID, DC: dc}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	response, err := r.client.Project.ListNodeDeployments(params, r.bearerToken)
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

func (r *TestClient) WaitForClusterNodeDeploymentsToExist(projectID, dc, clusterID string) error {
	timeout := 30 * time.Second
	before := time.Now()

	r.test.Logf("Waiting %v for NodeDeployment in cluster %s to exist...", timeout, clusterID)

	if !WaitFor(1*time.Second, timeout, func() bool {
		deployments, _ := r.GetClusterNodeDeployments(projectID, dc, clusterID)
		return len(deployments) > 0
	}) {
		return errors.New("NodeDeployment did not appear")
	}

	r.test.Logf("NodeDeployment appeared after %v", time.Since(before))
	return nil
}

func (r *TestClient) WaitForClusterNodeDeploymentsToByReady(projectID, dc, clusterID string, replicas int32) error {
	timeout := 10 * time.Minute
	before := time.Now()

	r.test.Logf("Waiting %v for NodeDeployment in cluster %s to become ready...", timeout, clusterID)

	if !WaitFor(5*time.Second, timeout, func() bool {
		deployments, _ := r.GetClusterNodeDeployments(projectID, dc, clusterID)
		return len(deployments) > 0 && deployments[0].Status.AvailableReplicas == replicas
	}) {
		return fmt.Errorf("NodeDeployment has not reached %d ready replicas", replicas)
	}

	r.test.Logf("All nodes became ready after %v.", time.Since(before))

	return nil
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
func (r *TestClient) ListGCPZones(credential, dc string) ([]string, error) {
	params := &gcp.ListGCPZonesParams{Credential: &credential, DC: dc}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

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
func (r *TestClient) ListGCPDiskTypes(credential, zone string) ([]string, error) {
	params := &gcp.ListGCPDiskTypesParams{Credential: &credential, Zone: &zone}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

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
func (r *TestClient) ListGCPSizes(credential, zone string) ([]apiv1.GCPMachineSize, error) {
	params := &gcp.ListGCPSizesParams{Credential: &credential, Zone: &zone}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

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

// IsHealthyCluster check if all cluster components are up
func IsHealthyCluster(healthStatus *apiv1.ClusterHealth) bool {
	return healthStatus != nil &&
		kubermaticv1.HealthStatusUp == healthStatus.UserClusterControllerManager &&
		kubermaticv1.HealthStatusUp == healthStatus.Scheduler &&
		kubermaticv1.HealthStatusUp == healthStatus.MachineController &&
		kubermaticv1.HealthStatusUp == healthStatus.Etcd &&
		kubermaticv1.HealthStatusUp == healthStatus.Controller &&
		kubermaticv1.HealthStatusUp == healthStatus.Apiserver
}

func (r *TestClient) DeleteUserFromProject(projectID, userID string) error {
	params := &users.DeleteUserFromProjectParams{ProjectID: projectID, UserID: userID}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	_, err := r.client.Users.DeleteUserFromProject(params, r.bearerToken)
	return err
}

func (r *TestClient) GetProjectUsers(projectID string) ([]apiv1.User, error) {
	params := &users.GetUsersForProjectParams{ProjectID: projectID}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute, http.StatusForbidden)

	response, err := r.client.Users.GetUsersForProject(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	users := make([]apiv1.User, 0)
	for _, user := range response.Payload {
		users = append(users, apiv1.User{
			Email: user.Email,
			ObjectMeta: apiv1.ObjectMeta{
				ID:   user.ID,
				Name: user.Name,
			},
		})
	}

	return users, nil
}

func (r *TestClient) AddProjectUser(projectID, email, name, group string) (*apiv1.User, error) {
	params := &users.AddUserToProjectParams{ProjectID: projectID, Body: &models.User{
		Email: email,
		Name:  name,
		Projects: []*models.ProjectGroup{
			{ID: projectID,
				GroupPrefix: group,
			},
		},
	}}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute, http.StatusForbidden)

	responseUser, err := r.client.Users.AddUserToProject(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	user := &apiv1.User{
		Email: responseUser.Payload.Email,
		ObjectMeta: apiv1.ObjectMeta{
			ID:   responseUser.Payload.ID,
			Name: responseUser.Payload.Name,
		},
	}

	return user, nil
}

func (r *TestClient) GetGlobalSettings() (*apiv1.GlobalSettings, error) {
	params := &admin.GetKubermaticSettingsParams{}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	responseSettings, err := r.client.Admin.GetKubermaticSettings(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return convertGlobalSettings(responseSettings.Payload), nil
}

func (r *TestClient) UpdateGlobalSettings(patch json.RawMessage) (*apiv1.GlobalSettings, error) {
	params := &admin.PatchKubermaticSettingsParams{
		Patch: &patch,
	}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

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

func (r *TestClient) SetAdmin(email string, isAdmin bool) error {
	params := &admin.SetAdminParams{
		Body: &models.Admin{
			Email:   email,
			IsAdmin: isAdmin,
		},
	}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	_, err := r.client.Admin.SetAdmin(params, r.bearerToken)
	return err
}

func (r *TestClient) GetRoles(projectID, dc, clusterID string) ([]apiv1.RoleName, error) {
	params := &project.ListRoleNamesParams{DC: dc, ProjectID: projectID, ClusterID: clusterID}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	response, err := r.client.Project.ListRoleNames(params, r.bearerToken)
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

func (r *TestClient) BindUserToRole(projectID, dc, clusterID, roleName, namespace, user string) (*apiv1.RoleBinding, error) {
	params := &project.BindUserToRoleParams{
		Body:      &models.RoleUser{UserEmail: user},
		ClusterID: clusterID,
		DC:        dc,
		Namespace: namespace,
		ProjectID: projectID,
		RoleID:    roleName,
	}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	response, err := r.client.Project.BindUserToRole(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return &apiv1.RoleBinding{
		Namespace:   response.Payload.Namespace,
		RoleRefName: response.Payload.RoleRefName,
	}, nil
}

func (r *TestClient) GetClusterRoles(projectID, dc, clusterID string) ([]apiv1.ClusterRoleName, error) {
	params := &project.ListClusterRoleNamesParams{DC: dc, ProjectID: projectID, ClusterID: clusterID}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	response, err := r.client.Project.ListClusterRoleNames(params, r.bearerToken)
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
func (r *TestClient) BindUserToClusterRole(projectID, dc, clusterID, roleName, user string) (*apiv1.ClusterRoleBinding, error) {
	params := &project.BindUserToClusterRoleParams{
		Body:      &models.ClusterRoleUser{UserEmail: user},
		ClusterID: clusterID,
		DC:        dc,
		ProjectID: projectID,
		RoleID:    roleName,
	}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	response, err := r.client.Project.BindUserToClusterRole(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return &apiv1.ClusterRoleBinding{
		RoleRefName: response.Payload.RoleRefName,
	}, nil
}

func (r *TestClient) GetClusterBindings(projectID, dc, clusterID string) ([]apiv1.ClusterRoleBinding, error) {
	params := &project.ListClusterRoleBindingParams{DC: dc, ProjectID: projectID, ClusterID: clusterID}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	response, err := r.client.Project.ListClusterRoleBinding(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	clusterRoleBindings := []apiv1.ClusterRoleBinding{}
	for _, roleBinding := range response.Payload {
		subjects := []rbacv1.Subject{}
		for _, subject := range roleBinding.Subjects {
			subjects = append(subjects, rbacv1.Subject{
				Kind:     subject.Kind,
				APIGroup: subject.APIGroup,
				Name:     subject.Name,
			})
		}

		clusterRoleBindings = append(clusterRoleBindings, apiv1.ClusterRoleBinding{
			RoleRefName: roleBinding.RoleRefName,
			Subjects:    subjects,
		})
	}

	return clusterRoleBindings, nil
}

func (r *TestClient) UpdateCluster(projectID, dc, clusterID string, patch PatchCluster) (*apiv1.Cluster, error) {
	params := &project.PatchClusterParams{ProjectID: projectID, DC: dc, ClusterID: clusterID, Patch: patch}
	SetupParams(r.test, params, 1*time.Second, 5*time.Minute, http.StatusConflict)

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
func (r *TestClient) CreateUserSSHKey(projectID, keyName, publicKey string) (*apiv1.SSHKey, error) {
	params := &project.CreateSSHKeyParams{
		Key: &models.SSHKey{
			Name: keyName,
			Spec: &models.SSHKeySpec{
				PublicKey: publicKey,
			},
		},
		ProjectID: projectID,
	}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	key, err := r.client.Project.CreateSSHKey(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return convertSSHKey(key.Payload), nil
}

// ListUserSSHKey list user SSH keys
func (r *TestClient) ListUserSSHKey(projectID string) ([]*apiv1.SSHKey, error) {
	params := &project.ListSSHKeysParams{
		ProjectID: projectID,
	}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	keyList, err := r.client.Project.ListSSHKeys(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	resultList := []*apiv1.SSHKey{}
	for _, key := range keyList.Payload {
		resultList = append(resultList, convertSSHKey(key))
	}

	return resultList, nil
}

// DeleteUserSSHKey deletes user SSH keys
func (r *TestClient) DeleteUserSSHKey(projectID, keyID string) error {
	params := &project.DeleteSSHKeyParams{
		ProjectID: projectID,
		SSHKeyID:  keyID,
	}
	// consider HTTP 403 (Forbidden) to be transient, as it can take a few
	// moments for the UserProjectBindings to be properly reconciled
	SetupParams(r.test, params, 1*time.Second, 1*time.Minute, http.StatusForbidden)

	_, err := r.client.Project.DeleteSSHKey(params, r.bearerToken)
	return err
}

// AssignSSHKeyToCluster adds user SSH key to the cluster
func (r *TestClient) AssignSSHKeyToCluster(projectID, clusterID, dc, keyID string) error {
	params := &project.AssignSSHKeyToClusterParams{
		ClusterID: clusterID,
		DC:        dc,
		KeyID:     keyID,
		ProjectID: projectID,
	}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	_, err := r.client.Project.AssignSSHKeyToCluster(params, r.bearerToken)
	return err
}

// DetachSSHKeyFromClusterParams detaches user SSH key from the cluster
func (r *TestClient) DetachSSHKeyFromClusterParams(projectID, clusterID, dc, keyID string) error {
	params := &project.DetachSSHKeyFromClusterParams{
		ClusterID: clusterID,
		DC:        dc,
		KeyID:     keyID,
		ProjectID: projectID,
	}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	_, err := r.client.Project.DetachSSHKeyFromCluster(params, r.bearerToken)
	return err
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

func (r *TestClient) ListDCForProvider(provider string) ([]*models.Datacenter, error) {
	params := &datacenter.ListDCForProviderParams{
		Provider: provider,
	}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	list, err := r.client.Datacenter.ListDCForProvider(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return list.GetPayload(), nil
}

func (r *TestClient) GetDCForProvider(provider, dc string) (*models.Datacenter, error) {
	params := &datacenter.GetDCForProviderParams{
		Provider:   provider,
		Datacenter: dc,
	}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	receivedDC, err := r.client.Datacenter.GetDCForProvider(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return receivedDC.GetPayload(), nil
}

func (r *TestClient) CreateDC(seed string, dc *models.Datacenter) (*models.Datacenter, error) {
	params := &datacenter.CreateDCParams{
		Body: datacenter.CreateDCBody{
			Name: dc.Metadata.Name,
			Spec: dc.Spec,
		},
		Seed: seed,
	}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	createdDC, err := r.client.Datacenter.CreateDC(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return createdDC.GetPayload(), nil
}

func (r *TestClient) DeleteDC(seed, dc string) error {
	params := &datacenter.DeleteDCParams{
		Seed: seed,
		DC:   dc,
	}
	// HTTP400 is returned when the DC is not yet available in the Seed
	SetupParams(r.test, params, 1*time.Second, 5*time.Minute, http.StatusBadRequest)

	_, err := r.client.Datacenter.DeleteDC(params, r.bearerToken)
	return err
}

func (r *TestClient) UpdateDC(seed, dcToUpdate string, dc *models.Datacenter) (*models.Datacenter, error) {
	params := &datacenter.UpdateDCParams{
		Body: datacenter.UpdateDCBody{
			Name: dc.Metadata.Name,
			Spec: dc.Spec,
		},
		DCToUpdate: dcToUpdate,
		Seed:       seed,
	}
	SetupRetryParams(r.test, params, Backoff{
		Duration: 1 * time.Second,
		Steps:    4,
		Factor:   1.5,
	}, http.StatusBadRequest)

	updatedDC, err := r.client.Datacenter.UpdateDC(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return updatedDC.GetPayload(), nil
}

func (r *TestClient) PatchDC(seed, dcToPatch, patch string) (*models.Datacenter, error) {
	params := &datacenter.PatchDCParams{
		Patch:     strings.NewReader(patch),
		DCToPatch: dcToPatch,
		Seed:      seed,
	}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	patchedDC, err := r.client.Datacenter.PatchDC(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return patchedDC.GetPayload(), nil
}

func (r *TestClient) GetDCForSeed(seed, dc string) (*models.Datacenter, error) {
	params := &datacenter.GetDCForSeedParams{
		Seed: seed,
		DC:   dc,
	}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute, http.StatusNotFound)

	receivedDC, err := r.client.Datacenter.GetDCForSeed(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return receivedDC.GetPayload(), nil
}

func (r *TestClient) ListDCForSeed(seed string) ([]*models.Datacenter, error) {
	params := &datacenter.ListDCForSeedParams{
		Seed: seed,
	}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	list, err := r.client.Datacenter.ListDCForSeed(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return list.GetPayload(), nil
}

func (r *TestClient) GetDC(dc string) (*models.Datacenter, error) {
	params := &datacenter.GetDatacenterParams{
		DC: dc,
	}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	receivedDC, err := r.client.Datacenter.GetDatacenter(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return receivedDC.GetPayload(), nil
}

func (r *TestClient) ListDC() ([]*models.Datacenter, error) {
	params := &datacenter.ListDatacentersParams{}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	list, err := r.client.Datacenter.ListDatacenters(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return list.GetPayload(), nil
}

func (r *TestClient) Logout() error {
	params := &users.LogoutCurrentUserParams{}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	_, err := r.client.Users.LogoutCurrentUser(params, r.bearerToken)
	return err
}

func (r *TestClient) GetKubeconfig(dc, projectID, clusterID string) (string, error) {
	params := &project.GetClusterKubeconfigParams{
		ClusterID: clusterID,
		DC:        dc,
		ProjectID: projectID,
	}
	SetupParams(r.test, params, 1*time.Second, 3*time.Minute)

	conf, err := r.client.Project.GetClusterKubeconfig(params, r.bearerToken)
	if err != nil {
		return "", err
	}

	return string(conf.Payload), nil
}

func (r *TestClient) GetUserClusterClient(dc, projectID, clusterID string) (ctrlruntimeclient.Client, error) {
	userClusterKubeconfig, err := r.GetKubeconfig(dc, projectID, clusterID)
	if err != nil {
		return nil, err
	}
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(userClusterKubeconfig))
	if err != nil {
		return nil, err
	}

	return ctrlruntimeclient.New(config, ctrlruntimeclient.Options{Scheme: scheme.Scheme})
}

// GetConstraint gets the constraint with the given name, project and cluster; it does not perform any
// retries if the API returns errors.
func (r *TestClient) GetConstraint(projectID, clusterID, name string) (*apiv2.Constraint, error) {
	params := &project.GetConstraintParams{
		ProjectID: projectID,
		ClusterID: clusterID,
		Name:      name,
	}

	SetupRetryParams(r.test, params, Backoff{
		Duration: 1 * time.Second,
		Steps:    4,
		Factor:   1.5,
	})

	project, err := r.client.Project.GetConstraint(params, r.bearerToken)
	if err != nil {
		return nil, err
	}

	return convertConstraint(project.Payload)
}

func convertConstraint(constraint *models.Constraint) (*apiv2.Constraint, error) {
	apiConstraint := &apiv2.Constraint{}
	apiConstraint.Name = constraint.Name
	apiConstraint.Spec = kubermaticv1.ConstraintSpec{
		ConstraintType: constraint.Spec.ConstraintType,
	}

	return apiConstraint, nil
}
