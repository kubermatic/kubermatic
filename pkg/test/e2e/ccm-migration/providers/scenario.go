/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package providers

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type errorPrinter struct {
	err error
}

func (p *errorPrinter) Errorf(format string, args ...interface{}) {
	p.err = fmt.Errorf(format, args...)
}

type TestScenario interface {
	ClusterJig() *jig.ClusterJig
	MachineJig() *jig.MachineJig
	Setup(ctx context.Context) (*kubermaticv1.Cluster, error)
	CheckComponents(ctx context.Context, cluster *kubermaticv1.Cluster, userClient ctrlruntimeclient.Client) (bool, error)
	Cleanup(ctx context.Context, cluster *kubermaticv1.Cluster, userClient ctrlruntimeclient.Client) error
}

type commmonScenario struct {
	seedClient ctrlruntimeclient.Client
	testJig    *jig.TestJig
}

func (s *commmonScenario) ClusterJig() *jig.ClusterJig {
	return s.testJig.ClusterJig
}

func (s *commmonScenario) MachineJig() *jig.MachineJig {
	return s.testJig.MachineJig
}

func (s *commmonScenario) Setup(ctx context.Context) (*kubermaticv1.Cluster, error) {
	_, cluster, err := s.testJig.Setup(ctx, jig.WaitForReadyPods)

	return cluster, err
}

func (s *commmonScenario) Cleanup(ctx context.Context, cluster *kubermaticv1.Cluster, userClient ctrlruntimeclient.Client) error {
	// Skip eviction to speed up the clean up process
	if err := s.MachineJig().SkipEvictionForAllNodes(ctx, userClient); err != nil {
		return fmt.Errorf("failed to skip node evictions: %w", err)
	}

	p := errorPrinter{}
	s.testJig.Cleanup(ctx, &p, true)

	return p.err
}
