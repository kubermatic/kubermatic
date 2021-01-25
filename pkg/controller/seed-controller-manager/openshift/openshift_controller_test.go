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

package openshift

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"go.uber.org/zap"

	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/semver"
	testhelper "k8c.io/kubermatic/v2/pkg/test"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	ctrlruntimezaplog "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

func init() {
	_ = ioutil.WriteFile(certLocation, []byte(`-----BEGIN CERTIFICATE-----
MIIDojCCAoqgAwIBAgIQE4Y1TR0/BvLB+WUF1ZAcYjANBgkqhkiG9w0BAQUFADBr
MQswCQYDVQQGEwJVUzENMAsGA1UEChMEVklTQTEvMC0GA1UECxMmVmlzYSBJbnRl
cm5hdGlvbmFsIFNlcnZpY2UgQXNzb2NpYXRpb24xHDAaBgNVBAMTE1Zpc2EgZUNv
bW1lcmNlIFJvb3QwHhcNMDIwNjI2MDIxODM2WhcNMjIwNjI0MDAxNjEyWjBrMQsw
CQYDVQQGEwJVUzENMAsGA1UEChMEVklTQTEvMC0GA1UECxMmVmlzYSBJbnRlcm5h
dGlvbmFsIFNlcnZpY2UgQXNzb2NpYXRpb24xHDAaBgNVBAMTE1Zpc2EgZUNvbW1l
cmNlIFJvb3QwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCvV95WHm6h
2mCxlCfLF9sHP4CFT8icttD0b0/Pmdjh28JIXDqsOTPHH2qLJj0rNfVIsZHBAk4E
lpF7sDPwsRROEW+1QK8bRaVK7362rPKgH1g/EkZgPI2h4H3PVz4zHvtH8aoVlwdV
ZqW1LS7YgFmypw23RuwhY/81q6UCzyr0TP579ZRdhE2o8mCP2w4lPJ9zcc+U30rq
299yOIzzlr3xF7zSujtFWsan9sYXiwGd/BmoKoMWuDpI/k4+oKsGGelT84ATB+0t
vz8KPFUgOSwsAGl0lUq8ILKpeeUYiZGo3BxN77t+Nwtd/jmliFKMAGzsGHxBvfaL
dXe6YJ2E5/4tAgMBAAGjQjBAMA8GA1UdEwEB/wQFMAMBAf8wDgYDVR0PAQH/BAQD
AgEGMB0GA1UdDgQWBBQVOIMPPyw/cDMezUb+B4wg4NfDtzANBgkqhkiG9w0BAQUF
AAOCAQEAX/FBfXxcCLkr4NWSR/pnXKUTwwMhmytMiUbPWU3J/qVAtmPN3XEolWcR
zCSs00Rsca4BIGsDoo8Ytyk6feUWYFN4PMCvFYP3j1IzJL1kk5fui/fbGKhtcbP3
LBfQdCVp9/5rPJS+TUtBjE7ic9DjkCJzQ83z7+pzzkWKsKZJ/0x9nXGIxHYdkFsd
7v3M9+79YKWxehZx0RbQfBI8bGmX265fOZpwLwU8GUYEmSA20GBuYQa7FkKMcPcw
++DbZqMAAb3mLNqRX6BGi01qnD093QVG/na/oAo85ADmJ7f/hC3euiInlhBx6yLt
398znM/jra6O1I7mT1GvFpLgXPYHDw==
-----END CERTIFICATE-----`), 0644)
}

const certLocation = "/tmp/kubermatic-testcert.pem"

var update = flag.Bool("update", false, "update .golden files")

