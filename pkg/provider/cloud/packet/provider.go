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

package packet

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/packethost/packngo"

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	defaultBillingCycle = "hourly"
)

type packet struct {
	secretKeySelector provider.SecretKeySelectorValueFunc
}

// NewCloudProvider creates a new packet provider.
func NewCloudProvider(secretKeyGetter provider.SecretKeySelectorValueFunc) provider.CloudProvider {
	return &packet{
		secretKeySelector: secretKeyGetter,
	}
}

var _ provider.CloudProvider = &packet{}

// DefaultCloudSpec adds defaults to the CloudSpec.
func (p *packet) DefaultCloudSpec(_ context.Context, _ *kubermaticv1.ClusterSpec) error {
	return nil
}

// ValidateCloudSpec validates the given CloudSpec.
func (p *packet) ValidateCloudSpec(_ context.Context, spec kubermaticv1.CloudSpec) error {
	_, _, err := GetCredentialsForCluster(spec, p.secretKeySelector)
	return err
}

// InitializeCloudProvider initializes a cluster, in particular
// updates BillingCycle to the defaultBillingCycle, if it is not set.
func (p *packet) InitializeCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	var err error
	if cluster.Spec.Cloud.Packet.BillingCycle == "" {
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.Packet.BillingCycle = defaultBillingCycle
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

func (p *packet) ReconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return p.InitializeCloudProvider(ctx, cluster, update)
}

// CleanUpCloudProvider.
func (p *packet) CleanUpCloudProvider(_ context.Context, cluster *kubermaticv1.Cluster, _ provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted.
func (p *packet) ValidateCloudSpecUpdate(_ context.Context, _ kubermaticv1.CloudSpec, _ kubermaticv1.CloudSpec) error {
	return nil
}

func GetCredentialsForCluster(cloudSpec kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (apiKey, projectID string, err error) {
	apiKey = cloudSpec.Packet.APIKey
	projectID = cloudSpec.Packet.ProjectID

	if apiKey == "" {
		if cloudSpec.Packet.CredentialsReference == nil {
			return "", "", errors.New("no credentials provided")
		}
		apiKey, err = secretKeySelector(cloudSpec.Packet.CredentialsReference, resources.PacketAPIKey)
		if err != nil {
			return "", "", err
		}
	}

	if projectID == "" {
		if cloudSpec.Packet.CredentialsReference == nil {
			return "", "", errors.New("no credentials provided")
		}
		projectID, err = secretKeySelector(cloudSpec.Packet.CredentialsReference, resources.PacketProjectID)
		if err != nil {
			return "", "", err
		}
	}

	return apiKey, projectID, nil
}

func ValidateCredentials(apiKey, projectID string) error {
	client := packngo.NewClientWithAuth("kubermatic", apiKey, nil)
	_, _, err := client.Projects.Get(projectID, nil)
	return err
}

// Used to decode response object.
type plansRoot struct {
	Plans []packngo.Plan `json:"plans"`
}

func DescribeSize(apiKey, projectID, instanceType string) (*provider.NodeCapacity, error) {
	if len(apiKey) == 0 {
		return nil, fmt.Errorf("missing required parameter: apiKey")
	}

	if len(projectID) == 0 {
		return nil, fmt.Errorf("missing required parameter: projectID")
	}

	packetclient := packngo.NewClientWithAuth("kubermatic", apiKey, nil)
	req, err := packetclient.NewRequest(http.MethodGet, "/projects/"+projectID+"/plans", nil)
	if err != nil {
		return nil, err
	}

	root := new(plansRoot)
	_, err = packetclient.Do(req, root)
	if err != nil {
		return nil, err
	}

	plans := root.Plans
	for _, currentPlan := range plans {
		if currentPlan.Slug == instanceType {
			var totalCPUs int
			for _, cpu := range currentPlan.Specs.Cpus {
				totalCPUs += cpu.Count
			}

			var storageReq, memReq resource.Quantity
			for _, drive := range currentPlan.Specs.Drives {
				if drive.Size == "" || drive.Count == 0 {
					continue
				}

				// trimming "B" as quantities must match the regular expression '^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$'.
				storage, err := resource.ParseQuantity(strings.TrimSuffix(drive.Size, "B"))
				if err != nil {
					return nil, fmt.Errorf("failed to parse plan disk size %q: %w", drive.Size, err)
				}

				// total storage for each types = drive count *drive Size.
				strDrive := strconv.FormatInt(storage.Value()*int64(drive.Count), 10)
				totalStorage, err := resource.ParseQuantity(strDrive)
				if err != nil {
					return nil, fmt.Errorf("error parsing machine storage request to quantity: %w", err)
				}

				// Adding storage value for all storage types like "SSD", "NVME".
				storageReq.Add(totalStorage)
			}

			if currentPlan.Specs.Memory.Total != "" {
				memReq, err = resource.ParseQuantity(strings.TrimSuffix(currentPlan.Specs.Memory.Total, "B"))
				if err != nil {
					return nil, fmt.Errorf("error parsing machine memory request to quantity: %w", err)
				}
			}

			capacity := provider.NewNodeCapacity()
			capacity.WithCPUCount(totalCPUs)
			capacity.Memory = &memReq
			capacity.Storage = &storageReq

			return capacity, nil
		}
	}

	return nil, fmt.Errorf("instance type %q not found", instanceType)
}
