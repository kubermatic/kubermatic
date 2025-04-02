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
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/resources"
	encryptionresources "k8c.io/kubermatic/v2/pkg/resources/encryption"
	"k8c.io/reconciler/pkg/reconciling"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *Reconciler) encryptData(ctx context.Context, client ctrlruntimeclient.Client, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	key, resourceList, err := getActiveConfiguration(ctx, r.Client, cluster)
	if err != nil {
		return &reconcile.Result{}, err
	}

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
		encryptionresources.SecretRevisionLabelKey: secret.ResourceVersion,
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

		// generate a Job that will run re-encryption on both configured and previously encrypted resources.
		// That is done to make sure that previously encrypted resources get re-encrypted or decrypted even
		// if they vanished from the resource list in ClusterSpec.
		job := encryptionresources.EncryptionJobCreator(
			data,
			cluster,
			&secret,
			mergeSlice(resourceList, cluster.Status.Encryption.EncryptedResources),
			key,
		)

		if err := r.Create(ctx, &job); err != nil {
			return &reconcile.Result{}, err
		}

		// wait for Job to appear in cache
		waiter := reconciling.WaitUntilObjectExistsInCacheConditionFunc(r.Client, log, ctrlruntimeclient.ObjectKeyFromObject(&job), &job)
		if err := wait.PollUntilContextTimeout(ctx, 100*time.Millisecond, 10*time.Second, false, waiter); err != nil {
			return &reconcile.Result{}, fmt.Errorf("failed waiting for the Job to appear in the cache: %w", err)
		}

		// Jobs are watched and queued by the controller, so we now wait for the reconcile loop
		// that is triggered by the Job updating.
		return &reconcile.Result{}, nil
	} else {
		job := jobList.Items[0]
		if job.Status.Succeeded == 1 {
			if err := util.UpdateClusterStatus(ctx, r.Client, cluster, func(c *kubermaticv1.Cluster) {
				cluster.Status.Encryption.Phase = kubermaticv1.ClusterEncryptionPhaseActive
				cluster.Status.Encryption.EncryptedResources = resourceList
				cluster.Status.Encryption.ActiveKey = key
			}); err != nil {
				return &reconcile.Result{}, err
			}

			return &reconcile.Result{}, nil
		} else if job.Status.Failed > 0 {
			if err := util.UpdateClusterStatus(ctx, r.Client, cluster, func(c *kubermaticv1.Cluster) {
				cluster.Status.Encryption.Phase = kubermaticv1.ClusterEncryptionPhaseFailed
			}); err != nil {
				return &reconcile.Result{}, err
			}
			return &reconcile.Result{}, nil
		}

		// no job result yet, job status will update at a later point and trigger another loop
		return &reconcile.Result{}, nil
	}
}
