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
	"strconv"
	"strings"

	"github.com/equinix/equinix-sdk-go/services/metalv1"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
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

func ValidateCredentials(ctx context.Context, apiKey, projectID string) error {
	client := getClient(apiKey)
	request := client.ProjectsApi.FindProjectById(ctx, projectID)

	_, response, err := client.ProjectsApi.FindProjectByIdExecute(request)
	defer response.Body.Close()

	return err
}

func getClient(apiKey string) *metalv1.APIClient {
	configuration := metalv1.NewConfiguration()
	configuration.UserAgent = fmt.Sprintf("kubermatic %s", configuration.UserAgent)
	configuration.AddDefaultHeader("X-Auth-Token", apiKey)

	return metalv1.NewAPIClient(configuration)
}

func parsePlanQuantity(s string) (resource.Quantity, error) {
	// trimming "B" as quantities must match the regular expression '^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$'.
	return resource.ParseQuantity(strings.TrimSuffix(s, "B"))
}

func DescribeSize(ctx context.Context, apiKey, projectID, instanceType string) (*provider.NodeCapacity, error) {
	if len(apiKey) == 0 {
		return nil, errors.New("missing required parameter: apiKey")
	}

	if len(projectID) == 0 {
		return nil, errors.New("missing required parameter: projectID")
	}

	client := getClient(apiKey)
	request := client.PlansApi.FindPlansByProject(ctx, projectID)

	plans, response, err := client.PlansApi.FindPlansByProjectExecute(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	for _, plan := range plans.Plans {
		if plan.Slug != nil && *plan.Slug == instanceType {
			var (
				totalCPUs          int
				storageReq, memReq resource.Quantity
			)

			if specs := plan.Specs; specs != nil {
				for _, cpu := range specs.Cpus {
					if cpu.Count != nil {
						totalCPUs += int(*cpu.Count)
					}
				}

				for _, drive := range specs.Drives {
					if drive.Size == nil || *drive.Size == "" || drive.Count == nil || *drive.Count == 0 {
						continue
					}

					storage, err := parsePlanQuantity(*drive.Size)
					if err != nil {
						return nil, fmt.Errorf("failed to parse plan disk size %q: %w", *drive.Size, err)
					}

					// total storage for each types = drive count * drive Size.
					strDrive := strconv.FormatInt(storage.Value()*int64(*drive.Count), 10)
					totalStorage, err := resource.ParseQuantity(strDrive)
					if err != nil {
						return nil, fmt.Errorf("error parsing plan storage request to quantity: %w", err)
					}

					// Adding storage value for all storage types like "SSD", "NVME".
					storageReq.Add(totalStorage)
				}

				if memory := specs.Memory; memory != nil && memory.Total != nil && *memory.Total != "" {
					memReq, err = parsePlanQuantity(*memory.Total)
					if err != nil {
						return nil, fmt.Errorf("error parsing plan memory request %q: %w", *memory.Total, err)
					}
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
