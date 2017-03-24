package cluster

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubermatic/api"
	"os"
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
	cc := &clusterController{
		masterResourcesPath: "../../../config/kubermatic/static/master/",
		dc:                  "host2",
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

	for s, r := range svcs {
		res, err := loadServiceFile(cc, c, s)
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
	cc := &clusterController{
		masterResourcesPath: "../../../config/kubermatic/static/master/",
		dc:                  "host2",
		externalURL:         "int.kubermatic.io",
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

	for s, r := range ing {
		res, err := loadIngressFile(cc, c, s)
		if err != nil {
			t.Fatal(err)
		}

		checkTestResult(t, r, res)
	}
}

func TestLoadDeploymentFile(t *testing.T) {
	if IsOnCi() {
		t.Skip("cannot load master files. Maybe on CI?")
	}
	cc := &clusterController{
		masterResourcesPath: "../../../config/kubermatic/static/master/",
		dc:                  "host2",
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
		"etcd":               "loaddeploymentfile-etcd-result",
		"scheduler":          "loaddeploymentfile-scheduler-result",
		"controller-manager": "loaddeploymentfile-controller-manager-result",
		"apiserver":          "loaddeploymentfile-apiserver-result",
	}

	for s, r := range deps {
		res, err := loadDeploymentFile(cc, c, "dc", s)
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
	cc := &clusterController{
		masterResourcesPath: "../../../config/kubermatic/static/master/",
		dc:                  "host2",
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
		"etcd":               "loaddeploymentfile-etcd-result",
		"scheduler":          "loaddeploymentfile-scheduler-result",
		"controller-manager": "loaddeploymentfile-aws-controller-manager-result",
		"apiserver":          "loaddeploymentfile-aws-apiserver-result",
	}

	for s, r := range deps {
		res, err := loadDeploymentFile(cc, c, "dc", s)
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
	cc := &clusterController{
		masterResourcesPath: "../../../config/kubermatic/static/master/",
		dc:                  "host2",
		externalURL:         "int.kubermatic.io",
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
		res, err := loadPVCFile(cc, c, s)
		if err != nil {
			t.Fatal(err)
		}

		checkTestResult(t, r, res)
	}
}

func checkTestResult(t *testing.T, resFile string, testObj interface{}) {
	exp, err := ioutil.ReadFile(filepath.Join("./fixtures/templates", resFile+".json"))
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
		t.Fatalf("Expected to get %v (%q), got %v from %T", expStr, resFile, resStr, testObj)
	}
}
