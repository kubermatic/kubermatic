package resources

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubermatic/api"
)

func IsOnCi() bool {
	_, err := os.Stat("../../../config/kubermatic/static/master/")
	if err != nil {
		if os.IsNotExist(err) {
			return true
		}
		panic(err)
	}

	return false
}

func TestLoadServiceFile(t *testing.T) {
	if IsOnCi() {
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
			URL:      "https://jh8j81chn.host2.int.kubermatic.io:8443",
			NodePort: 13000,
		},
	}

	svcs := map[string]string{
		"etcd":      "loadservicefile-etcd-result",
		"apiserver": "loadservicefile-apiserver-result",
	}

	for app, r := range svcs {
		res, err := LoadServiceFile(c, app, "../../../config/kubermatic/static/master/")
		if err != nil {
			t.Fatal(err)
		}

		checkTestResult(t, r, res)
	}
}

func TestLoadIngressFile(t *testing.T) {
	if IsOnCi() {
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
			URL:      "https://jh8j81chn.host2.int.kubermatic.io:8443",
			NodePort: 13000,
		},
	}

	ing := map[string]string{
		"k8sniff": "loadingressfile-k8sniff-result",
	}

	for app, r := range ing {
		res, err := LoadIngressFile(c, app, "../../../config/kubermatic/static/master/", "host2", "int.kubermatic.io")
		if err != nil {
			t.Fatal(err)
		}

		checkTestResult(t, r, res)
	}
}

func TestLoadPVCFile(t *testing.T) {
	if IsOnCi() {
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
			URL:      "https://jh8j81chn.host2.int.kubermatic.io:8443",
			NodePort: 13000,
		},
	}

	ing := map[string]string{
		"etcd": "loadpvcfile-etcd-result",
	}

	for s, r := range ing {
		res, err := LoadPVCFile(c, s, "../../../config/kubermatic/static/master/")
		if err != nil {
			t.Fatal(err)
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
	if IsOnCi() {
		t.Skip("cannot load master files. Maybe on CI?")
	}

	v := &api.MasterVersion{
		Name:                       "1.5.3",
		ID:                         "1.5.3",
		Default:                    true,
		AllowedNodeVersions:        []string{"1.3.0"},
		EtcdOperatorDeploymentYaml: "etcd-dep.yaml",
		ApiserverDeploymentYaml:    "apiserver-dep.yaml",
		ControllerDeploymentYaml:   "controller-manager-dep.yaml",
		SchedulerDeploymentYaml:    "scheduler-dep.yaml",
		Values: map[string]string{
			"k8s-version":  "v1.5.3",
			"etcd-version": "3.0.14-kubeadm",
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
			URL: "https://jh8j81chn.host2.int.kubermatic.io:8443",
		},
	}

	deps := map[string]string{
		v.EtcdOperatorDeploymentYaml: "loaddeploymentfile-etcd-result",
		v.SchedulerDeploymentYaml:    "loaddeploymentfile-scheduler-result",
		v.ControllerDeploymentYaml:   "loaddeploymentfile-controller-manager-result",
		v.ApiserverDeploymentYaml:    "loaddeploymentfile-apiserver-result",
	}

	for s, r := range deps {
		res, err := LoadDeploymentFile(c, v, "../../../config/kubermatic/static/master/", "host2", s)
		if err != nil {
			t.Fatal(err)
		}

		checkTestResult(t, r, res)
	}
}

func TestLoadDeploymentFileAWS(t *testing.T) {
	if IsOnCi() {
		t.Skip("cannot load master files. Maybe on CI?")
	}

	v := &api.MasterVersion{
		Name:                       "1.5.3",
		ID:                         "1.5.3",
		Default:                    true,
		AllowedNodeVersions:        []string{"1.3.0"},
		EtcdOperatorDeploymentYaml: "etcd-dep.yaml",
		ApiserverDeploymentYaml:    "apiserver-dep.yaml",
		ControllerDeploymentYaml:   "controller-manager-dep.yaml",
		SchedulerDeploymentYaml:    "scheduler-dep.yaml",
		Values: map[string]string{
			"k8s-version":  "v1.5.3",
			"etcd-version": "3.0.14-kubeadm",
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
			URL: "https://jh8j81chn.host2.int.kubermatic.io:8443",
		},
	}

	deps := map[string]string{
		v.EtcdOperatorDeploymentYaml: "loaddeploymentfile-etcd-result",
		v.SchedulerDeploymentYaml:    "loaddeploymentfile-scheduler-result",
		v.ControllerDeploymentYaml:   "loaddeploymentfile-aws-controller-manager-result",
		v.ApiserverDeploymentYaml:    "loaddeploymentfile-aws-apiserver-result",
	}

	for s, r := range deps {
		res, err := LoadDeploymentFile(c, v, "../../../config/kubermatic/static/master/", "host2", s)
		if err != nil {
			t.Fatal(err)
		}

		checkTestResult(t, r, res)
	}
}

func TestLoadAwsCloudConfigConfigMap(t *testing.T) {
	if IsOnCi() {
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
				},
			},
		},
		Metadata: api.Metadata{
			Name: "jh8j81chn",
		},
		Address: &api.ClusterAddress{
			URL: "https://jh8j81chn.host2.int.kubermatic.io:8443",
		},
	}

	res, err := LoadAwsCloudConfigConfigMap(c)
	if err != nil {
		t.Fatal(err)
	}

	checkTestResult(t, "loadawscloudconfigconfigmap-result", res)
}
