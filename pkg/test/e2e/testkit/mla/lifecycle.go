/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package mla

import (
	"context"
	"errors"
	"fmt"
	"time"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/mla"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// EnableMLA turns on user cluster MLA in the cluster's Seed and enables
// monitoring and logging for the cluster. Changing the Seed redeploys the
// seed-controller-manager, so call it once per suite, before any check.
func EnableMLA(ctx context.Context, in Input) error {
	return setMLAIntegration(ctx, in, true)
}

// DisableMLAAndCheckCleanup turns MLA off for the cluster and its Seed and
// verifies that the Grafana org, the Grafana user and the project annotation
// are removed. Nothing MLA related can be checked afterwards, so it must run
// last.
func DisableMLAAndCheckCleanup(ctx context.Context, in Input) error {
	if in.GrafanaAdminEmail == "" {
		return errors.New("Input.GrafanaAdminEmail is required for this check")
	}

	orgID, err := grafanaOrgID(ctx, in)
	if err != nil {
		return err
	}

	if err := setMLAIntegration(ctx, in, false); err != nil {
		return fmt.Errorf("failed to disable MLA: %w", err)
	}

	clusterJig := jig.NewClusterJig(in.SeedClient, in.Log).WithExistingCluster(in.Cluster.Name)
	if err := clusterJig.WaitForHealthyControlPlane(ctx, 2*time.Minute); err != nil {
		return fmt.Errorf("cluster did not get healthy: %w", err)
	}

	if err := waitForAgents(ctx, in, agentState{}, 5*time.Minute); err != nil {
		return fmt.Errorf("MLA agents not removed: %w", err)
	}

	err = wait.PollImmediate(ctx, 2*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		gateway := &appsv1.Deployment{}
		err := in.SeedClient.Get(ctx, types.NamespacedName{Namespace: in.Cluster.Status.NamespaceName, Name: gatewayName}, gateway)
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		if err != nil {
			return err, nil
		}

		return errors.New("MLA gateway Deployment still exists"), nil
	})
	if err != nil {
		return fmt.Errorf("MLA gateway not removed: %w", err)
	}

	// the client must not carry an X-Grafana-Org-Id header: once Grafana's org
	// cache expires, requests whose org header points to a deleted org are
	// rejected with 403, so the lookups below would never see the 404 they wait
	// for
	client, stop, err := newGrafanaClient(ctx, in)
	if err != nil {
		return err
	}
	defer stop()

	// require a real 404; a plain err != nil check would also pass on transport
	// errors, hiding a broken connection or a failed deletion
	err = wait.PollImmediate(ctx, time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		_, err := client.GetOrgById(ctx, orgID)
		switch {
		case errors.As(err, &grafanasdk.ErrNotFound{}):
			return nil, nil
		case err != nil:
			return fmt.Errorf("failed to look up Grafana org %d: %w", orgID, err), nil
		default:
			return fmt.Errorf("Grafana org %d still exists", orgID), nil
		}
	})
	if err != nil {
		return fmt.Errorf("Grafana org not cleaned up: %w", err)
	}

	// user cleanup is asynchronous: disabling MLA in the Seed makes the KKP
	// operator redeploy the seed-controller-manager without the
	// -enable-user-cluster-mla flag, and only then the one-shot cleanup
	// controller deletes the Grafana user; the timeout covers the redeploy,
	// leader election, and cleanup retries
	err = wait.PollImmediate(ctx, time.Second, 15*time.Minute, func(ctx context.Context) (error, error) {
		_, err := client.LookupUser(ctx, in.GrafanaAdminEmail)
		switch {
		case errors.As(err, &grafanasdk.ErrNotFound{}):
			return nil, nil
		case err != nil:
			return fmt.Errorf("failed to look up Grafana user %s: %w", in.GrafanaAdminEmail, err), nil
		default:
			return fmt.Errorf("Grafana user %s still exists", in.GrafanaAdminEmail), nil
		}
	})
	if err != nil {
		return fmt.Errorf("Grafana user not cleaned up: %w", err)
	}

	err = wait.PollImmediate(ctx, time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		project := &kubermaticv1.Project{}
		if err := in.SeedClient.Get(ctx, types.NamespacedName{Name: in.Project.Name}, project); err != nil {
			return fmt.Errorf("failed to get project: %w", err), nil
		}

		if _, ok := project.Annotations[mla.GrafanaOrgAnnotationKey]; ok {
			return errors.New("project still has the Grafana org annotation"), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("project annotation not cleaned up: %w", err)
	}

	return nil
}

func setMLAIntegration(ctx context.Context, in Input, enabled bool) error {
	seed, _, err := jig.Seed(ctx, in.SeedClient, in.Cluster.Spec.Cloud.DatacenterName)
	if err != nil {
		return fmt.Errorf("failed to get seed: %w", err)
	}

	oldSeed := seed.DeepCopy()
	seed.Spec.MLA = &kubermaticv1.SeedMLASettings{
		UserClusterMLAEnabled: enabled,
	}

	if err := in.SeedClient.Patch(ctx, seed, ctrlruntimeclient.MergeFrom(oldSeed)); err != nil {
		return fmt.Errorf("failed to update seed: %w", err)
	}

	cluster := &kubermaticv1.Cluster{}
	if err := in.SeedClient.Get(ctx, types.NamespacedName{Name: in.Cluster.Name}, cluster); err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	oldCluster := cluster.DeepCopy()
	cluster.Spec.MLA = &kubermaticv1.MLASettings{
		MonitoringEnabled: enabled,
		LoggingEnabled:    enabled,
	}

	if err := in.SeedClient.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
		return fmt.Errorf("failed to update cluster: %w", err)
	}

	return nil
}
