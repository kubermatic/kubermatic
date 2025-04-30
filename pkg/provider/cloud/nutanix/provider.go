/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package nutanix

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	nutanixv3 "github.com/nutanix-cloud-native/prism-go-client/v3"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/utils/ptr"
)

const (
	ClusterCategoryName = "KKPCluster"
	ProjectCategoryName = "KKPProject"
	categoryDescription = "automatically created by KKP"
	categoryValuePrefix = "kubernetes-"

	DefaultProject = "default"

	categoryCleanupFinalizer = "kubermatic.k8c.io/cleanup-nutanix-categories"
)

type Nutanix struct {
	dc                *kubermaticv1.DatacenterSpecNutanix
	log               *zap.SugaredLogger
	secretKeySelector provider.SecretKeySelectorValueFunc
}

var _ provider.ReconcilingCloudProvider = &Nutanix{}

func NewCloudProvider(dc *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc) (*Nutanix, error) {
	if dc.Spec.Nutanix == nil {
		return nil, errors.New("datacenter is not a Nutanix datacenter")
	}

	return &Nutanix{
		dc:                dc.Spec.Nutanix,
		log:               log.Logger,
		secretKeySelector: secretKeyGetter,
	}, nil
}

func (n *Nutanix) InitializeCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return n.reconcileCluster(ctx, cluster, update, false)
}

func (n *Nutanix) ReconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return n.reconcileCluster(ctx, cluster, update, true)
}

func (*Nutanix) ClusterNeedsReconciling(cluster *kubermaticv1.Cluster) bool {
	return false
}

func (n *Nutanix) CleanUpCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	if !kuberneteshelper.HasFinalizer(cluster, categoryCleanupFinalizer) {
		return cluster, nil
	}

	client, err := GetClientSet(n.dc, cluster.Spec.Cloud.Nutanix, n.secretKeySelector)
	if err != nil {
		return nil, err
	}

	logger := n.log.With("cluster", cluster.Name)
	logger.Info("removing category values")

	if err = deleteCategoryValues(ctx, client, cluster); err != nil {
		return nil, err
	}

	return update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		kuberneteshelper.RemoveFinalizer(cluster, categoryCleanupFinalizer)
	})
}

func (n *Nutanix) DefaultCloudSpec(_ context.Context, spec *kubermaticv1.ClusterSpec) error {
	// default csi
	if spec.Cloud.Nutanix.CSI != nil {
		if spec.Cloud.Nutanix.CSI.Port == nil {
			spec.Cloud.Nutanix.CSI.Port = ptr.To[int32](9440)
		}
	}

	return nil
}

func (n *Nutanix) ValidateCloudSpec(ctx context.Context, spec kubermaticv1.CloudSpec) error {
	if spec.Nutanix == nil {
		return errors.New("not a Nutanix spec")
	}

	client, err := GetClientSet(n.dc, spec.Nutanix, n.secretKeySelector)
	if err != nil {
		return err
	}

	if spec.Nutanix.ProjectName != "" {
		// check for project existence
		_, err = GetProjectByName(ctx, client, spec.Nutanix.ProjectName)
		if err != nil {
			return err
		}
	}

	// validate csi is set - required for new clusters
	if spec.Nutanix.CSI == nil {
		return errors.New("CSI not configured")
	}

	return nil
}

func (n *Nutanix) ValidateCloudSpecUpdate(_ context.Context, oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	if oldSpec.Nutanix == nil || newSpec.Nutanix == nil {
		return errors.New("'nutanix' spec is empty")
	}

	if oldSpec.Nutanix.ClusterName != newSpec.Nutanix.ClusterName {
		return fmt.Errorf("updating Nutanix cluster name is not supported (was %s, updated to %s)", oldSpec.Nutanix.ClusterName, newSpec.Nutanix.ClusterName)
	}

	if oldSpec.Nutanix.ProjectName != newSpec.Nutanix.ProjectName {
		return fmt.Errorf("updating Nutanix project name is not supported (was %s, updated to %s)", oldSpec.Nutanix.ProjectName, newSpec.Nutanix.ProjectName)
	}

	return nil
}

func (n *Nutanix) reconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, force bool) (*kubermaticv1.Cluster, error) {
	logger := n.log.With("cluster", cluster.Name)

	client, err := GetClientSet(n.dc, cluster.Spec.Cloud.Nutanix, n.secretKeySelector)
	if err != nil {
		return nil, err
	}

	logger.Info("reconciling category and value")
	if err := reconcileCategoryAndValue(ctx, client, cluster); err != nil {
		return nil, fmt.Errorf("failed to reconcile category and cluster value: %w", err)
	}

	cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		kuberneteshelper.AddFinalizer(cluster, categoryCleanupFinalizer)
	})

	if err != nil {
		return nil, err
	}

	return cluster, nil
}

