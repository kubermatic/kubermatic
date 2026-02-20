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

package seedsync

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcilingSeed(t *testing.T) {
	kubeconfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-seed-kubeconfig",
			Namespace: "kubermatic",
		},
		Data: map[string][]byte{
			provider.DefaultKubeconfigFieldPath: []byte("this-is-not-a-kubeconfig-but-that-doesnt-matter-here"),
		},
	}

	existingSeeds := []ctrlruntimeclient.Object{
		&kubermaticv1.Seed{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-seed",
				Namespace: "kubermatic",
				Labels: map[string]string{
					"i-should-be-removed": "please",
				},
				Annotations: map[string]string{
					"i-should-be-removed": "please",
				},
			},
			Spec: kubermaticv1.SeedSpec{
				Country: "Val Verde",
			},
		},
	}
	tests := []struct {
		name          string
		shouldFail    bool
		seed          *kubermaticv1.Seed
		config        *kubermaticv1.KubermaticConfiguration
		existingSeeds []ctrlruntimeclient.Object
		validate      func(input, result *kubermaticv1.Seed, masterClient, seedClient ctrlruntimeclient.Client) error
	}{
		{
			name: "happy path",
			seed: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-seed",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.SeedSpec{
					Country:        "Germany",
					ExposeStrategy: kubermaticv1.ExposeStrategyNodePort,
					Kubeconfig: corev1.ObjectReference{
						Name: kubeconfigSecret.Name,
					},
				},
			},
			config: &kubermaticv1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: "kubermatic",
				},
			},
			existingSeeds: existingSeeds,
			validate: func(input, result *kubermaticv1.Seed, _, _ ctrlruntimeclient.Client) error {
				if result == nil {
					return errors.New("seed CR should exist in seed cluster, but does not")
				}

				if l := result.Labels[ManagedByLabel]; l != ControllerName {
					return fmt.Errorf("seed CR should have a %s label with '%s', but has label '%s'", ManagedByLabel, ControllerName, l)
				}

				if _, exists := result.Labels["i-should-be-removed"]; exists {
					return errors.New("existing labels should have been removed, but were not")
				}

				if _, exists := result.Annotations["i-should-be-removed"]; exists {
					return errors.New("existing annotations should have been removed, but were not")
				}

				if input.Spec.Country != result.Spec.Country {
					return errors.New("existing spec should have been updated, but was not")
				}

				return nil
			},
		},
		{
			name: "empty expose strategy",
			seed: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-seed",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.SeedSpec{
					Country: "Germany",
					Kubeconfig: corev1.ObjectReference{
						Name: kubeconfigSecret.Name,
					},
				},
			},
			config: &kubermaticv1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: "kubermatic",
				},
			},
			existingSeeds: existingSeeds,
			validate: func(input, result *kubermaticv1.Seed, _, _ ctrlruntimeclient.Client) error {
				if result == nil {
					return errors.New("seed CR should exist in seed cluster, but does not")
				}

				if l := result.Labels[ManagedByLabel]; l != ControllerName {
					return fmt.Errorf("seed CR should have a %s label with '%s', but has label '%s'", ManagedByLabel, ControllerName, l)
				}

				if _, exists := result.Labels["i-should-be-removed"]; exists {
					return errors.New("existing labels should have been removed, but were not")
				}

				if _, exists := result.Annotations["i-should-be-removed"]; exists {
					return errors.New("existing annotations should have been removed, but were not")
				}

				if input.Spec.Country != result.Spec.Country {
					return errors.New("existing spec should have been updated, but was not")
				}

				return nil
			},
		},
		{
			name: "sync config into seed",
			seed: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-seed",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.SeedSpec{
					Country:        "Germany",
					ExposeStrategy: kubermaticv1.ExposeStrategyNodePort,
					Kubeconfig: corev1.ObjectReference{
						Name: kubeconfigSecret.Name,
					},
				},
			},
			config: &kubermaticv1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					ImagePullSecret: "hello world",
				},
			},
			existingSeeds: existingSeeds,
			validate: func(_, _ *kubermaticv1.Seed, masterClient, seedClient ctrlruntimeclient.Client) error {
				cfg := &kubermaticv1.KubermaticConfiguration{}
				err := seedClient.Get(context.Background(), types.NamespacedName{
					Namespace: "kubermatic",
					Name:      "kubermatic",
				}, cfg)

				if err != nil {
					return fmt.Errorf("failed to get config: %w", err)
				}

				if cfg.Spec.ImagePullSecret == "" {
					return fmt.Errorf("ImagePullSecret should be set on the config copy in the seed cluster, but is empty")
				}

				return nil
			},
		},
	}

	rawLog := zap.NewNop()
	log := rawLog.Sugar()

	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog).WithName("controller_runtime"))

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			masterClient := fake.NewClientBuilder().WithObjects(test.seed, kubeconfigSecret).Build()
			seedClient := fake.
				NewClientBuilder().
				WithObjects(test.existingSeeds...).
				Build()

			ctx := context.Background()

			reconciler := Reconciler{
				Client:   masterClient,
				recorder: events.NewFakeRecorder(10),
				log:      log,
				seedClientGetter: func(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error) {
					return seedClient, nil
				},
			}

			err := reconciler.reconcile(ctx, test.config, test.seed, seedClient, log)
			if test.shouldFail && err == nil {
				t.Fatalf("check for %s failed", test.name)
			}
			if !test.shouldFail && err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			key := ctrlruntimeclient.ObjectKeyFromObject(test.seed)

			result := &kubermaticv1.Seed{}
			if err := seedClient.Get(ctx, key, result); err != nil && apierrors.IsNotFound(err) {
				t.Fatalf("could not find seed CR in seed cluster: %v", err)
			}
			if test.validate != nil {
				if err := test.validate(test.seed, result, masterClient, seedClient); err != nil {
					t.Fatalf("reconciling did not lead to a valid end state: %v", err)
				}
			}
		})
	}
}

