package addon

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	testManifest1 = `kind: ConfigMap
apiVersion: v1
metadata:
  name: test1
  namespace: kube-system
  labels:
    app: test
data:
  foo: bar
`
	testManifest2 = `kind: ConfigMap
apiVersion: v1
metadata:
  name: test2
  namespace: kube-system
  labels:
    app: test
data:
  foo: bar`
	testManifest3 = `kind: ConfigMap
apiVersion: v1
metadata:
  name: test3
  namespace: kube-system
  labels:
    app: test
data:
  foo: bar
`

	testManifest1WithLabel = `apiVersion: v1
data:
  foo: bar
kind: ConfigMap
metadata:
  labels:
    app: test
    kubermatic-addon: test
  name: test1
  namespace: kube-system
`

	testManifest1WithDeployment = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
spec:
	template:
	  spec:
	    containers:
	    - name: nginx
	      image: {{default "foo.io/" .OverwriteRegistry}}test:1.2.3
`

	testManifestKubeDNS = `apiVersion: v1
kind: Service
metadata:
  name: kube-dns
  namespace: kube-system
  labels:
    k8s-app: kube-dns
    kubernetes.io/cluster-service: "true"
    addonmanager.kubernetes.io/mode: Reconcile
    kubernetes.io/name: "KubeDNS"
spec:
  selector:
    k8s-app: kube-dns
  clusterIP: {{.DNSClusterIP}}
	clusterCIDR: "{{first .Cluster.Spec.ClusterNetwork.Pods.CIDRBlocks}}"
