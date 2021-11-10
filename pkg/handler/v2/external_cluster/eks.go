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

package externalcluster

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go/service/eks"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/aws"
	awsprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/aws"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

func createEKSCluster(ctx context.Context, name string, userInfoGetter provider.UserInfoGetter, project *kubermaticapiv1.Project, cloud *apiv2.ExternalClusterCloudSpec, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider) (*kubermaticapiv1.ExternalCluster, error) {
	if cloud.EKS.Name == "" || cloud.EKS.Region == "" || cloud.EKS.AccessKeyID == "" || cloud.EKS.SecretAccessKey == "" {
		return nil, errors.NewBadRequest("the EKS cluster name, region or credentials can not be empty")
	}

	newCluster := genExternalCluster(name, project.Name)
	newCluster.Spec.CloudSpec = &kubermaticapiv1.ExternalClusterCloudSpec{
		EKS: &kubermaticapiv1.ExternalClusterEKSCloudSpec{
			Name:   cloud.EKS.Name,
			Region: cloud.EKS.Region,
		},
	}
	keyRef, err := clusterProvider.CreateOrUpdateCredentialSecretForCluster(ctx, cloud, project.Name, newCluster.Name)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	kuberneteshelper.AddFinalizer(newCluster, apiv1.CredentialsSecretsCleanupFinalizer)
	newCluster.Spec.CloudSpec.EKS.CredentialsReference = keyRef

	return createNewCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, newCluster, project)
}

func patchEKSCluster(ctx context.Context, old, new *apiv2.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticapiv1.ExternalClusterCloudSpec) (*string, error) {

	accessKeyID, secretAccessKey, err := aws.GetCredentialsForEKSCluster(*cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}

	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, cloudSpec.EKS.Region)
	if err != nil {
		return nil, err
	}

	newVersion := new.Spec.Version.Semver()
	newVersionString := strings.TrimSuffix(newVersion.String(), ".0")

	updateInput := eks.UpdateClusterVersionInput{
		Name:    &cloudSpec.EKS.Name,
		Version: &newVersionString,
	}
	updateOutput, err := client.EKS.UpdateClusterVersion(&updateInput)
	if err != nil {
		return nil, err
	}

	status := "Cluster Upgrade " + *updateOutput.Update.Status

	return &status, nil
}
