/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package clients

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	httptransport "github.com/go-openapi/runtime/client"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/metrics"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/scenarios"
	ctypes "k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/project"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/dex"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// apiClient uses the KKP REST API to interact with KKP.
type apiClient struct {
	opts *ctypes.Options
}

var _ Client = &apiClient{}

func NewAPIClient(opts *ctypes.Options) Client {
	return &apiClient{
		opts: opts,
	}
}

func (c *apiClient) Setup(ctx context.Context, log *zap.SugaredLogger) error {
	kubermaticClient, err := utils.NewKubermaticClient(c.opts.KubermaticEndpoint)
	if err != nil {
		log.Fatalw("Failed to create Kubermatic API client", zap.Error(err))
	}
	c.opts.KubermaticClient = kubermaticClient

	if !c.opts.CreateOIDCToken {
		if c.opts.KubermaticOIDCToken == "" {
			log.Fatal("An existing OIDC token must be set via the -kubermatic-oidc-token flag")
		}

		c.opts.KubermaticAuthenticator = httptransport.BearerToken(c.opts.KubermaticOIDCToken)
	} else {
		dexClient, err := dex.NewClientFromHelmValues(c.opts.DexHelmValuesFile, "kubermatic", log)
		if err != nil {
			log.Fatalw("Failed to create OIDC client", zap.Error(err))
		}

		// OIDC credentials are passed in as environment variables instead of
		// CLI flags because having the password as a flag might be a security
		// issue and then it also makes sense to handle the login name in the
		// same way.
		// Also, the API E2E tests use environment variables to get the values
		// into the `go test` runs.
		login, password, err := utils.OIDCCredentials()
		if err != nil {
			log.Fatalf("Invalid OIDC credentials: %v", err)
		}

		log.Infow("Creating login token", "login", login, "provider", dexClient.ProviderURI, "client", dexClient.ClientID)

		var token string

		if err := metrics.MeasureTime(
			metrics.KubermaticLoginDurationMetric.WithLabelValues(),
			log,
			func() error {
				token, err = dexClient.Login(ctx, login, password, dex.OIDCLocalConnector)
				return err
			},
		); err != nil {
			log.Fatalw("Failed to get master token", zap.Error(err))
		}

		log.Info("Successfully retrieved master token")

		c.opts.KubermaticAuthenticator = httptransport.BearerToken(token)
	}

	return nil
}

func (c *apiClient) CreateProject(ctx context.Context, log *zap.SugaredLogger, name string) (string, error) {
	log.Info("Creating project...")

	params := &project.CreateProjectParams{
		Context: ctx,
		Body:    project.CreateProjectBody{Name: name},
	}
	utils.SetupParams(nil, params, 3*time.Second, 1*time.Minute, http.StatusConflict)

	result, err := c.opts.KubermaticClient.Project.CreateProject(params, c.opts.KubermaticAuthenticator)
	if err != nil {
		return "", fmt.Errorf("failed to create project: %w", err)
	}
	projectID := result.Payload.ID

	// we have to wait a moment for the RBAC stuff to be reconciled, and to try to avoid
	// logging a misleading error in the following loop, we just wait a few seconds;
	time.Sleep(3 * time.Second)

	getProjectParams := &project.GetProjectParams{Context: ctx, ProjectID: projectID}
	utils.SetupParams(nil, getProjectParams, 2*time.Second, 1*time.Minute, http.StatusConflict)

	if err := wait.PollImmediate(2*time.Second, 1*time.Minute, func() (bool, error) {
		response, err := c.opts.KubermaticClient.Project.GetProject(getProjectParams, c.opts.KubermaticAuthenticator)
		if err != nil {
			log.Errorw("Failed to get project", zap.Error(err))
			return false, nil
		}
		if response.Payload.Status != string(kubermaticv1.ProjectActive) {
			log.Warnw("Project not active yet", "project-status", response.Payload.Status)
			return false, nil
		}
		return true, nil
	}); err != nil {
		return "", fmt.Errorf("failed to wait for project to be ready: %w", err)
	}

	return projectID, nil
}

