package cloudcontroller

import (
  "fmt"
  "k8c.io/kubermatic/v2/pkg/resources"
  "k8c.io/kubermatic/v2/pkg/resources/reconciling"
  "k8c.io/kubermatic/v2/pkg/resources/vpnsidecar"
  appsv1 "k8s.io/api/apps/v1"
  corev1 "k8s.io/api/core/v1"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "k8s.io/apimachinery/pkg/util/intstr"
)

const AnexiaCCMDeploymentName = "anx-cloud-controller-manager"

func anexiaDeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {

  return func() (name string, create reconciling.DeploymentCreator) {
    return AnexiaCCMDeploymentName, func(deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
      deployment.Labels = resources.BaseAppLabels(AnexiaCCMDeploymentName, nil)
      deployment.Spec.Replicas = resources.Int32(1)

      deployment.Spec.Selector = &metav1.LabelSelector{
        MatchLabels: resources.BaseAppLabels(AnexiaCCMDeploymentName, nil),
      }

      podLabels, err := data.GetPodTemplateLabels(AnexiaCCMDeploymentName, deployment.Spec.Template.Spec.Volumes, nil)

      deployment.Spec.Template.ObjectMeta = metav1.ObjectMeta{
        Labels: podLabels,
      }

      f := false
      deployment.Spec.Template.Spec.AutomountServiceAccountToken = &f

      openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, openvpnClientContainerName)
      if err != nil {
        return nil, fmt.Errorf("failed to get openvpn sidecar: %v", err)
      }

      credentials, err := resources.GetCredentials(data)
      if err != nil {
        return nil, fmt.Errorf("failed to get credentials: %v", err)
      }

      deployment.Spec.Template.Spec.Volumes = append(getVolumes())

      deployment.Spec.Template.Spec.Containers = []corev1.Container{
        *openvpnSidecar,
        {
          Name:            ccmContainerName,
          Image:           data.ImageRegistry(resources.RegistryAnexia) + "/anexia/anx-cloud-controller-manager:0.1.0",
          Command: []string{
            "/app/ccm",
            "--cloud-provider=anexia",
            "--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
          },
          Env: []corev1.EnvVar{
            {
              Name:  "ANEXIA_TOKEN",
              Value: credentials.Anexia.Token,
            },
          },
          Ports: []corev1.ContainerPort{
            {
              Name:          "http",
              ContainerPort: 8080,
              Protocol:      "TCP",
            },
          },
          LivenessProbe: &corev1.Probe{
            Handler: corev1.Handler{
              HTTPGet: &corev1.HTTPGetAction{
                Path:   "/healthz",
                Port:   intstr.FromString("http"),
                Scheme: "HTTPS",
              },
            },
            InitialDelaySeconds: 5,
            TimeoutSeconds:      10,
            PeriodSeconds:       20,
            SuccessThreshold:    1,
            FailureThreshold:    3,
          },
          ReadinessProbe: &corev1.Probe{
           Handler: corev1.Handler{
             HTTPGet: &corev1.HTTPGetAction{
               Path:   "/healthz",
               Port:   intstr.FromString("http"),
               Scheme: "HTTPS",
             },
           },
            InitialDelaySeconds: 5,
            TimeoutSeconds:      10,
            PeriodSeconds:       20,
            SuccessThreshold:    1,
            FailureThreshold:    3,
          },
          VolumeMounts: getVolumeMounts(),
        },
      }

      if err != nil {
        return nil, err
      }
      return deployment, nil
    }
  }
}