`
)

var (
	// testManifest1 & testManifest3 have a linebreak at the end, testManifest2 not
	combinedTestManifest = fmt.Sprintf("%s---\n%s\n---\n%s", testManifest1, testManifest2, testManifest3)
)

type fakeKubeconfigProvider struct{}

func (f *fakeKubeconfigProvider) GetAdminKubeconfig(c *kubermaticv1.Cluster) ([]byte, error) {
	return []byte("foo"), nil
}

func TestController_combineManifests(t *testing.T) {
	controller := &Controller{}

	manifests := []*bytes.Buffer{
		bytes.NewBufferString(testManifest1),
		bytes.NewBufferString(testManifest2),
		bytes.NewBufferString(testManifest3),
	}

	manifest := controller.combineManifests(manifests)

	if manifest.String() != combinedTestManifest {
		t.Fatalf("invalid combined manifest returned. Expected \n%s, Got \n%s", combinedTestManifest, manifest.String())
	}
}

func setupTestCluster(CIDRBlock string) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cluster",
		},
		Spec: kubermaticv1.ClusterSpec{
			Version: "v1.10.2",
			ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
				Services: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{
						CIDRBlock,
					},
				},
				Pods: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{
						"172.25.0.0/16",
					},
				},
				DNSDomain: "cluster.local",
			},
			Cloud: kubermaticv1.CloudSpec{
				Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
					Token: "1234567890",
				},
				DatacenterName: "us-central1a",
			},
		},
	}
}

func setupTestAddon(name string) *kubermaticv1.Addon {
	return &kubermaticv1.Addon{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.AddonSpec{
			Name: name,
		},
	}
}

func TestController_getAddonKubeDNStManifests(t *testing.T) {
	cluster := setupTestCluster("10.10.10.0/24")
	addon := setupTestAddon("kube-dns")

	addonDir, err := ioutil.TempDir("/tmp", "kubermatic-tests-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(addonDir)

	if err := os.Mkdir(path.Join(addonDir, addon.Spec.Name), 0777); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(path.Join(addonDir, addon.Spec.Name, "testManifest.yaml"), []byte(testManifestKubeDNS), 0644); err != nil {
		t.Fatal(err)
	}

	controller := &Controller{
		addonDir:           addonDir,
		KubeconfigProvider: &fakeKubeconfigProvider{},
	}
	manifests, err := controller.getAddonManifests(addon, cluster)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifests) != 1 {
		t.Fatalf("invalid number of manifests returned. Expected 1, Got %d", len(manifests))
	}
	fmt.Println(manifests)
	expectedIP := "10.10.10.10"
	if !strings.Contains(manifests[0].String(), expectedIP) {
		t.Fatalf("invalid IP returned. Expected \n%s, Got \n%s", expectedIP, manifests[0].String())
	}
	expectedCIDR := "172.25.0.0/16"
	if !strings.Contains(manifests[0].String(), expectedCIDR) {
		t.Fatalf("invalid CIDR returned. Expected \n%s, Got \n%s", expectedCIDR, manifests[0].String())
	}

	cluster = setupTestCluster("172.25.0.0/16")
	manifests, err = controller.getAddonManifests(addon, cluster)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifests) != 1 {
		t.Fatalf("invalid number of manifests returned. Expected 1, Got %d", len(manifests))
	}
	expectedIP = "172.25.0.10"
	if !strings.Contains(manifests[0].String(), expectedIP) {
		t.Fatalf("invalid registryURI returned. Expected \n%s, Got \n%s", expectedIP, manifests[0].String())
	}
}

func TestController_getAddonDeploymentManifests(t *testing.T) {
	cluster := setupTestCluster("10.10.10.0/24")
	addon := setupTestAddon("test")

	addonDir, err := ioutil.TempDir("/tmp", "kubermatic-tests-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(addonDir)

	if err := os.Mkdir(path.Join(addonDir, addon.Spec.Name), 0777); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(path.Join(addonDir, addon.Spec.Name, "testManifest.yaml"), []byte(testManifest1WithDeployment), 0644); err != nil {
		t.Fatal(err)
	}

	controller := &Controller{
		addonDir:           addonDir,
		registryURI:        parceRegistryURI("bar.io"),
		KubeconfigProvider: &fakeKubeconfigProvider{},
	}
	manifests, err := controller.getAddonManifests(addon, cluster)
	if err != nil {
		t.Fatal(err)
	}

	if len(manifests) != 1 {
		t.Fatalf("invalid number of manifests returned. Expected 1, Got %d", len(manifests))
	}

	expectedRegURL := "bar.io/test:1.2.3"
	if !strings.Contains(manifests[0].String(), expectedRegURL) {
		t.Fatalf("invalid registryURI returned. Expected \n%s, Got \n%s", expectedRegURL, manifests[0].String())
	}
}

func TestController_getAddonDeploymentManifestsDefault(t *testing.T) {
	cluster := setupTestCluster("10.10.10.0/24")
	addon := setupTestAddon("test")

	addonDir, err := ioutil.TempDir("/tmp", "kubermatic-tests-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(addonDir)

	if err := os.Mkdir(path.Join(addonDir, addon.Spec.Name), 0777); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(path.Join(addonDir, addon.Spec.Name, "testManifest.yaml"), []byte(testManifest1WithDeployment), 0644); err != nil {
		t.Fatal(err)
	}

	controller := &Controller{
		addonDir:           addonDir,
		KubeconfigProvider: &fakeKubeconfigProvider{},
	}
	manifests, err := controller.getAddonManifests(addon, cluster)
	if err != nil {
		t.Fatal(err)
	}

	if len(manifests) != 1 {
		t.Fatalf("invalid number of manifests returned. Expected 1, Got %d", len(manifests))
	}

	expectedRegURL := "foo.io/test:1.2.3"
	if !strings.Contains(manifests[0].String(), expectedRegURL) {
		t.Fatalf("invalid registryURI returned. Expected \n%s, Got \n%s", expectedRegURL, manifests[0].String())
	}
}

func TestController_getAddonManifests(t *testing.T) {
	cluster := setupTestCluster("10.10.10.0/24")
	addon := setupTestAddon("test")
	addonDir, err := ioutil.TempDir("/tmp", "kubermatic-tests-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(addonDir)

	if err := os.Mkdir(path.Join(addonDir, addon.Spec.Name), 0777); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(path.Join(addonDir, addon.Spec.Name, "testManifest.yaml"), []byte(testManifest1), 0644); err != nil {
		t.Fatal(err)
	}
	multilineManifest := fmt.Sprintf(`%s
