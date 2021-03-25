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

package kubermaticmaster

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	v1 "github.com/jetstack/cert-manager/pkg/apis/meta/v1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func waitForCertManagerWebhook(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client) error {
	logger.Debug("Waiting for webhook to become ready…")

	certName := "kubermatic-installer-test"

	// delete any leftovers from previous installer runs
	if err := deleteCertificate(ctx, kubeClient, CertManagerNamespace, certName); err != nil {
		return fmt.Errorf("failed to prepare webhook: %v", err)
	}

	// always clean up on a best-effort basis
	defer func() {
		_ = deleteCertificate(ctx, kubeClient, CertManagerNamespace, certName)
	}()

	// create a dummy cert to see if the webhook is alive and well
	dummyCert := &certmanagerv1alpha2.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certName,
			Namespace: CertManagerNamespace,
		},
		Spec: certmanagerv1alpha2.CertificateSpec{
			SecretName: certName,
			DNSNames:   []string{"www.example.com"},
			IssuerRef: v1.ObjectReference{
				Name: "dummy-issuer", // does not have to actually exist
			},
		},
	}

	var lastCreateErr error
	err := wait.PollImmediate(1*time.Second, 1*time.Minute, func() (bool, error) {
		lastCreateErr = kubeClient.Create(ctx, dummyCert)
		return lastCreateErr == nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for webhook to become ready: %v", lastCreateErr)
	}

	return nil
}

func deleteCertificate(ctx context.Context, kubeClient ctrlruntimeclient.Client, namespace string, name string) error {
	cert := &certmanagerv1alpha2.Certificate{}
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	if err := kubeClient.Get(ctx, key, cert); err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("failed to probe for leftover test certificate: %v", err)
	}

	if err := kubeClient.Delete(ctx, cert); err != nil {
		return fmt.Errorf("failed to delete test certificate: %v", err)
	}

	return nil
}