func TestResources(t *testing.T) {
	testCases := []struct {
		name         string
		reconciler   Reconciler
		object       runtime.Object
		validateFunc func(runtime.Object) error
	}{
		{
			name: "Kubermatic API image is overwritten",
			reconciler: Reconciler{
				kubermaticImage:          "my.corp/kubermatic",
				log:                      zap.NewNop().Sugar(),
				concurrentClusterUpdates: 10,
			},
			object: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "usercluster-controller",
				},
			},
			validateFunc: func(o runtime.Object) error {
				deployment := o.(*appsv1.Deployment)
				expectedImage := "docker.io/my.corp/kubermatic:"
				if deployment.Spec.Template.Spec.Containers[0].Image != expectedImage {
					return fmt.Errorf("Expected image to be %q, was %q",
						expectedImage, deployment.Spec.Template.Spec.Containers[0].Image)
				}
				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			ctrlruntimelog.SetLogger(ctrlruntimezaplog.Logger(false))
			if err := autoscalingv1beta2.AddToScheme(scheme.Scheme); err != nil {
				t.Fatalf("failed to add the autoscaling.k8s.io scheme to mgr: %v", err)
			}

			tc.reconciler.recorder = &record.FakeRecorder{}
			tc.reconciler.Client = fake.NewFakeClient(
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster",
						Annotations: map[string]string{
							"kubermatic.io/openshift": "true",
						},
					},
					Spec: kubermaticv1.ClusterSpec{
						Cloud: kubermaticv1.CloudSpec{
							DatacenterName: "alias-europe-west3-c",
						},
						ExposeStrategy: kubermaticv1.ExposeStrategyNodePort,
						Openshift:      &kubermaticv1.Openshift{},
						Version:        *semver.NewSemverOrDie("4.1.9"),
					},
					Status: kubermaticv1.ClusterStatus{
						NamespaceName: "test-cluster-ns",
						ExtendedHealth: kubermaticv1.ExtendedClusterHealth{
							CloudProviderInfrastructure: kubermaticv1.HealthStatusUp,
						},
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "apiserver-external",
						Namespace: "test-cluster-ns",
					},
					Spec: corev1.ServiceSpec{
						Type:           corev1.ServiceTypeLoadBalancer,
						LoadBalancerIP: "1.2.3.4",
					},
				},
			)
			tc.reconciler.seedGetter = func() (*kubermaticv1.Seed, error) {
				return &kubermaticv1.Seed{
					ObjectMeta: metav1.ObjectMeta{
						Name: "alias-europe-west3-c",
					},
					Spec: kubermaticv1.SeedSpec{
						Datacenters: map[string]kubermaticv1.Datacenter{
							"alias-europe-west3-c": {},
						},
					},
				}, nil
			}
			tc.reconciler.externalURL = "dev.kubermatic.io"
			tc.reconciler.oidc.CAFile = certLocation
			var err error
			tc.reconciler.userClusterConnProvider, err = clusterclient.NewInternal(tc.reconciler.Client)
			if err != nil {
				t.Fatalf("error getting usercluster connection provider: %v", err)
			}

			if _, err := tc.reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "test-cluster"}}); err != nil {
				if !strings.HasPrefix(err.Error(), "failed to check if cluster is reachable: failed to create restMapper:") {
					t.Fatalf("failed to run reconcile: %v", err)
				}
			}

			object := tc.object.DeepCopyObject()
			metav1object, ok := object.(metav1.Object)
			if !ok {
				t.Fatal("testcase object can not be asserted as metav1.Object")
			}

			if err := tc.reconciler.Get(ctx,
				types.NamespacedName{Namespace: "test-cluster-ns", Name: metav1object.GetName()},
				object); err != nil {
				t.Fatalf("failed to get object %q: %v", metav1object.GetName(), err)
			}

			if err := tc.validateFunc(object); err != nil {
				t.Fatal(err.Error())
			}

			serializedObject, err := yaml.Marshal(object)
			if err != nil {
				t.Fatalf("failed to serialize object %q: %v", metav1object.GetName(), err)
			}

			serializedObject = append([]byte("# This file has been generated, DO NOT EDIT.\n"), serializedObject...)

			testhelper.CompareOutput(t, fmt.Sprintf("%s-%s", strings.Replace(tc.name, " ", "_", -1), metav1object.GetName()), string(serializedObject), *update, ".yaml")
		})
	}

}
