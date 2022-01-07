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
	"errors"
	"fmt"
	"strings"

	nutanixv3 "github.com/embik/nutanix-client-go/pkg/client/v3"
	"go.uber.org/zap"
	"k8s.io/utils/pointer"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
)

const (
	ClusterCategoryName = "KKPCluster"
	ProjectCategoryName = "KKPProject"
	categoryDescription = "automatically created by KKP"
	categoryValuePrefix = "kubernetes-"

	clusterKind = "cluster"
	projectKind = "project"

	entityNotFoundError = "ENTITY_NOT_FOUND"
)

type Nutanix struct {
	dc                *kubermaticv1.DatacenterSpecNutanix
	log               *zap.SugaredLogger
	secretKeySelector provider.SecretKeySelectorValueFunc
}

var _ provider.CloudProvider = &Nutanix{}

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

func (n *Nutanix) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return n.reconcileCluster(cluster, update, false)
}

func (n *Nutanix) ReconcileCluster(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return n.reconcileCluster(cluster, update, true)
}

func (n *Nutanix) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return nil, nil
}

func (n *Nutanix) DefaultCloudSpec(spec *kubermaticv1.CloudSpec) error {
	return nil
}

func (n *Nutanix) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
	if spec.Nutanix == nil {
		return errors.New("not a Nutanix spec")
	}

	client, err := GetClientSet(n.dc, spec.Nutanix, n.secretKeySelector)
	if err != nil {
		return err
	}

	if spec.Nutanix.ProjectName != "" {
		// check for project existence
		_, err = getProjectByName(client, spec.Nutanix.ProjectName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *Nutanix) ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}

func (n *Nutanix) reconcileCluster(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, force bool) (*kubermaticv1.Cluster, error) {
	logger := n.log.With("cluster", cluster.Name)

	client, err := GetClientSet(n.dc, cluster.Spec.Cloud.Nutanix, n.secretKeySelector)
	if err != nil {
		return nil, err
	}

	logger.Infow("reconciling category and value")
	if err := reconcileCategoryAndValue(client, cluster); err != nil {
		return nil, fmt.Errorf("failed to reconcile category and cluster value: %v", err)
	}

	return cluster, nil
}

func reconcileCategoryAndValue(client *ClientSet, cluster *kubermaticv1.Cluster) error {
	// check if category (key) is present, create it if not
	_, err := client.Prism.V3.GetCategoryKey(ClusterCategoryName)
	if err != nil {
		if strings.Contains(err.Error(), entityNotFoundError) {
			_, err := client.Prism.V3.CreateOrUpdateCategoryKey(&nutanixv3.CategoryKey{
				Name:        pointer.String(ClusterCategoryName),
				Description: pointer.String(categoryDescription),
			})

			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// check if category value is present, create it if not
	_, err = client.Prism.V3.GetCategoryValue(ClusterCategoryName, CategoryValue(cluster.Name))
	if err != nil {
		if strings.Contains(err.Error(), entityNotFoundError) {
			_, err := client.Prism.V3.CreateOrUpdateCategoryValue(ClusterCategoryName, &nutanixv3.CategoryValue{
				Value:       pointer.String(CategoryValue(cluster.Name)),
				Description: pointer.String(fmt.Sprintf("value for Kubernetes cluster %s", cluster.Name)),
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

		_, err = client.Prism.V3.GetCategoryKey(ProjectCategoryName)
		if err != nil {
			if strings.Contains(err.Error(), entityNotFoundError) {
				_, err := client.Prism.V3.CreateOrUpdateCategoryKey(&nutanixv3.CategoryKey{
					Name:        pointer.String(ProjectCategoryName),
					Description: pointer.String(categoryDescription),
				})

				if err != nil {
					return err
				}
			} else {
				return err
			}
		}

		_, err = client.Prism.V3.GetCategoryValue(ProjectCategoryName, projectID)
		if err != nil {
			if strings.Contains(err.Error(), entityNotFoundError) {
				_, err := client.Prism.V3.CreateOrUpdateCategoryValue(ClusterCategoryName, &nutanixv3.CategoryValue{
					Value:       pointer.String(projectID),
					Description: pointer.String(fmt.Sprintf("value for KKP project %s", projectID)),
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
