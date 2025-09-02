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
	"errors"
	"fmt"
	"path"
	"path/filepath"

	vapitags "github.com/vmware/govmomi/vapi/tags"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
)

const (
	folderCleanupFinalizer = "kubermatic.k8c.io/cleanup-vsphere-folder"
	// tagCleanupFinalizer will instruct the deletion of the default category tag.
	tagCleanupFinalizer = "kubermatic.k8c.io/cleanup-vsphere-tags"
	// tagCategoryCleanupFinalizer is a legacy finalizer that needs to be removed unconditionally.
	tagCategoryCleanupFinalizer = "kubermatic.k8c.io/cleanup-vsphere-tag-category"
)

// VSphere represents the vsphere provider.
type VSphere struct {
	dc                *kubermaticv1.DatacenterSpecVSphere
	log               *zap.SugaredLogger
	secretKeySelector provider.SecretKeySelectorValueFunc
	caBundle          *x509.CertPool
}

// NewCloudProvider creates a new vSphere provider.
func NewCloudProvider(dc *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc, caBundle *x509.CertPool) (*VSphere, error) {
	if dc.Spec.VSphere == nil {
		return nil, errors.New("datacenter is not a vSphere datacenter")
	}
	return &VSphere{
		dc:                dc.Spec.VSphere,
		log:               log.Logger,
		secretKeySelector: secretKeyGetter,
		caBundle:          caBundle,
	}, nil
}

var _ provider.ReconcilingCloudProvider = &VSphere{}

func (v *VSphere) ReconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return v.reconcileCluster(ctx, cluster, update)
}

func (*VSphere) ClusterNeedsReconciling(cluster *kubermaticv1.Cluster) bool {
	return false
}

func (v *VSphere) reconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	logger := v.log.With("cluster", cluster.Name)
	username, password, err := getCredentialsForCluster(cluster.Spec.Cloud, v.secretKeySelector, v.dc)
	if err != nil {
		return nil, err
	}

	restSession, err := newRESTSession(ctx, v.dc, username, password, v.caBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to create REST client session: %w", err)
	}

	if cluster.Spec.Cloud.VSphere.Tags != nil {
		cluster, err = reconcileTags(ctx, restSession, cluster, update)
		if err != nil {
			return nil, fmt.Errorf("failed to reconcile cluster tags: %w", err)
		}
	}

	session, err := newSession(ctx, v.dc, username, password, v.caBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to create vCenter session: %w", err)
	}
	defer session.Logout(ctx)

	rootPath := getVMRootPath(v.dc)

	clusterFolder := path.Join(rootPath, cluster.Name)

	if cluster.Spec.Cloud.VSphere.BasePath != "" {
		if filepath.IsAbs(cluster.Spec.Cloud.VSphere.BasePath) {
			clusterFolder = path.Join(cluster.Spec.Cloud.VSphere.BasePath, cluster.Name)
		} else {
			clusterFolder = path.Join(rootPath, cluster.Spec.Cloud.VSphere.BasePath, cluster.Name)
		}
	}

	// Only reconcile folders that are KKP managed at the clusterFolder location.
	if cluster.Spec.Cloud.VSphere.Folder == "" || cluster.Spec.Cloud.VSphere.Folder == clusterFolder {
		logger.Infow("reconciling vsphere folder", "folder", clusterFolder)
		session, err := newSession(ctx, v.dc, username, password, v.caBundle)
		if err != nil {
			return nil, fmt.Errorf("failed to create vCenter session: %w", err)
		}
		defer session.Logout(ctx)

		cluster, err = reconcileFolder(ctx, session, restSession, clusterFolder, cluster, update)
		if err != nil {
			return nil, fmt.Errorf("failed to reconcile cluster folder: %w", err)
		}
	}

	return cluster, nil
}

// InitializeCloudProvider initializes the vsphere cloud provider by setting up vm folders for the cluster.
func (v *VSphere) InitializeCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return v.reconcileCluster(ctx, cluster, update)
}

// DefaultCloudSpec adds defaults to the cloud spec.
func (v *VSphere) DefaultCloudSpec(ctx context.Context, spec *kubermaticv1.ClusterSpec) error {
	if spec.Cloud.VSphere == nil {
		return errors.New("no vSphere cloud spec found")
	}

	if spec.Cloud.VSphere.Tags == nil {
		if v.dc.DefaultTagCategoryID != "" {
			spec.Cloud.VSphere.Tags = &kubermaticv1.VSphereTag{
				CategoryID: v.dc.DefaultTagCategoryID,
				Tags:       []string{},
			}
		}
	} else {
		// If tags were specified without a CategoryID then use the default CategoryID, if configured at seed level.
		if v.dc.DefaultTagCategoryID != "" && spec.Cloud.VSphere.Tags.CategoryID == "" {
			spec.Cloud.VSphere.Tags.CategoryID = v.dc.DefaultTagCategoryID
		}
	}

	return nil
}

