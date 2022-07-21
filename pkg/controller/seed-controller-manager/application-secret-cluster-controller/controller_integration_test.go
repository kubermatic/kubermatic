//go:build integration

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
package applicationsecretclustercontroller

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	applicationsecretsynchronizer "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/application-secret-synchronizer"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	timeout  = time.Second * 10
	interval = time.Second * 1
)

var _ = Describe("kkp-application-secret-cluster-controller", func() {
	Context("when an application secrert is created", func() {
		var secret *corev1.Secret

		BeforeEach(func() {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "app-cred",
					Annotations:  map[string]string{applicationsecretsynchronizer.SecretTypeAnnotation: ""},
					Namespace:    kubermaticNS.Name,
				},
				Data: map[string][]byte{"pass": []byte("a3ViZXJtYXRpYwo=")},
			}

			Expect(client.Create(ctx, secret)).To(Succeed())
		})
		It("should create secret in cluster namespace", func() {
			expectSecretSync(clusterWithWorkerName.Status.NamespaceName, secret)
			expectSecretHasFinalizer(secret)
		})
		It("should not sync secret with paused cluster", func() {
			expectSecretNevertExist(pauseClusterWithWorkerName.Status.NamespaceName, secret.Name)
		})
		It("should not sync secret for cluster with workername different from controller", func() {
			expectSecretNevertExist(clusterWithoutWorkerName.Status.NamespaceName, secret.Name)
		})
	})

	Context("when an application secrert is updated", func() {
		var secret *corev1.Secret
		BeforeEach(func() {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "app-cred",
					Annotations:  map[string]string{applicationsecretsynchronizer.SecretTypeAnnotation: ""},
					Labels:       map[string]string{"foo": "bar"},
					Namespace:    kubermaticNS.Name,
				},
				Data: map[string][]byte{"pass": []byte("a3ViZXJtYXRpYwo=")},
			}

			Expect(client.Create(ctx, secret)).To(Succeed())

			By("wait for secret to sync for the first time")
			expectSecretSync(clusterWithWorkerName.Status.NamespaceName, secret)
			expectSecretNevertExist(pauseClusterWithWorkerName.Status.NamespaceName, secret.Name)

			By("updating secret")
			original := secret.DeepCopy()
			secret.Data = map[string][]byte{"pass": []byte("bG9vZHNlCg==")}
			secret.Labels["new"] = "val"
			Expect(client.Patch(ctx, secret, ctrlruntimeclient.MergeFrom(original))).To(Succeed())
		})
		It("should update secret in cluster namespace", func() {
			expectSecretSync(clusterWithWorkerName.Status.NamespaceName, secret)
			expectSecretHasFinalizer(secret)
		})
		It("should not sync secret with paused cluster", func() {
			expectSecretNevertExist(pauseClusterWithWorkerName.Status.NamespaceName, secret.Name)
		})
		It("should not sync secret cluster with workername different from controller", func() {
			expectSecretNevertExist(clusterWithoutWorkerName.Status.NamespaceName, secret.Name)
		})
	})

	Context("when an application secrert is deleted", func() {
		var secret *corev1.Secret
		BeforeEach(func() {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "app-cred",
					Annotations:  map[string]string{applicationsecretsynchronizer.SecretTypeAnnotation: ""},
					Labels:       map[string]string{"foo": "bar"},
					Namespace:    kubermaticNS.Name,
				},
				Data: map[string][]byte{"pass": []byte("a3ViZXJtYXRpYwo=")},
			}

			Expect(client.Create(ctx, secret)).To(Succeed())

			By("wait for secret to sync for the first time")
			expectSecretSync(clusterWithWorkerName.Status.NamespaceName, secret)
			expectSecretNevertExist(pauseClusterWithWorkerName.Status.NamespaceName, secret.Name)

			By("deleting Secret")
			Expect(client.Delete(ctx, secret)).To(Succeed())
		})
		It("deleted secret from cluster namespace", func() {
			expectSecretIsDeleted(clusterWithWorkerName.Status.NamespaceName, secret.Name)
		})
		It("should not sync secret with paused cluster", func() {
			expectSecretNevertExist(pauseClusterWithWorkerName.Status.NamespaceName, secret.Name)
		})
		It("should not sync secret with cluster having different workername from controller", func() {
			expectSecretNevertExist(clusterWithoutWorkerName.Status.NamespaceName, secret.Name)
		})
		It("delete the secret (testing finalizer removed)", func() {
			expectSecretIsDeleted(secret.Namespace, secret.Name)
		})
	})

	Context("when cluster is being deleted", func() {
		var cluster *kubermaticv1.Cluster

		BeforeEach(func() {
			cluster = createCluster(ctx, client, "deleting-cluster", workerLabel, false, []string{"something-to-keep-object"})
			Expect(client.Delete(ctx, cluster)).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(cluster), cluster)).To(Succeed())
				g.Expect(cluster.DeletionTimestamp.IsZero()).To(BeTrue())
			}, timeout, interval)
		})
		AfterEach(func() {
			original := cluster.DeepCopy()
			cluster.Finalizers = []string{}
			Expect(client.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(original))).To(Succeed())
		})

		It("secret should not be synced", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "app-cred",
					Annotations:  map[string]string{applicationsecretsynchronizer.SecretTypeAnnotation: ""},
					Labels:       map[string]string{"foo": "bar"},
					Namespace:    kubermaticNS.Name,
				},
				Data: map[string][]byte{"pass": []byte("a3ViZXJtYXRpYwo=")},
			}
			Expect(client.Create(ctx, secret)).To(Succeed())
			expectSecretNevertExist(cluster.Status.NamespaceName, secret.Name)
		})
	})

	Context("when a non application secret (i.e. without annotation applicationsecretsynchronizer.SecretTypeAnnotatio) is created '", func() {
		var secret *corev1.Secret

		BeforeEach(func() {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "a-secret",
					Namespace:    kubermaticNS.Name,
				},
				Data: map[string][]byte{"pass": []byte("a3ViZXJtYXRpYwo=")},
			}

			Expect(client.Create(ctx, secret)).To(Succeed())
		})
		It("should not be synced with  any cluster", func() {
			expectSecretNevertExist(clusterWithWorkerName.Status.NamespaceName, secret.Name)
			expectSecretNevertExist(pauseClusterWithWorkerName.Status.NamespaceName, secret.Name)
			expectSecretNevertExist(clusterWithoutWorkerName.Status.NamespaceName, secret.Name)
		})
	})

	Context("when application secret is created in another namespace than kubermatic'", func() {
		var secret *corev1.Secret

		BeforeEach(func() {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "a-secret",
					Namespace:    "default",
					Annotations:  map[string]string{applicationsecretsynchronizer.SecretTypeAnnotation: ""},
				},
				Data: map[string][]byte{"pass": []byte("a3ViZXJtYXRpYwo=")},
			}

			Expect(client.Create(ctx, secret)).To(Succeed())
		})
		It("should not be synced with  any cluster", func() {
			expectSecretNevertExist(clusterWithWorkerName.Status.NamespaceName, secret.Name)
			expectSecretNevertExist(pauseClusterWithWorkerName.Status.NamespaceName, secret.Name)
			expectSecretNevertExist(clusterWithoutWorkerName.Status.NamespaceName, secret.Name)
		})
	})
})

