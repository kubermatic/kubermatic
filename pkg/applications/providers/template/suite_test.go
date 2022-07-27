//go:build integration

/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package template

import (
	"context"
	"os"
	"path"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var testEnv *envtest.Environment
var userClient ctrlruntimeclient.Client
var ctx context.Context
var tmpDir string
var testEnvKubeConfig string

func TestHelmTemplate(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Helm template test suite")
}

var _ = BeforeSuite(func() {
	ctx = context.Background()

	kubermaticlog.Logger = kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{"../../../crd/k8c.io"},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = kubermaticv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	userClient, err = ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(userClient).ToNot(BeNil())

	tmpDir, err := os.MkdirTemp("", "test-env-helm")
	Expect(err).NotTo(HaveOccurred())
	testEnvKubeConfig = path.Join(tmpDir, "test-env-kubeconfig")

	restConfig := testEnv.Config

	config := *clientcmdapi.NewConfig()
	config.Clusters["testenv"] = &clientcmdapi.Cluster{
		Server:                   restConfig.Host,
		CertificateAuthorityData: restConfig.CAData,
	}
	config.CurrentContext = "default-context"
	config.Contexts["default-context"] = &clientcmdapi.Context{
		Cluster:   "testenv",
		Namespace: "default",
		AuthInfo:  "auth-info",
	}
	config.AuthInfos["auth-info"] = &clientcmdapi.AuthInfo{ClientCertificateData: restConfig.CertData, ClientKeyData: restConfig.KeyData}

	err = clientcmd.WriteToFile(config, testEnvKubeConfig)
	Expect(err).NotTo(HaveOccurred())

}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	os.RemoveAll(tmpDir)

	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
