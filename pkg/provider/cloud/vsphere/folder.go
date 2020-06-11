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
