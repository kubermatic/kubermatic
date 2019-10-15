package clustercomponentdefaulter

import (
	"fmt"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-test/deep"
	"go.uber.org/zap"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilpointer "k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const clusterName = "test-cluster"

func exampleOverrides() []kubermaticv1.ComponentSettings {
	return []kubermaticv1.ComponentSettings{
		{
			Apiserver: kubermaticv1.APIServerSettings{
				DeploymentSettings:          kubermaticv1.DeploymentSettings{Replicas: utilpointer.Int32Ptr(1)},
				EndpointReconcilingDisabled: utilpointer.BoolPtr(true),
			},
			Scheduler:         kubermaticv1.DeploymentSettings{Replicas: utilpointer.Int32Ptr(2)},
			ControllerManager: kubermaticv1.DeploymentSettings{Replicas: utilpointer.Int32Ptr(3)},
		},
	}
}

func TestReconciliation(t *testing.T) {
	testCases := []struct {
		name     string
		cluster  kubermaticv1.Cluster
		override kubermaticv1.ComponentSettings
		verify   func(error, *kubermaticv1.Cluster) error
	}{
		{
			name:    "Defaulting without EndpointReconcilingDisabled",
			cluster: kubermaticv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: clusterName}},
			override: kubermaticv1.ComponentSettings{
				Apiserver: kubermaticv1.APIServerSettings{},
			},
			verify: func(err error, cluster *kubermaticv1.Cluster) error {
				if err != nil {
					return fmt.Errorf("expected err to be nil, was: %v", err)
				}
				if val := cluster.Spec.ComponentsOverride.Apiserver.EndpointReconcilingDisabled; val != nil {
					return fmt.Errorf("expected EndpointReconcilingDisabled to be nil, was %v", val)
				}
				return nil
			},
		},
		{
			name:    "Defaulting without EndpointReconcilingDisabled with Replicas",
			cluster: kubermaticv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: clusterName}},
			override: kubermaticv1.ComponentSettings{
				Apiserver: kubermaticv1.APIServerSettings{
					DeploymentSettings: kubermaticv1.DeploymentSettings{Replicas: utilpointer.Int32Ptr(1)},
				},
				Scheduler:         kubermaticv1.DeploymentSettings{Replicas: utilpointer.Int32Ptr(2)},
				ControllerManager: kubermaticv1.DeploymentSettings{Replicas: utilpointer.Int32Ptr(3)},
			},
			verify: func(err error, cluster *kubermaticv1.Cluster) error {
				if err != nil {
					return fmt.Errorf("expected err to be nil, was: %v", err)
				}
				if val := cluster.Spec.ComponentsOverride.Apiserver.EndpointReconcilingDisabled; val != nil {
					return fmt.Errorf("expected EndpointReconcilingDisabled to be nil, was %v", val)
				}
				if cluster.Spec.ComponentsOverride.Apiserver.Replicas == nil ||
					cluster.Spec.ComponentsOverride.Scheduler.Replicas == nil ||
					cluster.Spec.ComponentsOverride.ControllerManager.Replicas == nil ||
					*cluster.Spec.ComponentsOverride.Apiserver.Replicas != 1 ||
					*cluster.Spec.ComponentsOverride.Scheduler.Replicas != 2 ||
					*cluster.Spec.ComponentsOverride.ControllerManager.Replicas != 3 {
					return fmt.Errorf("expected unmodified Replica counts, at least one was modified")
				}
				return nil
			},
		},
		{
			name:     "Defaulting with EndpointReconcilingDisabled with Replicas",
			cluster:  kubermaticv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: clusterName}},
			override: exampleOverrides()[0],
			verify: func(err error, cluster *kubermaticv1.Cluster) error {
				if err != nil {
					return fmt.Errorf("expected err to be nil, was: %v", err)
				}
				if diff := deep.Equal(cluster.Spec.ComponentsOverride, exampleOverrides()[0]); diff != nil {
					return fmt.Errorf("unexpected difference in components override: %v", diff)
				}
				return nil
			},
		},
		{
			name:    "Defaulting with EndpointReconcilingDisabled: true",
			cluster: kubermaticv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: clusterName}},
			override: kubermaticv1.ComponentSettings{
				Apiserver: kubermaticv1.APIServerSettings{EndpointReconcilingDisabled: utilpointer.BoolPtr(true)},
			},
			verify: func(err error, cluster *kubermaticv1.Cluster) error {
				if err != nil {
					return fmt.Errorf("expected err to be nil, was: %v", err)
				}
				if val := cluster.Spec.ComponentsOverride.Apiserver.EndpointReconcilingDisabled; val == nil || *val != true {
					return fmt.Errorf("expected EndpointReconcilingDisabled to be true, was %v", val)
				}
				return nil
			},
		},
		{
			name:    "Defaulting with EndpointReconcilingDisabled: false",
			cluster: kubermaticv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: clusterName}},
			override: kubermaticv1.ComponentSettings{
				Apiserver: kubermaticv1.APIServerSettings{EndpointReconcilingDisabled: utilpointer.BoolPtr(false)},
			},
			verify: func(err error, cluster *kubermaticv1.Cluster) error {
				if err != nil {
					return fmt.Errorf("expected err to be nil, was: %v", err)
				}
				if val := cluster.Spec.ComponentsOverride.Apiserver.EndpointReconcilingDisabled; val == nil || *val != false {
					return fmt.Errorf("expected EndpointReconcilingDisabled to be false, was %v", val)
				}
				return nil
			},
		},
		{
			name: "No override when EndpointReconcilingDisabled is specified in cluster",
			cluster: kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: clusterName},
				Spec: kubermaticv1.ClusterSpec{
					ComponentsOverride: kubermaticv1.ComponentSettings{
						Apiserver: kubermaticv1.APIServerSettings{
							EndpointReconcilingDisabled: utilpointer.BoolPtr(false),
						},
					},
				},
			},
			override: kubermaticv1.ComponentSettings{
				Apiserver: kubermaticv1.APIServerSettings{EndpointReconcilingDisabled: utilpointer.BoolPtr(true)},
			},
			verify: func(err error, cluster *kubermaticv1.Cluster) error {
				if err != nil {
					return fmt.Errorf("expected err to be nil, was: %v", err)
				}
				if diff := deep.Equal(&cluster.Spec.ComponentsOverride.Apiserver,
					&kubermaticv1.APIServerSettings{EndpointReconcilingDisabled: utilpointer.BoolPtr(false)}); diff != nil {
					return fmt.Errorf("unexpected difference in Apiserver override: %v", diff)
				}
				return nil
			},
		},
	}

	logger := zap.NewExample().Sugar()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewFakeClient([]runtime.Object{&tc.cluster}...)
			r := &Reconciler{client: client, log: logger, defaults: tc.override}
			reconcileErr := r.reconcile(logger, &tc.cluster)
			reconciledCluster := &kubermaticv1.Cluster{}
			if err := r.client.Get(r.ctx, types.NamespacedName{Name: clusterName}, reconciledCluster); err != nil {
				t.Fatalf("failed to get reconciledCluster %s: %v", clusterName, err)
			}
			if err := tc.verify(reconcileErr, reconciledCluster); err != nil {
				t.Fatalf("verification failed: %v", err)
			}
		})
	}
}
