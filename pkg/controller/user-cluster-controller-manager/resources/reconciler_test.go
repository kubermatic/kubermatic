// +build integration

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

package resources

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"testing"

	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func init() {
	if err := apiextensionsv1beta1.AddToScheme(scheme.Scheme); err != nil {
		panic(fmt.Sprintf("failed to register apiextensionsv1beta1 with scheme: %v", err))
	}
	if err := apiregistrationv1beta1.AddToScheme(scheme.Scheme); err != nil {
		panic(fmt.Sprintf("failed to register apiregistrationv1beta1 with scheme: %v", err))
	}
}

func TestResourceReconciliationIdempotency(t *testing.T) {
	kubermaticlog.Logger = kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()
	env := &envtest.Environment{}
	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("failed to start testenv: %v", err)
	}
	defer func() {
		if err := env.Stop(); err != nil {
			t.Fatalf("failed to stop testenv: %v", err)
		}
	}()

	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		t.Fatalf("failed to construct manager: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if err := mgr.Start(ctx); err != nil {
			t.Errorf("failed to start manager: %v", err)
		}
	}()

	// Used during reconciliation and seems the apiserver we use
	// doesn't create the initial rolebings/is too slow
	basicUserClusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "system:basic-user",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "system:basic-user",
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Name:     "system:authenticated",
				Kind:     rbacv1.GroupKind,
			},
			{
				APIGroup: rbacv1.GroupName,
				Name:     "system:unauthenticated",
				Kind:     rbacv1.GroupKind,
			},
		},
	}
	if err := mgr.GetClient().Create(ctx, basicUserClusterRoleBinding); err != nil {
		if !kerrors.IsConflict(err) {
			t.Fatalf("failed to create the `system:basic-user` clusterrolebinding: %v", err)
		}
	}

	const seedNamespace = "cluster-namespace"
	caSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.CASecretName,
			Namespace: seedNamespace,
			// Used for encrypting the OAuthBootstrapSecret, so must be set
			UID: types.UID("567a5759-20ce-11ea-b48e-42010a9c0115"),
		},
		Data: getRSACAData(),
	}
	openVPNCASecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.OpenVPNCASecretName,
			Namespace: seedNamespace,
		},
		Data: getECDSACAData(),
	}
	sshKeySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.UserSSHKeys,
			Namespace: seedNamespace,
		},
	}
	cloudConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.CloudConfigConfigMapName,
			Namespace: seedNamespace,
		},
		Data: map[string]string{resources.CloudConfigConfigMapKey: "some-cloud-config-content"},
	}
	seedClient := fakectrlruntimeclient.NewFakeClient(
		caSecret,
		openVPNCASecret,
		sshKeySecret,
		cloudConfigMap,
	)
	r := reconciler{
		Client:     mgr.GetClient(),
		seedClient: seedClient,
		// Openshift means that we deploy some more resources, so enable it.
		openshift:       true,
		version:         "4.1.18",
		namespace:       seedNamespace,
		platform:        "aws",
		clusterURL:      &url.URL{},
		rLock:           &sync.Mutex{},
		log:             kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar(),
		userSSHKeyAgent: true,
	}

	if err := mgr.GetClient().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "openshift-infra",
		},
	}); err != nil {
		t.Fatalf("Failed to create openshift base namespace: %v", err)
	}

	if err := r.reconcile(ctx); err != nil {
		t.Fatalf("Initial resource deployment failed, this indicates that some resources are invalid. Error: %v", err)
	}

	if err := r.reconcile(ctx); err != nil {
		t.Fatalf("The second resource reconciliation failed, indicating we don't properly default some fields. Check the `Object differs from generated one` error for the object for which we timed out. Original error: %v", err)
	}

	// A very simplistic smoketest that we actually did something
	daemonSets := &appsv1.DaemonSetList{}
	if err := mgr.GetClient().List(ctx, daemonSets); err != nil {
		t.Fatalf("failed to list DaemonSets: %v", err)
	}
	if len(daemonSets.Items) == 0 {
		t.Error("expected to find at least one DaemonSet after reconciliation, but there was none")
	}
}

