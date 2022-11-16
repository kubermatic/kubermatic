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
	"crypto/x509"
	"fmt"
	"path"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
)

// Folder represents a vsphere folder.
type Folder struct {
	Path string
}

// createVMFolder creates the specified vm folder if it does not exist yet. It returns true if a new folder has been created
// and false if no new folder is created or on reported errors, other than not found error.
func createVMFolder(ctx context.Context, session *Session, fullPath string) (bool, error) {
	rootPath, newFolder := path.Split(fullPath)

	rootFolder, err := session.Finder.Folder(ctx, rootPath)
	if err != nil {
		return false, fmt.Errorf("couldn't find rootpath, see: %w", err)
	}

	if _, err = session.Finder.Folder(ctx, newFolder); err != nil {
		if !isNotFound(err) {
			return false, fmt.Errorf("failed to get folder %s: %w", fullPath, err)
		}

		if _, err = rootFolder.CreateFolder(ctx, newFolder); err != nil {
			return false, fmt.Errorf("failed to create folder %s: %w", fullPath, err)
		}

		return true, err
	}

	return false, nil
}

// deleteVMFolder deletes the specified folder.
func deleteVMFolder(ctx context.Context, session *Session, path string) error {
	folder, err := session.Finder.Folder(ctx, path)
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("couldn't open folder %q: %w", path, err)
	}

	task, err := folder.Destroy(ctx)
	if err != nil {
		return fmt.Errorf("failed to trigger folder deletion: %w", err)
	}
	if err := task.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for deletion of folder: %w", err)
	}

	return nil
}

// getVMFolders returns a slice of VSphereFolders of the datacenter from the passed cloudspec.
func getVMFolders(ctx context.Context, dc *kubermaticv1.DatacenterSpecVSphere, username, password string, caBundle *x509.CertPool) ([]Folder, error) {
	session, err := newSession(ctx, dc, username, password, caBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to create vCenter session: %w", err)
	}
	defer session.Logout(ctx)

	// We simply list all folders & filter out afterwards.
	// Filtering here is not possible as vCenter only lists the first level when giving a path.
	// vCenter only lists folders recursively if you just specify "*".
	folderRefs, err := session.Finder.FolderList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("couldn't retrieve folder list: %w", err)
	}

	rootPath := getVMRootPath(dc)
	var folders []Folder
	for _, folderRef := range folderRefs {
		// We filter by rootPath. If someone configures it, we should respect it.
		if !strings.HasPrefix(folderRef.InventoryPath, rootPath+"/") && folderRef.InventoryPath != rootPath {
			continue
		}
		folder := Folder{Path: folderRef.Common.InventoryPath}
		folders = append(folders, folder)
	}

	return folders, nil
}
