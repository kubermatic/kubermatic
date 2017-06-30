package template

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"

	"fmt"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/test"
)

const (
	tplPath         = "coreos/cloud-init.yaml"
	masterFilesPath = "../../config/kubermatic/static/master/"
)

var (
	awsCluster = &api.Cluster{
		Spec: api.ClusterSpec{
			Cloud: &api.CloudSpec{
				AWS: &api.AWSCloudSpec{
					AccessKeyID:      "my_AccessKeyID",
					SecretAccessKey:  "my_SecretAccessKey",
					VPCId:            "my_VPCId",
					AvailabilityZone: "my_AvailabilityZone",
					ContainerLinux: api.ContainerLinuxClusterSpec{
						AutoUpdate: false,
					},
				},
			},
			MasterVersion: "1.6.6",
		},
		Metadata: api.Metadata{
			Name: "jh8j81chn",
		},
		Address: &api.ClusterAddress{
			URL:          "https://jh8j81chn.host2.int.kubermatic.io:8443",
			ExternalName: "jh8j81chn.host2.int.kubermatic.io",
			ExternalPort: 8443,
			KubeletToken: "kubelet-token",
		},
		Status: api.ClusterStatus{
			RootCA: api.SecretKeyCert{
				Cert: []byte("RootCA_CERT"),
				Key:  []byte("RootCA_KEY"),
			},
			ApiserverSSHKey: api.SecretRSAKeys{
				PrivateKey: []byte("ApiserverSSHKey_PRIVATE_KEY"),
				PublicKey:  []byte("ApiserverSSHKey_PUBLIC_KEY"),
			},
		},
	}

	doCluster = &api.Cluster{
		Spec: api.ClusterSpec{
			Cloud: &api.CloudSpec{
				Digitalocean: &api.DigitaloceanCloudSpec{
					Token: "DO_TOKEN",
				},
			},
			MasterVersion: "1.6.6",
		},
		Metadata: api.Metadata{
			Name: "jh8j81chn",
		},
		Address: &api.ClusterAddress{
			URL:          "https://jh8j81chn.host2.int.kubermatic.io:8443",
			ExternalName: "jh8j81chn.host2.int.kubermatic.io",
			ExternalPort: 8443,
			KubeletToken: "kubelet-token",
		},
		Status: api.ClusterStatus{
			RootCA: api.SecretKeyCert{
				Cert: []byte("RootCA_CERT"),
				Key:  []byte("RootCA_KEY"),
			},
			ApiserverSSHKey: api.SecretRSAKeys{
				PrivateKey: []byte("ApiserverSSHKey_PRIVATE_KEY"),
				PublicKey:  []byte("ApiserverSSHKey_PUBLIC_KEY"),
			},
		},
	}
)

func TestNodeTemplate(t *testing.T) {
	if test.IsOnCi(masterFilesPath) {
		t.Skip("cannot load master files. Maybe on CI?")
	}

	cases := []struct {
		Name        string
		Cluster     *api.Cluster
		FixtureFile string
	}{
		{
			Name:        "AWS Cluster",
			Cluster:     awsCluster,
			FixtureFile: "aws-cloud-init.yaml",
		},
		{
			Name:        "DO Cluster",
			Cluster:     doCluster,
			FixtureFile: "do-cloud-init.yaml",
		},
	}

	for _, c := range cases {
		res, err := parseTemplate(c.Cluster)
		if err != nil {
			t.Fatalf("failed to parse template for test %s: %v", c.Name, err)
		}

		f, err := ioutil.ReadFile(filepath.Join("./fixtures", c.FixtureFile))
		if err != nil {
			t.Fatalf("failed to load fixture file %s for test %s: %v", c.FixtureFile, c.Name, err)
		}

		if string(f) != res {
			t.Fatalf("\nExpected to get \n%s \n(%q)\n\n Got \n%s \n", f, c.FixtureFile, res)
		}
	}
}

func parseTemplate(c *api.Cluster) (string, error) {
	data := api.NodeTemplateData{
		SSHAuthorizedKeys: []string{"key1", "key2"},
		Cluster:           c,
	}

	tpl, err := ParseFile(tplPath)
	if err != nil {
		return "", fmt.Errorf("failed to parse %q: %v", tplPath, err)
	}

	var buf bytes.Buffer
	if err = tpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
