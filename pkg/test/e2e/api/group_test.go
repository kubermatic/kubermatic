//go:build e2e

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

package api

import (
	"context"
	"testing"
	"time"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	"k8s.io/apimachinery/pkg/util/rand"
)

func TestOidcGroupSupport(t *testing.T) {
	ctx := context.Background()

	// Login as an administrator.
	masterToken, err := utils.RetrieveMasterToken(ctx)
	if err != nil {
		t.Fatalf("failed to get master token: %v", err)
	}
	masterClient := utils.NewTestClient(masterToken, t)

	// Create some project with administrator's account.
	testProject, err := masterClient.CreateProject(rand.String(10))
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	defer cleanupProject(t, testProject.ID)

	// Login as Jane (member of "developers" group).
	janeToken, err := utils.RetrieveLDAPToken(ctx)
	if err != nil {
		t.Fatalf("failed to get Jane's token: %v", err)
	}
	janeClient := utils.NewTestClient(janeToken, t)

	// Try to access the project created by the administrator.
	_, err = janeClient.GetProject(testProject.ID)
	if err == nil {
		t.Fatalf("expected error, shouldn't be able to get admin's project")
	}

	// Create a binding between "developers" group and administrator's project.
	binding, err := masterClient.CreateGroupProjectBinding("developers", "owners", testProject.ID)
	if err != nil {
		t.Fatalf("failed to create project group binding: %s", err.Error())
	}

	// We have to wait a moment for the RBAC stuff to be reconciled.
	time.Sleep(3 * time.Second)

	// Try to access the project again.
	_, err = janeClient.GetProject(testProject.ID)
	if err != nil {
		t.Fatalf("failed to get the project: %s", err.Error())
	}

	// Remove GroupProjectBinding.
	err = janeClient.DeleteGroupProjectBinding(binding.Name, testProject.ID)
	if err != nil {
		t.Fatalf("failed to delete group project binding: %s", err.Error())
	}

	// Again, some sleep for the RBAC stuff to be reconciled.
	time.Sleep(3 * time.Second)

	// Try to access the project one last time.
	_, err = janeClient.GetProject(testProject.ID)
	if err == nil {
		t.Fatalf("expected error, shouldn't be able to get admin's project")
	}
}
