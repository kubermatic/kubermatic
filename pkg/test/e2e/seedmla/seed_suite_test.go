//go:build seedmla

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

package seedmla

import (
	"flag"
	"testing"

	"github.com/go-logr/zapr"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/log"
	mlafeatures "k8c.io/kubermatic/v2/pkg/test/e2e/features/mla"
	mlatestkit "k8c.io/kubermatic/v2/pkg/test/e2e/testkit/mla"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	logOptions  = utils.DefaultLogOptions
	grafanaUser string

	// input is filled on every Ginkgo process before any spec runs.
	input mlatestkit.SeedInput
)

func init() {
	logOptions.AddFlags(flag.CommandLine)
	flag.StringVar(&grafanaUser, "seed-grafana-user", "", "seed Grafana admin user name (chart value grafana.user, default admin)")
}

func TestSeedMLA(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Seed MLA e2e")
}

var _ = ginkgo.BeforeSuite(func() {
	rawLogger := log.NewFromOptions(logOptions)
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))

	seedClient, seedConfig, err := utils.GetClients()
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to get seed clients")

	input = mlatestkit.SeedInput{
		SeedClient:  seedClient,
		SeedConfig:  seedConfig,
		GrafanaUser: grafanaUser,
		Log:         rawLogger.Sugar(),
	}
})

var _ = mlafeatures.SeedSpecs(func() mlatestkit.SeedInput {
	return input
})
