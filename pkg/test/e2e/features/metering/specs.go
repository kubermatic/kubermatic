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

// Package metering registers the metering testkit checks as Ginkgo specs for
// the KKP e2e suites. The checks themselves live in
// pkg/test/e2e/testkit/metering.
package metering

import (
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	meteringtestkit "k8c.io/kubermatic/v2/pkg/test/e2e/testkit/metering"
)

// Specs registers one spec per entry in Features. getInput is called when a
// spec runs, so the suite can fill the input in its BeforeSuite.
func Specs(getInput func() meteringtestkit.Input) bool {
	var regular, last []Feature
	for _, feature := range Features {
		if feature.Mode == ModeLast {
			last = append(last, feature)
		} else {
			regular = append(regular, feature)
		}
	}

	ginkgo.Describe("Metering", func() {
		for _, feature := range regular {
			registerSpec(feature, getInput)
		}
	})

	// Ginkgo orders top-level containers by the highest SpecPriority of their
	// specs, so a feature that has to run last gets its own container
	for _, feature := range last {
		ginkgo.Describe("Metering", ginkgo.SpecPriority(lastPriority), func() {
			registerSpec(feature, getInput)
		})
	}

	return true
}

func registerSpec(feature Feature, getInput func() meteringtestkit.Input) {
	args := append(feature.decorators(), func(ctx ginkgo.SpecContext) {
		in := getInput()
		gomega.Expect(in.Validate()).To(gomega.Succeed(), "suite input is incomplete")
		gomega.Expect(feature.Check(ctx, in)).To(gomega.Succeed())
	})

	ginkgo.It(fmt.Sprintf("%s %s", feature.ID, feature.Name), args...)
}
