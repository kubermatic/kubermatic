package resources

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubermatic/kubermatic/api"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	masterResourcePath = "../../../../config/kubermatic/static/master/"
)

func TestLoadServiceFile(t *testing.T) {
	c := &kubermaticv1.Cluster{
		Spec: kubermaticv1.ClusterSpec{
			Cloud: &kubermaticv1.CloudSpec{
				BareMetal: &kubermaticv1.BareMetalCloudSpec{
					Name: "test",
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "de-test-01",
		},
		Address: &kubermaticv1.ClusterAddress{
			URL:          "https://jh8j81chn.europe-west3-c.dev.kubermatic.io:8443",
			ExternalName: "jh8j81chn.europe-west3-c.dev.kubermatic.io",
			ExternalPort: 8443,
		},
	}

	svcs := map[string]string{
		"apiserver": "loadservicefile-apiserver-result",
	}

	for app, r := range svcs {
		res, err := LoadServiceFile(c, app, masterResourcePath)
		if err != nil {
			t.Fatalf("failed to load %q: %v", app, err)
		}

		checkTestResult(t, r, res)
	}
}

func TestLoadPVCFile(t *testing.T) {
	c := &kubermaticv1.Cluster{
		Spec: kubermaticv1.ClusterSpec{
			Cloud: &kubermaticv1.CloudSpec{
				BareMetal: &kubermaticv1.BareMetalCloudSpec{
					Name: "test",
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "de-test-01",
		},
		Address: &kubermaticv1.ClusterAddress{
			URL:          "https://jh8j81chn.europe-west3-c.dev.kubermatic.io:8443",
			ExternalName: "jh8j81chn.europe-west3-c.dev.kubermatic.io",
			ExternalPort: 8443,
		},
	}

	ing := map[string]string{}

	for s, r := range ing {
		res, err := LoadPVCFile(c, s, masterResourcePath)
		if err != nil {
			t.Fatalf("failed to load %q: %v", s, err)
		}

		checkTestResult(t, r, res)
	}
}

func checkTestResult(t *testing.T, resFile string, testObj interface{}) {
	exp, err := ioutil.ReadFile(filepath.Join("./fixtures", resFile+".json"))
	if err != nil {
		t.Fatal(err)
	}

	res, err := json.Marshal(testObj)
	if err != nil {
		t.Fatal(err)
	}

	resStr := strings.TrimSpace(string(res))
	expStr := strings.TrimSpace(string(exp))

	if resStr != expStr {
		t.Fatalf("\nExpected to get \n%v (%q)\n Got \n%v \nfrom %T", expStr, resFile, resStr, testObj)
	}
}

func TestLoadDeploymentFile(t *testing.T) {
	v := &api.MasterVersion{
		Name:                       "1.5.3",
		ID:                         "1.5.3",
		Default:                    true,
		AllowedNodeVersions:        []string{"1.3.0"},
		EtcdOperatorDeploymentYaml: "etcd-operator-dep.yaml",
		ApiserverDeploymentYaml:    "apiserver-dep.yaml",
		ControllerDeploymentYaml:   "controller-manager-dep.yaml",
		SchedulerDeploymentYaml:    "scheduler-dep.yaml",
		Values: map[string]string{
			"k8s-version":           "v1.5.3",
			"etcd-operator-version": "v0.2.4fix",
			"etcd-cluster-version":  "3.1.5",
			"pod-network-bridge":    "v0.1",
		},
	}

	c := &kubermaticv1.Cluster{
		Spec: kubermaticv1.ClusterSpec{
			Cloud: &kubermaticv1.CloudSpec{
				BareMetal: &kubermaticv1.BareMetalCloudSpec{
					Name: "test",
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "de-test-01",
		},
		Address: &kubermaticv1.ClusterAddress{
			URL:          "https://jh8j81chn.europe-west3-c.dev.kubermatic.io:8443",
			ExternalName: "jh8j81chn.europe-west3-c.dev.kubermatic.io",
			ExternalPort: 8443,
		},
	}

	deps := map[string]string{
		v.EtcdOperatorDeploymentYaml: "loaddeploymentfile-etcd-operator-result",
		v.SchedulerDeploymentYaml:    "loaddeploymentfile-scheduler-result",
		v.ControllerDeploymentYaml:   "loaddeploymentfile-controller-manager-result",
		v.ApiserverDeploymentYaml:    "loaddeploymentfile-apiserver-result",
	}

	for s, r := range deps {
		res, err := LoadDeploymentFile(c, v, masterResourcePath, "us-central1", s)
		if err != nil {
			t.Fatalf("failed to load %q: %v", s, err)
		}

		checkTestResult(t, r, res)
	}
}

func TestLoadDeploymentFileAWS(t *testing.T) {
	v := &api.MasterVersion{
		Name:                       "1.5.3",
		ID:                         "1.5.3",
		Default:                    true,
		AllowedNodeVersions:        []string{"1.3.0"},
		EtcdOperatorDeploymentYaml: "etcd-operator-dep.yaml",
		ApiserverDeploymentYaml:    "apiserver-dep.yaml",
		ControllerDeploymentYaml:   "controller-manager-dep.yaml",
		SchedulerDeploymentYaml:    "scheduler-dep.yaml",
		Values: map[string]string{
			"k8s-version":           "v1.5.3",
			"etcd-operator-version": "v0.2.4fix",
			"etcd-cluster-version":  "3.1.5",
			"pod-network-bridge":    "v0.1",
		},
	}

	c := &kubermaticv1.Cluster{
		Spec: kubermaticv1.ClusterSpec{
			Cloud: &kubermaticv1.CloudSpec{
				AWS: &kubermaticv1.AWSCloudSpec{
					AccessKeyID:      "my_AccessKeyID",
					SecretAccessKey:  "my_SecretAccessKey",
					VPCID:            "my_VPCId",
					AvailabilityZone: "my_AvailabilityZone",
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "de-test-01",
		},
		Address: &kubermaticv1.ClusterAddress{
			URL:          "https://jh8j81chn.europe-west3-c.dev.kubermatic.io:8443",
			ExternalName: "jh8j81chn.europe-west3-c.dev.kubermatic.io",
			ExternalPort: 8443,
		},
	}

	deps := map[string]string{
		v.EtcdOperatorDeploymentYaml: "loaddeploymentfile-etcd-operator-result",
		v.SchedulerDeploymentYaml:    "loaddeploymentfile-scheduler-result",
		v.ControllerDeploymentYaml:   "loaddeploymentfile-aws-controller-manager-result",
		v.ApiserverDeploymentYaml:    "loaddeploymentfile-aws-apiserver-result",
	}

	for s, r := range deps {
		res, err := LoadDeploymentFile(c, v, masterResourcePath, "us-central1", s)
		if err != nil {
			t.Fatalf("failed to load %q: %v", s, err)
		}

		checkTestResult(t, r, res)
	}
}

func TestLoadAwsCloudConfigConfigMap(t *testing.T) {
	c := &kubermaticv1.Cluster{
		Spec: kubermaticv1.ClusterSpec{
			Cloud: &kubermaticv1.CloudSpec{
				AWS: &kubermaticv1.AWSCloudSpec{
					AccessKeyID:      "my_AccessKeyID",
					SecretAccessKey:  "my_SecretAccessKey",
					VPCID:            "my_VPCId",
					AvailabilityZone: "my_AvailabilityZone",
					RouteTableID:     "my-routetableID",
					SubnetID:         "my-subnetID",
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "de-test-01",
		},
		Address: &kubermaticv1.ClusterAddress{
			URL:          "https://jh8j81chn.europe-west3-c.dev.kubermatic.io:8443",
			ExternalName: "jh8j81chn.europe-west3-c.dev.kubermatic.io",
			ExternalPort: 8443,
		},
	}

	res, err := LoadAwsCloudConfigConfigMap(c, nil)
	if err != nil {
		t.Fatal(err)
	}

	checkTestResult(t, "loadawscloudconfigconfigmap-result", res)
}

func TestLoadServiceAccountFile(t *testing.T) {
	apps := map[string]string{
		"etcd-operator": "loadserviceaccountfile-etcd-operator-result",
	}

	for app, r := range apps {
		res, err := LoadServiceAccountFile(app, masterResourcePath)
		if err != nil {
			t.Fatalf("failed to load %q: %v", app, err)
		}

		checkTestResult(t, r, res)
	}
}

func TestLoadClusterRoleBindingFile(t *testing.T) {
	apps := map[string]string{
		"etcd-operator": "loadclusterrolebindingfile-etcd-operator-result",
	}

	for app, r := range apps {
		res, err := LoadClusterRoleBindingFile("cluster-jh8j81chn", app, masterResourcePath)
		if err != nil {
			t.Fatalf("failed to load %q: %v", app, err)
		}

		checkTestResult(t, r, res)
	}
}
