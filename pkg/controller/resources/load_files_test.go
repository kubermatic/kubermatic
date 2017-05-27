package resources

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"

	"reflect"

	api "github.com/kubermatic/api/pkg/types"
)

func IsOnCi() bool {
	//_, err := os.Stat("../../../config/kubermatic/static/master/")
	//if err != nil {
	//	if os.IsNotExist(err) {
	//		return true
	//	}
	//	panic(err)
	//}
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
			URL: "https://jh8j81chn.host2.int.kubermatic.io:8443",
			ApiserverExternalPort: 13000,
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

		checkEqualJSONResult(t, r, res)
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
			URL: "https://jh8j81chn.host2.int.kubermatic.io:8443",
			ApiserverExternalPort: 13000,
		},
	}

	ing := map[string]string{}

	for s, r := range ing {
		res, err := LoadPVCFile(c, s, "../../../config/kubermatic/static/master/")
		if err != nil {
			t.Fatalf("failed to load %q: %v", s, err)
		}

		checkEqualJSONResult(t, r, res)
	}
}

func checkEqualJSONResult(t *testing.T, resFile string, testObj interface{}) {
	var o1, o2 interface{}

	exp, err := ioutil.ReadFile(filepath.Join("./fixtures", resFile+".json"))
	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal(exp, &o1)
	if err != nil {
		t.Fatal(err)
	}

	res, err := json.Marshal(testObj)
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal(res, &o2)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(o1, o2) {
		t.Fatalf("\nExpected to get \n%v \nFrom(%q)\n Got \n%v \nfrom %T", string(exp), resFile, string(res), testObj)
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
			URL: "https://jh8j81chn.host2.int.kubermatic.io:8443",
			ApiserverExternalPort: 3000,
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

		checkEqualJSONResult(t, r, res)
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
			URL: "https://jh8j81chn.host2.int.kubermatic.io:8443",
			ApiserverExternalPort: 3000,
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

		checkEqualJSONResult(t, r, res)
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
					RouteTableID:     "my-routetableID",
					SubnetID:         "my-subnetID",
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

	checkEqualJSONResult(t, "loadawscloudconfigconfigmap-result", res)
}

func TestLoadServiceAccountFile(t *testing.T) {
	if IsOnCi() {
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

		checkEqualJSONResult(t, r, res)
	}
}

func TestLoadClusterRoleBindingFile(t *testing.T) {
	if IsOnCi() {
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

		checkEqualJSONResult(t, r, res)
	}
}
