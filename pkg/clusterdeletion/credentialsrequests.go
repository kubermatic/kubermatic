/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package clusterdeletion

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (d *Deletion) cleanupCredentialsRequests(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (deletedSomething bool, err error) {
	log = log.Named("CredentialsRequestsCleanup")

	userClusterClient, err := d.userClusterClientGetter()
	if err != nil {
		return false, err
	}

	credentialRequests := &unstructured.UnstructuredList{}
	credentialRequests.SetAPIVersion("cloudcredential.openshift.io/v1")
	credentialRequests.SetKind("CredentialsRequest")

	if err := userClusterClient.List(
		ctx,
		credentialRequests,
		&ctrlruntimeclient.ListOptions{Namespace: "openshift-cloud-credential-operator"},
	); err != nil {
		if meta.IsNoMatchError(err) {
			log.Debug("Got a NoMatchError when listing CredentialsRequests, skipping their cleanup")
			return false, nil
		}
		return false, fmt.Errorf("failed to list CredentialsRequests: %v", err)
	}

	if len(credentialRequests.Items) == 0 {
		log.Debug("No CredentialsRequests found, nothing to clean up")
		return false, nil
	}

	log.Debug("Found CredentialsRequests", "num-credentials-requests", len(credentialRequests.Items))

	for _, credentialRequest := range credentialRequests.Items {
		if err := userClusterClient.Delete(ctx, &credentialRequest); err != nil {
			return false, fmt.Errorf("failed to delete CredentialsRequest: %v", err)
		}
	}

	log.Debug("Successfully issued DELETE for all CredentialsRequests")
	return true, nil
}
