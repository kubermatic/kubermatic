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

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
)

func cleanupProject(t *testing.T, id string) {
	// use a dedicated context so that cleanups always run, even
	// if the context inside a test was already cancelled
	token, err := utils.RetrieveAdminMasterToken(context.Background())
	if err != nil {
		t.Fatalf("failed to get master token: %v", err)
	}

	utils.NewTestClient(token, t).CleanupProject(t, id)
}
