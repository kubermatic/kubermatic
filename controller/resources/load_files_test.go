package resources

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/test"
)

const masterFilesPath = "../../../config/kubermatic/static/master/"

func TestLoadServiceFile(t *testing.T) {
	if test.IsOnCi(masterFilesPath) {
		t.Skip("cannot load master files. Maybe on CI?")
	}

	c := &api.Cluster{
		Spec: api.ClusterSpec{
			Cloud: &api.CloudSpec{
				BareMetal: &api.BareMetalCloudSpec{
					Name: "test",
				},
			},
		},
		Metadata: api.Metadata{
			Name: "de-test-01",
		},
		Address: &api.ClusterAddress{
			URL:          "https://jh8j81chn.host2.int.kubermatic.io:8443",
			ExternalName: "jh8j81chn.host2.int.kubermatic.io",
			ExternalPort: 8443,
		},
	}

	svcs := map[string]string{
		"etcd":      "loadservicefile-etcd-result",
		"apiserver": "loadservicefile-apiserver-result",
	}

	for app, r := range svcs {
		res, err := LoadServiceFile(c, app, "../../../config/kubermatic/static/master/")
		if err != nil {
			t.Fatalf("failed to load %q: %v", app, err)
		}

		checkTestResult(t, r, res)
	}
}

func TestLoadPVCFile(t *testing.T) {
	if test.IsOnCi(masterFilesPath) {
		t.Skip("cannot load master files. Maybe on CI?")
	}

	c := &api.Cluster{
		Spec: api.ClusterSpec{
			Cloud: &api.CloudSpec{
				BareMetal: &api.BareMetalCloudSpec{
					Name: "test",
				},
			},
		},
		Metadata: api.Metadata{
			Name: "jh8j81chn",
		},
		Address: &api.ClusterAddress{
			URL:          "https://jh8j81chn.host2.int.kubermatic.io:8443",
			ExternalName: "jh8j81chn.host2.int.kubermatic.io",
			ExternalPort: 8443,
		},
	}

	ing := map[string]string{}

	for s, r := range ing {
		res, err := LoadPVCFile(c, s, "../../../config/kubermatic/static/master/")
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
	if test.IsOnCi(masterFilesPath) {
		t.Skip("cannot load master files. Maybe on CI?")
	}

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
		},
	}

	c := &api.Cluster{
		Spec: api.ClusterSpec{
			Cloud: &api.CloudSpec{
				BareMetal: &api.BareMetalCloudSpec{
					Name: "test",
				},
			},
		},
		Metadata: api.Metadata{
			Name: "jh8j81chn",
		},
		Address: &api.ClusterAddress{
			URL:          "https://jh8j81chn.host2.int.kubermatic.io:8443",
			ExternalName: "jh8j81chn.host2.int.kubermatic.io",
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
		res, err := LoadDeploymentFile(c, v, "../../../config/kubermatic/static/master/", "host2", s)
		if err != nil {
			t.Fatalf("failed to load %q: %v", s, err)
		}

		checkTestResult(t, r, res)
	}
}

func TestLoadDeploymentFileAWS(t *testing.T) {
	if test.IsOnCi(masterFilesPath) {
		t.Skip("cannot load master files. Maybe on CI?")
	}

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
		},
	}

	c := &api.Cluster{
		Spec: api.ClusterSpec{
			Cloud: &api.CloudSpec{
				AWS: &api.AWSCloudSpec{
					AccessKeyID:      "my_AccessKeyID",
					SecretAccessKey:  "my_SecretAccessKey",
					VPCId:            "my_VPCId",
					AvailabilityZone: "my_AvailabilityZone",
				},
			},
		},
		Metadata: api.Metadata{
			Name: "jh8j81chn",
		},
		Address: &api.ClusterAddress{
			URL:          "https://jh8j81chn.host2.int.kubermatic.io:8443",
			ExternalName: "jh8j81chn.host2.int.kubermatic.io",
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
		res, err := LoadDeploymentFile(c, v, "../../../config/kubermatic/static/master/", "host2", s)
		if err != nil {
			t.Fatalf("failed to load %q: %v", s, err)
		}

		checkTestResult(t, r, res)
	}
}

func TestLoadAwsCloudConfigConfigMap(t *testing.T) {
	if test.IsOnCi(masterFilesPath) {
		t.Skip("cannot load master files. Maybe on CI?")
	}

	c := &api.Cluster{
		Spec: api.ClusterSpec{
			Cloud: &api.CloudSpec{
				AWS: &api.AWSCloudSpec{
					AccessKeyID:      "my_AccessKeyID",
					SecretAccessKey:  "my_SecretAccessKey",
					VPCId:            "my_VPCId",
					AvailabilityZone: "my_AvailabilityZone",
					RouteTableID:     "my-routetableID",
					SubnetID:         "my-subnetID",
				},
			},
		},
		Metadata: api.Metadata{
			Name: "jh8j81chn",
		},
		Address: &api.ClusterAddress{
			URL:          "https://jh8j81chn.host2.int.kubermatic.io:8443",
			ExternalName: "jh8j81chn.host2.int.kubermatic.io",
			ExternalPort: 8443,
		},
	}

	res, err := LoadAwsCloudConfigConfigMap(c)
	if err != nil {
		t.Fatal(err)
	}

	checkTestResult(t, "loadawscloudconfigconfigmap-result", res)
}

func TestLoadServiceAccountFile(t *testing.T) {
	if test.IsOnCi(masterFilesPath) {
		t.Skip("cannot load master files. Maybe on CI?")
	}

	apps := map[string]string{
		"etcd-operator": "loadserviceaccountfile-etcd-operator-result",
	}

	for app, r := range apps {
		res, err := LoadServiceAccountFile(app, "../../../config/kubermatic/static/master/")
		if err != nil {
			t.Fatalf("failed to load %q: %v", app, err)
		}

		checkTestResult(t, r, res)
	}
}

func TestLoadClusterRoleBindingFile(t *testing.T) {
	if test.IsOnCi(masterFilesPath) {
		t.Skip("cannot load master files. Maybe on CI?")
	}

	apps := map[string]string{
		"etcd-operator": "loadclusterrolebindingfile-etcd-operator-result",
	}

	for app, r := range apps {
		res, err := LoadClusterRoleBindingFile("cluster-jh8j81chn", app, "../../../config/kubermatic/static/master/")
		if err != nil {
			t.Fatalf("failed to load %q: %v", app, err)
		}

		checkTestResult(t, r, res)
	}
}