func TestConfigRemainsOnSharedSeedCleanup(t *testing.T) {
	kubeconfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-seed-kubeconfig",
			Namespace: "kubermatic",
		},
		Data: map[string][]byte{
			provider.DefaultKubeconfigFieldPath: []byte("this-is-not-a-kubeconfig-but-that-doesnt-matter-here"),
		},
	}

	rawLog := zap.NewNop()
	log := rawLog.Sugar()

	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog).WithName("controller_runtime"))

	config := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubermatic",
			Namespace: "kubermatic",
			UID:       "config-uid",
		},
	}

	seed := &kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-seed",
			Namespace: "kubermatic",
			UID:       "seed-uid",
		},
		Spec: kubermaticv1.SeedSpec{
			Country: "Val Verde",
			Kubeconfig: corev1.ObjectReference{
				Name: kubeconfigSecret.Name,
			},
		},
		Status: kubermaticv1.SeedStatus{
			Conditions: map[kubermaticv1.SeedConditionType]kubermaticv1.SeedCondition{
				kubermaticv1.SeedConditionClusterInitialized: {
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	masterSeedClient := fake.NewClientBuilder().WithObjects(config, seed, kubeconfigSecret).Build()

	ctx := context.Background()

	// create the reconciler
	reconciler := Reconciler{
		Client:   masterSeedClient,
		recorder: events.NewFakeRecorder(10),
		log:      log,
		seedClientGetter: func(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error) {
			return masterSeedClient, nil
		},
	}

	reconcile := func(ctx context.Context) {
		if _, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(seed),
		}); err != nil {
			t.Fatalf("reconciling failed: %v", err)
		}
	}

	// reconcile once to add the finalizers
	reconcile(ctx)

	// ensure finalizer was added
	currentSeed := &kubermaticv1.Seed{}
	if err := masterSeedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(seed), currentSeed); err != nil {
		t.Fatalf("failed to get current seed on master cluster: %v", err)
	}

	// delete the seed object on the master cluster
	toDelete := seed.DeepCopy()
	if err := masterSeedClient.Delete(ctx, toDelete); err != nil {
		t.Fatalf("failed to delete seed on master cluster: %v", err)
	}

	// if all finalizers are correct, the seed should still exist
	currentSeed = &kubermaticv1.Seed{}
	if err := masterSeedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(seed), currentSeed); err != nil {
		t.Fatalf("failed to get seed on master cluster: %v", err)
	}

	// reconcile again, this should *not* cleanup the config
	reconcile(ctx)

	// cleanup takes 2 rounds of reconciling
	reconcile(ctx)

	// in general: seed should be gone in both clusters, config should remain on the master;
	// for shared master/seed this means the seed is gone and the config remains
	currentSeed = &kubermaticv1.Seed{}
	if err := masterSeedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(seed), currentSeed); err == nil {
		t.Errorf("expected seed to be deleted, but found it: %+v", currentSeed)
	}

	currentConfig := &kubermaticv1.KubermaticConfiguration{}
	if err := masterSeedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(config), currentConfig); err != nil {
		t.Errorf("config should exist, but failed to get it: %v", err)
	}
}
