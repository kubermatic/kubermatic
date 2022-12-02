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

package reconciling

import (
	"fmt"

	"github.com/go-test/deep"
	"go.uber.org/zap"

	reconcilercompare "k8c.io/reconciler/pkg/compare"
	reconcilerlog "k8c.io/reconciler/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Configure(log *zap.SugaredLogger) {
	reconcilerlog.SetLogger(log)
	reconcilercompare.DiffReporter = newLogDiffHandler(log)
}

func newLogDiffHandler(log *zap.SugaredLogger) reconcilercompare.DiffReporterFunc {
	return func(a, b metav1.Object) {
		maxDepth := deep.MaxDepth
		logErrors := deep.LogErrors

		// Kubernetes Objects can be deeper than the default 10 levels.
		deep.MaxDepth = 50
		deep.LogErrors = true

		// For informational purpose we use deep.equal as it tells us what the difference is.
		// We need to calculate the difference in both ways as deep.equal only does a one-way comparison
		diff := deep.Equal(a, b)
		if diff == nil {
			diff = deep.Equal(b, a)
		}

		deep.MaxDepth = maxDepth
		deep.LogErrors = logErrors

		log.Debugw("Object differs from generated one", "type", fmt.Sprintf("%T", a), "namespace", a.GetNamespace(), "name", a.GetName(), "diff", diff)
	}
}
