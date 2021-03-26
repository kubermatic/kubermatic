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
	"context"
	"fmt"
	"strings"

	grafanasdk "github.com/aborilov/sdk"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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
	ctx context.Context,
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	grafanaURL string,
	grafanaHeader string,
	grafanaSecret string,
) error {

	split := strings.Split(grafanaSecret, "/")
	if n := len(split); n != 2 {
		return fmt.Errorf("splitting value of %q didn't yield two but %d results",
			grafanaSecret, n)
	}
	secret := corev1.Secret{}
	mgr.GetConfig()
	client, err := ctrlruntimeclient.New(mgr.GetConfig(), ctrlruntimeclient.Options{})
	if err != nil {
		return err
	}
	if err := client.Get(ctx, types.NamespacedName{Name: split[1], Namespace: split[0]}, &secret); err != nil {
		return fmt.Errorf("failed to get Grafana Secret: %v", err)
	}
	auth, ok := secret.Data["auth"]
	if !ok {
		return fmt.Errorf("Grafana Secret %q does not contain auth", grafanaSecret)
	}
	grafanaClient := grafanasdk.NewClient(grafanaURL, string(auth), grafanasdk.DefaultHTTPClient)
	if err := newProjectReconciler(mgr, log, numWorkers, workerName, versions, grafanaClient); err != nil {
		return fmt.Errorf("failed to create mla project controller: %v", err)
	}
	return nil
}
