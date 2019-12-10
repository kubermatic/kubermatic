package test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/ghodss/yaml"
	"github.com/pmezard/go-difflib/difflib"

	kubernetescontroller "github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-manager/kubernetes"
	monitoringcontroller "github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-manager/monitoring"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	metricsserver "github.com/kubermatic/kubermatic/api/pkg/resources/metrics-server"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	ksemver "github.com/kubermatic/kubermatic/api/pkg/semver"
	"github.com/kubermatic/kubermatic/api/pkg/version"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var update = flag.Bool("update", false, "Update test fixtures")

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

	if *update {
		if err := ioutil.WriteFile(path, res, 0644); err != nil {
			t.Fatalf("failed to update fixtures: %v", err)
		}
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
		t.Errorf("\nDeployment file changed and does not match fixture(%q) anymore: \n %s\n\nMake sure you update all fixtures after changing templates. If the diff seems valid, run the tests again with '-update'", path, diffStr)
	}
}

func TestLoadFiles(t *testing.T) {
	versions := []*version.Version{
		{
			Version: semver.MustParse("1.10.0"),
		},
		{
			Version: semver.MustParse("1.10.6"),
		},
		{
			Version: semver.MustParse("1.11.0"),
		},
		{
			Version: semver.MustParse("1.11.1"),
		},
		{
			Version: semver.MustParse("1.12.0"),
		},
		{
			Version: semver.MustParse("1.13.0"),
		},
	}

	clouds := map[string]kubermaticv1.CloudSpec{
		"azure": {
			Azure: &kubermaticv1.AzureCloudSpec{
				TenantID:        "az-tenant-id",
				SubscriptionID:  "az-subscription-id",
				ClientID:        "az-client-id",
				ClientSecret:    "az-client-secret",
				ResourceGroup:   "az-res-group",
				VNetName:        "az-vnet-name",
				SubnetName:      "az-subnet-name",
				RouteTableName:  "az-route-table-name",
				SecurityGroup:   "az-sec-group",
				AvailabilitySet: "az-availability-set",
			},
		},
		"vsphere": {
			VSphere: &kubermaticv1.VSphereCloudSpec{
				Username: "vs-username",
				Password: "vs-password",
			},
		},
		"digitalocean": {
			Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
				Token: "do-token",
			},
		},
		"aws": {
			AWS: &kubermaticv1.AWSCloudSpec{
				AccessKeyID:         "aws-access-key-id",
				SecretAccessKey:     "aws-secret-access-key",
				InstanceProfileName: "aws-instance-profile-name",
				RoleName:            "aws-role-name",
				RouteTableID:        "aws-route-table-id",
				SecurityGroupID:     "aws-security-group",
				VPCID:               "aws-vpn-id",
				ControlPlaneRoleARN: "aws-role-arn",
			},
		},
		"openstack": {
			Openstack: &kubermaticv1.OpenstackCloudSpec{
				SubnetID:       "openstack-subnet-id",
				Username:       "openstack-username",
				Tenant:         "openstack-tenant",
				Domain:         "openstack-domain",
				FloatingIPPool: "openstack-floating-ip-pool",
				Network:        "openstack-network",
				Password:       "openstack-password",
				RouterID:       "openstack-router-id",
				SecurityGroups: "openstack-security-group1,openstack-security-group2",
			},
		},
		"bringyourown": {
			BringYourOwn: &kubermaticv1.BringYourOwnCloudSpec{},
		},
	}

	dc := &kubermaticv1.Datacenter{
		Spec: kubermaticv1.DatacenterSpec{
			Azure: &kubermaticv1.DatacenterSpecAzure{
				Location: "az-location",
			},
			VSphere: &kubermaticv1.DatacenterSpecVSphere{
				Endpoint:      "https://vs-endpoint.io",
				AllowInsecure: false,
				Datastore:     "vs-datastore",
				Datacenter:    "vs-datacenter",
				Cluster:       "vs-cluster",
				RootPath:      "vs-cluster",
			},
			AWS: &kubermaticv1.DatacenterSpecAWS{
				Images: kubermaticv1.ImageList{
					providerconfig.OperatingSystemUbuntu: "ubuntu-ami",
					providerconfig.OperatingSystemCentOS: "centos-ami",
					providerconfig.OperatingSystemCoreos: "coreos-ami",
				},
				Region: "us-central1",
			},
			Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{
				Region: "fra1",
			},
			Openstack: &kubermaticv1.DatacenterSpecOpenstack{
				AuthURL:          "https://example.com:8000/v3",
				AvailabilityZone: "zone1",
				DNSServers:       []string{"8.8.8.8", "8.8.4.4"},
				IgnoreVolumeAZ:   true,
				Region:           "cbk",
			},
		},
	}

	for _, ver := range versions {
		for prov, cloudspec := range clouds {
			t.Run(fmt.Sprintf("resources-%s-%s", prov, ver.Version.String()), func(t *testing.T) {
				cluster := &kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "de-test-01",
						UID:  types.UID("1234567890"),
						Labels: map[string]string{
							"my-label": "my-value",
						},
					},
					Spec: kubermaticv1.ClusterSpec{
						ExposeStrategy: corev1.ServiceTypeLoadBalancer,
						Cloud:          cloudspec,
						Version:        *ksemver.NewSemverOrDie(ver.Version.String()),
						ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
							Services: kubermaticv1.NetworkRanges{
								CIDRBlocks: []string{"10.240.16.0/20"},
							},
							Pods: kubermaticv1.NetworkRanges{
								CIDRBlocks: []string{"172.25.0.0/16"},
							},
							DNSDomain: "cluster.local",
							ProxyMode: resources.IPVSProxyMode,
						},
						MachineNetworks: []kubermaticv1.MachineNetworkingConfig{
							{
								CIDR: "192.168.1.1/24",
								DNSServers: []string{
									"8.8.8.8",
								},
								Gateway: "192.168.1.1",
							},
						},
					},
					Address: kubermaticv1.ClusterAddress{
						ExternalName: "jh8j81chn.europe-west3-c.dev.kubermatic.io",
						IP:           "35.198.93.90",
						AdminToken:   "6hzr76.u8txpkk4vhgmtgdp",
						InternalName: "apiserver-external.cluster-de-test-01.svc.cluster.local.",
						Port:         30000,
					},
					Status: kubermaticv1.ClusterStatus{
						NamespaceName: "cluster-de-test-01",
					},
				}

				dynamicClient := ctrlruntimefakeclient.NewFakeClientWithScheme(scheme.Scheme,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            metricsserver.ServingCertSecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.DexCASecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.TokensSecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.ServiceAccountKeySecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.ApiserverTLSSecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.KubeletClientCertificatesSecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.CASecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.OpenVPNCASecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.ApiserverEtcdClientCertificateSecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.ApiserverFrontProxyClientCertificateSecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.EtcdTLSCertificateSecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.MachineControllerKubeconfigSecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.OpenVPNServerCertificatesSecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.OpenVPNClientCertificatesSecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.ControllerManagerKubeconfigSecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.KubeStateMetricsKubeconfigSecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.SchedulerKubeconfigSecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.KubeletDnatControllerKubeconfigSecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.FrontProxyCASecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.MetricsServerKubeconfigSecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.PrometheusApiserverClientCertificateSecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.MachineControllerWebhookServingCertSecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.InternalUserClusterAdminKubeconfigSecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.KubernetesDashboardKubeconfigSecretName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.UserSSHKeys,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.OpenVPNClientConfigsConfigMapName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.CloudConfigConfigMapName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.PrometheusConfigConfigMapName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.DNSResolverConfigMapName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.AuditConfigMapName,
							Namespace:       cluster.Status.NamespaceName,
						},
					},
					&corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      resources.ApiserverExternalServiceName,
							Namespace: cluster.Status.NamespaceName,
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{
								{
									NodePort: 30000,
								},
							},
							ClusterIP: "192.0.2.10",
						},
					},
					&corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      resources.ApiserverInternalServiceName,
							Namespace: cluster.Status.NamespaceName,
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{
								{
									NodePort: 30001,
								},
							},
							ClusterIP: "192.0.2.11",
						},
					},
					&corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      resources.OpenVPNServerServiceName,
							Namespace: cluster.Status.NamespaceName,
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{
								{
									NodePort: 30003,
								},
							},
							ClusterIP: "192.0.2.13",
						},
					},
					&corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      resources.DNSResolverServiceName,
							Namespace: cluster.Status.NamespaceName,
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{
								{
									NodePort: 30003,
								},
							},
							ClusterIP: "192.0.2.14",
						},
					},
				)

				var group wait.Group
				defer group.Wait()
				stopCh := make(chan struct{})
				defer func() {
					close(stopCh)
				}()

				tmpFile, err := ioutil.TempFile("", "kubermatic")
				if err != nil {
					t.Fatalf("couldnt create temp file, see: %v", err)
				}

				tmpFilePath := tmpFile.Name()
				_, err = tmpFile.WriteString(`- job_name: custom-test-config
  scheme: https
  metrics_path: '/metrics'
  static_configs:
  - targets:
    - 'foo.bar:12345'
`)
				if err != nil {
					t.Fatalf("couldnt write to temp file, see: %v", err)
				}
				defer (func() {
					err = os.Remove(tmpFilePath)
					if err != nil {
						t.Fatalf("couldn't delete temp file, see: %v", err)
					}
				})()

				ctx := context.Background()
				data := resources.NewTemplateData(
					ctx,
					dynamicClient,
					cluster,
					dc,
					&kubermaticv1.Seed{
						ObjectMeta: metav1.ObjectMeta{Name: "testdc"},
						Spec: kubermaticv1.SeedSpec{
							ProxySettings: &kubermaticv1.ProxySettings{
								HTTPProxy: kubermaticv1.NewProxyValue("http://my-corp"),
							},
						},
					},
					"",
					"",
					"192.0.2.0/24",
					resource.MustParse("5Gi"),
					"kubermatic_io_monitoring",
					"",
					false,
					false,
					tmpFilePath,
					"test",
					"https://dev.kubermatic.io/dex",
					"kubermaticIssuer",
					true,
					"quay.io/kubermatic/api",
					"quay.io/kubermatic/kubeletdnat-controller",
					false)

				var deploymentCreators []reconciling.NamedDeploymentCreatorGetter
				deploymentCreators = append(deploymentCreators, kubernetescontroller.GetDeploymentCreators(data, true)...)
				deploymentCreators = append(deploymentCreators, monitoringcontroller.GetDeploymentCreators(data)...)
				for _, create := range deploymentCreators {
					_, creator := create()
					res, err := creator(&appsv1.Deployment{})
					if err != nil {
						t.Fatalf("failed to create Deployment: %v", err)
					}
					fixturePath := fmt.Sprintf("deployment-%s-%s-%s", prov, ver.Version.String(), res.Name)

					verifyContainerResources(fmt.Sprintf("Deployment/%s", res.Name), res.Spec.Template, t)

					checkTestResult(t, fixturePath, res)
				}

				var namedConfigMapCreatorGetters []reconciling.NamedConfigMapCreatorGetter
				namedConfigMapCreatorGetters = append(namedConfigMapCreatorGetters, kubernetescontroller.GetConfigMapCreators(data)...)
				namedConfigMapCreatorGetters = append(namedConfigMapCreatorGetters, monitoringcontroller.GetConfigMapCreators(data)...)
				for _, namedGetter := range namedConfigMapCreatorGetters {
					name, create := namedGetter()
					res, err := create(&corev1.ConfigMap{})
					if err != nil {
						t.Fatalf("failed to create ConfigMap: %v", err)
					}

					fixturePath := fmt.Sprintf("configmap-%s-%s-%s", prov, ver.Version.String(), name)
					checkTestResult(t, fixturePath, res)
				}

				serviceCreators := kubernetescontroller.GetServiceCreators(data)
				for _, creatorGetter := range serviceCreators {
					name, create := creatorGetter()
					res, err := create(&corev1.Service{})
					if err != nil {
						t.Fatalf("failed to create Service: %v", err)
					}

					fixturePath := fmt.Sprintf("service-%s-%s-%s", prov, ver.Version.String(), name)
					checkTestResult(t, fixturePath, res)
				}

				var statefulSetCreators []reconciling.NamedStatefulSetCreatorGetter
				statefulSetCreators = append(statefulSetCreators, kubernetescontroller.GetStatefulSetCreators(data, false)...)
				statefulSetCreators = append(statefulSetCreators, monitoringcontroller.GetStatefulSetCreators(data)...)
				for _, creatorGetter := range statefulSetCreators {
					_, create := creatorGetter()
					res, err := create(&appsv1.StatefulSet{})
					if err != nil {
						t.Fatalf("failed to create StatefulSet: %v", err)
					}

					fixturePath := fmt.Sprintf("statefulset-%s-%s-%s", prov, ver.Version.String(), res.Name)
					if err != nil {
						t.Fatalf("failed to create StatefulSet for %s: %v", fixturePath, err)
					}

					// Verify that every StatefulSet has the ImagePullSecret set
					if len(res.Spec.Template.Spec.ImagePullSecrets) == 0 {
						t.Errorf("StatefulSet %s is missing the ImagePullSecret on the PodTemplate", res.Name)
					}

					verifyContainerResources(fmt.Sprintf("StatefulSet/%s", res.Name), res.Spec.Template, t)

					checkTestResult(t, fixturePath, res)
				}

				for _, creatorGetter := range kubernetescontroller.GetPodDisruptionBudgetCreators(data) {
					name, create := creatorGetter()
					res, err := create(&policyv1beta1.PodDisruptionBudget{})
					if err != nil {
						t.Fatalf("failed to create PodDisruptionBudget: %v", err)
					}

					fixturePath := fmt.Sprintf("poddisruptionbudget-%s-%s-%s", prov, ver.Version.String(), name)
					if err != nil {
						t.Fatalf("failed to create PodDisruptionBudget for %s: %v", fixturePath, err)
					}

					checkTestResult(t, fixturePath, res)
				}

				for _, creatorGetter := range kubernetescontroller.GetCronJobCreators(data) {
					_, create := creatorGetter()
					res, err := create(&batchv1beta1.CronJob{})
					if err != nil {
						t.Fatalf("failed to create CronJob: %v", err)
					}

					fixturePath := fmt.Sprintf("cronjob-%s-%s-%s", prov, ver.Version.String(), res.Name)
					if err != nil {
						t.Fatalf("failed to create CronJob for %s: %v", fixturePath, err)
					}

					// Verify that every CronJob has the ImagePullSecret set
					if len(res.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets) == 0 {
						t.Errorf("CronJob %s is missing the ImagePullSecret on the PodTemplate", res.Name)
					}

					checkTestResult(t, fixturePath, res)
				}
			})
		}
	}
}

func verifyContainerResources(owner string, podTemplateSpec corev1.PodTemplateSpec, t *testing.T) {
	// Verify that every pod has resource request's & limit's set.
	for _, container := range podTemplateSpec.Spec.Containers {
		resourceLists := map[string]corev1.ResourceList{
			"Limit":    container.Resources.Limits,
			"Requests": container.Resources.Requests,
		}
		for listKind, resourceList := range resourceLists {
			for _, resourceName := range []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceMemory} {
				if _, exists := resourceList[resourceName]; !exists {
					t.Errorf("Container '%s' of %s is missing the %s %s!", container.Name, owner, resourceName, listKind)
				}
			}
		}
	}
}