---
%s`, testManifest2, testManifest3)
	if err := ioutil.WriteFile(path.Join(addonDir, addon.Spec.Name, "testManifest2.yaml"), []byte(multilineManifest), 0644); err != nil {
		t.Fatal(err)
	}

	controller := &Controller{
		addonDir:           addonDir,
		KubeconfigProvider: &fakeKubeconfigProvider{},
	}
	manifests, err := controller.getAddonManifests(addon, cluster)
	if err != nil {
		t.Fatal(err)
	}

	if len(manifests) != 3 {
		t.Fatalf("invalid number of manifests returned. Expected 3, Got %d", len(manifests))
	}

	if manifests[0].String() != testManifest1 {
		t.Fatalf("invalid manifest returned. Expected \n%s, Got \n%s", manifests[0].String(), testManifest1)
	}
	if manifests[1].String() != testManifest2 {
		t.Fatalf("invalid manifest returned. Expected \n%s, Got \n%s", manifests[1].String(), testManifest2)
	}
	if manifests[2].String() != testManifest3 {
		t.Fatalf("invalid manifest returned. Expected \n%s, Got \n%s", manifests[2].String(), testManifest3)
	}
}

func TestController_ensureAddonLabelOnManifests(t *testing.T) {
	controller := &Controller{
		KubeconfigProvider: &fakeKubeconfigProvider{},
	}

	manifests := []*bytes.Buffer{
		bytes.NewBufferString(testManifest1),
	}

	addon := &kubermaticv1.Addon{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-addon",
		},
		Spec: kubermaticv1.AddonSpec{
			Name: "test",
		},
	}
	labeledManifests, err := controller.ensureAddonLabelOnManifests(addon, manifests)
	if err != nil {
		t.Fatal(err)
	}

	if labeledManifests[0].String() != testManifest1WithLabel {
		t.Fatalf("invalid labeled manifest returned. Expected \n%s, Got \n%s", testManifest1WithLabel, labeledManifests[0].String())
	}
}

func TestController_getDeleteCommand(t *testing.T) {
	controller := &Controller{}
	cmd := controller.getDeleteCommand("/opt/kubeconfig", "/opt/manifest.yaml")
	expected := "kubectl --kubeconfig /opt/kubeconfig delete -f /opt/manifest.yaml"
	got := strings.Join(cmd.Args, " ")
	if got != expected {
		t.Fatalf("invalid delete command returned. Expected \n%s, Got \n%s", expected, got)
	}
}

func TestController_getApplyCommand(t *testing.T) {
	controller := &Controller{}
	cmd := controller.getApplyCommand("/opt/kubeconfig", "/opt/manifest.yaml", labels.SelectorFromSet(map[string]string{"foo": "bar"}))
	expected := "kubectl --kubeconfig /opt/kubeconfig apply --prune -f /opt/manifest.yaml -l foo=bar"
	got := strings.Join(cmd.Args, " ")
	if got != expected {
		t.Fatalf("invalid apply command returned. Expected \n%s, Got \n%s", expected, got)
	}
}

func TestController_wasKubectlDeleteSuccessful(t *testing.T) {
	tests := []struct {
		name    string
		out     string
		success bool
	}{
		{
			name: "everything was deleted successfully",
			out: `clusterrolebinding.rbac.authorization.k8s.io "metrics-server:system:auth-delegator" deleted
rolebinding.rbac.authorization.k8s.io "metrics-server-auth-reader" deleted
apiservice.apiregistration.k8s.io "v1beta1.metrics.k8s.io" deleted
service "metrics-server" deleted
clusterrole.rbac.authorization.k8s.io "system:metrics-server" deleted
clusterrolebinding.rbac.authorization.k8s.io "system:metrics-server" deleted`,
			success: true,
		},
		{
			name: "some thing where not found - but everything else was successfully deleted",
			out: `clusterrolebinding.rbac.authorization.k8s.io "metrics-server:system:auth-delegator" deleted
rolebinding.rbac.authorization.k8s.io "metrics-server-auth-reader" deleted
apiservice.apiregistration.k8s.io "v1beta1.metrics.k8s.io" deleted
service "metrics-server" deleted
clusterrole.rbac.authorization.k8s.io "system:metrics-server" deleted
clusterrolebinding.rbac.authorization.k8s.io "system:metrics-server" deleted
Error from server (NotFound): error when deleting "/tmp/cluster-rwhxp9j5j-metrics-server.yaml": serviceaccounts "metrics-server" not found
Error from server (NotFound): error when stopping "/tmp/cluster-rwhxp9j5j-metrics-server.yaml": deployments.extensions "metrics-server" not found`,
			success: true,
		},
		{
			name:    "failed to delete",
			out:     `connection refused`,
			success: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := wasKubectlDeleteSuccessful(test.out)
			if res != test.success {
				t.Errorf("expected to get %v, got %v", test.success, res)
			}
		})
	}
}
