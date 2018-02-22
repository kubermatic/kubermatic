package resources

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/pmezard/go-difflib/difflib"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	masterResourcePath = "../../../../config/kubermatic/static/master/"
)

func TestLoadServiceFile(t *testing.T) {
	c := &kubermaticv1.Cluster{
		Spec: kubermaticv1.ClusterSpec{
			Cloud: &kubermaticv1.CloudSpec{
				BringYourOwn: &kubermaticv1.BringYourOwnCloudSpec{},
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
				BringYourOwn: &kubermaticv1.BringYourOwnCloudSpec{},
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
	path := filepath.Join("./fixtures", resFile+".yaml")
	jsonRes, err := json.Marshal(testObj)
	if err != nil {
		t.Fatal(err)
	}
	res, err := yaml.JSONToYAML(jsonRes)
	if err != nil {
		t.Fatal(err)
	}

	exp, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	resStr := strings.TrimSpace(string(res))
	expStr := strings.TrimSpace(string(exp))

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(expStr),
		B:        difflib.SplitLines(resStr),
		FromFile: "Fixture",
		ToFile:   "Current",
		Context:  3,
	}
	diffStr, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		t.Fatal(err)
	}

	if resStr != expStr {
		t.Errorf("\nDeployment file changed and does not match fixture(%q) anymore: \n %s\n\nMake sure you update all fixtures after changing templates.", path, diffStr)
	}
}

func TestLoadDeploymentFile(t *testing.T) {
	versions := []*apiv1.MasterVersion{
		{
			Name:                         "1.7.0",
			ID:                           "1.7.0",
			Default:                      true,
			AllowedNodeVersions:          []string{"1.3.0"},
			EtcdOperatorDeploymentYaml:   "etcd-operator-dep.yaml",
			ApiserverDeploymentYaml:      "apiserver-dep.yaml",
			ControllerDeploymentYaml:     "controller-manager-dep.yaml",
			SchedulerDeploymentYaml:      "scheduler-dep.yaml",
			AddonManagerDeploymentYaml:   "addon-manager-dep.yaml",
			NodeControllerDeploymentYaml: "node-controller-dep.yaml",
			Values: map[string]string{
				"k8s-version":           "v1.7.11",
				"etcd-operator-version": "v0.6.0",
				"etcd-cluster-version":  "3.2.7",
				"kube-machine-version":  "v0.2.3",
				"addon-manager-version": "v1.7.6",
				"pod-network-bridge":    "v0.1",
			},
		},
		{
			Name:                         "1.8.0",
			ID:                           "1.8.0",
			Default:                      true,
			AllowedNodeVersions:          []string{"1.3.0"},
			EtcdOperatorDeploymentYaml:   "etcd-operator-dep.yaml",
			ApiserverDeploymentYaml:      "apiserver-dep.yaml",
			ControllerDeploymentYaml:     "controller-manager-dep.yaml",
			SchedulerDeploymentYaml:      "scheduler-dep.yaml",
			AddonManagerDeploymentYaml:   "addon-manager-dep.yaml",
			NodeControllerDeploymentYaml: "node-controller-dep.yaml",
			Values: map[string]string{
				"k8s-version":           "v1.8.5",
				"etcd-operator-version": "v0.6.0",
				"etcd-cluster-version":  "3.2.7",
				"kube-machine-version":  "v0.2.3",
				"addon-manager-version": "v1.8.2",
				"pod-network-bridge":    "v0.2",
			},
		},
		{
			Name:                         "1.9.0",
			ID:                           "1.9.0",
			Default:                      true,
			AllowedNodeVersions:          []string{"1.3.0"},
			EtcdOperatorDeploymentYaml:   "etcd-operator-dep.yaml",
			ApiserverDeploymentYaml:      "apiserver-dep.yaml",
			ControllerDeploymentYaml:     "controller-manager-dep.yaml",
			SchedulerDeploymentYaml:      "scheduler-dep.yaml",
			AddonManagerDeploymentYaml:   "addon-manager-dep.yaml",
			NodeControllerDeploymentYaml: "node-controller-dep.yaml",
			Values: map[string]string{
				"k8s-version":           "v1.9.0",
				"etcd-operator-version": "v0.6.0",
				"etcd-cluster-version":  "3.2.7",
				"kube-machine-version":  "v0.2.3",
				"addon-manager-version": "v1.9.0",
				"pod-network-bridge":    "v0.2",
			},
		},
	}

	clouds := map[string]*kubermaticv1.CloudSpec{
		"digitalocean": {
			Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
				Token: "do-token",
			},
		},
		"aws": {
			AWS: &kubermaticv1.AWSCloudSpec{
				AccessKeyID:         "aws-access-key-id",
				SecretAccessKey:     "aws-secret-access-key",
				AvailabilityZone:    "aws-availability-zone",
				InstanceProfileName: "aws-instance-profile-name",
				RoleName:            "aws-role-name",
				RouteTableID:        "aws-route-table-id",
				SecurityGroup:       "aws-security-group",
				SubnetID:            "aws-subnet-id",
				VPCID:               "aws-vpn-id",
			},
		},
		"openstack": {
			Openstack: &kubermaticv1.OpenstackCloudSpec{
				NetworkCreated:       true,
				SecurityGroupCreated: true,
				SubnetID:             "openstack-subnet-id",
				Username:             "openstack-username",
				Tenant:               "openstack-tenant",
				Domain:               "openstack-domain",
				FloatingIPPool:       "openstack-floating-ip-pool",
				Network:              "openstack-network",
				Password:             "openstack-password",
				RouterID:             "openstack-router-id",
				SecurityGroups:       "openstack-security-group1,openstack-security-group2",
			},
		},
		"bringyourown": {
			BringYourOwn: &kubermaticv1.BringYourOwnCloudSpec{},
		},
	}

	for _, version := range versions {
		for provider, cloudspec := range clouds {
			deps := map[string]string{
				version.EtcdOperatorDeploymentYaml:   fmt.Sprintf("deployment-%s-%s-etcd-operator", provider, version.ID),
				version.SchedulerDeploymentYaml:      fmt.Sprintf("deployment-%s-%s-scheduler", provider, version.ID),
				version.ControllerDeploymentYaml:     fmt.Sprintf("deployment-%s-%s-controller-manager", provider, version.ID),
				version.ApiserverDeploymentYaml:      fmt.Sprintf("deployment-%s-%s-apiserver", provider, version.ID),
				version.NodeControllerDeploymentYaml: fmt.Sprintf("deployment-%s-%s-node-controller", provider, version.ID),
				version.AddonManagerDeploymentYaml:   fmt.Sprintf("deployment-%s-%s-addon-manager", provider, version.ID),
			}

			cluster := &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Cloud:              cloudspec,
					SeedDatacenterName: "us-central1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "de-test-01",
				},
				Address: &kubermaticv1.ClusterAddress{
					URL:          "https://jh8j81chn.europe-west3-c.dev.kubermatic.io:30000",
					ExternalName: "jh8j81chn.europe-west3-c.dev.kubermatic.io",
					ExternalPort: 30000,
				},
			}
			for s, r := range deps {
				res, err := LoadDeploymentFile(cluster, version, masterResourcePath, s)
				if err != nil {
					t.Fatalf("failed to load %q: %v", s, err)
				}

				checkTestResult(t, r, res)
			}
		}
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

	res, err := LoadAwsCloudConfigConfigMap(c, nil, nil)
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

