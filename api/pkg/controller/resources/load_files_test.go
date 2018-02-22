package resources

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/pmezard/go-difflib/difflib"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

const (
	masterResourcePath = "../../../../config/kubermatic/static/master/"
)

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

func TestLoadFiles(t *testing.T) {
	versions := []*apiv1.MasterVersion{
		{
			Name:                            "1.7.0",
			ID:                              "1.7.0",
			Default:                         true,
			AllowedNodeVersions:             []string{"1.3.0"},
			EtcdOperatorDeploymentYaml:      "etcd-operator-dep.yaml",
			ApiserverDeploymentYaml:         "apiserver-dep.yaml",
			ControllerDeploymentYaml:        "controller-manager-dep.yaml",
			SchedulerDeploymentYaml:         "scheduler-dep.yaml",
			AddonManagerDeploymentYaml:      "addon-manager-dep.yaml",
			NodeControllerDeploymentYaml:    "node-controller-dep.yaml",
			MachineControllerDeploymentYaml: "machine-controller.yaml",
			Values: map[string]string{
				"k8s-version":                "v1.7.11",
				"etcd-operator-version":      "v0.6.0",
				"etcd-cluster-version":       "3.2.7",
				"kube-machine-version":       "v0.2.3",
				"addon-manager-version":      "v1.7.6",
				"pod-network-bridge":         "v0.1",
				"machine-controller-version": "v0.1.2",
			},
		},
		{
			Name:                            "1.8.0",
			ID:                              "1.8.0",
			Default:                         true,
			AllowedNodeVersions:             []string{"1.3.0"},
			EtcdOperatorDeploymentYaml:      "etcd-operator-dep.yaml",
			ApiserverDeploymentYaml:         "apiserver-dep.yaml",
			ControllerDeploymentYaml:        "controller-manager-dep.yaml",
			SchedulerDeploymentYaml:         "scheduler-dep.yaml",
			AddonManagerDeploymentYaml:      "addon-manager-dep.yaml",
			NodeControllerDeploymentYaml:    "node-controller-dep.yaml",
			MachineControllerDeploymentYaml: "machine-controller.yaml",
			Values: map[string]string{
				"k8s-version":                "v1.8.5",
				"etcd-operator-version":      "v0.6.0",
				"etcd-cluster-version":       "3.2.7",
				"kube-machine-version":       "v0.2.3",
				"addon-manager-version":      "v1.8.2",
				"pod-network-bridge":         "v0.2",
				"machine-controller-version": "v0.1.2",
			},
		},
		{
			Name:                            "1.9.0",
			ID:                              "1.9.0",
			Default:                         true,
			AllowedNodeVersions:             []string{"1.3.0"},
			EtcdOperatorDeploymentYaml:      "etcd-operator-dep.yaml",
			ApiserverDeploymentYaml:         "apiserver-dep.yaml",
			ControllerDeploymentYaml:        "controller-manager-dep.yaml",
			SchedulerDeploymentYaml:         "scheduler-dep.yaml",
			AddonManagerDeploymentYaml:      "addon-manager-dep.yaml",
			NodeControllerDeploymentYaml:    "node-controller-dep.yaml",
			MachineControllerDeploymentYaml: "machine-controller.yaml",
			Values: map[string]string{
				"k8s-version":                "v1.9.0",
				"etcd-operator-version":      "v0.6.0",
				"etcd-cluster-version":       "3.2.7",
				"kube-machine-version":       "v0.2.3",
				"addon-manager-version":      "v1.9.0",
				"pod-network-bridge":         "v0.2",
				"machine-controller-version": "v0.1.2",
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

	dc := provider.DatacenterMeta{
		Spec: provider.DatacenterSpec{
			AWS: &provider.AWSSpec{
				AMI:           "ami-aujakj",
				Region:        "us-central1",
				ZoneCharacter: "a",
			},
			Digitalocean: &provider.DigitaloceanSpec{
				Region: "fra1",
			},
			Openstack: &provider.OpenstackSpec{
				AuthURL:          "https://example.com:8000/v3",
				AvailabilityZone: "zone1",
				DNSServers:       []string{"8.8.8.8", "8.8.4.4"},
				IgnoreVolumeAZ:   true,
				Region:           "cbk",
			},
		},
	}

	for _, version := range versions {
		for prov, cloudspec := range clouds {
			t.Run(fmt.Sprintf("resources-%s-%s", prov, version.ID), func(t *testing.T) {
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
						IP:           "35.198.93.90",
					},
					Status: kubermaticv1.ClusterStatus{
						NamespaceName: "cluster-de-test-01",
					},
				}

				kubeClient := kubefake.NewSimpleClientset(&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						ResourceVersion: "123456",
						Name:            "token-users",
						Namespace:       cluster.Status.NamespaceName,
					},
				})

				var group wait.Group
				defer group.Wait()
				stopCh := make(chan struct{})
				defer func() {
					close(stopCh)
				}()

				informerFactory := informers.NewSharedInformerFactory(kubeClient, 30*time.Second)
				secretInformer := informerFactory.Core().V1().Secrets().Informer()
				secretLister := informerFactory.Core().V1().Secrets().Lister()
				group.StartWithChannel(stopCh, secretInformer.Run)
				cache.WaitForCacheSync(stopCh, secretInformer.HasSynced)

				data := &TemplateData{Cluster: cluster, Version: version, SecretLister: secretLister, DC: &dc}
				deps := map[string]string{
					version.EtcdOperatorDeploymentYaml:      fmt.Sprintf("deployment-%s-%s-etcd-operator", prov, version.ID),
					version.SchedulerDeploymentYaml:         fmt.Sprintf("deployment-%s-%s-scheduler", prov, version.ID),
					version.ControllerDeploymentYaml:        fmt.Sprintf("deployment-%s-%s-controller-manager", prov, version.ID),
					version.ApiserverDeploymentYaml:         fmt.Sprintf("deployment-%s-%s-apiserver", prov, version.ID),
					version.NodeControllerDeploymentYaml:    fmt.Sprintf("deployment-%s-%s-node-controller", prov, version.ID),
					version.AddonManagerDeploymentYaml:      fmt.Sprintf("deployment-%s-%s-addon-manager", prov, version.ID),
					version.MachineControllerDeploymentYaml: fmt.Sprintf("deployment-%s-%s-machine-controller", prov, version.ID),
				}

				for path, fixture := range deps {
					res, _, err := LoadDeploymentFile(data, masterResourcePath, path)
					if err != nil {
						t.Fatalf("failed to load deployment %q: %v", path, err)
					}

					checkTestResult(t, fixture, res)
				}

				configmaps := map[string]string{
					"cloud-config": fmt.Sprintf("configmap-%s-%s-cloud-config", prov, version.ID),
				}
				for name, fixture := range configmaps {
					res, _, err := LoadConfigMapFile(data, name, masterResourcePath)
					if err != nil {
						t.Fatalf("failed to load configmap %q: %v", name, err)
					}

					checkTestResult(t, fixture, res)
				}

				svcs := map[string]string{
					"apiserver":          fmt.Sprintf("service-%s-%s-apiserver", prov, version.ID),
					"apiserver-external": fmt.Sprintf("service-%s-%s-apiserver-external", prov, version.ID),
				}
				for name, fixture := range svcs {
					res, _, err := LoadServiceFile(data, name, masterResourcePath)
					if err != nil {
						t.Fatalf("failed to load service %q: %v", name, err)
					}

					checkTestResult(t, fixture, res)
				}

				clusterRoleBindings := map[string]string{
					"etcd-operator": fmt.Sprintf("cluster-role-binding-%s-%s-etcd-operator", prov, version.ID),
				}
				for name, fixture := range clusterRoleBindings {
					res, _, err := LoadClusterRoleBindingFile(&TemplateData{Cluster: cluster}, name, masterResourcePath)
					if err != nil {
						t.Fatalf("failed to load cluster role binding %q: %v", name, err)
					}

					checkTestResult(t, fixture, res)
				}
			})
		}
	}
}

func TestLoadServiceAccountFile(t *testing.T) {
	apps := map[string]string{
		"etcd-operator": "loadserviceaccountfile-etcd-operator-result",
	}

	for app, r := range apps {
		res, _, err := LoadServiceAccountFile(&TemplateData{}, app, masterResourcePath)
		if err != nil {
			t.Fatalf("failed to load %q: %v", app, err)
		}

		checkTestResult(t, r, res)
	}
}
