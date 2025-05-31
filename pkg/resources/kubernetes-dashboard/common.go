/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package kubernetesdashboard

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// CertUsername is the name of the user coming from kubeconfig cert.
	CertUsername = "kubermatic:kubernetes-dashboard"

	CSRFSecretName       = "kubernetes-dashboard-csrf"
	KubeconfigSecretName = "kubernetes-dashboard-kubeconfig"
	crsfKeyName          = "private.key"
)

var (
	defaultResourceRequirements = map[string]*corev1.ResourceRequirements{
		webContainerName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("32Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("128Mi"),
				corev1.ResourceCPU:    resource.MustParse("250m"),
			},
		},
		apiContainerName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("200Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("400Mi"),
				corev1.ResourceCPU:    resource.MustParse("250m"),
			},
		},
		authContainerName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("200Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("400Mi"),
				corev1.ResourceCPU:    resource.MustParse("250m"),
			},
		},
		kongContainerName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("200Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("400Mi"),
				corev1.ResourceCPU:    resource.MustParse("250m"),
			},
		},
	}
)

// kubernetesDashboardData is the data needed to construct the Kubernetes Dashboard components.
type kubernetesDashboardData interface {
	Cluster() *kubermaticv1.Cluster
	RewriteImage(string) (string, error)
}

func CSRFSecretReconciler() reconciling.NamedSecretReconcilerFactory {
	return func() (name string, reconciler reconciling.SecretReconciler) {
		return CSRFSecretName, func(secret *corev1.Secret) (*corev1.Secret, error) {
			if secret.Data == nil {
				secret.Data = map[string][]byte{}
			}

			if _, ok := secret.Data[crsfKeyName]; !ok {
				newKey := make([]byte, 256)

				if _, err := rand.Read(newKey); err != nil {
					return nil, fmt.Errorf("failed to read enough random bytes for the CSRF key: %w", err)
				}

				// base64 encode because the key is used as an environment variable and those
				// cannot contain special characters; additionally the dashboard explicitly
				// decodes the env variable to get to the raw key bytes.
				secret.Data[crsfKeyName] = []byte(base64.StdEncoding.EncodeToString(newKey))
			}

			return secret, nil
		}
	}
}

type internalKubeconfigReconcilerData interface {
	GetRootCA() (*triple.KeyPair, error)
	Cluster() *kubermaticv1.Cluster
}

func KubeconfigReconciler(namespace string, data internalKubeconfigReconcilerData, log *zap.SugaredLogger) reconciling.NamedSecretReconcilerFactory {
	return resources.GetInternalKubeconfigReconciler(namespace, KubeconfigSecretName, CertUsername, nil, data, log)
}

func HealthStatus(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, versions kubermatic.Versions) (kubermaticv1.HealthStatus, error) {
	// The Kubernetes dashboard consists of multiple Deployments, which all need to be healthy for us to
	// consider the dashboard as a whole healthy.
	deployments := []string{
		webDeploymentName,
		apiDeploymentName,
		authDeploymentName,
		kongDeploymentName,
	}

	var worstStatus *kubermaticv1.HealthStatus

	for _, deployment := range deployments {
		status, err := resources.HealthyDeployment(ctx, client, types.NamespacedName{Name: deployment, Namespace: cluster.Status.NamespaceName}, 1)
		if err != nil {
			return kubermaticv1.HealthStatusDown, fmt.Errorf("failed to determine health: %w", err)
		}
		status = util.GetHealthStatus(status, cluster, versions)

		// only remember the smallest (worst) common status among the deployments
		worstStatus = worseStatus(worstStatus, &status)
	}

	return *worstStatus, nil
}

var healthValues = map[kubermaticv1.HealthStatus]int{
	kubermaticv1.HealthStatusDown:         0,
	kubermaticv1.HealthStatusProvisioning: 1,
	kubermaticv1.HealthStatusUp:           2,
}

func worseStatus(prev, current *kubermaticv1.HealthStatus) *kubermaticv1.HealthStatus {
	if prev == nil {
		return current
	}

	scorePrev := healthValues[*prev]
	scoreCurrent := healthValues[*current]

	if scorePrev < scoreCurrent {
		return prev
	}

	return current
}
