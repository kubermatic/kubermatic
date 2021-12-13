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
	"time"

	nutanixv3 "github.com/terraform-providers/terraform-provider-nutanix/client/v3"
	"go.uber.org/zap"
	"k8s.io/utils/pointer"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
)

const (
	categoryName        = "kkp-cluster"
	categoryDescription = "automatically created by KKP"
	categoryValuePrefix = "kubernetes-"
	subnetNamePrefix    = "kubernetes-"

	subnetKind  = "subnet"
	clusterKind = "cluster"
	projectKind = "project"

	entityNotFoundError = "ENTITY_NOT_FOUND"

	FinalizerSubnet = "kubermatic.io/cleanup-nutanix-subnet"
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

	// check for project existence
	_, err = getProjectByName(client, spec.Nutanix.ProjectName)
	if err != nil {
		return err
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

	logger.Infow("reconciling category and cluster value")
	err = reconcileCategoryAndValue(client, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to reconcile category and cluster value: %v", err)
	}

	if force || cluster.Spec.Cloud.Nutanix.SubnetName == "" {
		logger.Infow("reconciling subnet", "subnet", cluster.Spec.Cloud.Nutanix.SubnetName)
		cluster, err = n.reconcileSubnet(client, cluster, update)
		if err != nil {
			return nil, fmt.Errorf("failed to reconcile subnet: %v", err)
		}
	}

	return cluster, nil
}

func reconcileCategoryAndValue(client *ClientSet, cluster *kubermaticv1.Cluster) error {
	// check if category (key) is present, create it if not
	_, err := client.Prism.V3.GetCategoryKey(categoryName)
	if err != nil {
		if strings.Contains(err.Error(), entityNotFoundError) {
			_, err := client.Prism.V3.CreateOrUpdateCategoryKey(&nutanixv3.CategoryKey{
				Name:        pointer.String(categoryName),
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
	_, err = client.Prism.V3.GetCategoryValue(categoryName, categoryValue(cluster.Name))
	if err != nil {
		if strings.Contains(err.Error(), entityNotFoundError) {
			_, err := client.Prism.V3.CreateOrUpdateCategoryValue(categoryName, &nutanixv3.CategoryValue{
				Value:       pointer.String(categoryValue(cluster.Name)),
				Description: pointer.String(fmt.Sprintf("value for Kubernetes cluster %s", cluster.Name)),
			})
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func (n *Nutanix) reconcileSubnet(client *ClientSet, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	if cluster.Spec.Cloud.Nutanix.SubnetName != "" {
		_, err := getSubnetByName(client, subnetName(cluster.Name))

		// subnet exists, we can return early
		// TODO: check status
		if err == nil {
			return cluster, nil
		}

		// if there's an error different from "not found", we should return that error
		if err != nil && !strings.Contains(err.Error(), entityNotFoundError) {
			return nil, err
		}
	}

	dcCluster, err := getClusterByName(client, n.dc.ClusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster: %v", err)
	}

	project, err := getProjectByName(client, cluster.Spec.Cloud.Nutanix.ProjectName)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %v", err)
	}

	subnetInput := &nutanixv3.SubnetIntentInput{
		Metadata: &nutanixv3.Metadata{
			Kind: pointer.String(subnetKind),
			ProjectReference: &nutanixv3.Reference{
				Kind: pointer.String(projectKind),
				UUID: project.Metadata.UUID,
			},
			Categories: map[string]string{
				categoryName: categoryValue(cluster.Name),
			},
			Name: pointer.String(subnetName(cluster.Name)),
		},
		Spec: &nutanixv3.Subnet{
			ClusterReference: &nutanixv3.Reference{
				Kind: pointer.String(clusterKind),
				UUID: dcCluster.Metadata.UUID,
			},
			Resources: &nutanixv3.SubnetResources{
				IPConfig: &nutanixv3.IPConfig{
					SubnetIP:     pointer.String("192.168.1.0"),
					PrefixLength: pointer.Int64(24),
				},
				SubnetType: pointer.String("VLAN"),
			},
		},
	}

	if len(n.dc.DNSServers) > 0 {
		dnsServers := []*string{}
		for _, server := range n.dc.DNSServers {
			dnsServers = append(dnsServers, pointer.String(server))
		}

		if subnetInput.Spec.Resources.IPConfig.DHCPOptions == nil {
			subnetInput.Spec.Resources.IPConfig.DHCPOptions = &nutanixv3.DHCPOptions{}
		}

		subnetInput.Spec.Resources.IPConfig.DHCPOptions.DomainNameServerList = dnsServers
	}

	resp, err := client.Prism.V3.CreateSubnet(subnetInput)
	if err != nil {
		return nil, err
	}

	cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		updatedCluster.Spec.Cloud.Nutanix.SubnetName = *resp.Metadata.Name
		kuberneteshelper.AddFinalizer(updatedCluster, FinalizerSubnet)
	})

	taskID := resp.Status.ExecutionContext.TaskUUID.(string)
	if err = waitForCompletion(client, taskID, 10*time.Second, 10*time.Minute); err != nil {
		return cluster, err
	}

	return cluster, nil
}

func categoryValue(clusterName string) string {
	return categoryValuePrefix + clusterName
}

func subnetName(clusterName string) string {
	return subnetNamePrefix + clusterName
}