func deleteCategoryValues(ctx context.Context, client *ClientSet, cluster *kubermaticv1.Cluster) error {
	projectID, ok := cluster.Labels[kubermaticv1.ProjectIDLabelKey]
	if ok {
		_, err := client.Prism.V3.GetCategoryValue(ctx, ProjectCategoryName, projectID)
		if err != nil {
			nutanixError, parseErr := ParseNutanixError(err)

			// failed to parse nutanix error? likely auth issues
			if parseErr != nil {
				return parseErr
			}

			if nutanixError.Code != http.StatusNotFound {
				return err
			}
		} else {
			// we need to make sure that no resources are attached
			// to the project category before trying to delete it.
			query, err := client.Prism.V3.GetCategoryQuery(ctx, &nutanixv3.CategoryQueryInput{
				UsageType:        ptr.To("APPLIED_TO"),
				GroupMemberCount: ptr.To[int64](0),
				CategoryFilter: &nutanixv3.CategoryFilter{
					Type:     ptr.To("CATEGORIES_MATCH_ANY"),
					KindList: []*string{},
					Params: map[string][]string{
						ProjectCategoryName: {projectID},
					},
				},
			})

			if err != nil {
				return err
			}

			// no results mean it is safe to clean up this category value.
			if len(query.Results) == 0 {
				if err = client.Prism.V3.DeleteCategoryValue(ctx, ProjectCategoryName, projectID); err != nil {
					return err
				}
			} else {
				// we will loop over the results and see if any of them match our category for the cluster.
				// if we hit a matching resource, an error is returned, as there is some leftover resource.
				// other entities not linked to our cluster are fine and can be ignored.
				for _, result := range query.Results {
					if result != nil {
						for _, entity := range result.EntityAnyReferenceList {
							if entity != nil {
								if val, ok := entity.Categories[ClusterCategoryName]; ok && val == CategoryValue(cluster.Name) {
									return fmt.Errorf("entities associated with category '%s=%s' still exist", ClusterCategoryName, CategoryValue(cluster.Name))
								}
							}
						}
					}
				}
			}
		}
	}

	_, err := client.Prism.V3.GetCategoryValue(ctx, ClusterCategoryName, CategoryValue(cluster.Name))
	if err != nil {
		nutanixError, parseErr := ParseNutanixError(err)

		// failed to parse nutanix error? likely auth issues
		if parseErr != nil {
			return parseErr
		}

		if nutanixError.Code != http.StatusNotFound {
			return err
		}
	} else if err = client.Prism.V3.DeleteCategoryValue(ctx, ClusterCategoryName, CategoryValue(cluster.Name)); err != nil {
		return err
	}

	return nil
}

func reconcileCategoryAndValue(ctx context.Context, client *ClientSet, cluster *kubermaticv1.Cluster) error {
	// check if category (key) is present, create it if not
	_, err := client.Prism.V3.GetCategoryKey(ctx, ClusterCategoryName)
	if err != nil {
		nutanixError, err := ParseNutanixError(err)

		// failed to parse nutanix error? likely auth issues
		if err != nil {
			return err
		}

		if nutanixError.Code == http.StatusNotFound {
			_, err := client.Prism.V3.CreateOrUpdateCategoryKey(ctx, &nutanixv3.CategoryKey{
				Name:        ptr.To(ClusterCategoryName),
				Description: ptr.To(categoryDescription),
			})

			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// check if category value is present, create it if not
	_, err = client.Prism.V3.GetCategoryValue(ctx, ClusterCategoryName, CategoryValue(cluster.Name))
	if err != nil {
		nutanixError, err := ParseNutanixError(err)

		// failed to parse nutanix error? likely auth issues
		if err != nil {
			return err
		}

		if nutanixError.Code == http.StatusNotFound {
			_, err := client.Prism.V3.CreateOrUpdateCategoryValue(ctx, ClusterCategoryName, &nutanixv3.CategoryValue{
				Value:       ptr.To(CategoryValue(cluster.Name)),
				Description: ptr.To(fmt.Sprintf("value for Kubernetes cluster %s", cluster.Name)),
			})
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	projectID, ok := cluster.Labels[kubermaticv1.ProjectIDLabelKey]
	if ok {
		_, err = client.Prism.V3.GetCategoryKey(ctx, ProjectCategoryName)
		if err != nil {
			nutanixError, err := ParseNutanixError(err)

			// failed to parse nutanix error? likely auth issues
			if err != nil {
				return err
			}

			if nutanixError.Code == http.StatusNotFound {
				_, err := client.Prism.V3.CreateOrUpdateCategoryKey(ctx, &nutanixv3.CategoryKey{
					Name:        ptr.To(ProjectCategoryName),
					Description: ptr.To(categoryDescription),
				})

				if err != nil {
					return err
				}
			} else {
				return err
			}
		}

		_, err = client.Prism.V3.GetCategoryValue(ctx, ProjectCategoryName, projectID)
		if err != nil {
			nutanixError, err := ParseNutanixError(err)

			// failed to parse nutanix error? likely auth issues
			if err != nil {
				return err
			}

			if nutanixError.Code == http.StatusNotFound {
				_, err := client.Prism.V3.CreateOrUpdateCategoryValue(ctx, ProjectCategoryName, &nutanixv3.CategoryValue{
					Value:       ptr.To(projectID),
					Description: ptr.To(fmt.Sprintf("value for KKP project %s", projectID)),
				})
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	return nil
}

func CategoryValue(clusterName string) string {
	return categoryValuePrefix + clusterName
}

func ValidateCredentials(ctx context.Context, endpoint string, port *int32, allowInsecure *bool, proxyURL, username, password string) error {
	cli, err := GetClientSetWithCreds(endpoint, port, allowInsecure, proxyURL, username, password)
	if err != nil {
		return err
	}
	_, err = cli.Prism.V3.ListAllImage(ctx, "")

	return err
}
