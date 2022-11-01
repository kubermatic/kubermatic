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

package test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	semverlib "github.com/Masterminds/semver/v3"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubernetescontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/kubernetes"
	monitoringcontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/monitoring"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	metricsserver "k8c.io/kubermatic/v2/pkg/resources/metrics-server"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	ksemver "k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/version"
	"k8c.io/kubermatic/v2/pkg/version/cni"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

var (
	update     = flag.Bool("update", false, "Update test fixtures")
	fixtureDir = "fixtures"

	kubernetesVersions = []*version.Version{
		{
			Version: semverlib.MustParse("1.22.1"),
		},
		{
			Version: semverlib.MustParse("1.23.5"),
		},
		{
			Version: semverlib.MustParse("1.24.0"),
		},
		{
			Version: semverlib.MustParse("1.25.0"),
		},
	}

	featureSets = []map[string]bool{
		{},
		{kubermaticv1.ClusterFeatureExternalCloudProvider: true},
	}

	cloudProviders = map[string]kubermaticv1.CloudSpec{
		"azure": {
			ProviderName: string(kubermaticv1.AzureCloudProvider),
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
				LoadBalancerSKU: kubermaticv1.AzureBasicLBSKU,
			},
		},
		"vsphere": {
			ProviderName: string(kubermaticv1.VSphereCloudProvider),
			VSphere: &kubermaticv1.VSphereCloudSpec{
				Username: "vs-username",
				Password: "vs-password",
			},
		},
		"digitalocean": {
			ProviderName: string(kubermaticv1.DigitaloceanCloudProvider),
			Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
				Token: "do-token",
			},
		},
		"aws": {
			ProviderName: string(kubermaticv1.AWSCloudProvider),
			AWS: &kubermaticv1.AWSCloudSpec{
				AccessKeyID:          "aws-access-key-id",
				SecretAccessKey:      "aws-secret-access-key",
				AssumeRoleARN:        "aws-assume-role-arn",
				AssumeRoleExternalID: "aws-assume-role-external-id",
				InstanceProfileName:  "aws-instance-profile-name",
				RouteTableID:         "aws-route-table-id",
				SecurityGroupID:      "aws-security-group",
				VPCID:                "aws-vpn-id",
				ControlPlaneRoleARN:  "aws-role-arn",
			},
		},
		"gcp": {
			ProviderName: string(kubermaticv1.GCPCloudProvider),
			GCP: &kubermaticv1.GCPCloudSpec{
				ServiceAccount: "eyJ0aGlzaXMiOiJqc29uIn0=",
			},
		},
		"openstack": {
			ProviderName: string(kubermaticv1.OpenstackCloudProvider),
			Openstack: &kubermaticv1.OpenstackCloudSpec{
				SubnetID:       "openstack-subnet-id",
				Username:       "openstack-username",
				Project:        "openstack-project",
				Domain:         "openstack-domain",
				FloatingIPPool: "openstack-floating-ip-pool",
				Network:        "openstack-network",
				Password:       "openstack-password",
				RouterID:       "openstack-router-id",
				SecurityGroups: "openstack-security-group1,openstack-security-group2",
			},
		},
		"bringyourown": {
			ProviderName: string(kubermaticv1.BringYourOwnCloudProvider),
			BringYourOwn: &kubermaticv1.BringYourOwnCloudSpec{},
		},
	}

	config = &kubermaticv1.KubermaticConfiguration{
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
				Monitoring: kubermaticv1.KubermaticUserClusterMonitoringConfiguration{
					ScrapeAnnotationPrefix: defaulting.DefaultUserClusterScrapeAnnotationPrefix,
					CustomScrapingConfigs: `
- job_name: custom-test-config
  scheme: https
  metrics_path: '/metrics'
  static_configs:
  - targets:
    - 'foo.bar:12345'
`,
				},
			},
		},
	}

	datacenter = &kubermaticv1.Datacenter{
		Spec: kubermaticv1.DatacenterSpec{
			Azure: &kubermaticv1.DatacenterSpecAzure{
				Location: "az-location",
			},
			VSphere: &kubermaticv1.DatacenterSpecVSphere{
				Endpoint:         "https://vs-endpoint.io",
				AllowInsecure:    false,
				DefaultDatastore: "vs-datastore",
				Datacenter:       "vs-datacenter",
				Cluster:          "vs-cluster",
				RootPath:         "vs-cluster",
			},
			AWS: &kubermaticv1.DatacenterSpecAWS{
				Images: kubermaticv1.ImageList{
					providerconfig.OperatingSystemUbuntu:  "ubuntu-ami",
					providerconfig.OperatingSystemCentOS:  "centos-ami",
					providerconfig.OperatingSystemSLES:    "sles-ami",
					providerconfig.OperatingSystemRHEL:    "rhel-ami",
					providerconfig.OperatingSystemFlatcar: "flatcar-ami",
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
)

func checkTestResult(t *testing.T, resFile string, testObj interface{}) {
	path := filepath.Join(fixtureDir, resFile+".yaml")
	jsonRes, err := json.Marshal(testObj)
	if err != nil {
		t.Fatal(err)
	}
	res, err := yaml.JSONToYAML(jsonRes)
	if err != nil {
		t.Fatal(err)
	}

	res = append([]byte("# This file has been generated, DO NOT EDIT.\n\n"), res...)

	if *update {
		if err := os.WriteFile(path, res, 0644); err != nil {
			t.Fatalf("failed to update fixtures: %v", err)
		}
	}

	exp, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	resStr := strings.TrimSpace(string(res))
	expStr := strings.TrimSpace(string(exp))

	if resStr != expStr {
		t.Errorf("Deployment file changed and does not match fixture(%q) anymore. Make sure you update all fixtures after changing templates. If the diff seems valid, run the tests again with '-update':\n%v", path, diff.StringDiff(expStr, resStr))
	}
}

type testCase struct {
	provider string
	version  semverlib.Version
	features map[string]bool
}

// name returns the name for the current test case.
func (tc testCase) enabledFeatures() string {
	features := sets.NewString()
	for f, active := range tc.features {
		if active {
			features.Insert(f)
		}
	}

	return strings.Join(features.List(), "-")
}

// name returns the name for the current test case.
func (tc testCase) name() string {
	name := fmt.Sprintf("resources-%s-%s", tc.provider, tc.version)
	if features := tc.enabledFeatures(); len(features) > 0 {
		name += "-" + features
	}
	return name
}

// fixturePath returns the path to the fixture for the targeted resource.
func (tc testCase) fixturePath(resType, resName string) string {
	path := fmt.Sprintf("%s-%s-%s-%s", resType, tc.provider, tc.version, resName)
	if features := tc.enabledFeatures(); len(features) > 0 {
		path += "-" + features
	}
	return path
}

func createClusterObject(version semverlib.Version, cloudSpec kubermaticv1.CloudSpec, features map[string]bool) *kubermaticv1.Cluster {
	sversion := *ksemver.NewSemverOrDie(version.String())

	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "de-test-01",
			UID:  types.UID("1234567890"),
			Labels: map[string]string{
				"my-label":                     "my-value",
				kubermaticv1.ProjectIDLabelKey: "my-project",
			},
		},
		Spec: kubermaticv1.ClusterSpec{
			Features:       features,
			ExposeStrategy: kubermaticv1.ExposeStrategyLoadBalancer,
			Cloud:          cloudSpec,
			Version:        sversion,
			ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
				Services: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{"10.240.16.0/20"},
				},
				Pods: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{"172.25.0.0/16"},
				},
				DNSDomain:                "cluster.local",
				ProxyMode:                resources.IPVSProxyMode,
				NodeLocalDNSCacheEnabled: pointer.Bool(true),
			},
			CNIPlugin: &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCanal,
				Version: cni.GetDefaultCNIPluginVersion(kubermaticv1.CNIPluginTypeCanal),
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
			ServiceAccount: &kubermaticv1.ServiceAccountSettings{
				TokenVolumeProjectionEnabled: true,
			},
			MLA: &kubermaticv1.MLASettings{
				MonitoringEnabled: true,
				LoggingEnabled:    false,
			},
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: "cluster-de-test-01",
			Versions: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      sversion,
				Apiserver:         sversion,
				ControllerManager: sversion,
				Scheduler:         sversion,
			},
			Address: kubermaticv1.ClusterAddress{
				ExternalName: "jh8j81chn.europe-west3-c.dev.kubermatic.io",
				IP:           "35.198.93.90",
				AdminToken:   "6hzr76.u8txpkk4vhgmtgdp",
				InternalName: "apiserver-external.cluster-de-test-01.svc.cluster.local.",
				URL:          "https://jh8j81chn.europe-west3-c.dev.kubermatic.io:30000",
				Port:         30000,
			},
		},
	}
}

