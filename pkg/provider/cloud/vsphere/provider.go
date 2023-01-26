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

	"github.com/vmware/govmomi/vapi/tags"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
)

const (
	folderCleanupFinalizer = "kubermatic.k8c.io/cleanup-vsphere-folder"
	// categoryCleanupFinilizer will instruct the deletion of the default category tag.
	tagCategoryCleanupFinilizer = "kubermatic.k8c.io/cleanup-vsphere-tag-category"

	defaultCategoryPrefix = "kubermatic.k8c.io/tag-category"
	defaultTagPrefix      = "kubermatic.k8c.io/tag"
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
	return v.reconcileCluster(ctx, cluster, update, true)
}

func (v *VSphere) reconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, force bool) (*kubermaticv1.Cluster, error) {
	logger := v.log.With("cluster", cluster.Name)

	username, password, err := getCredentialsForCluster(cluster.Spec.Cloud, v.secretKeySelector, v.dc)
	if err != nil {
		return nil, err
	}

	restSession, err := newRESTSession(ctx, v.dc, username, password, v.caBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to create REST client session: %w", err)
	}

	if force || cluster.Spec.Cloud.VSphere.TagCategory == nil {
		cluster, err = reconcileTagCategory(ctx, restSession, cluster, update)
		if err != nil {
			return nil, fmt.Errorf("failed to reconcile cluster tag category: %w", err)
		}
	}

	if force || cluster.Spec.Cloud.VSphere.Tags == nil {
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
	if force || cluster.Spec.Cloud.VSphere.Folder == "" {
		logger.Infow("reconciling vsphere folder", "folder", cluster.Spec.Cloud.VSphere.Folder)
		session, err := newSession(ctx, v.dc, username, password, v.caBundle)
		if err != nil {
			return nil, fmt.Errorf("failed to create vCenter session: %w", err)
		}
		defer session.Logout(ctx)

		cluster, err = reconcileFolder(ctx, session, restSession, rootPath, cluster, update)
		if err != nil {
			return nil, fmt.Errorf("failed to reconcile cluster folder: %w", err)
		}
	}

	return cluster, nil
}

// InitializeCloudProvider initializes the vsphere cloud provider by setting up vm folders for the cluster.
func (v *VSphere) InitializeCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return v.reconcileCluster(ctx, cluster, update, false)
}

// DefaultCloudSpec adds defaults to the cloud spec.
func (v *VSphere) DefaultCloudSpec(ctx context.Context, spec *kubermaticv1.ClusterSpec) error {
	if spec.Cloud.VSphere.TagCategory == nil {
		spec.Cloud.VSphere.TagCategory = &kubermaticv1.TagCategory{
			ID: v.dc.DefaultTagCategoryID,
		}
	}

	if tagCategory := spec.Cloud.VSphere.TagCategory; tagCategory != nil {
		username, password, err := getCredentialsForCluster(spec.Cloud, v.secretKeySelector, v.dc)
		if err != nil {
			return err
		}

		restSession, err := newRESTSession(ctx, v.dc, username, password, v.caBundle)
		if err != nil {
			return fmt.Errorf("failed to create REST client session: %w", err)
		}
		defer restSession.Logout(ctx)

		return defaultClusterSpecTagCategory(ctx, spec, tagCategory, restSession)
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

	session, err := newSession(ctx, v.dc, username, password, v.caBundle)
	if err != nil {
		return fmt.Errorf("failed to create vCenter session: %w", err)
	}
	defer session.Logout(ctx)

	if ds := v.dc.DefaultDatastore; ds != "" {
		if _, err := session.Finder.Datastore(ctx, ds); err != nil {
			return fmt.Errorf("failed to get default datastore provided by datacenter spec %q: %w", ds, err)
		}
	}

	if rp := spec.VSphere.ResourcePool; rp != "" {
		if _, err := session.Finder.ResourcePool(ctx, rp); err != nil {
			return fmt.Errorf("failed to get resource pool %s: %w", rp, err)
		}
	}

	if dc := spec.VSphere.DatastoreCluster; dc != "" {
		if _, err := session.Finder.DatastoreCluster(ctx, spec.VSphere.DatastoreCluster); err != nil {
			return fmt.Errorf("failed to get datastore cluster provided by cluster spec %q: %w", dc, err)
		}
	}

	if ds := spec.VSphere.Datastore; ds != "" {
		if _, err = session.Finder.Datastore(ctx, ds); err != nil {
			return fmt.Errorf("failed to get datastore cluster provided by cluste spec %q: %w", ds, err)
		}
	}

	if tagCategory := spec.VSphere.TagCategory; tagCategory != nil {
		restSession, err := newRESTSession(ctx, v.dc, username, password, v.caBundle)
		if err != nil {
			return fmt.Errorf("failed to create REST client session: %w", err)
		}
		defer restSession.Logout(ctx)

		if tagCategory.ID != "" {
			tagManager := tags.NewManager(restSession.Client)
			if _, err := tagManager.GetCategory(ctx, tagCategory.ID); err != nil {
				return fmt.Errorf("failed to get tag categories %w", err)
			}
		}

		if tagCategory.Name != "" {
			tagManager := tags.NewManager(restSession.Client)
			if _, err := tagManager.GetCategory(ctx, tagCategory.Name); err != nil {
				return fmt.Errorf("failed to get tag categories %w", err)
			}
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

	if err := deleteVMFolder(ctx, session, restSession, cluster.Name, cluster.Spec.Cloud.VSphere.Folder); err != nil {
		return nil, err
	}
	cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		kuberneteshelper.RemoveFinalizer(cluster, folderCleanupFinalizer)
	})
	if err != nil {
		return nil, err
	}

	if kuberneteshelper.HasFinalizer(cluster, tagCategoryCleanupFinilizer) {
		if err := deleteTagCategory(ctx, restSession, cluster); err != nil {
			return nil, err
		}
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, tagCategoryCleanupFinilizer)
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