// ValidateCloudSpec validates whether a vsphere client can be constructed for
// the passed cloudspec and perform some additional checks on datastore config.
func (v *VSphere) ValidateCloudSpec(ctx context.Context, spec kubermaticv1.CloudSpec) error {
	username, password, err := getCredentialsForCluster(spec, v.secretKeySelector, v.dc)
	if err != nil {
		return err
	}

	if v.dc.DefaultDatastore == "" && spec.VSphere.DatastoreCluster == "" && spec.VSphere.Datastore == "" {
		return errors.New("no default datastore provided at datacenter nor datastore/datastore cluster at cluster level")
	}

	if spec.VSphere.DatastoreCluster != "" && spec.VSphere.Datastore != "" {
		return errors.New("either datastore or datastore cluster can be selected")
	}

	if spec.VSphere.Tags != nil && spec.VSphere.Tags.CategoryID == "" && v.dc.DefaultTagCategoryID == "" {
		return errors.New("CategoryID must be specified with tags")
	}

	session, err := newSession(ctx, v.dc, username, password, v.caBundle)
	if err != nil {
		return fmt.Errorf("failed to create vCenter session: %w", err)
	}
	defer session.Logout(ctx)

	if dc := spec.VSphere.DatastoreCluster; dc != "" {
		if _, err := session.Finder.DatastoreCluster(ctx, spec.VSphere.DatastoreCluster); err != nil {
			return fmt.Errorf("failed to get datastore cluster provided by cluster spec %q: %w", dc, err)
		}
	} else {
		effectiveDatastore := v.dc.DefaultDatastore
		if ds := spec.VSphere.Datastore; ds != "" {
			effectiveDatastore = ds
		}

		if effectiveDatastore != "" {
			if _, err := session.Finder.Datastore(ctx, effectiveDatastore); err != nil {
				return fmt.Errorf("failed to get effective datastore %q: %w", effectiveDatastore, err)
			}
		}
	}

	if rp := spec.VSphere.ResourcePool; rp != "" {
		if _, err := session.Finder.ResourcePool(ctx, rp); err != nil {
			return fmt.Errorf("failed to get resource pool %s: %w", rp, err)
		}
	}

	if spec.VSphere.Tags != nil && spec.VSphere.Tags.CategoryID != "" {
		restSession, err := newRESTSession(ctx, v.dc, username, password, v.caBundle)
		if err != nil {
			return fmt.Errorf("failed to create REST client session: %w", err)
		}
		defer restSession.Logout(ctx)

		if _, err := vapitags.NewManager(restSession.Client).GetCategory(ctx, spec.VSphere.Tags.CategoryID); err != nil {
			return fmt.Errorf("failed to fetch tag category: %w", err)
		}
	}

	return nil
}

// CleanUpCloudProvider we always check if the folder is there and remove it if yes because we know its absolute path
// This covers cases where the finalizer was not added
// We also remove the finalizer if either the folder is not present or we successfully deleted it.
func (v *VSphere) CleanUpCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	username, password, err := getCredentialsForCluster(cluster.Spec.Cloud, v.secretKeySelector, v.dc)
	if err != nil {
		return nil, err
	}

	session, err := newSession(ctx, v.dc, username, password, v.caBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to create vCenter session: %w", err)
	}
	defer session.Logout(ctx)

	restSession, err := newRESTSession(ctx, v.dc, username, password, v.caBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to create REST client session: %w", err)
	}
	defer restSession.Logout(ctx)

	if kuberneteshelper.HasFinalizer(cluster, folderCleanupFinalizer) {
		if err := deleteVMFolder(ctx, session, cluster.Spec.Cloud.VSphere.Folder); err != nil {
			return nil, err
		}
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, folderCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.VSphere.Tags != nil {
		// During cleanup, we only care about the side effect of deleting the tags from vSphere.
		// We can safely ignore the returned list of managed tags because the cluster is being
		// deleted, so there's no need to update its annotations.
		_, err = syncDeletedClusterTags(ctx, restSession, cluster, getManagedTags(cluster))
		if err != nil {
			return nil, fmt.Errorf("failed to cleanup cluster tags: %w", err)
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, tagCleanupFinalizer) {
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, tagCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	// remove orphaned Category finalizer
	if kuberneteshelper.HasFinalizer(cluster, tagCategoryCleanupFinalizer) {
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, tagCategoryCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted.
func (v *VSphere) ValidateCloudSpecUpdate(_ context.Context, oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	if oldSpec.VSphere == nil || newSpec.VSphere == nil {
		return errors.New("'vsphere' spec is empty")
	}

	if oldSpec.VSphere.Folder != "" && oldSpec.VSphere.Folder != newSpec.VSphere.Folder {
		return fmt.Errorf("updating vSphere folder is not supported (was %s, updated to %s)", oldSpec.VSphere.Folder, newSpec.VSphere.Folder)
	}

	return nil
}