func TestLoadFiles(t *testing.T) {
	kubermaticVersions := kubermatic.NewFakeVersions()
	caBundle := certificates.NewFakeCABundle()

	if *update {
		if err := os.RemoveAll(fixtureDir); err != nil {
			t.Fatalf("Failed to remove all old fixtures: %v", err)
		}
		if err := os.MkdirAll(fixtureDir, 0755); err != nil {
			t.Fatalf("Failed to create fixture directory: %v", err)
		}
	}

	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatalf("Failed to list existing fixtures: %v", err)
	}

	allFiles := sets.NewString()
	for _, e := range entries {
		allFiles.Insert(e.Name())
	}

	markFixtureUsed := func(fixtureName string) {
		filename := fixtureName + ".yaml"
		allFiles.Delete(filename)
	}

	for _, ver := range kubernetesVersions {
		for prov, cloudspec := range cloudProviders {
			for _, features := range featureSets {
				tc := testCase{
					provider: prov,
					version:  *ver.Version,
					features: features,
				}
				t.Run(tc.name(), func(t *testing.T) {
					cluster := createClusterObject(*ver.Version, cloudspec, features)

					caBundleConfigMap := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							ResourceVersion: "123456",
							Name:            resources.CABundleConfigMapName,
							Namespace:       cluster.Status.NamespaceName,
						},
					}

					if features[kubermaticv1.ClusterFeatureExternalCloudProvider] && !resources.ExternalCloudControllerFeatureSupported(datacenter, &cluster.Spec.Cloud, cluster.Spec.Version) {
						t.Log("Unsupported configuration")
						return
					}

					dynamicClient := ctrlruntimefakeclient.
						NewClientBuilder().
						WithScheme(scheme.Scheme).
						WithObjects(
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
									Name:            resources.OperatingSystemManagerKubeconfigSecretName,
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
							&corev1.Secret{
								ObjectMeta: metav1.ObjectMeta{
									ResourceVersion: "123456",
									Name:            resources.GatekeeperWebhookServerCertSecretName,
									Namespace:       cluster.Status.NamespaceName,
								},
							},
							&corev1.Secret{
								ObjectMeta: metav1.ObjectMeta{
									ResourceVersion: "123456",
									Name:            resources.AdminKubeconfigSecretName,
									Namespace:       cluster.Status.NamespaceName,
								},
							},
							&corev1.Secret{
								ObjectMeta: metav1.ObjectMeta{
									ResourceVersion: "123456",
									Name:            resources.CloudConfigSeedSecretName,
									Namespace:       cluster.Status.NamespaceName,
								},
							},
							&corev1.Secret{
								ObjectMeta: metav1.ObjectMeta{
									ResourceVersion: "123456",
									Name:            resources.GoogleServiceAccountSecretName,
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
							caBundleConfigMap,
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
							&corev1.ConfigMap{
								ObjectMeta: metav1.ObjectMeta{
									ResourceVersion: "123456",
									Name:            resources.AdmissionControlConfigMapName,
									Namespace:       cluster.Status.NamespaceName,
								},
							},
							&corev1.Service{
								ObjectMeta: metav1.ObjectMeta{
									Name:      resources.ApiserverServiceName,
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
							&corev1.Service{
								ObjectMeta: metav1.ObjectMeta{
									Name:      resources.MLAGatewayExternalServiceName,
									Namespace: cluster.Status.NamespaceName,
								},
								Spec: corev1.ServiceSpec{
									Ports: []corev1.ServicePort{
										{
											NodePort: 30005,
										},
									},
									ClusterIP: "192.0.2.15",
								},
							},
						).
						Build()

					var group wait.Group
					defer group.Wait()
					stopCh := make(chan struct{})
					defer func() {
						close(stopCh)
					}()

					ctx := context.Background()
					data := resources.NewTemplateDataBuilder().
						WithContext(ctx).
						WithClient(dynamicClient).
						WithCluster(cluster).
						WithDatacenter(datacenter).
						WithSeed(&kubermaticv1.Seed{
							ObjectMeta: metav1.ObjectMeta{Name: "testdc"},
							Spec: kubermaticv1.SeedSpec{
								ProxySettings: &kubermaticv1.ProxySettings{
									HTTPProxy: kubermaticv1.NewProxyValue("http://my-corp"),
								},
								MLA: &kubermaticv1.SeedMLASettings{
									UserClusterMLAEnabled: true,
								},
							},
						}).
						WithKubermaticConfiguration(config).
						WithNodeAccessNetwork("192.0.2.0/24").
						WithEtcdDiskSize(resource.MustParse("5Gi")).
						WithBackupPeriod(20 * time.Minute).
						WithUserClusterMLAEnabled(true).
						WithCABundle(caBundle).
						WithOIDCIssuerURL("https://dev.kubermatic.io/dex").
						WithOIDCIssuerClientID("kubermaticIssuer").
						WithKubermaticImage("quay.io/kubermatic/kubermatic").
						WithEtcdLauncherImage("quay.io/kubermatic/etcd-launcher").
						WithDnatControllerImage("quay.io/kubermatic/kubeletdnat-controller").
						WithVersions(kubermaticVersions).
						Build()

					generateAndVerifyResources(t, data, tc, markFixtureUsed)
				})
			}
		}
	}

	if leftover := allFiles.List(); len(leftover) > 0 {
		t.Fatalf("Leftover fixtures found that do not belong to any of the configured testcases: %v", leftover)
	}
}

func generateAndVerifyResources(t *testing.T, data *resources.TemplateData, tc testCase, fixtureDone func(string)) {
	cluster := data.Cluster()

	var deploymentCreators []reconciling.NamedDeploymentCreatorGetter
	deploymentCreators = append(deploymentCreators, kubernetescontroller.GetDeploymentCreators(data, true)...)
	deploymentCreators = append(deploymentCreators, monitoringcontroller.GetDeploymentCreators(data)...)
	for _, create := range deploymentCreators {
		name, creator := create()
		res, err := creator(&appsv1.Deployment{})
		if err != nil {
			t.Fatalf("failed to create Deployment: %v", err)
		}
		res.Name = name
		res.Namespace = cluster.Status.NamespaceName

		fixturePath := tc.fixturePath("deployment", res.Name)

		verifyContainerResources(fmt.Sprintf("Deployment/%s", res.Name), res.Spec.Template, t)
		fixtureDone(fixturePath)
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
		res.Name = name
		res.Namespace = cluster.Status.NamespaceName

		fixturePath := tc.fixturePath("configmap", res.Name)
		fixtureDone(fixturePath)
		checkTestResult(t, fixturePath, res)
	}

	serviceCreators := kubernetescontroller.GetServiceCreators(data)
	for _, creatorGetter := range serviceCreators {
		name, create := creatorGetter()
		res, err := create(&corev1.Service{})
		if err != nil {
			t.Fatalf("failed to create Service: %v", err)
		}
		res.Name = name
		res.Namespace = cluster.Status.NamespaceName

		fixturePath := tc.fixturePath("service", res.Name)
		fixtureDone(fixturePath)
		checkTestResult(t, fixturePath, res)
	}

	var statefulSetCreators []reconciling.NamedStatefulSetCreatorGetter
	statefulSetCreators = append(statefulSetCreators, kubernetescontroller.GetStatefulSetCreators(data, false, false)...)
	statefulSetCreators = append(statefulSetCreators, monitoringcontroller.GetStatefulSetCreators(data)...)
	for _, creatorGetter := range statefulSetCreators {
		name, create := creatorGetter()
		res, err := create(&appsv1.StatefulSet{})
		if err != nil {
			t.Fatalf("failed to create StatefulSet: %v", err)
		}
		res.Name = name
		res.Namespace = cluster.Status.NamespaceName

		fixturePath := tc.fixturePath("statefulset", res.Name)
		if err != nil {
			t.Fatalf("failed to create StatefulSet for %s: %v", fixturePath, err)
		}

		fixtureDone(fixturePath)

		// Verify that every StatefulSet has the ImagePullSecret set
		if len(res.Spec.Template.Spec.ImagePullSecrets) == 0 {
			t.Errorf("StatefulSet %s is missing the ImagePullSecret on the PodTemplate", res.Name)
		}

		verifyContainerResources(fmt.Sprintf("StatefulSet/%s", res.Name), res.Spec.Template, t)

		checkTestResult(t, fixturePath, res)
	}

	for _, creatorGetter := range kubernetescontroller.GetPodDisruptionBudgetCreators(data) {
		name, create := creatorGetter()
		res, err := create(&policyv1.PodDisruptionBudget{})
		if err != nil {
			t.Fatalf("failed to create PodDisruptionBudget: %v", err)
		}
		res.Name = name
		res.Namespace = cluster.Status.NamespaceName

		fixturePath := tc.fixturePath("poddisruptionbudget", name)
		if err != nil {
			t.Fatalf("failed to create PodDisruptionBudget for %s: %v", fixturePath, err)
		}

		fixtureDone(fixturePath)
		checkTestResult(t, fixturePath, res)
	}

	for _, creatorGetter := range kubernetescontroller.GetCronJobCreators(data) {
		name, create := creatorGetter()
		res, err := create(&batchv1.CronJob{})
		if err != nil {
			t.Fatalf("failed to create CronJob: %v", err)
		}
		res.Name = name
		res.Namespace = cluster.Status.NamespaceName

		fixturePath := tc.fixturePath("cronjob", res.Name)
		if err != nil {
			t.Fatalf("failed to create CronJob for %s: %v", fixturePath, err)
		}

		// Verify that every CronJob has the ImagePullSecret set
		if len(res.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets) == 0 {
			t.Errorf("CronJob %s is missing the ImagePullSecret on the PodTemplate", res.Name)
		}

		fixtureDone(fixturePath)
		checkTestResult(t, fixturePath, res)
	}

	for _, creatorGetter := range kubernetescontroller.GetEtcdBackupConfigCreators(data, test.GenTestSeed()) {
		name, create := creatorGetter()
		res, err := create(&kubermaticv1.EtcdBackupConfig{})
		if err != nil {
			t.Fatalf("failed to create EtcdBackupConfig: %v", err)
		}
		res.Name = name
		res.Namespace = cluster.Status.NamespaceName

		fixturePath := tc.fixturePath("etcdbackupconfig", res.Name)
		if err != nil {
			t.Fatalf("failed to create EtcdBackupConfig for %s: %v", fixturePath, err)
		}

		fixtureDone(fixturePath)
		checkTestResult(t, fixturePath, res)
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
