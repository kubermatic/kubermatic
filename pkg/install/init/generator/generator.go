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

	"github.com/sirupsen/logrus"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/util/yaml"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Config struct {
	DNS            string
	ExposeStrategy kubermaticv1.ExposeStrategy
}

func Start(in <-chan Config, log *logrus.Logger, outputDir string) <-chan interface{} {
	done := make(chan interface{})
	go runGenerator(in, done, log, outputDir)

	return done
}

func runGenerator(in <-chan Config, done chan<- interface{}, log *logrus.Logger, outputDir string) {
	defer close(done)

	// wait for a generator config to come in.
	config := <-in

	if err := Generate(config, outputDir); err != nil {
		log.Errorf("failed to generate configuration files: %v", err)
	}
}

func Generate(config Config, outputDir string) error {
	f, _ := os.Create(path.Join(outputDir, "kubermatic.yaml"))
	defer f.Close()

	kkpConfig, err := generateKubermaticConfiguration(config)
	if err != nil {
		return fmt.Errorf("failed to generate KubermaticConfiguration: %v", err)
	}

	yaml.Encode(kkpConfig, f)

	return nil
}

func generateKubermaticConfiguration(config Config) (*kubermaticv1.KubermaticConfiguration, error) {
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
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: config.DNS,
				CertificateIssuer: corev1.TypedLocalObjectReference{
					Kind: "ClusterIssuer",
					Name: "letsencrypt-prod",
				},
			},
			FeatureGates: map[string]bool{
				features.OIDCKubeCfgEndpoint: true,
				features.OpenIDAuthPlugin:    true,
			},
			API: kubermaticv1.KubermaticAPIConfiguration{},
		},
	}

	kkpConfig.Spec.Auth = kubermaticv1.KubermaticAuthConfiguration{
		TokenIssuer:        fmt.Sprintf("https://%s/dex", config.DNS),
		IssuerClientSecret: "<to-be-generated>",
		IssuerCookieKey:    "<to-be-generated>",
		ServiceAccountKey:  "<to-be-generated>",
	}

	return kkpConfig, nil
}