func getRSACAData() map[string][]byte {
	return map[string][]byte{
		"ca.crt": []byte(`
-----BEGIN CERTIFICATE-----
MIIDGjCCAgKgAwIBAgIBADANBgkqhkiG9w0BAQsFADA+MTwwOgYDVQQDEzNyb290
LWNhLjRtY3NmYjl2NXAuZXVyb3BlLXdlc3QzLWMuZGV2Lmt1YmVybWF0aWMuaW8w
HhcNMTkxMjAzMTEzMzM1WhcNMjkxMTMwMTEzMzM1WjA+MTwwOgYDVQQDEzNyb290
LWNhLjRtY3NmYjl2NXAuZXVyb3BlLXdlc3QzLWMuZGV2Lmt1YmVybWF0aWMuaW8w
ggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDGCoaNNVCSzqQRh+xh+I+R
bQbJoOEanMTiubHBNkL6mcpPRfacsRRzH4dOq3hhCb4KGqyN7tJ/yQ0bVqU+7/Lg
x6VFMr0a+qtVtNqZgAhicOItBRGgeaPsDlIIbITVWsHKchZcUgiUDKyXJh66hel9
1vqTZv4O116+GUkp+owm6jXH263p5o5OOlsvJCxJzdfB852xYw/FGlYF/JTY8yB2
UIa5xkIP2gs2rLzICznwQ08Ysr1WFBGC8z12lWaf74v8ygESNuDk1Xnyr2bSloa7
tCQe+HXazsy63laBMaJbtke+Bey7N+sUwuR4m+/xyL/2skoRbnIvu6FY/IrG4JEh
AgMBAAGjIzAhMA4GA1UdDwEB/wQEAwICpDAPBgNVHRMBAf8EBTADAQH/MA0GCSqG
SIb3DQEBCwUAA4IBAQC/IQXGMLI4fSZv3JhTUHif4ZTzebuUU596bcBmpgXCctdH
7SPfsXUtys4sOYXFgPWnTFaslmOek86ftylXb+zwLBTeD5th5twluPbjtBQrFTxO
Ws1l5SdZZgma+MoeLQuSaHYQHbcK0o0TBQ1kV8yH6xFcCqiBha9GNdSzKfsN6Y/G
KZ3CZK59Z6wbaABlNX0mGRPeDT5VF6rzA9Os1verpMNqC6aknsptQd41r3WKcTZh
ypAcZxLSiairF5PnDoS9Gbxrte11U3TKRShgMLrw3rAQAPw2gZgS8mnEzXHxDPVg
J4XmVajIvOz2hHzI5j8hMkrbDfXX40iERkvxUe2J
-----END CERTIFICATE-----
		`),
		"ca.key": []byte(`
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAxgqGjTVQks6kEYfsYfiPkW0GyaDhGpzE4rmxwTZC+pnKT0X2
nLEUcx+HTqt4YQm+Chqsje7Sf8kNG1alPu/y4MelRTK9GvqrVbTamYAIYnDiLQUR
oHmj7A5SCGyE1VrBynIWXFIIlAyslyYeuoXpfdb6k2b+DtdevhlJKfqMJuo1x9ut
6eaOTjpbLyQsSc3XwfOdsWMPxRpWBfyU2PMgdlCGucZCD9oLNqy8yAs58ENPGLK9
VhQRgvM9dpVmn++L/MoBEjbg5NV58q9m0paGu7QkHvh12s7Mut5WgTGiW7ZHvgXs
uzfrFMLkeJvv8ci/9rJKEW5yL7uhWPyKxuCRIQIDAQABAoIBAH1n1OQ+SpNsPwDK
7ajsWR1hyNaNBX5wn0xjAmizD57ZG/8u8ocuqyBraqcqdcQdAzYqxfHqtWktyxrw
txsvnsEwKzuycYVQDobrSzHAnY6YpOCVQSA5Zs/oEZI8BbGFEwo7TGWRnNUDYZcl
EHhUrBJ/u5TztxV21AvUvzvR6EYLEpETDvRP+MB7aRLSO0NVX1y3PJO/Flogffri
UQGpPzDK7MaYJ05Nv/JNElXXtEdKBcYM2aFyK5egL3f8TfRD6ZHNobHGtB8Yunos
fkSeiUzRhLKrCdcuQFErtEzJklqgI0n+VNS6rMTcYu+mnwQwLELXVEiaLg+mkDl5
0hj3K0ECgYEAyMewn8uSLvoJBKpxYqNaX4+Qd570QNc4XCyslKBaFa20P6B8pXlE
RxEWqGnCHpGGamCCZFcujq/ydCRSBrodHipebdQ0g73aW09lp+/ezPyXrBYnk5Pm
H707yfo3LqXl0sp6jp2L2lOMk3Jwc0uYXyNDysBOV0Y3swkGWKuE8k0CgYEA/IH/
Faw9iq31UcquHH+I/nBg+CieEnTV9W+asR04PO9JjveAjO3m+z6/h3UyHybJP7gl
4zWNXdm3STvidxBTMea1LtHh2wgO7HsfvyAgx1JT51DxFMwNkUjeNApu/vLFJN5p
saVDPPMyLCKcNYtGHSY3v+QY7ttGMhGe7IgdvCUCgYAizXpwOHk+a1jk1iaRUn93
0QrZsVPlaWj7kULZAHKdD18PKGJyzHJpXyzuRGbBpbgG+HOhsPsBAL6mIyWUxO4H
LJDxuLbhnycabMjSM2ILZj1kNvLlQd3B4qtad2TZUShlQXO9BNIZJiJX7J5RXekr
lJFLs8nglutQvV+8Pv7sgQKBgCFT3yEzLKRDNZ/oCncEdkZu+r2GTubMuPi+FELQ
Qn34b1MJY9Q2CkibDZiJZWYrLmDUo1uL0+7RL5ng55EjfHeXpc5aMV9BfwbDcXs+
eFUWjrB3RHqkPB4y6fEgd2n7DP5CxNyHnYpL5xFgOBHxIf3y72TKbGgKVQeCv+Ek
ThhpAoGBAJCCTh5if2Y9G0+JYsYwRPWSoXB74wuaEK4xullUNdJKcgDUkft3NNAB
mfhnQDDKk7KGgerhPnoF0DgiRHpAPBQmFuLFPVeA7pbdVquQERgqHxMmk7yoIvc+
7zjWmqk/FSSYn8qll1vbtD3O25yNax3tlpWBGh9rouGBJK5pLvh/
-----END RSA PRIVATE KEY-----
		`),
	}
}

