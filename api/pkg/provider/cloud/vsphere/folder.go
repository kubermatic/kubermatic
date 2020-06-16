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
	"fmt"
	"path"
)

// createVMFolder creates the specified vm folder if it does not exist yet.
func createVMFolder(ctx context.Context, session *Session, fullPath string) error {
	rootPath, newFolder := path.Split(fullPath)

	rootFolder, err := session.Finder.Folder(ctx, rootPath)
	if err != nil {
		return fmt.Errorf("couldn't find rootpath, see: %v", err)
	}

	if _, err = session.Finder.Folder(ctx, newFolder); err != nil {
		if !isNotFound(err) {
			return fmt.Errorf("failed to get folder %s: %v", fullPath, err)
		}

		if _, err = rootFolder.CreateFolder(ctx, newFolder); err != nil {
			return fmt.Errorf("failed to create folder %s: %v", fullPath, err)
		}
	}

	return nil
}

// deleteVMFolder deletes the specified folder.
func deleteVMFolder(ctx context.Context, session *Session, path string) error {
	folder, err := session.Finder.Folder(ctx, path)
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("couldn't open folder %q: %v", path, err)
	}

	task, err := folder.Destroy(ctx)
	if err != nil {
		return fmt.Errorf("failed to trigger folder deletion: %v", err)
	}
	if err := task.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for deletion of folder: %v", err)
	}

	return nil
}