func TestLoadOpenstackCloudConfigConfigMap(t *testing.T) {
	c := &kubermaticv1.Cluster{
		Spec: kubermaticv1.ClusterSpec{
			Cloud: &kubermaticv1.CloudSpec{
				Openstack: &kubermaticv1.OpenstackCloudSpec{
					SubnetID:       "openstack-subnetid",
					Username:       "openstack-username",
					Tenant:         "openstack-tenant",
					SecurityGroups: "openstack-security-groups",
					RouterID:       "openstack-router-id",
					Password:       "openstack-password",
					Network:        "openstack-network",
					Domain:         "openstack-domain",
					FloatingIPPool: "openstack-floating-ip-pool",
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "de-test-01",
		},
		Address: &kubermaticv1.ClusterAddress{
			URL:          "https://jh8j81chn.europe-west3-c.dev.kubermatic.io:8443",
			ExternalName: "jh8j81chn.europe-west3-c.dev.kubermatic.io",
			ExternalPort: 8002,
		},
	}

	dc := &provider.DatacenterMeta{
		Spec: provider.DatacenterSpec{
			Openstack: &provider.OpenstackSpec{
				AvailabilityZone: "az1",
				AuthURL:          "https://my.openstack.foo/v3",
				DNSServers:       []string{"8.8.8.8", "8.8.4.4"},
				IgnoreVolumeAZ:   true,
				Region:           "os-region",
			},
		},
	}

	version := &apiv1.MasterVersion{
		Name:                         "1.9.0",
		ID:                           "1.9.0",
		Default:                      true,
		AllowedNodeVersions:          []string{"1.3.0"},
		EtcdOperatorDeploymentYaml:   "etcd-operator-dep.yaml",
		ApiserverDeploymentYaml:      "apiserver-dep.yaml",
		ControllerDeploymentYaml:     "controller-manager-dep.yaml",
		SchedulerDeploymentYaml:      "scheduler-dep.yaml",
		AddonManagerDeploymentYaml:   "addon-manager-dep.yaml",
		NodeControllerDeploymentYaml: "node-controller-dep.yaml",
		Values: map[string]string{
			"k8s-version":           "v1.9.0",
			"etcd-operator-version": "v0.6.0",
			"etcd-cluster-version":  "3.2.7",
			"kube-machine-version":  "v0.2.3",
			"addon-manager-version": "v1.9.0",
			"pod-network-bridge":    "v0.2",
		},
	}

	res, err := LoadOpenstackCloudConfigConfigMap(c, dc, version)
	if err != nil {
		t.Fatal(err)
	}

	checkTestResult(t, "loadopenstackcloudconfigconfigmap-result", res)

}
