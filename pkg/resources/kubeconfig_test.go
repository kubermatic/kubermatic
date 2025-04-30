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

package resources

import (
	"crypto/x509"
	"encoding/pem"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetBaseKubeconfig(t *testing.T) {
	caString := `-----BEGIN CERTIFICATE-----
MIIFeDCCA2CgAwIBAgIMW4/qBithXLxSQLAeMA0GCSqGSIb3DQEBCwUAMGAxFzAV
BgNVBAMTDmV4YW1wbGUuY29tIENBMRQwEgYDVQQKEwtMb29kc2UgR21iSDEQMA4G
A1UEBxMHSGFtYnVyZzEQMA4GA1UECBMHSGFtYnVyZzELMAkGA1UEBhMCREUwHhcN
MTgwOTA1MTQzNjU0WhcNMTkwOTA1MTQzNjU0WjBgMRcwFQYDVQQDEw5leGFtcGxl
LmNvbSBDQTEUMBIGA1UEChMLTG9vZHNlIEdtYkgxEDAOBgNVBAcTB0hhbWJ1cmcx
EDAOBgNVBAgTB0hhbWJ1cmcxCzAJBgNVBAYTAkRFMIICIjANBgkqhkiG9w0BAQEF
AAOCAg8AMIICCgKCAgEAxrELs+aJmPNo1bQL9afQhvbb7u37KWLHracoIYYJD3e+
21BqzNVgZZBEu7gLTxd2c0+p9BOo9RqBjNBjxpzSWiLo/Kqsvfzg/Raou3V2AcjY
MXv2pw9IYI6sxjzmF71zWpuUyHdjsoO5P+/WmZHZxKA8NfqMduExz817Ie93ucci
PikRgSLOQKQn3Yn6PPZcG1z6lTWX8QZlZTWZ+I3B2BQ/c8704XR/9Qh6KNZhKK1N
lai6BexaTPQ1yIkH7ECweZ8cJGEQ0fTgCA9YmjSCC+9SCpNLIbVyKzlfQjYI4T/i
NbyooeO6iiZZ7a2NYXBIRn1r2maYlpKb722AiEIK5Li1mPX6526F1KDuB83tIw7P
nCm+K8aFuNbY1imLBALdf86Uvmg6x0sTTRejiCF/RNkviWwTZbz35C7mXpfPcImw
KJQLz1FUEzUw3mhbT6rlwjTgEpCT78uxqaHlMTDNmXyxuRzdD46VjU1oXDRjoWDf
rPVIpF7Z1aDYP++Jv8nEos5m/GtXJlPv9TI6QF7EXuW5paBRPNxISoyhX49PQKyS
sqb+K2pb6WYXji5MA/JAIwKLbiInBKR8BFi3w4rDFafhx/TrTrILG6mn5vj2tpqJ
1OZ5YTFUUN9kGBcw51Md7v1b7VNyFy2dMHuxWGSOIYIx+XQkGksO5F9z8oS41pMC
AwEAAaMyMDAwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUi2gGbqSxAJKFtwCF
/rgOPVwXQMEwDQYJKoZIhvcNAQELBQADggIBAD7CICX/2yvGYf0sOtq2vIfNRc6j
2MMOJSzmHjEeBIGXNxu3ZhFQzlFFindWpI2QgaD7wHDFndoKPS37m4d0AtRLKYMn
yfoYzSszHR9FO3Y3lTZ0FCfqY09ixzCDneShd6ns/ki12meNh7Zk4HQ0iEMmwjnZ
G8EpzknhVMW8bIUXdctAhF5bVRFdNIyHkcgoE9UFElzvegZes5fREncpRG3y23zf
xSvZm20gx4ZzjZtlEOwp6YdtEiDytiz8xgsJmY6Uz7AYNTgiFO8HB3SLqOiA4+MC
DfsiRvyqyVKWV+QNy+bEo7jxOSvM/nWgHtKKYWa01CNm88j7iXXnjxpDgliUApOY
/zpUX2fnUrSIJRelMqZoqwv5Gft4ZvWnuh3WHRNXBbch6yuMYbvyaQHP/TK4Cg4k
ulRg8yZgX8Biba7/p9eHjNbtQwUHqurY6SlDFOyyC+nWAi6c5pJT0fVn3sMNCH07
Ug/L6GonVLonbp+GSMcQ8dDlFv6Nksyexan5RkKeyn5vBc2Vagv8em7vpXv5QHu0
wBTOazRcKJBBPXrLbpUi4G03OkNh5ZlqBC+BG4286ZndVzH1gOjtA8ShHUiJQFJY
0Kp6Y3QoM3CUNPAuORCWmaIMT1bWHrM50BYaRe6pQSq1xMeGk9PYlldmg/iRL1VD
Y1OSU+4JRXF62VQY
-----END CERTIFICATE-----
`
	clusterName := "d3adb33f"

	pemBlock, _ := pem.Decode([]byte(caString))
	assert.NotNil(t, pemBlock)

	caCert, err := x509.ParseCertificate(pemBlock.Bytes)
	assert.NotNil(t, caCert)
	assert.NoError(t, err)

	c := GetBaseKubeconfig(caCert, "example.com", clusterName)
	assert.NotNil(t, c)

	assert.Len(t, c.Clusters, 1)
	assert.Contains(t, c.Clusters, clusterName)
	assert.Equal(t, []byte(caString), c.Clusters[clusterName].CertificateAuthorityData)
}

func TestGetInternalKubeconfigReconciler(t *testing.T) {
	checkKubeConfigRegeneration(t, nil)
}

func TestGetInternalKubeconfigReconcilerWithOrgs(t *testing.T) {
	checkKubeConfigRegeneration(t, []string{"org1", "org2"})
}

// fakeDataProvider provides just enough for testing kubeconfig creation.
type fakeDataProvider struct {
	caPair *triple.KeyPair
}

func (fake *fakeDataProvider) Cluster() *kubermaticv1.Cluster { return &kubermaticv1.Cluster{} }

func (fake *fakeDataProvider) ExternalIP() (*net.IP, error) { return nil, nil }

func (fake *fakeDataProvider) GetClusterRef() metav1.OwnerReference { return metav1.OwnerReference{} }

func (fake *fakeDataProvider) GetFrontProxyCA() (*triple.KeyPair, error) { return nil, nil }

func (fake *fakeDataProvider) GetRootCA() (*triple.KeyPair, error) { return fake.caPair, nil }

func (fake *fakeDataProvider) GetOpenVPNCA() (*ECDSAKeyPair, error) { return &ECDSAKeyPair{}, nil }

func (fake *fakeDataProvider) InClusterApiserverAddress() (string, error) { return "", nil }

func checkKubeConfigRegeneration(t *testing.T, orgs []string) {
	// get a ca for testing and setup fake data
	ca, err := triple.NewCA("test-ca")
	if err != nil {
		t.Fatalf("Failed to generate test root ca: %v", err)
	}
	data := &fakeDataProvider{caPair: ca}
	assert.NotNil(t, data)

	_, create := GetInternalKubeconfigReconciler("some-namespace", "some-name", "test-creator-cn", orgs, data, zap.NewNop().Sugar())()
	secret, err := create(&corev1.Secret{})
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, secret)

	secret2, err := create(secret.DeepCopy())
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, secret2)
	// kubeconfig should be unmodified
	assert.Equal(t, string(secret.Data[KubeconfigSecretKey]), string(secret2.Data[KubeconfigSecretKey]))
}
