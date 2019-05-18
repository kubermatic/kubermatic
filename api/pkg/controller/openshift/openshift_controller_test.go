package openshift

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	testhelper "github.com/kubermatic/kubermatic/api/pkg/test"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
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
				kubermaticAPIImage: "my.corp/kubermatic",
			},
			object: &appsv1.Deployment{
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

			ctrlruntimelog.SetLogger(ctrlruntimelog.ZapLogger(true))
			if err := kubermaticv1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
				t.Fatalf("failed to add kubermatic scheme to mgr: %v", err)
			}
			if err := autoscalingv1beta2.AddToScheme(scheme.Scheme); err != nil {
				t.Fatalf("failed to add the autoscaling.k8s.io scheme to mgr: %v", err)
			}

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
					},
					Status: kubermaticv1.ClusterStatus{
						NamespaceName: "test-cluster-ns",
						Health: kubermaticv1.ClusterHealth{
							ClusterHealthStatus: kubermaticv1.ClusterHealthStatus{
								CloudProviderInfrastructure: true,
							},
						},
					},
				},
			)
			tc.reconciler.dcs = map[string]provider.DatacenterMeta{
				"alias-europe-west3-c": {},
			}
			tc.reconciler.externalURL = "dev.kubermatic.io"
			tc.reconciler.dc = "alias-europe-west3-c"
			tc.reconciler.oidc.CAFile = certLocation

			if _, err := tc.reconciler.Reconcile(reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "test-cluster"}}); err != nil {
				t.Fatalf("failed to run reconcile: %v", err)
			}

			object := tc.object.DeepCopyObject()
			metav1object, ok := object.(metav1.Object)
			if !ok {
				t.Fatal("testcase object can not be asserted as metav1.Object")
			}

			if err := tc.reconciler.Get(context.Background(),
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

			testhelper.CompareOutput(t, fmt.Sprintf("%s-%s", tc.name, metav1object.GetName()), string(serializedObject), *update, ".yaml")
		})
	}

}
