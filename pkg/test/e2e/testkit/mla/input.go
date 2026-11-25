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

// Package mla holds the user cluster MLA checks used by the KKP e2e suites.
//
// Every check is a plain function that takes an Input and returns an error,
// so other suites (for example conformance-ee) can run the same checks on
// clusters they created themselves. Checks never create or delete clusters,
// never read command line flags and do not depend on a test framework.
package mla

import (
	"errors"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Input is everything a check needs. The caller owns the lifecycle of the
// seed and the user cluster.
type Input struct {
	// SeedClient and SeedConfig point to the seed cluster that runs the MLA
	// stack in the "mla" namespace and the user cluster control plane.
	SeedClient ctrlruntimeclient.Client
	SeedConfig *rest.Config

	// UserClient points to the user cluster under test.
	UserClient ctrlruntimeclient.Client

	Cluster *kubermaticv1.Cluster
	Project *kubermaticv1.Project

	// GrafanaAdminEmail is the KKP admin user that is expected to be
	// provisioned into Grafana as a Grafana admin.
	GrafanaAdminEmail string

	Log *zap.SugaredLogger
}

// Validate returns an error if a required field is missing.
func (in Input) Validate() error {
	var errs []error

	if in.SeedClient == nil {
		errs = append(errs, errors.New("SeedClient is required"))
	}
	if in.SeedConfig == nil {
		errs = append(errs, errors.New("SeedConfig is required"))
	}
	if in.UserClient == nil {
		errs = append(errs, errors.New("UserClient is required"))
	}
	if in.Cluster == nil {
		errs = append(errs, errors.New("Cluster is required"))
	}
	if in.Project == nil {
		errs = append(errs, errors.New("Project is required"))
	}
	if in.Log == nil {
		errs = append(errs, errors.New("Log is required"))
	}

	return errors.Join(errs...)
}
