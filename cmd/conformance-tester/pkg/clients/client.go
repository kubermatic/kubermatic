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
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/scenarios"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Client represents a client that can talk to a KKP system.
// This interface and the single KubeClient implementation are
// vestigial from the pre-2.22 era where tests could also be
// run using the KKP REST API.
type Client interface {
	Setup(ctx context.Context, log *zap.SugaredLogger) error
	CreateProject(ctx context.Context, log *zap.SugaredLogger, name string) (string, error)
	EnsureSSHKeys(ctx context.Context, log *zap.SugaredLogger) error
	CreateCluster(ctx context.Context, log *zap.SugaredLogger, scenario scenarios.Scenario) (*kubermaticv1.Cluster, error)
	CreateMachineDeployments(ctx context.Context, log *zap.SugaredLogger, scenario scenarios.Scenario, userClusterClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error
	DeleteMachineDeployments(ctx context.Context, log *zap.SugaredLogger, scenario scenarios.Scenario, userClusterClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error
	DeleteCluster(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, timeout time.Duration) error
	DeleteProject(ctx context.Context, log *zap.SugaredLogger, id string, timeout time.Duration) error
}
