/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package generator

import (
	"fmt"
	"os"
	"path"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/install/init/values"
	"k8c.io/kubermatic/v2/pkg/util/yaml"
	"k8c.io/kubermatic/v2/pkg/util/yamled"

	"github.com/sirupsen/logrus"
	yamlv3 "gopkg.in/yaml.v3"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Config is supposed to be a de-facto stable interface in the sense that
// the init command has flags to pass those (or you put them in via the interactive
// wizard), and we take those options and generate a working set of KKP
// configuration out of it.
//
// Think twice before removing/changing a value here because that likely means
// you also need to update the wizard steps and the command line flags, and we
// should consider that a breaking change.
type Config struct {
	DNS             string
	ExposeStrategy  kubermaticv1.ExposeStrategy
	GenerateSecrets bool
	FeatureGate     features.FeatureGate
}

func Start(in <-chan Config, log *logrus.Logger, outputDir string) <-chan bool {
	done := make(chan bool)
	go runGenerator(in, done, log, outputDir)

	return done
}

func runGenerator(in <-chan Config, done chan<- bool, log *logrus.Logger, outputDir string) {
	defer close(done)

	// wait for a generator config to come in.
	config := <-in

	log.Debugf("received configuration: %+v", config)

	if err := Generate(config, outputDir, log); err != nil {
		log.Errorf("failed to generate configuration files: %v", err)
	}

	log.Debug("finished generating configuration")

	// send out message that we are done.
	done <- true

	log.Debug("sent out message that we are done")
}

func Generate(config Config, outputDir string, log *logrus.Logger) error {
	kkpConfigFile, _ := os.Create(path.Join(outputDir, "kubermatic.yaml"))
	defer kkpConfigFile.Close()

	valuesFile, _ := os.Create(path.Join(outputDir, "values.yaml"))
	defer valuesFile.Close()

	secrets, err := generateSecrets(config)
	if err != nil {
		return fmt.Errorf("failed to generate secrets: %v", err)
	}

	kkpConfig, err := generateKubermaticConfiguration(config, secrets)
	if err != nil {
		return fmt.Errorf("failed to generate KubermaticConfiguration: %v", err)
	}

	chartValues, err := values.Values()
	if err != nil {
		return err
	}

	if err := setHelmValues(chartValues, config, secrets); err != nil {
		return fmt.Errorf("failed to set Helm values: %v", err)
	}

	if err := yaml.Encode(kkpConfig, kkpConfigFile); err != nil {
		return err
	}

	if err := yamlv3.NewEncoder(valuesFile).Encode(chartValues); err != nil {
		return err
	}

	return nil
}

func generateKubermaticConfiguration(config Config, secrets kkpSecrets) (*kubermaticv1.KubermaticConfiguration, error) {
	kkpConfig := &kubermaticv1.KubermaticConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
			Kind:       "KubermaticConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubermatic",
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			ExposeStrategy: config.ExposeStrategy,
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: config.DNS,
				CertificateIssuer: corev1.TypedLocalObjectReference{
					Kind: "ClusterIssuer",
					Name: "letsencrypt-prod",
				},
			},
			FeatureGates: config.FeatureGate,
			API:          kubermaticv1.KubermaticAPIConfiguration{},
		},
	}

	kkpConfig.Spec.Auth = kubermaticv1.KubermaticAuthConfiguration{
		TokenIssuer:        fmt.Sprintf("https://%s/dex", config.DNS),
		IssuerClientSecret: secrets.KubermaticIssuerClientSecret,
		IssuerCookieKey:    secrets.IssuerCookieKey,
		ServiceAccountKey:  secrets.ServiceAccountKey,
	}

	return kkpConfig, nil
}

func setHelmValues(chartValues *yamled.Document, config Config, secrets kkpSecrets) error {
	// set RedirectURIs for the 'kubermatic' dex client.
	if ok := chartValues.Set(yamled.Path{"dex", "clients", 0, "RedirectURIs"}, []string{
		fmt.Sprintf("https://%s", config.DNS),
		fmt.Sprintf("https://%s/projects", config.DNS),
	}); !ok {
		return fmt.Errorf("failed to set 'kubermatic' redirect URIs")
	}

	if ok := chartValues.Set(yamled.Path{"dex", "clients", 0, "secret"}, secrets.KubermaticClientSecret); !ok {
		return fmt.Errorf("failed to set secret for 'kubermatic' client")
	}

	// set kubermaticIssuer secret.
	if ok := chartValues.Set(yamled.Path{"dex", "clients", 1, "secret"}, secrets.KubermaticIssuerClientSecret); !ok {
		return fmt.Errorf("failed to set secret for 'kubermaticIssuer' client")
	}

	// set RedirectURIs for the 'kubermaticIssuer' dex client.
	if ok := chartValues.Set(yamled.Path{"dex", "clients", 1, "RedirectURIs"}, []string{
		fmt.Sprintf("https://%s/api/v1/kubeconfig", config.DNS),
		fmt.Sprintf("https://%s/api/v2/kubeconfig/secret", config.DNS),
		fmt.Sprintf("https://%s/api/v2/dashboard/login", config.DNS),
	}); !ok {
		return fmt.Errorf("failed to set 'kubermaticIssuer' redirect URIs")
	}

	// set Ingress domain.
	if ok := chartValues.Set(yamled.Path{"dex", "ingress", "host"}, config.DNS); !ok {
		return fmt.Errorf("failed to set dex ingress domain")
	}

	return nil
}
