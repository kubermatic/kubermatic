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
	"bytes"
	"fmt"
	"strconv"
	"text/template"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/servingcerthelper"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	utilpointer "k8s.io/utils/pointer"
)

var (
	consoleTemplate = template.Must(template.New("base").Parse(consoleTemplateRaw))
)

const (
	ConsoleOAuthSecretName       = "openshift-console-oauth-client-secret"
	consoleServingCertSecretName = "openshift-console-serving-cert"
	consoleConfigMapName         = "openshift-console-config"
	consoleConfigMapKey          = "console-config.yaml"
	consoleDeploymentName        = "openshift-console"
	// ConsoleAdminPasswordSecretName is the name of the secret that contains
	// the bootstrap admin user for Openshift OAuth
	ConsoleAdminPasswordSecretName = "openshift-bootstrap-password"
	// ConsoleAdminUserName is the name of the bootstrap admin user for oauth/the console
	ConsoleAdminUserName = "kubeadmin"
	// ConsoleListenPort is the port the console listens on
	ConsoleListenPort  = 8443
	consoleTemplateRaw = `apiVersion: console.openshift.io/v1
auth:
  clientID: console
  clientSecretFile: /var/oauth-config/clientSecret
  logoutRedirect: https://{{ .ExternalURL }}/api/v1/projects/{{ .ProjectID }}/dc/{{ .SeedName }}/clusters/{{ .ClusterName }}/openshift/console/login
  oauthEndpointCAFile: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
clusterInfo:
  consoleBaseAddress: https://{{ .ExternalURL }}
  consoleBasePath: /api/v1/projects/{{ .ProjectID }}/dc/{{ .SeedName }}/clusters/{{ .ClusterName }}/openshift/console/proxy/
  masterPublicURL: {{ .APIServerURL }}
customization:
  branding: ocp
  documentationBaseURL: https://docs.openshift.com/container-platform/4.1/
kind: ConsoleConfig
servingInfo:
  bindAddress: http://0.0.0.0:{{ .ListenPort }}
  certFile: /var/serving-cert/serving.crt
  keyFile: /var/serving-cert/serving.key`
)

func ConsoleDeployment(data openshiftData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return consoleDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {

			d.Spec.Template.Spec.AutomountServiceAccountToken = utilpointer.BoolPtr(false)
			d.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: openshiftImagePullSecretName}}
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(consoleDeploymentName, nil),
			}
			image, err := consoleImage(data.Cluster().Spec.Version.String(), data.ImageRegistry(""))
			if err != nil {
				return nil, err
			}

			d.Spec.Template.Spec.InitContainers = []corev1.Container{}
			d.Spec.Template.Spec.Containers = []corev1.Container{{
				Name:  "console",
				Image: image,
				Command: []string{
					"/opt/bridge/bin/bridge",
					"--public-dir=/opt/bridge/static",
					"--config=/etc/console-config/console-config.yaml",
				},
				Env: []corev1.EnvVar{
					{
						Name:  "KUBERNETES_SERVICE_HOST",
						Value: data.Cluster().Address.InternalName,
					},
					{
						Name:  "KUBERNETES_SERVICE_PORT",
						Value: strconv.Itoa(int(data.Cluster().Address.Port)),
					},
					{
						Name:  "KUBECONFIG",
						Value: "/etc/kubernetes/kubeconfig/kubeconfig",
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      ConsoleOAuthSecretName,
						MountPath: "/var/oauth-config",
					},
					{
						Name:      consoleConfigMapName,
						MountPath: "/etc/console-config",
					},
					{
						Name:      consoleServingCertSecretName,
						MountPath: "/var/serving-cert",
					},
					{
						Name:      resources.CASecretName,
						MountPath: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
						SubPath:   "ca.crt",
					},
					{
						Name:      resources.AdminKubeconfigSecretName,
						MountPath: "/var/run/secrets/kubernetes.io/serviceaccount/token",
						SubPath:   "token",
					},
					// Used by the http prober
					{
						Name:      resources.AdminKubeconfigSecretName,
						MountPath: "/etc/kubernetes/kubeconfig",
					},
				},
			}}
			d.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: resources.AdminKubeconfigSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{SecretName: resources.AdminKubeconfigSecretName},
					},
				},
				{
					Name: ConsoleOAuthSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{SecretName: ConsoleOAuthSecretName},
					},
				},
				{
					Name: consoleServingCertSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{SecretName: consoleServingCertSecretName},
					},
				},
				{
					Name: consoleConfigMapName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: consoleConfigMapName},
						},
					},
				},
				{
					Name: resources.CASecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.CASecretName,
						},
					},
				},
			}

			podLabels, err := data.GetPodTemplateLabels(consoleDeploymentName, d.Spec.Template.Spec.Volumes, nil)
			if err != nil {
				return nil, err
			}
			d.Spec.Template.Labels = podLabels

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, d.Spec.Template.Spec, sets.NewString("console"), "OAuthClient,oauth.openshift.io/v1")
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			d.Spec.Template.Spec = *wrappedPodSpec
			return d, nil
		}
	}
}

func ConsoleConfigCreator(data openshiftData) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return consoleConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {

			data := struct {
				APIServerURL string
				ExternalURL  string
				ProjectID    string
				SeedName     string
				ClusterName  string
				ListenPort   string
			}{
				APIServerURL: data.Cluster().Address.URL,
				ExternalURL:  data.ExternalURL(),
				ProjectID:    data.Cluster().Labels[kubermaticv1.ProjectIDLabelKey],
				SeedName:     data.Seed().Name,
				ClusterName:  data.Cluster().Name,
				ListenPort:   strconv.Itoa(ConsoleListenPort),
			}
			buffer := bytes.NewBuffer([]byte{})
			if err := consoleTemplate.Execute(buffer, data); err != nil {
				return nil, fmt.Errorf("failed to render template for openshift console: %v", err)
			}

			if cm.Data == nil {
				cm.Data = map[string]string{}
			}
			cm.Data[consoleConfigMapKey] = buffer.String()
			return cm, nil
		}
	}
}

func ConsoleServingCertCreator(caGetter servingcerthelper.CAGetter) reconciling.NamedSecretCreatorGetter {
	return servingcerthelper.ServingCertSecretCreator(caGetter,
		consoleServingCertSecretName,
		// We proxy this from the API
		"console.openshift.seed.tld",
		[]string{"console.openshift.seed.tld"},
		nil)
}
