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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

func init() {
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme.Scheme))
}

func TestReconcilingSeed(t *testing.T) {

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
		config        *operatorv1alpha1.KubermaticConfiguration
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
				},
			},
			config: &operatorv1alpha1.KubermaticConfiguration{
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
				},
			},
			config: &operatorv1alpha1.KubermaticConfiguration{
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
			name:       "invalid expose strategy",
			shouldFail: true,
			seed: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-seed",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.SeedSpec{
					Country:        "Germany",
					ExposeStrategy: kubermaticv1.ExposeStrategy("wtf"),
				},
			},
			config: &operatorv1alpha1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: "kubermatic",
				},
			},
			existingSeeds: existingSeeds,
			// actual check is done in the reconciler, nothing to validate here
			validate: nil,
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
				},
			},
			config: &operatorv1alpha1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: "kubermatic",
				},
				Spec: operatorv1alpha1.KubermaticConfigurationSpec{
					ImagePullSecret: "hello world",
				},
			},
			existingSeeds: existingSeeds,
			validate: func(_, _ *kubermaticv1.Seed, masterClient, seedClient ctrlruntimeclient.Client) error {
				cfg := &operatorv1alpha1.KubermaticConfiguration{}
				err := seedClient.Get(context.TODO(), types.NamespacedName{
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
			masterClient := fakectrlruntimeclient.NewClientBuilder().WithObjects(test.seed).Build()
			seedClient := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(test.existingSeeds...).
				Build()

			ctx := context.Background()

			reconciler := Reconciler{
				Client:   masterClient,
				recorder: record.NewFakeRecorder(10),
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
			if err := seedClient.Get(ctx, key, result); err != nil && kerrors.IsNotFound(err) {
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
