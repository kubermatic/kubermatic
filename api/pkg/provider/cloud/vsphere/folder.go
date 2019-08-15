package vsphere

import (
	"context"
	"fmt"
	"path"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
)

// createVMFolder creates the specified vm folder if it does not exist yet.
func createVMFolder(ctx context.Context, client *govmomi.Client, fullPath string) error {
	finder := find.NewFinder(client.Client, true)
	rootPath, newFolder := path.Split(fullPath)

	rootFolder, err := finder.Folder(ctx, rootPath)
	if err != nil {
		return fmt.Errorf("couldn't find rootpath, see: %v", err)
	}

	if _, err = finder.Folder(ctx, newFolder); err != nil {
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
func deleteVMFolder(ctx context.Context, client *govmomi.Client, path string) error {
	finder := find.NewFinder(client.Client, true)

	folder, err := finder.Folder(ctx, path)
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
