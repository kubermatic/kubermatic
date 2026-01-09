//go:build integration

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

package vsphere

import (
	"context"
	"strings"
	"testing"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/simulator"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/rand"
)

func TestProvider_GetVMFolders(t *testing.T) {
	// Set up vcsim simulator
	model := simulator.VPX()

	err := model.Create()
	if err != nil {
		t.Fatal(err)
	}
	defer model.Remove()

	server := model.Service.NewServer()
	defer server.Close()

	username := simulator.DefaultLogin.Username()
	password, _ := simulator.DefaultLogin.Password()

	dc := &kubermaticv1.DatacenterSpecVSphere{
		Datacenter:    "DC0",
		Endpoint:      strings.TrimSuffix(server.URL.String(), "/sdk"),
		AllowInsecure: true,
		RootPath:      "/DC0/vm",
	}

	t.Run("create and list folders", func(t *testing.T) {
		ctx := context.Background()
		session, err := newSession(ctx, dc, username, password, nil)
		if err != nil {
			t.Fatal(err)
		}

		finder := find.NewFinder(session.Client.Client, true)
		datacenter, err := finder.Datacenter(ctx, dc.Datacenter)
		if err != nil {
			t.Fatalf("failed to find datacenter: %v", err)
		}
		finder.SetDatacenter(datacenter)

		vmFolder, err := finder.Folder(ctx, dc.RootPath)
		if err != nil {
			t.Fatalf("failed to find vm folder: %v", err)
		}

		testFolderName := "kubermatic-e2e-" + rand.String(8)
		expectedFolderPath := dc.RootPath + "/" + testFolderName

		_, err = vmFolder.CreateFolder(ctx, testFolderName)
		if err != nil {
			t.Fatalf("failed to create folder: %v", err)
		}

		// Clean up
		defer func() {
			testFolder, err := finder.Folder(ctx, expectedFolderPath)
			if err != nil {
				t.Logf("warning: failed to find test folder for cleanup: %v", err)
				return
			}
			task, err := testFolder.Destroy(ctx)
			if err != nil {
				t.Logf("warning: failed to destroy test folder: %v", err)
				return
			}
			_ = task.Wait(ctx)
		}()

		folders, err := GetVMFolders(ctx, dc, username, password, nil)
		if err != nil {
			t.Fatalf("GetVMFolders failed: %v", err)
		}

		if len(folders) == 0 {
			t.Fatal("expected at least one folder, got none")
		}

		// Verify that we have the folder in the list
		folderFound := false
		for _, folder := range folders {
			if folder.Path == "" {
				t.Error("folder Path should not be empty")
			}

			if folder.Path == expectedFolderPath {
				folderFound = true
			}
		}

		if !folderFound {
			t.Errorf("created folder %s not found in GetVMFolders results", expectedFolderPath)
		}
	})
}
