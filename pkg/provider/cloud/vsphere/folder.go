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

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vapi/tags"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
)

// Folder represents a vsphere folder.
type Folder struct {
	Path string
}

// reconcileFolder reconciles a vSphere folder.
func reconcileFolder(ctx context.Context, s *Session, restSession *RESTSession, folderPath string,
	cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	err := ensureVMFolder(ctx, s, restSession, folderPath, cluster.Spec.Cloud.VSphere.Tags)
	if err != nil {
		return nil, fmt.Errorf("failed to create the VM folder %q: %w", folderPath, err)
	}

	cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		if !kuberneteshelper.HasFinalizer(cluster, folderCleanupFinalizer) {
			kuberneteshelper.AddFinalizer(cluster, folderCleanupFinalizer)
		}

		cluster.Spec.Cloud.VSphere.Folder = folderPath
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add finalizer %s on vsphere cluster object: %w", folderCleanupFinalizer, err)
	}
	return cluster, nil
}

// ensureVMFolder creates the specified vm folder if it does not exist yet.
func ensureVMFolder(ctx context.Context, session *Session, restSession *RESTSession, fullPath string, tags *kubermaticv1.VSphereTag) error {
	rootPath, newFolder := path.Split(fullPath)

	rootFolder, err := session.Finder.Folder(ctx, rootPath)
	if err != nil {
		return fmt.Errorf("couldn't find rootpath, see: %w", err)
	}

	folder, err := session.Finder.Folder(ctx, newFolder)
	if err != nil {
		if !isNotFound(err) {
			return fmt.Errorf("failed to get folder %s: %w", fullPath, err)
		}

		folder, err = rootFolder.CreateFolder(ctx, newFolder)
		if err != nil {
			return fmt.Errorf("failed to create folder %s: %w", fullPath, err)
		}
	}
	return ensureFolderTags(ctx, session, restSession, fullPath, tags, folder)
}

// deleteVMFolder deletes the specified folder.
func deleteVMFolder(ctx context.Context, session *Session, folderPath string) error {
	folder, err := session.Finder.Folder(ctx, folderPath)
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("couldn't open folder %q: %w", folderPath, err)
	}

	task, err := folder.Destroy(ctx)
	if err != nil {
		return fmt.Errorf("failed to trigger folder deletion: %w", err)
	}
	if err := task.WaitEx(ctx); err != nil {
		return fmt.Errorf("failed to wait for deletion of folder: %w", err)
	}

	return nil
}

// GetVMFolders returns a slice of VSphereFolders of the datacenter from the passed cloudspec.
func GetVMFolders(ctx context.Context, dc *kubermaticv1.DatacenterSpecVSphere, username, password string, caBundle *x509.CertPool) ([]Folder, error) {
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
		folder := Folder{Path: folderRef.InventoryPath}
		folders = append(folders, folder)
	}

	return folders, nil
}

func ensureFolderTags(ctx context.Context, session *Session, restSession *RESTSession, fullPath string,
	desiredTags *kubermaticv1.VSphereTag, folder *object.Folder) error {
	if desiredTags == nil {
		return nil
	}

	tagManager := tags.NewManager(restSession.Client)
	// Fetch tags associated with the folder.
	folderTags, err := tagManager.GetAttachedTags(ctx, folder.Reference())
	if err != nil {
		return fmt.Errorf("failed to retrieve tags for folder %q: %w", fullPath, err)
	}

	var tagsToDelete, tagsToCreate []string
	// Check if the folder has all tags that are specified in the cluster spec.
	for _, tag := range desiredTags.Tags {
		tagID, err := determineTagID(ctx, tagManager, tag, desiredTags.CategoryID)
		if err != nil {
			return err
		}

		// Check if the tag is already attached to the folder.
		var found bool
		for _, folderTag := range folderTags {
			if folderTag.CategoryID == desiredTags.CategoryID && folderTag.ID == tagID {
				found = true
				break
			}
		}

		if !found {
			tagsToCreate = append(tagsToCreate, tagID)
		}
	}

	// Check if the folder has tags that are not specified in the cluster spec.
	for _, folderTag := range folderTags {
		// We only care about tags that are in the same category as the tags specified in the cluster spec. Tags from other categories are ignored.
		if folderTag.CategoryID == desiredTags.CategoryID {
			var found bool
			for _, tag := range desiredTags.Tags {
				if tag == folderTag.Name {
					found = true
					break
				}
			}

			if !found {
				tagsToDelete = append(tagsToDelete, folderTag.ID)
			}
		}
	}
	// At this point we have lists of tags that need to be attached and detached from the folder.
	// Attach tags to the folder.
	for _, tagID := range tagsToCreate {
		if err := tagManager.AttachTag(ctx, tagID, folder.Reference()); err != nil {
			return fmt.Errorf("failed to attach tag %q to folder: %w", tagID, err)
		}
	}

	// Detach tags from the folder.
	for _, tagID := range tagsToDelete {
		if err := tagManager.DetachTag(ctx, tagID, folder.Reference()); err != nil {
			return fmt.Errorf("failed to detach tag %q from folder: %w", tagID, err)
		}
	}
	return err
}

func determineTagID(ctx context.Context, tagManager *tags.Manager, tag, tagCategoryID string) (string, error) {
	apiTag, err := tagManager.GetTagForCategory(ctx, tag, tagCategoryID)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve tag %v: %w", tag, err)
	}
	return apiTag.ID, nil
}