func (c *apiClient) CreateSSHKeys(ctx context.Context, log *zap.SugaredLogger) error {
	for i, key := range c.opts.PublicKeys {
		log.Infow("Creating UserSSHKey...", "pubkey", string(key))

		body := &project.CreateSSHKeyParams{
			Context:   ctx,
			ProjectID: c.opts.KubermaticProject,
			Key: &models.SSHKey{
				Name: fmt.Sprintf("SSH Key No. %d", i+1),
				Spec: &models.SSHKeySpec{
					PublicKey: string(key),
				},
			},
		}
		utils.SetupParams(nil, body, 3*time.Second, 1*time.Minute, http.StatusConflict)

		if _, err := c.opts.KubermaticClient.Project.CreateSSHKey(body, c.opts.KubermaticAuthenticator); err != nil {
			return fmt.Errorf("failed to create SSH key: %w", err)
		}
	}

	return nil
}

func (c *apiClient) CreateCluster(ctx context.Context, log *zap.SugaredLogger, scenario scenarios.Scenario) (*kubermaticv1.Cluster, error) {
	log.Info("Creating cluster...")

	cluster := scenario.APICluster(c.opts.Secrets)
	// The cluster name must be unique per project;
	// we build up a readable name with the various cli parameters annd
	// add a random string in the end to ensure we really have a unique name.
	if c.opts.NamePrefix != "" {
		cluster.Cluster.Name = c.opts.NamePrefix + "-"
	}
	if c.opts.WorkerName != "" {
		cluster.Cluster.Name += c.opts.WorkerName + "-"
	}
	cluster.Cluster.Name += scenario.Name() + "-"
	cluster.Cluster.Name += rand.String(8)

	cluster.Cluster.Spec.UsePodSecurityPolicyAdmissionPlugin = c.opts.PspEnabled
	cluster.Cluster.Spec.EnableOperatingSystemManager = c.opts.OperatingSystemManagerEnabled

	params := &project.CreateClusterParams{
		Context:   ctx,
		ProjectID: c.opts.KubermaticProject,
		DC:        c.opts.Seed.Name,
		Body:      cluster,
	}
	utils.SetupParams(nil, params, 3*time.Second, 1*time.Minute, http.StatusConflict)

	response, err := c.opts.KubermaticClient.Project.CreateCluster(params, c.opts.KubermaticAuthenticator)
	if err != nil {
		return nil, errors.New(getErrorResponse(err))
	}

	clusterID := response.Payload.ID
	crCluster := &kubermaticv1.Cluster{}

	if err := wait.PollImmediate(2*time.Second, 1*time.Minute, func() (bool, error) {
		key := types.NamespacedName{Name: clusterID}

		if err := c.opts.SeedClusterClient.Get(ctx, key, crCluster); err != nil {
			return false, ctrlruntimeclient.IgnoreNotFound(err)
		}

		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to wait for Cluster to appear: %w", err)
	}

	// fetch all existing SSH keys
	listKeysBody := &project.ListSSHKeysParams{
		Context:   ctx,
		ProjectID: c.opts.KubermaticProject,
	}
	utils.SetupParams(nil, listKeysBody, 3*time.Second, 1*time.Minute, http.StatusConflict, http.StatusNotFound)

	result, err := c.opts.KubermaticClient.Project.ListSSHKeys(listKeysBody, c.opts.KubermaticAuthenticator)
	if err != nil {
		return nil, fmt.Errorf("failed to list project's SSH keys: %w", err)
	}

	keyIDs := []string{}
	for _, key := range result.Payload {
		keyIDs = append(keyIDs, key.ID)
	}

	// assign all keys to the new cluster
	for _, keyID := range keyIDs {
		assignKeyBody := &project.AssignSSHKeyToClusterParams{
			Context:   ctx,
			ProjectID: c.opts.KubermaticProject,
			DC:        c.opts.Seed.Name,
			ClusterID: crCluster.Name,
			KeyID:     keyID,
		}
		utils.SetupParams(nil, assignKeyBody, 3*time.Second, 1*time.Minute, http.StatusConflict, http.StatusNotFound, http.StatusForbidden)

		if _, err := c.opts.KubermaticClient.Project.AssignSSHKeyToCluster(assignKeyBody, c.opts.KubermaticAuthenticator); err != nil {
			return nil, fmt.Errorf("failed to assign SSH key to cluster: %w", err)
		}
	}

	log.Infof("Successfully created cluster %s", crCluster.Name)
	return crCluster, nil
}

func (c *apiClient) CreateNodeDeployments(ctx context.Context, log *zap.SugaredLogger, scenario scenarios.Scenario, userClusterClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	nodeDeploymentGetParams := &project.ListNodeDeploymentsParams{
		Context:   ctx,
		ProjectID: c.opts.KubermaticProject,
		ClusterID: cluster.Name,
		DC:        c.opts.Seed.Name,
	}
	utils.SetupParams(nil, nodeDeploymentGetParams, 5*time.Second, 1*time.Minute)

	log.Info("Getting existing NodeDeployments…")
	resp, err := c.opts.KubermaticClient.Project.ListNodeDeployments(nodeDeploymentGetParams, c.opts.KubermaticAuthenticator)
	if err != nil {
		return fmt.Errorf("failed to get existing NodeDeployments: %w", err)
	}

	existingReplicas := 0
	for _, nodeDeployment := range resp.Payload {
		existingReplicas += int(*nodeDeployment.Spec.Replicas)
	}
	log.Infof("Found %d pre-existing node replicas", existingReplicas)

	nodeCount := c.opts.NodeCount - existingReplicas
	if nodeCount < 0 {
		return fmt.Errorf("found %d existing replicas and want %d, scaledown not supported", existingReplicas, c.opts.NodeCount)
	}
	if nodeCount == 0 {
		return nil
	}

	log.Info("Preparing NodeDeployments")
	var nodeDeployments []models.NodeDeployment
	if err := wait.PollImmediate(10*time.Second, time.Minute, func() (bool, error) {
		var err error
		nodeDeployments, err = scenario.NodeDeployments(ctx, nodeCount, c.opts.Secrets)
		if err != nil {
			log.Warnw("Getting NodeDeployments from scenario failed", zap.Error(err))
			return false, nil
		}
		return true, nil
	}); err != nil {
		return fmt.Errorf("didn't get NodeDeployments from scenario within a minute: %w", err)
	}

	log.Info("Creating NodeDeployments…")
	for _, nd := range nodeDeployments {
		params := &project.CreateNodeDeploymentParams{
			Context:   ctx,
			ProjectID: c.opts.KubermaticProject,
			ClusterID: cluster.Name,
			DC:        c.opts.Seed.Name,
			Body:      &nd,
		}
		utils.SetupParams(nil, params, 5*time.Second, 1*time.Minute, http.StatusConflict)

		if _, err := c.opts.KubermaticClient.Project.CreateNodeDeployment(params, c.opts.KubermaticAuthenticator); err != nil {
			return fmt.Errorf("failed to create NodeDeployment %s: %w", nd.Name, err)
		}
	}

	log.Infof("Successfully created %d NodeDeployments", nodeCount)
	return nil
}

func (c *apiClient) DeleteCluster(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, timeout time.Duration) error {
	var (
		selector labels.Selector
		err      error
	)

	if c.opts.WorkerName != "" {
		selector, err = labels.Parse(fmt.Sprintf("worker-name=%s", c.opts.WorkerName))
		if err != nil {
			return fmt.Errorf("failed to parse selector: %w", err)
		}
	}

	return wait.PollImmediate(1*time.Second, timeout, func() (bool, error) {
		clusterList := &kubermaticv1.ClusterList{}
		listOpts := &ctrlruntimeclient.ListOptions{LabelSelector: selector}
		if err := c.opts.SeedClusterClient.List(ctx, clusterList, listOpts); err != nil {
			log.Errorw("Listing clusters failed", zap.Error(err))
			return false, nil
		}

		// Success!
		if len(clusterList.Items) == 0 {
			return true, nil
		}

		// Should never happen
		if len(clusterList.Items) > 1 {
			return false, fmt.Errorf("expected to find zero or one cluster, got %d", len(clusterList.Items))
		}

		// Cluster is currently being deleted
		if clusterList.Items[0].DeletionTimestamp != nil {
			return false, nil
		}

		// Issue Delete call
		log.With("cluster", clusterList.Items[0].Name).Info("Deleting user cluster now...")

		deleteParams := &project.DeleteClusterParams{
			Context:   ctx,
			ProjectID: c.opts.KubermaticProject,
			ClusterID: clusterList.Items[0].Name,
			DC:        c.opts.Seed.Name,
		}
		utils.SetupParams(nil, deleteParams, 3*time.Second, timeout)

		if _, err := c.opts.KubermaticClient.Project.DeleteCluster(deleteParams, c.opts.KubermaticAuthenticator); err != nil {
			log.Warnw("Failed to delete cluster", zap.Error(err))
		}

		return false, nil
	})
}

// getErrorResponse converts the client error response to string.
func getErrorResponse(err error) string {
	rawData, newErr := json.Marshal(err)
	if newErr != nil {
		return err.Error()
	}

	return string(rawData)
}
