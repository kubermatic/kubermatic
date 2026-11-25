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

// Package metering holds the KKP metering checks used by the KKP e2e suites.
//
// Metering is an Enterprise Edition feature: the KKP operator deploys a
// metering Prometheus and one report CronJob per report configuration of a
// Seed, and the reports land in an S3 bucket. Every check is a plain function
// that takes an Input and returns an error, so other suites can run the same
// checks. Checks never create or delete clusters, never read command line
// flags and do not depend on a test framework.
package metering

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Names of the objects the KKP operator creates for metering, see
// pkg/ee/metering. They are repeated here because the checks must not
// depend on the EE build.
const (
	prometheusName    = "metering-prometheus"
	prometheusPVCName = "storage-metering-prometheus-0"
	s3SecretName      = "metering-s3"
	cronJobPrefix     = "metering-"

	nameLabel      = "app.kubernetes.io/name"
	componentLabel = "app.kubernetes.io/component"
	componentValue = "metering"

	// defaultRetentionDays and defaultStorageSize are the Seed defaults from
	// pkg/defaulting/seed.go.
	defaultRetentionDays = 90
	defaultStorageSize   = "100Gi"
)

// Input is everything a check needs. The caller owns the lifecycle of the
// seed and the user cluster.
type Input struct {
	// SeedClient and SeedConfig point to the cluster that holds the Seed
	// object and runs metering. The checks expect master and seed to be the
	// same cluster, as in the KKP e2e and ks environments.
	SeedClient ctrlruntimeclient.Client
	SeedConfig *rest.Config

	// SeedName and SeedNamespace identify the Seed under test. Metering
	// objects live in SeedNamespace.
	SeedName      string
	SeedNamespace string

	// ReportPrefix is the unique prefix of the report configurations the
	// suite adds to the Seed. Checks that change report configurations only
	// touch names with this prefix.
	ReportPrefix string

	// Cluster, Project and UserClient are the user cluster whose usage the
	// report checks look for. They are optional; checks that need them
	// return an error when they are missing.
	Cluster    *kubermaticv1.Cluster
	Project    *kubermaticv1.Project
	UserClient ctrlruntimeclient.Client

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
	if in.SeedName == "" || in.SeedNamespace == "" {
		errs = append(errs, errors.New("SeedName and SeedNamespace are required"))
	}
	if in.ReportPrefix == "" {
		errs = append(errs, errors.New("ReportPrefix is required"))
	}
	if in.Log == nil {
		errs = append(errs, errors.New("Log is required"))
	}

	return errors.Join(errs...)
}

func (in Input) requireCluster() error {
	if in.Cluster == nil || in.Project == nil || in.UserClient == nil {
		return errors.New("this check needs a user cluster (Cluster, Project and UserClient)")
	}

	return nil
}

// CSVReport is the report configuration the suite adds for CSV reports of
// both types, with a retention.
func (in Input) CSVReport() string {
	return in.ReportPrefix + "-csv"
}

// JSONReport is the report configuration the suite adds for JSON reports of
// both types, without a retention.
func (in Input) JSONReport() string {
	return in.ReportPrefix + "-json"
}

// adhocDir is the bucket prefix of the reports the checks generate for their
// own time window.
func (in Input) adhocDir() string {
	return in.ReportPrefix + "-adhoc"
}

func (in Input) seed(ctx context.Context) (*kubermaticv1.Seed, error) {
	seed := &kubermaticv1.Seed{}
	if err := in.SeedClient.Get(ctx, types.NamespacedName{Namespace: in.SeedNamespace, Name: in.SeedName}, seed); err != nil {
		return nil, fmt.Errorf("failed to get Seed %s: %w", in.SeedName, err)
	}

	return seed, nil
}

// meteringConfig returns the metering configuration of the Seed, or an error
// when metering is not enabled.
func (in Input) meteringConfig(ctx context.Context) (*kubermaticv1.Seed, *kubermaticv1.MeteringConfiguration, error) {
	seed, err := in.seed(ctx)
	if err != nil {
		return nil, nil, err
	}

	if seed.Spec.Metering == nil || !seed.Spec.Metering.Enabled {
		return nil, nil, fmt.Errorf("metering is not enabled on Seed %s", seed.Name)
	}

	return seed, seed.Spec.Metering, nil
}
