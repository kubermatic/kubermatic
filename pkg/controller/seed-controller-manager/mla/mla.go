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

package mla

import (
	"fmt"

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	grafanasdk "github.com/aborilov/sdk"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	ControllerName = "kubermatic_mla_controller"
	mlaFinalizer   = "kubermatic.io/mla"
)

// Add creates a new MLA controller that is responsible for
// managing Monitoring, Logging and Alerting for user clusters.
// * project controller - create/update Grafana organizations based on Kubermatic Projects
func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	grafanaURL string,
	grafanaHeader string,
) error {
	grafanaClient := grafanasdk.NewClient(grafanaURL, "admin:admin", grafanasdk.DefaultHTTPClient)
	if err := newProjectReconciler(mgr, log, numWorkers, workerName, versions, grafanaClient); err != nil {
		return fmt.Errorf("failed to create mla project controller: %v", err)
	}
	return nil
}
