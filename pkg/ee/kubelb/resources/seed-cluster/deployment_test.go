//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2026 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package resources

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestKubeLBDeploymentImages(t *testing.T) {
	for _, test := range []struct {
		name              string
		overwriteRegistry string
		repository        string
		tag               string
		wantCCM           string
		wantProxyRegistry string
	}{
		{
			name: "defaults", wantCCM: "quay.io/kubermatic/kubelb-ccm-ee:v1.5.0", wantProxyRegistry: "docker.io",
		},
		{
			name: "registry override", overwriteRegistry: "registry.example.com/cache",
			wantCCM: "registry.example.com/cache/kubermatic/kubelb-ccm-ee:v1.5.0", wantProxyRegistry: "registry.example.com/cache",
		},
		{
			name: "custom CCM repository and tag", overwriteRegistry: "registry.example.com/cache",
			repository: "custom.example.com/kubelb/ccm", tag: "v1.5.1-test",
			wantCCM: "custom.example.com/kubelb/ccm:v1.5.1-test", wantProxyRegistry: "registry.example.com/cache",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := resources.NewTemplateDataBuilder().
				WithCluster(&kubermaticv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"}}).
				WithDatacenter(&kubermaticv1.Datacenter{}).
				WithOverwriteRegistry(test.overwriteRegistry).
				WithKubeLBImageRepository(test.repository).
				WithKubeLBImageTag(test.tag).
				Build()

			_, reconcile := DeploymentReconcilerWithoutInitWrapper(data)()
			deployment, err := reconcile(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "cluster-test-cluster"}})
			if err != nil {
				t.Fatal(err)
			}
			container := deployment.Spec.Template.Spec.Containers[0]
			if container.Image != test.wantCCM {
				t.Errorf("CCM image = %q, want %q", container.Image, test.wantCCM)
			}
			flags := kubeLBFlagValues(container.Args)
			for name, want := range map[string]string{
				"tenant-proxy-xds-writer-image":       test.wantCCM,
				"tenant-proxy-envoy-image":            test.wantProxyRegistry + "/envoyproxy/envoy:distroless-v1.36.4",
				"tenant-proxy-shutdown-manager-image": test.wantProxyRegistry + "/envoyproxy/gateway:v1.8.3",
				"leader-election-namespace":           metav1.NamespaceSystem,
				"cluster-name":                        "tenant-test-cluster",
			} {
				if flags[name] != want {
					t.Errorf("flag %s = %q, want %q", name, flags[name], want)
				}
			}
			if !reflect.DeepEqual(container.Env, []corev1.EnvVar{{Name: "NAMESPACE", Value: metav1.NamespaceSystem}}) {
				t.Errorf("proxy namespace must be user-cluster kube-system, got %v", container.Env)
			}

			updated, err := reconcile(deployment.DeepCopy())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(deployment, updated) {
				t.Error("reconciliation changed an up-to-date deployment")
			}
		})
	}
}

func TestKubeLBDeploymentExtraArgs(t *testing.T) {
	datacenterArgs := map[string]string{
		"tenant-proxy-envoy-image":        "custom.example.com/envoy:dc",
		"tenant-proxy-image-pull-secrets": "dc-credentials",
	}
	for _, test := range []struct {
		name        string
		clusterArgs map[string]string
		wantEnvoy   string
		wantSecrets string
	}{
		{
			name: "datacenter fallback", wantEnvoy: datacenterArgs["tenant-proxy-envoy-image"], wantSecrets: "dc-credentials",
		},
		{
			name: "empty cluster map suppresses datacenter args", clusterArgs: map[string]string{},
			wantEnvoy: "docker.io/envoyproxy/envoy:distroless-v1.36.4",
		},
		{
			name: "cluster overrides defaults and datacenter", clusterArgs: map[string]string{
				"tenant-proxy-envoy-image":        "custom.example.com/envoy:cluster",
				"tenant-proxy-image-pull-secrets": "user-registry,another-registry",
			},
			wantEnvoy: "custom.example.com/envoy:cluster", wantSecrets: "user-registry,another-registry",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cluster := &kubermaticv1.Cluster{Spec: kubermaticv1.ClusterSpec{KubeLB: &kubermaticv1.KubeLB{
				EnableGatewayAPI: ptr.To(true), UseLoadBalancerClass: ptr.To(true), ExtraArgs: test.clusterArgs,
			}}}
			dc := &kubermaticv1.Datacenter{Spec: kubermaticv1.DatacenterSpec{KubeLB: &kubermaticv1.KubeLBDatacenterSettings{
				EnableSecretSynchronizer: true, DisableIngressClass: true, NodeAddressType: "InternalIP", ExtraArgs: datacenterArgs,
			}}}
			data := resources.NewTemplateDataBuilder().WithCluster(cluster).WithDatacenter(dc).Build()
			_, reconcile := DeploymentReconcilerWithoutInitWrapper(data)()
			deployment, err := reconcile(&appsv1.Deployment{})
			if err != nil {
				t.Fatal(err)
			}
			flags := kubeLBFlagValues(deployment.Spec.Template.Spec.Containers[0].Args)
			for name, want := range map[string]string{
				"tenant-proxy-envoy-image": test.wantEnvoy, "tenant-proxy-image-pull-secrets": test.wantSecrets,
				"enable-gateway-api": "true", "install-gateway-api-crds": "true", "use-loadbalancer-class": "true",
				"enable-secret-synchronizer": "true", "use-ingress-class": "false", "node-address-type": "InternalIP",
			} {
				if flags[name] != want {
					t.Errorf("flag %s = %q, want %q", name, flags[name], want)
				}
			}
			updated, err := reconcile(deployment.DeepCopy())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(deployment, updated) {
				t.Error("extra args must have stable ordering between reconciliations")
			}
		})
	}
}

func TestKubeLBDeploymentImageRewriteErrors(t *testing.T) {
	for _, failImage := range []string{"kubelb-ccm-ee", "envoyproxy/envoy", "envoyproxy/gateway"} {
		t.Run(failImage, func(t *testing.T) {
			data := failingKubeLBImageData{
				TemplateData: resources.NewTemplateDataBuilder().WithCluster(&kubermaticv1.Cluster{}).WithDatacenter(&kubermaticv1.Datacenter{}).Build(),
				failImage:    failImage,
			}
			_, reconcile := DeploymentReconcilerWithoutInitWrapper(data)()
			if _, err := reconcile(&appsv1.Deployment{}); !errors.Is(err, errKubeLBImageRewrite) {
				t.Fatalf("expected image rewrite error, got %v", err)
			}
		})
	}
}

var errKubeLBImageRewrite = errors.New("image rewrite failed")

type failingKubeLBImageData struct {
	*resources.TemplateData
	failImage string
}

func (d failingKubeLBImageData) RewriteImage(image string) (string, error) {
	if strings.Contains(image, d.failImage) {
		return "", errKubeLBImageRewrite
	}
	return d.TemplateData.RewriteImage(image)
}

// kubeLBFlagValues resolves flag values in argument order, including ExtraArgs overrides.
func kubeLBFlagValues(args []string) map[string]string {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		name, value, found := strings.Cut(strings.TrimLeft(args[i], "-"), "=")
		if !found {
			value = "true"
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				value = args[i]
			}
		}
		values[name] = value
	}
	return values
}
