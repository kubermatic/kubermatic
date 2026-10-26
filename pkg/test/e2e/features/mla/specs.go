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

// Package mla registers the MLA testkit checks as Ginkgo specs for the KKP
// e2e suites. The checks themselves live in pkg/test/e2e/testkit/mla.
package mla

import (
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	mlatestkit "k8c.io/kubermatic/v2/pkg/test/e2e/testkit/mla"
)

// validator is implemented by every testkit input type.
type validator interface {
	Validate() error
}

// UserClusterSpecs registers one spec per entry in UserClusterFeatures.
// getInput is called when a spec runs, so the suite can fill the input after
// its BeforeSuite created or attached the cluster.
func UserClusterSpecs(getInput func() mlatestkit.Input) bool {
	return registerSpecs("User cluster MLA", UserClusterFeatures, getInput)
}

// SeedSpecs registers one spec per entry in SeedFeatures.
func SeedSpecs(getInput func() mlatestkit.SeedInput) bool {
	return registerSpecs("Seed MLA", SeedFeatures, getInput)
}

// registerSpecs registers the features as specs. Ginkgo orders top-level
// containers by the highest SpecPriority of their specs, so a feature that has
// to run last gets its own top-level container.
func registerSpecs[I validator](text string, features []Feature[I], getInput func() I) bool {
	var regular, last []Feature[I]
	for _, feature := range features {
		if feature.Mode == ModeLast {
			last = append(last, feature)
		} else {
			regular = append(regular, feature)
		}
	}

	ginkgo.Describe(text, func() {
		for _, feature := range regular {
			registerSpec(feature, getInput)
		}
	})

	for _, feature := range last {
		ginkgo.Describe(text, ginkgo.SpecPriority(lastPriority), func() {
			registerSpec(feature, getInput)
		})
	}

	return true
}

func registerSpec[I validator](feature Feature[I], getInput func() I) {
	args := append(feature.decorators(), func(ctx ginkgo.SpecContext) {
		in := getInput()
		gomega.Expect(in.Validate()).To(gomega.Succeed(), "suite input is incomplete")
		gomega.Expect(feature.Check(ctx, in)).To(gomega.Succeed())
	})

	ginkgo.It(fmt.Sprintf("%s %s", feature.ID, feature.Name), args...)
}
