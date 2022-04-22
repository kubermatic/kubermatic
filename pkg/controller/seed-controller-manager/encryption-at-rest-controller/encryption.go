/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package encryptionatrestcontroller

import (
	"context"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/resources"
	encryptionresources "k8c.io/kubermatic/v2/pkg/resources/encryption"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *Reconciler) encryptData(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, key string) (*reconcile.Result, error) {
	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{
		Name:      resources.EncryptionConfigurationSecretName,
		Namespace: cluster.Status.NamespaceName,
	}, &secret); err != nil {
		return &reconcile.Result{}, err
	}

	var jobList batchv1.JobList

	if err := r.List(ctx, &jobList, ctrlruntimeclient.MatchingLabels{
		encryptionresources.ClusterLabelKey:        cluster.Name,
		encryptionresources.SecretRevisionLabelKey: secret.ObjectMeta.ResourceVersion,
	}); err != nil {
		return &reconcile.Result{}, err
	}

	if len(jobList.Items) == 0 {
		seed, err := r.seedGetter()
		if err != nil {
			return nil, err
		}
		config, err := r.configGetter(ctx)
		if err != nil {
			return nil, err
		}
		data, err := r.getClusterTemplateData(ctx, cluster, seed, config)
		if err != nil {
			return nil, err
		}

		job := encryptionresources.EncryptionJobCreator(data, cluster, &secret, key)

		if err := r.Create(ctx, &job); err != nil {
			return &reconcile.Result{}, err
		}

		// we just created the job and need to check in with it later
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	} else {
		job := jobList.Items[0]
		if job.Status.Succeeded == 1 {
			if err := kubermaticv1helper.UpdateClusterStatus(ctx, r.Client, cluster, func(c *kubermaticv1.Cluster) {
				cluster.Status.Encryption.Phase = kubermaticv1.ClusterEncryptionPhaseActive
				cluster.Status.Encryption.ActiveKey = key
			}); err != nil {
				return &reconcile.Result{}, err
			}

			return &reconcile.Result{}, nil
		} else if job.Status.Failed > 0 {
			if err := kubermaticv1helper.UpdateClusterStatus(ctx, r.Client, cluster, func(c *kubermaticv1.Cluster) {
				cluster.Status.Encryption.Phase = kubermaticv1.ClusterEncryptionPhaseFailed
			}); err != nil {
				return &reconcile.Result{}, err
			}
			return &reconcile.Result{}, nil
		}

		// no job result yet, requeue to read job status again in 10 seconds
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}
}