func getECDSACAData() map[string][]byte {
	return map[string][]byte{
		"ca.crt": []byte(`
-----BEGIN CERTIFICATE-----
MIIBOzCB4qADAgECAhEAmAjNj5Q3VZ8PZAfV6YMPOTAKBggqhkjOPQQDAjANMQsw
CQYDVQQDEwJDQTAeFw0xOTEyMDMxMTMzMzVaFw0yOTExMzAxMTMzMzVaMA0xCzAJ
BgNVBAMTAkNBMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEYnsP4XV/3ADshz27
ew6EgAnGQ15fXZkSmsgNncyLW9nYr9ebjYdYkioYiJbweFChJufnIUqLtEEADcNG
KErSKaMjMCEwDgYDVR0PAQH/BAQDAgKkMA8GA1UdEwEB/wQFMAMBAf8wCgYIKoZI
zj0EAwIDSAAwRQIgdUW++x4CtDlpO7L5uGfzs9xv0R/PCnS6DlX+RNmt5LkCIQDI
NU25kYiwIUo8Q/DE3EoqIe04Q4KzhJPBN6hf4MUxrw==
-----END CERTIFICATE-----
		`),
		"ca.key": []byte(`
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIN/u9ZN9n91aN9SW/TXMmcck66kNs8a09XuI3PrsWqRpoAoGCCqGSM49
AwEHoUQDQgAEYnsP4XV/3ADshz27ew6EgAnGQ15fXZkSmsgNncyLW9nYr9ebjYdY
kioYiJbweFChJufnIUqLtEEADcNGKErSKQ==
-----END EC PRIVATE KEY-----
		`),
	}
}
