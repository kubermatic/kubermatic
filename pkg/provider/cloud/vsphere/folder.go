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

	vapitags "github.com/vmware/govmomi/vapi/tags"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
)

// Folder represents a vsphere folder.
type Folder struct {
	Path string
}

func reconcileFolder(ctx context.Context, s *Session, restSession *RESTSession, rootPath string,
	cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	// If the user did not specify a folder, we create a own folder for this cluster to improve
	// the VM management in vCenter
	clusterFolder := path.Join(rootPath, cluster.Name)
	err := createVMFolder(ctx, s, restSession, clusterFolder)
	if err != nil {
		return nil, fmt.Errorf("failed to create the VM folder %q: %w", clusterFolder, err)
	}

	cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		if !kuberneteshelper.HasFinalizer(cluster, folderCleanupFinalizer) {
			kuberneteshelper.AddFinalizer(cluster, folderCleanupFinalizer)
		}

		cluster.Spec.Cloud.VSphere.Folder = clusterFolder
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add finalizer %s on vsphere cluster object: %w", tagCategoryCleanupFinilizer, err)
	}
	return cluster, nil
}

// createVMFolder creates the specified vm folder if it does not exist yet. It returns true if a new folder has been created
// and false if no new folder is created or on reported errors, other than not found error.
func createVMFolder(ctx context.Context, session *Session, restSession *RESTSession, fullPath string) error {
	rootPath, newFolder := path.Split(fullPath)

	rootFolder, err := session.Finder.Folder(ctx, rootPath)
	if err != nil {
		return fmt.Errorf("couldn't find rootpath, see: %w", err)
	}

	if _, err = session.Finder.Folder(ctx, newFolder); err != nil {
		if !isNotFound(err) {
			return fmt.Errorf("failed to get folder %s: %w", fullPath, err)
		}

		_, err := rootFolder.CreateFolder(ctx, newFolder)
		if err != nil {
			return fmt.Errorf("failed to create folder %s: %w", fullPath, err)
		}
	}

	return nil
}

// deleteVMFolder deletes the specified folder.
func deleteVMFolder(ctx context.Context, session *Session, restSession *RESTSession, clusterName, folderPath string) error {
	folder, err := session.Finder.Folder(ctx, folderPath)
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("couldn't open folder %q: %w", folderPath, err)
	}

	tagManager := vapitags.NewManager(restSession.Client)
	attachedTags, err := tagManager.GetAttachedTags(ctx, folder)
	if err != nil {
		return fmt.Errorf("failed to fetch attached tags on folder %s: %w", folder.Name(), err)
	}

	for _, tag := range attachedTags {
		if tag.Name == controllerOwnershipTag(clusterName) {
			task, err := folder.Destroy(ctx)
			if err != nil {
				return fmt.Errorf("failed to trigger folder deletion: %w", err)
			}
			if err := task.Wait(ctx); err != nil {
				return fmt.Errorf("failed to wait for deletion of folder: %w", err)
			}
		}
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
		folder := Folder{Path: folderRef.Common.InventoryPath}
		folders = append(folders, folder)
	}

	return folders, nil
}
