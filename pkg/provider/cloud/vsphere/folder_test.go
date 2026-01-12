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
	"testing"

	"github.com/vmware/govmomi/find"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/rand"
)

func TestProvider_GetVMFolders(t *testing.T) {
	sim := vSphereSimulator{t: t}
	sim.setUp()
	defer sim.tearDown()

	dc := &kubermaticv1.DatacenterSpecVSphere{
		RootPath: "/DC0/vm",
	}
	sim.fillClientInfo(dc)

	ctx := context.Background()
	session, err := newSession(ctx, dc, sim.username(), sim.password(), nil)
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

	if _, err = vmFolder.CreateFolder(ctx, testFolderName); err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}
	defer cleanupFolder(t, finder, ctx, expectedFolderPath)

	folders, err := GetVMFolders(ctx, dc, sim.username(), sim.password(), nil)
	if err != nil {
		t.Fatalf("GetVMFolders failed: %v", err)
	}

	if len(folders) == 0 {
		t.Fatal("expected at least one folder, got none")
	}

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
}

func cleanupFolder(t *testing.T, finder *find.Finder, ctx context.Context, path string) {
	folder, err := finder.Folder(ctx, path)
	if err != nil {
		t.Logf("warning: failed to find test folder for cleanup: %v", err)
		return
	}

	task, err := folder.Destroy(ctx)
	if err != nil {
		t.Logf("warning: failed to destroy test folder: %v", err)
		return
	}
	_ = task.Wait(ctx)
}