func expectSecretSync(clusterNamespace string, expectedSecert *corev1.Secret) {
	syncedSecret := &corev1.Secret{}
	EventuallyWithOffset(1, func(g Gomega) {
		g.Expect(client.Get(ctx, types.NamespacedName{Namespace: clusterNamespace, Name: expectedSecert.Name}, syncedSecret)).To(Succeed())
		g.ExpectWithOffset(1, syncedSecret.Data).To(Equal(expectedSecert.Data))
		g.ExpectWithOffset(1, syncedSecret.Labels).To(Equal(expectedSecert.Labels))
		g.ExpectWithOffset(1, syncedSecret.Annotations).To(Equal(expectedSecert.Annotations))
	}, timeout, interval).Should(Succeed())
}

func expectSecretNevertExist(clusterNamespace string, name string) {
	syncedSecret := &corev1.Secret{}
	ConsistentlyWithOffset(1, func() error {
		return client.Get(ctx, types.NamespacedName{Namespace: clusterNamespace, Name: name}, syncedSecret)
	}, timeout, interval).ShouldNot(Succeed())
}

func expectSecretIsDeleted(clusterNamespace string, name string) {
	syncedSecret := &corev1.Secret{}
	EventuallyWithOffset(1, func() error {
		return client.Get(ctx, types.NamespacedName{Namespace: clusterNamespace, Name: name}, syncedSecret)
	}, timeout, interval).ShouldNot(Succeed())
}

func expectSecretHasFinalizer(secret *corev1.Secret) {
	currentSecret := &corev1.Secret{}
	EventuallyWithOffset(1, func(g Gomega) {
		g.Expect(client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(secret), currentSecret)).To(Succeed())
		g.ExpectWithOffset(1, currentSecret.Finalizers).To(Equal([]string{appskubermaticv1.ApplicationSecretCleanupFinalizer}))
	}, timeout, interval).Should(Succeed())
}
