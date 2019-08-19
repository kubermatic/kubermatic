package resources

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	testhelper "github.com/kubermatic/kubermatic/api/pkg/test"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

var update = flag.Bool("update", false, "update .golden files")

type fakeMasterConfigData struct {
	cluster  *kubermaticv1.Cluster
	nodePort int32
	oidc     oidcData
}

func (d *fakeMasterConfigData) Cluster() *kubermaticv1.Cluster {
	return d.cluster
}

func (d *fakeMasterConfigData) GetApiserverExternalNodePort(context.Context) (int32, error) {
	return d.nodePort, nil
}

func (d *fakeMasterConfigData) OIDCIssuerURL() string {
	return d.oidc.IssuerURL
}

func (d *fakeMasterConfigData) OIDCClientID() string {
	return d.oidc.ClientID
}

func (d *fakeMasterConfigData) OIDCClientSecret() string {
	return d.oidc.ClientSecret
}

func TestGetMasterConfig(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		cluster  *kubermaticv1.Cluster
		nodePort int32
	}{
		{
			name:     "aws-cluster",
			nodePort: 32593,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "aws-1",
				},
				Address: kubermaticv1.ClusterAddress{
					URL: "https://henrik-os1.europe-west3-c.dev.kubermatic.io:32593",
					IP:  "35.198.93.90",
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-aws-1",
				},
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Services: kubermaticv1.NetworkRanges{
							CIDRBlocks: []string{"10.10.10.0/24"},
						},
						Pods: kubermaticv1.NetworkRanges{
							CIDRBlocks: []string{"172.25.0.0/16"},
						},
						DNSDomain: "cluster.local",
					},
					Cloud: kubermaticv1.CloudSpec{
						DatacenterName: "aws-dummy",
						AWS: &kubermaticv1.AWSCloudSpec{
							AccessKeyID:     "access-key-id",
							SecretAccessKey: "secret-access-key",
						},
					},
				},
			},
		},
		{
			name:     "digitalocean-cluster",
			nodePort: 32594,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "digitalocean-1",
				},
				Address: kubermaticv1.ClusterAddress{
					URL: "https://henrik-os2.europe-west3-c.dev.kubermatic.io:32594",
					IP:  "35.198.93.90",
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-digitalocean-1",
				},
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Services: kubermaticv1.NetworkRanges{
							CIDRBlocks: []string{"10.10.10.0/24"},
						},
						Pods: kubermaticv1.NetworkRanges{
							CIDRBlocks: []string{"172.25.0.0/16"},
						},
						DNSDomain: "cluster.local",
					},
					Cloud: kubermaticv1.CloudSpec{
						DatacenterName: "digitalocean-dummy",
						Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
							Token: "token",
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := &fakeMasterConfigData{
				nodePort: test.nodePort,
				cluster:  test.cluster,
				oidc: oidcData{
					ClientSecret: "client-secret",
					ClientID:     "client-id",
					IssuerURL:    "https://dev.kubermatic.io/dex",
				},
			}

			apiserverConfig, err := getMasterConfig(ctx, data, "apiserver")
			if err != nil {
				log.Fatal(err)
			}
			testhelper.CompareOutput(t, fmt.Sprintf("master-config-%s-apiserver", test.name), apiserverConfig, *update, ".yaml")

			controllerConfig, err := getMasterConfig(ctx, data, "controller")
			if err != nil {
				log.Fatal(err)
			}
			testhelper.CompareOutput(t, fmt.Sprintf("master-config-%s-controller", test.name), controllerConfig, *update, ".yaml")
		})
	}
}

type openshiftAPIServerCreatorDataFake struct {
	cluster *kubermaticv1.Cluster
}

func (o *openshiftAPIServerCreatorDataFake) Cluster() *kubermaticv1.Cluster {
	return o.cluster
}

func TestOpenshiftAPIServerConfigMapCreator(t *testing.T) {
	testCases := []struct {
		name string
	}{
		{
			name: "Generate simple config",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := &openshiftAPIServerCreatorDataFake{cluster: &kubermaticv1.Cluster{}}
			creatorGetter := OpenshiftAPIServerConfigMapCreator(data)
			name, creator := creatorGetter()
			if name != openshiftAPIServerConfigMapName {
				t.Errorf("expected name to be %q was %q", openshiftAPIServerConfigMapName, name)
			}

			configMap, err := creator(&corev1.ConfigMap{})
			if err != nil {
				t.Fatalf("failed calling creator: %v", err)
			}

			serializedConfigmap, err := yaml.Marshal(configMap)
			if err != nil {
				t.Fatalf("failed to marshal configmap: %v", err)
			}

			testhelper.CompareOutput(t, strings.ReplaceAll(tc.name, " ", "_"), string(serializedConfigmap), *update, ".yaml")
		})
	}
}
