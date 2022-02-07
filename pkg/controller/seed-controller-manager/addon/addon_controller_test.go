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

package addon

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/controller/operator/defaults"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/util/kubectl"
	"k8c.io/kubermatic/v2/pkg/version/cni"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var testManifests = []string{
	`kind: ConfigMap
apiVersion: v1
metadata:
  name: test1
  namespace: kube-system
  labels:
    app: test
data:
  foo: bar
`,

	`kind: ConfigMap
apiVersion: v1
metadata:
  name: test2
  namespace: kube-system
  labels:
    app: test
data:
  foo: bar`,

	`kind: ConfigMap
apiVersion: v1
metadata:
  name: test3
  namespace: kube-system
  labels:
    app: test
data:
  foo: bar
`}

const (
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
        image: {{ Registry "foo.io" }}/test:1.2.3
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
  clusterIP: {{ .Cluster.Network.DNSClusterIP }}
  clusterCIDR: "{{ first .Cluster.Network.PodCIDRBlocks }}"
`
)

var (
	// testManifest1 & testManifest3 have a linebreak at the end, testManifest2 not.
	combinedTestManifest = fmt.Sprintf("%s---\n%s\n---\n%s", testManifests[0], testManifests[1], testManifests[2])
)

type fakeKubeconfigProvider struct{}

func (f *fakeKubeconfigProvider) GetAdminKubeconfig(_ context.Context, c *kubermaticv1.Cluster) ([]byte, error) {
	return []byte("foo"), nil
}

func (f *fakeKubeconfigProvider) GetClient(_ context.Context, c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error) {
	return nil, errors.New("not implemented")
}

func TestController_combineManifests(t *testing.T) {
	controller := &Reconciler{}

	var manifests []*bytes.Buffer
	for _, m := range testManifests {
		manifests = append(manifests, bytes.NewBufferString(m))
	}

	manifest := controller.combineManifests(manifests)

	if manifest.String() != combinedTestManifest {
		t.Fatalf("invalid combined manifest returned. Expected \n%s, Got \n%s", combinedTestManifest, manifest.String())
	}
}

func setupTestCluster(cidrBlock string) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cluster",
		},
		Spec: kubermaticv1.ClusterSpec{
			Version: *semver.NewSemverOrDie("v1.11.1"),
			ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
				Services: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{
						cidrBlock,
					},
				},

				Pods: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{
						"172.25.0.0/16",
					},
				},
				DNSDomain: "cluster.local",
				ProxyMode: resources.IPVSProxyMode,
			},
			CNIPlugin: &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCanal,
				Version: cni.GetDefaultCNIPluginVersion(kubermaticv1.CNIPluginTypeCanal),
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
	cluster := setupTestCluster("10.240.16.0/20")
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

	log := kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar()
	ctx := context.Background()

	controller := &Reconciler{
		kubernetesAddonDir: addonDir,
		KubeconfigProvider: &fakeKubeconfigProvider{},
	}
	manifests, err := controller.getAddonManifests(ctx, log, addon, cluster)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifests) != 1 {
		t.Fatalf("invalid number of manifests returned. Expected 1, Got %d", len(manifests))
	}

	expectedIP := "10.240.16.10"
	if !strings.Contains(string(manifests[0].Raw), expectedIP) {
		t.Fatalf("invalid IP returned. Expected \n%s, Got \n%s", expectedIP, manifests[0].String())
	}
	expectedCIDR := "172.25.0.0/16"
	if !strings.Contains(string(manifests[0].Raw), expectedCIDR) {
		t.Fatalf("invalid CIDR returned. Expected \n%s, Got \n%s", expectedCIDR, manifests[0].String())
	}

	cluster = setupTestCluster("172.25.0.0/16")
	manifests, err = controller.getAddonManifests(ctx, log, addon, cluster)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifests) != 1 {
		t.Fatalf("invalid number of manifests returned. Expected 1, Got %d", len(manifests))
	}
	expectedIP = "172.25.0.10"
	if !strings.Contains(string(manifests[0].Raw), expectedIP) {
		t.Fatalf("invalid registryURI returned. Expected \n%s, Got \n%s", expectedIP, manifests[0].String())
	}
}

func TestController_getAddonDeploymentManifests(t *testing.T) {
	cluster := setupTestCluster("10.240.16.0/20")
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

	log := kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar()

	controller := &Reconciler{
		kubernetesAddonDir: addonDir,
		overwriteRegistry:  "bar.io",
		KubeconfigProvider: &fakeKubeconfigProvider{},
	}
	manifests, err := controller.getAddonManifests(context.Background(), log, addon, cluster)
	if err != nil {
		t.Fatal(err)
	}

	if len(manifests) != 1 {
		t.Fatalf("invalid number of manifests returned. Expected 1, Got %d", len(manifests))
	}

	expectedRegURL := "bar.io/test:1.2.3"
	if !strings.Contains(string(manifests[0].Raw), expectedRegURL) {
		t.Fatalf("invalid registryURI returned. Expected \n%s, Got \n%s", expectedRegURL, manifests[0].String())
	}
}

func TestController_getAddonDeploymentManifestsDefault(t *testing.T) {
	cluster := setupTestCluster("10.240.16.0/20")
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

	log := kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar()

	controller := &Reconciler{
		kubernetesAddonDir: addonDir,
		KubeconfigProvider: &fakeKubeconfigProvider{},
	}
	manifests, err := controller.getAddonManifests(context.Background(), log, addon, cluster)
	if err != nil {
		t.Fatal(err)
	}

	if len(manifests) != 1 {
		t.Fatalf("invalid number of manifests returned. Expected 1, Got %d", len(manifests))
	}

	expectedRegURL := "foo.io/test:1.2.3"
	if !strings.Contains(string(manifests[0].Raw), expectedRegURL) {
		t.Fatalf("invalid registryURI returned. Expected \n%s, Got \n%s", expectedRegURL, manifests[0].String())
	}
}

func TestController_getAddonManifests(t *testing.T) {
	cluster := setupTestCluster("10.240.16.0/20")
	addon := setupTestAddon("test")
	addonDir, err := ioutil.TempDir("/tmp", "kubermatic-tests-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(addonDir)

	if err := os.Mkdir(path.Join(addonDir, addon.Spec.Name), 0777); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(path.Join(addonDir, addon.Spec.Name, "testManifest.yaml"), []byte(testManifests[0]), 0644); err != nil {
		t.Fatal(err)
	}

	log := kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar()

	multilineManifest := fmt.Sprintf(`%s
---
%s`, testManifests[1], testManifests[2])
	if err := ioutil.WriteFile(path.Join(addonDir, addon.Spec.Name, "testManifest2.yaml"), []byte(multilineManifest), 0644); err != nil {
		t.Fatal(err)
	}

	controller := &Reconciler{
		kubernetesAddonDir: addonDir,
		KubeconfigProvider: &fakeKubeconfigProvider{},
	}
	manifests, err := controller.getAddonManifests(context.Background(), log, addon, cluster)
	if err != nil {
		t.Fatal(err)
	}

	if len(manifests) != 3 {
		t.Fatalf("invalid number of manifests returned. Expected 3, Got %d", len(manifests))
	}

	for idx := range testManifests {
		testManifestDecoder := kyaml.NewYAMLToJSONDecoder(bytes.NewBuffer([]byte(testManifests[idx])))
		expected := runtime.RawExtension{}
		if err := testManifestDecoder.Decode(&expected); err != nil {
			t.Fatalf("decoding of test manifest failed: %v", err)
		}

		if string(expected.Raw) != string(manifests[idx].Raw) {
			t.Errorf("Invalid manifest returned, expected \n%q\n, got \n%q", string(expected.Raw), string(manifests[idx].Raw))
		}
	}
}

func TestController_ensureAddonLabelOnManifests(t *testing.T) {
	controller := &Reconciler{
		KubeconfigProvider: &fakeKubeconfigProvider{},
	}

	manifest := runtime.RawExtension{}
	decoder := kyaml.NewYAMLToJSONDecoder(bytes.NewBuffer([]byte(testManifests[0])))
	if err := decoder.Decode(&manifest); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}

	addon := &kubermaticv1.Addon{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-addon",
		},
		Spec: kubermaticv1.AddonSpec{
			Name: "test",
		},
	}
	labeledManifests, err := controller.ensureAddonLabelOnManifests(addon, []runtime.RawExtension{manifest})
	if err != nil {
		t.Fatal(err)
	}
	if labeledManifests[0].String() != testManifest1WithLabel {
		t.Fatalf("invalid labeled manifest returned. Expected \n%q, Got \n%q", testManifest1WithLabel, labeledManifests[0].String())
	}
}

func TestController_getApplyCommand(t *testing.T) {
	controller := &Reconciler{}
	clusterVersion := defaults.DefaultKubernetesVersioning.Default.Semver()

	binary, err := kubectl.BinaryForClusterVersion(clusterVersion)
	if err != nil {
		t.Fatalf("Should be able to determine a kubectl binary for %q, but got %v", clusterVersion, err)
	}

	cmd, err := controller.getApplyCommand(context.Background(), "/opt/kubeconfig", "/opt/manifest.yaml", labels.SelectorFromSet(map[string]string{"foo": "bar"}), clusterVersion)
	if err != nil {
		t.Fatalf("Should be able to determine the command, but got %v", err)
	}

	expected := fmt.Sprintf("%s --kubeconfig /opt/kubeconfig apply --prune --filename /opt/manifest.yaml --selector foo=bar", binary)
	got := strings.Join(cmd.Args, " ")
	if got != expected {
		t.Fatalf("invalid apply command returned. Expected \n%s, Got \n%s", expected, got)
	}
}

func TestHugeManifest(t *testing.T) {
	log := kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar()
	cluster := setupTestCluster("10.240.16.0/20")
	addon := setupTestAddon("istio")
	r := &Reconciler{
		kubernetesAddonDir: "./testdata",
		KubeconfigProvider: &fakeKubeconfigProvider{},
	}
	if _, _, _, err := r.setupManifestInteraction(context.Background(), log, addon, cluster); err != nil {
		t.Fatalf("failed to setup manifest interaction: %v", err)
	}
}
