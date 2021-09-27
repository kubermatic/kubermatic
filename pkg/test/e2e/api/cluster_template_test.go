//go:build e2e

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

package api

import (
	"context"
	"testing"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	"k8s.io/apimachinery/pkg/util/rand"
)

func TestCreateClusterTemplateAndInstance(t *testing.T) {
	tests := []struct {
		name       string
		scope      string
		dc         string
		location   string
		version    string
		credential string
		newName    string
		replicas   int64
	}{
		{
			name:       "create cluster template in user scope",
			scope:      kubermaticv1.UserClusterTemplateScope,
			dc:         "kubermatic",
			location:   "do-fra1",
			version:    utils.KubernetesVersion(),
			credential: "e2e-digitalocean",
			newName:    "newName",
			replicas:   3,
		},
		{
			name:       "create cluster template in project scope",
			scope:      kubermaticv1.ProjectClusterTemplateScope,
			dc:         "kubermatic",
			location:   "do-fra1",
			version:    utils.KubernetesVersion(),
			credential: "e2e-digitalocean",
			newName:    "newName",
			replicas:   3,
		},

		{
			name:       "create cluster template in global scope",
			scope:      kubermaticv1.GlobalClusterTemplateScope,
			dc:         "kubermatic",
			location:   "do-fra1",
			version:    utils.KubernetesVersion(),
			credential: "e2e-digitalocean",
			newName:    "newName",
			replicas:   3,
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := utils.RetrieveMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get master token: %v", err)
			}

			testClient := utils.NewTestClient(masterToken, t)
			project, err := testClient.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("failed to create project: %v", err)
			}
			defer cleanupProject(t, project.ID)

			var clusterTemplate *apiv2.ClusterTemplate

			if tc.scope == kubermaticv1.GlobalClusterTemplateScope {
				adminMasterToken, err := utils.RetrieveAdminMasterToken(ctx)
				if err != nil {
					t.Fatalf("failed to get admin master token: %v", err)
				}

				adminTestClient := utils.NewTestClient(adminMasterToken, t)
				clusterTemplate, err = adminTestClient.CreateClusterTemplate(project.ID, tc.newName, tc.scope, tc.credential, tc.version, tc.location)
				if err != nil {
					t.Fatalf("failed to create cluster template: %v", getErrorResponse(err))
				}

			} else {
				clusterTemplate, err = testClient.CreateClusterTemplate(project.ID, tc.newName, tc.scope, tc.credential, tc.version, tc.location)
				if err != nil {
					t.Fatalf("failed to create cluster template: %v", getErrorResponse(err))
				}
			}

			if clusterTemplate.Name != tc.newName {
				t.Fatalf("expected name %v, but got %v", tc.newName, clusterTemplate.Name)
			}

			if clusterTemplate.Scope != tc.scope {
				t.Fatalf("expected scope %v, but got %v", tc.scope, clusterTemplate.Scope)
			}

		})
	}
}
