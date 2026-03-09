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

package clustercredentialscontroller

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/machine-controller/sdk/providerconfig"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	const (
		clusterName    = "my-cluster"
		seedNamespace  = "kubermatic"
		credentialName = "credential-digitalocean-" + clusterName
	)

	testCases := []struct {
		name string

		cloudSpec     kubermaticv1.CloudSpec
		kkpSecret     map[string][]byte
		clusterSecret map[string][]byte

		expectedCloudSpec     kubermaticv1.CloudSpec
		expectedKKPSecret     map[string][]byte
		expectedClusterSecret map[string][]byte
	}{
		{
			name: "vanilla, nothing to do",
			cloudSpec: kubermaticv1.CloudSpec{
				Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
					CredentialsReference: &providerconfig.GlobalSecretKeySelector{
						ObjectReference: corev1.ObjectReference{
							Namespace: seedNamespace,
							Name:      credentialName,
						},
					},
				},
			},
			kkpSecret: map[string][]byte{
				resources.DigitaloceanToken: []byte("not-a-real-token"),
			},
			clusterSecret: map[string][]byte{
				resources.DigitaloceanToken: []byte("not-a-real-token"),
			},

			expectedCloudSpec: kubermaticv1.CloudSpec{
				Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
					CredentialsReference: &providerconfig.GlobalSecretKeySelector{
						ObjectReference: corev1.ObjectReference{
							Namespace: seedNamespace,
							Name:      credentialName,
						},
					},
				},
			},
			expectedKKPSecret: map[string][]byte{
				resources.DigitaloceanToken: []byte("not-a-real-token"),
			},
			expectedClusterSecret: map[string][]byte{
				resources.DigitaloceanToken: []byte("not-a-real-token"),
			},
		},

		{
			name: "new Cluster with inline credentials, both Secrets need to be created",
			cloudSpec: kubermaticv1.CloudSpec{
				Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
					Token: "not-a-real-token",
				},
			},

			expectedCloudSpec: kubermaticv1.CloudSpec{
				Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
					CredentialsReference: &providerconfig.GlobalSecretKeySelector{
						ObjectReference: corev1.ObjectReference{
							Namespace: seedNamespace,
							Name:      credentialName,
						},
					},
				},
			},
			expectedKKPSecret: map[string][]byte{
				resources.DigitaloceanToken: []byte("not-a-real-token"),
			},
			expectedClusterSecret: map[string][]byte{
				resources.DigitaloceanToken: []byte("not-a-real-token"),
			},
		},

		{
			name: "Cluster credentials are being updated and should be synced into the cluster namespace",
			cloudSpec: kubermaticv1.CloudSpec{
				Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
					CredentialsReference: &providerconfig.GlobalSecretKeySelector{
						ObjectReference: corev1.ObjectReference{
							Namespace: seedNamespace,
							Name:      credentialName,
						},
					},
				},
			},
			kkpSecret: map[string][]byte{
				resources.DigitaloceanToken: []byte("updated-token"),
			},
			clusterSecret: map[string][]byte{
				resources.DigitaloceanToken: []byte("old-token"),
			},

			expectedCloudSpec: kubermaticv1.CloudSpec{
				Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
					CredentialsReference: &providerconfig.GlobalSecretKeySelector{
						ObjectReference: corev1.ObjectReference{
							Namespace: seedNamespace,
							Name:      credentialName,
						},
					},
				},
			},
			expectedKKPSecret: map[string][]byte{
				resources.DigitaloceanToken: []byte("updated-token"),
			},
			expectedClusterSecret: map[string][]byte{
				resources.DigitaloceanToken: []byte("updated-token"),
			},
		},

		{
			name: "Cluster *inline* credentials are being updated and should be synced into the cluster namespace",
			cloudSpec: kubermaticv1.CloudSpec{
				Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
					CredentialsReference: &providerconfig.GlobalSecretKeySelector{
						ObjectReference: corev1.ObjectReference{
							Namespace: seedNamespace,
							Name:      credentialName,
						},
					},
					Token: "new-token",
				},
			},
			kkpSecret: map[string][]byte{
				resources.DigitaloceanToken: []byte("old-token"),
			},
			clusterSecret: map[string][]byte{
				resources.DigitaloceanToken: []byte("old-token"),
			},

			expectedCloudSpec: kubermaticv1.CloudSpec{
				Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
					CredentialsReference: &providerconfig.GlobalSecretKeySelector{
						ObjectReference: corev1.ObjectReference{
							Namespace: seedNamespace,
							Name:      credentialName,
						},
					},
				},
			},
			expectedKKPSecret: map[string][]byte{
				resources.DigitaloceanToken: []byte("new-token"),
			},
			expectedClusterSecret: map[string][]byte{
				resources.DigitaloceanToken: []byte("new-token"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			///////////////////////////////
			// setup preconditions

			dummyCluster := &kubermaticv1.Cluster{}
			dummyCluster.Name = clusterName
			dummyCluster.Labels = map[string]string{
				kubermaticv1.ProjectIDLabelKey: "test",
			}
			dummyCluster.Spec.Cloud = tc.cloudSpec
			dummyCluster.Status.NamespaceName = "cluster-" + clusterName

			builder := fake.NewClientBuilder().WithObjects(dummyCluster)

			if tc.kkpSecret != nil {
				ref, err := resources.GetCredentialsReference(dummyCluster)
				if err != nil {
					t.Fatalf("Expected existing cluster to already have a credentials ref, because a kkpSecret is also defined in the testcase, but failed to get current ref: %v", err)
				}
				if ref == nil {
					t.Fatal("Expected a credential ref on the existing Cluster, but got nil.")
				}

				builder.WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ref.Name,
						Namespace: seedNamespace,
					},
					Data: tc.kkpSecret,
				})
			}

			if tc.clusterSecret != nil {
				builder.WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resources.ClusterCloudCredentialsSecretName,
						Namespace: dummyCluster.Status.NamespaceName,
					},
					Data: tc.clusterSecret,
				})
			}

			seedClient := builder.Build()

			///////////////////////////////
			// setup controller

			ctx := context.Background()
			r := &reconciler{
				Client:     seedClient,
				workerName: "",
				recorder:   &events.FakeRecorder{},
				log:        kubermaticlog.Logger,
				versions:   kubermatic.GetFakeVersions(),
			}

			///////////////////////////////
			// reconcile once

			request := reconcile.Request{NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(dummyCluster)}
			// the controller will requeue under normal operations,
			// so we simply reconcile a few times to be sure we got it all
			for range 3 {
				if _, err := r.Reconcile(ctx, request); err != nil {
					t.Fatalf("Reconciling failed: %v", err)
				}
			}

			///////////////////////////////
			// check assertions

			// get current cluster state
			currentCluster := &kubermaticv1.Cluster{}
			if err := seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(dummyCluster), currentCluster); err != nil {
				t.Fatalf("Failed to get current cluster state: %v", err)
			}

			if changes := diff.ObjectDiff(tc.expectedCloudSpec, currentCluster.Spec.Cloud); changes != "" {
				t.Fatalf("CloudSpec is not as expected:\n\n%s", changes)
			}

			ref, err := resources.GetCredentialsReference(currentCluster)
			if err != nil {
				t.Fatalf("Failed to get secret reference: %v", err)
			}

			if ref == nil {
				t.Fatal("Expected a credential reference in the Cluster object, but none was found.")
			}

			currentKKPSecret := &corev1.Secret{}
			if err := seedClient.Get(ctx, types.NamespacedName{Namespace: seedNamespace, Name: credentialName}, currentKKPSecret); err != nil {
				t.Fatalf("Failed to get current KKP credential secret: %v", err)
			}

			if changes := diff.ObjectDiff(tc.expectedKKPSecret, currentKKPSecret.Data); changes != "" {
				t.Fatalf("KKP Secret is not as expected:\n\n%s", changes)
			}

			currentClusterSecret := &corev1.Secret{}
			if err := seedClient.Get(ctx, types.NamespacedName{Namespace: dummyCluster.Status.NamespaceName, Name: resources.ClusterCloudCredentialsSecretName}, currentClusterSecret); err != nil {
				t.Fatalf("Failed to get current cluster credential secret: %v", err)
			}

			if changes := diff.ObjectDiff(tc.expectedClusterSecret, currentClusterSecret.Data); changes != "" {
				t.Fatalf("Cluster Secret is not as expected:\n\n%s", changes)
			}
		})
	}
}
