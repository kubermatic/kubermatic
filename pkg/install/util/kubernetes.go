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

package util

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func EnsureNamespace(ctx context.Context, log logrus.FieldLogger, kubeClient ctrlruntimeclient.Client, namespace string) error {
	log.WithField("namespace", namespace).Debug("Ensuring namespace…")

	err := kubeClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})

	return ctrlruntimeclient.IgnoreAlreadyExists(err)
}

func EnsureNamespaceLabel(ctx context.Context, kubeClient ctrlruntimeclient.Client, namespace, key, value string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ns := &corev1.Namespace{}

		if err := wait.PollUntilContextTimeout(ctx, 250*time.Millisecond, 15*time.Second, true, func(ctx context.Context) (bool, error) {
			if err := kubeClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: namespace}, ns); err != nil {
				if apierrors.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}
			return true, nil
		}); err != nil {
			return err
		}

		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}

		if ns.Labels[key] == value {
			return nil
		}

		ns.Labels[key] = value

		return kubeClient.Update(ctx, ns)
	})
}
