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

package node

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/label"
	commonv2 "k8c.io/kubermatic/v2/pkg/handler/v2/common"
	machineconversions "k8c.io/kubermatic/v2/pkg/machine"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	machineresource "k8c.io/kubermatic/v2/pkg/resources/machine"
	k8cerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

// createMachineDeploymentReq defines HTTP request for createMachineDeployment
// swagger:parameters createMachineDeployment
type createMachineDeploymentReq struct {
	commonv2.GetClusterReq
	// in: body
	Body apiv1.MachineDeployment
}

// GetSeedCluster returns the SeedCluster object
func (req createMachineDeploymentReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func DecodeCreateMachineDeployment(c context.Context, r *http.Request) (interface{}, error) {
	var req createMachineDeploymentReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	if err = json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func CreateMachineDeployment(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createMachineDeploymentReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}

		isBYO, err := common.IsBringYourOwnProvider(cluster.Spec.Cloud)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if isBYO {
			return nil, k8cerrors.NewBadRequest("You cannot create a node deployment for KubeAdm provider")
		}

		keys, err := sshKeyProvider.List(project, &provider.SSHKeyListOptions{ClusterName: req.ClusterID})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, project.Name)
		if err != nil {
			return nil, err
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		_, dc, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, cluster.Spec.Cloud.DatacenterName)
		if err != nil {
			return nil, fmt.Errorf("error getting dc: %v", err)
		}

		nd, err := machineresource.Validate(&req.Body, cluster.Spec.Version.Semver())
		if err != nil {
			return nil, k8cerrors.NewBadRequest(fmt.Sprintf("machine deployment validation failed: %s", err.Error()))
		}

		assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
		if !ok {
			return nil, k8cerrors.New(http.StatusInternalServerError, "clusterprovider is not a kubernetesprovider.Clusterprovider, can not create secret")
		}

		data := common.CredentialsData{
			Ctx:               ctx,
			KubermaticCluster: cluster,
			Client:            assertedClusterProvider.GetSeedClusterAdminRuntimeClient(),
		}

		md, err := machineresource.Deployment(cluster, nd, dc, keys, data)
		if err != nil {
			return nil, fmt.Errorf("failed to create machine deployment from template: %v", err)
		}

		if err := client.Create(ctx, md); err != nil {
			return nil, fmt.Errorf("failed to create machine deployment: %v", err)
		}

		return outputMachineDeployment(md)
	}
}

func outputMachineDeployment(md *clusterv1alpha1.MachineDeployment) (*apiv1.MachineDeployment, error) {
	nodeStatus := apiv1.NodeStatus{}
	nodeStatus.MachineName = md.Name

	var deletionTimestamp *apiv1.Time
	if md.DeletionTimestamp != nil {
		dt := apiv1.NewTime(md.DeletionTimestamp.Time)
		deletionTimestamp = &dt
	}

	operatingSystemSpec, err := machineconversions.GetAPIV1OperatingSystemSpec(md.Spec.Template.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to get operating system spec from machine deployment: %v", err)
	}

	cloudSpec, err := machineconversions.GetAPIV2NodeCloudSpec(md.Spec.Template.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to get node cloud spec from machine deployment: %v", err)
	}

	taints := make([]apiv1.TaintSpec, len(md.Spec.Template.Spec.Taints))
	for i, taint := range md.Spec.Template.Spec.Taints {
		taints[i] = apiv1.TaintSpec{
			Effect: string(taint.Effect),
			Key:    taint.Key,
			Value:  taint.Value,
		}
	}

	hasDynamicConfig := md.Spec.Template.Spec.ConfigSource != nil

	return &apiv1.MachineDeployment{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                md.Name,
			Name:              md.Name,
			DeletionTimestamp: deletionTimestamp,
			CreationTimestamp: apiv1.NewTime(md.CreationTimestamp.Time),
		},
		Spec: apiv1.MachineDeploymentSpec{
			Replicas: *md.Spec.Replicas,
			Template: apiv1.MachineSpec{
				Labels: label.FilterLabels(label.NodeDeploymentResourceType, md.Spec.Template.Spec.Labels),
				Taints: taints,
				Versions: apiv1.MachineVersionInfo{
					Kubelet: md.Spec.Template.Spec.Versions.Kubelet,
				},
				OperatingSystem: *operatingSystemSpec,
				Cloud:           *cloudSpec,
			},
			Paused:        &md.Spec.Paused,
			DynamicConfig: &hasDynamicConfig,
		},
		Status: md.Status,
	}, nil
}
