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

package generator

import (
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"io"
	"time"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	appskubermaticv1 "k8c.io/api/v3/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/api/v3/pkg/semver"
	"k8c.io/kubermatic/v3/pkg/cni"
	kubermaticlog "k8c.io/kubermatic/v3/pkg/log"
	"k8c.io/kubermatic/v3/pkg/provider/kubernetes"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	// We call this in init because even thought it is possible to register the same
	// scheme multiple times it is an unprotected concurrent map access and these tests
	// are very good at making that panic
	if err := clusterv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		kubermaticlog.Logger.Fatalw("Failed to register scheme", zap.Stringer("api", clusterv1alpha1.SchemeGroupVersion), zap.Error(err))
	}
	if err := kubermaticv1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		kubermaticlog.Logger.Fatalw("Failed to register scheme", zap.Stringer("api", kubermaticv1.SchemeGroupVersion), zap.Error(err))
	}
	if err := v1beta1.AddToScheme(scheme.Scheme); err != nil {
		kubermaticlog.Logger.Fatalw("Failed to register scheme", zap.Stringer("api", v1beta1.SchemeGroupVersion), zap.Error(err))
	}
	if err := apiextensionsv1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		kubermaticlog.Logger.Fatalw("Failed to register scheme", zap.Stringer("api", apiextensionsv1.SchemeGroupVersion), zap.Error(err))
	}
	if err := osmv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		kubermaticlog.Logger.Fatalw("Failed to register scheme", zap.Stringer("api", osmv1alpha1.SchemeGroupVersion), zap.Error(err))
	}
}

const (
	// DefaultClusterID holds the test default cluster ID.
	DefaultClusterID = "defClusterID"
	// DefaultClusterName holds the test default cluster name.
	DefaultClusterName = "defClusterName"
	// TestOSdomain OpenStack domain.
	TestOSdomain = "OSdomain"
	// TestOSuserPass OpenStack user password.
	TestOSuserPass = "OSpass"
	// TestOSuserName OpenStack user name.
	TestOSuserName = "OSuser"
	// TestFakeCredential Fake provider credential name.
	TestFakeCredential = "fake"
)

var (
	// UserLastSeen hold a time the user was last seen.
	UserLastSeen = time.Date(2020, time.December, 31, 23, 0, 0, 0, time.UTC)
)

// GenUser generates a User resource
// note if the id is empty then it will be auto generated.
func GenUser(id, name, email string) *kubermaticv1.User {
	if len(id) == 0 {
		// the name of the object is derived from the email address and encoded as sha256
		id = fmt.Sprintf("%x", sha256.Sum256([]byte(email)))
	}

	h := sha512.New512_224()
	if _, err := io.WriteString(h, email); err != nil {
		// not nice, better to use t.Error
		panic("unable to generate a test user: " + err.Error())
	}

	return &kubermaticv1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: id,
			UID:  types.UID(fmt.Sprintf("fake-uid-%s", id)),
		},
		Spec: kubermaticv1.UserSpec{
			Name:  name,
			Email: email,
		},
		Status: kubermaticv1.UserStatus{
			LastSeen: metav1.NewTime(UserLastSeen),
		},
	}
}

// GenDefaultUser generates a default user.
func GenDefaultUser() *kubermaticv1.User {
	userEmail := "bob@acme.com"
	return GenUser("", "Bob", userEmail)
}

// GenDefaultKubermaticObjects generates default kubermatic object.
func GenDefaultKubermaticObjects(objs ...ctrlruntimeclient.Object) []ctrlruntimeclient.Object {
	defaultsObjs := []ctrlruntimeclient.Object{
		// add a user
		GenDefaultUser(),
		// add presets
		GenDefaultPreset(),
	}

	return append(defaultsObjs, objs...)
}

func GenCluster(id string, name string, creationTime time.Time, modifiers ...func(*kubermaticv1.Cluster)) *kubermaticv1.Cluster {
	version := *semver.NewSemverOrDie("9.9.9") // initTestEndpoint() configures KKP to know 8.8.8 and 9.9.9
	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: id,
			CreationTimestamp: func() metav1.Time {
				return metav1.NewTime(creationTime)
			}(),
		},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: "private-do1",
				ProviderName:   kubermaticv1.CloudProviderBringYourOwn,
				BringYourOwn:   &kubermaticv1.BringYourOwnCloudSpec{},
			},
			Version:               version,
			HumanReadableName:     name,
			EnableUserSSHKeyAgent: pointer.Bool(false),
			ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort,
			ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
				DNSDomain: "cluster.local",
				ProxyMode: "ipvs",
				IPVS: &kubermaticv1.IPVSConfiguration{
					StrictArp: pointer.Bool(true),
				},
				IPFamily: kubermaticv1.IPFamilyIPv4,
				Pods: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{"1.2.3.4/8"},
				},
				Services: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{"5.6.7.8/8"},
				},
				NodeCIDRMaskSizeIPv4: pointer.Int32(24),
			},
			CNIPlugin: &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCanal,
				Version: cni.GetDefaultCNIPluginVersion(kubermaticv1.CNIPluginTypeCanal),
			},
		},
		Status: kubermaticv1.ClusterStatus{
			ExtendedHealth: kubermaticv1.ExtendedClusterHealth{
				Apiserver:                    kubermaticv1.HealthStatusUp,
				ApplicationController:        kubermaticv1.HealthStatusUp,
				Scheduler:                    kubermaticv1.HealthStatusUp,
				Controller:                   kubermaticv1.HealthStatusUp,
				MachineController:            kubermaticv1.HealthStatusUp,
				Etcd:                         kubermaticv1.HealthStatusUp,
				UserClusterControllerManager: kubermaticv1.HealthStatusUp,
				CloudProviderInfrastructure:  kubermaticv1.HealthStatusUp,
			},
			Address: kubermaticv1.ClusterAddress{
				AdminToken:   "drphc2.g4kq82pnlfqjqt65",
				ExternalName: "w225mx4z66.asia-east1-a-1.cloud.kubermatic.io",
				IP:           "35.194.142.199",
				URL:          "https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885",
			},
			NamespaceName: kubernetes.NamespaceName(id),
			Versions: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      version,
				Apiserver:         version,
				ControllerManager: version,
				Scheduler:         version,
			},
		},
	}

	for _, modifier := range modifiers {
		modifier(cluster)
	}

	return cluster
}

func GenDefaultCluster() *kubermaticv1.Cluster {
	return GenCluster(DefaultClusterID, DefaultClusterName, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
}

func GenDefaultPreset() *kubermaticv1.Preset {
	return &kubermaticv1.Preset{
		ObjectMeta: metav1.ObjectMeta{
			Name: TestFakeCredential,
		},
		Spec: kubermaticv1.PresetSpec{
			OpenStack: &kubermaticv1.OpenStackPreset{
				Username: TestOSuserName, Password: TestOSuserPass, Domain: TestOSdomain,
			},
			Fake: &kubermaticv1.FakePreset{Token: "dummy_pluton_token"},
		},
	}
}

func GenAdminUser(name, email string, isAdmin bool) *kubermaticv1.User {
	user := GenUser("", name, email)
	user.Spec.IsAdmin = isAdmin
	return user
}

func GenClusterTemplate(name, id, scope, userEmail string) *kubermaticv1.ClusterTemplate {
	return &kubermaticv1.ClusterTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:        id,
			Labels:      map[string]string{kubermaticv1.ClusterTemplateScopeLabelKey: scope, kubermaticv1.ClusterTemplateHumanReadableNameLabelKey: name},
			Annotations: map[string]string{kubermaticv1.ClusterTemplateUserAnnotationKey: userEmail},
		},
		ClusterLabels:          nil,
		InheritedClusterLabels: nil,
		Credential:             "",
		Spec: kubermaticv1.ClusterSpec{
			HumanReadableName: name,
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: "fake-dc",
				Fake:           &kubermaticv1.FakeCloudSpec{},
			},
		},
	}
}

func GenClusterTemplateInstance(templateID, owner string, replicas int64) *kubermaticv1.ClusterTemplateInstance {
	return &kubermaticv1.ClusterTemplateInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:        templateID,
			Labels:      map[string]string{kubermaticv1.ClusterTemplateLabelKey: templateID},
			Annotations: map[string]string{kubermaticv1.ClusterTemplateInstanceOwnerAnnotationKey: owner},
		},
		Spec: kubermaticv1.ClusterTemplateInstanceSpec{
			ClusterTemplateID: templateID,
			Replicas:          replicas,
		},
	}
}

func GenRuleGroup(name, clusterName string, ruleGroupType kubermaticv1.RuleGroupType, isDefault bool) *kubermaticv1.RuleGroup {
	return &kubermaticv1.RuleGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: kubernetes.NamespaceName(clusterName),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "RuleGroup",
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		},
		Spec: kubermaticv1.RuleGroupSpec{
			RuleGroupType: ruleGroupType,
			IsDefault:     isDefault,
			Cluster: corev1.ObjectReference{
				Name: clusterName,
			},
			Data: GenerateTestRuleGroupData(name),
		},
	}
}

func GenerateTestRuleGroupData(ruleGroupName string) []byte {
	return []byte(fmt.Sprintf(`
name: %s
rules:
# Alert for any instance that is unreachable for >5 minutes.
- alert: InstanceDown
  expr: up == 0
  for: 5m
  labels:
    severity: page
  annotations:
    summary: "Instance  down"
`, ruleGroupName))
}

func GenApplicationDefinition(name string) *appskubermaticv1.ApplicationDefinition {
	return &appskubermaticv1.ApplicationDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       appskubermaticv1.ApplicationDefinitionKindName,
			APIVersion: appskubermaticv1.SchemeGroupVersion.String(),
		},
		Spec: appskubermaticv1.ApplicationDefinitionSpec{
			Method: appskubermaticv1.HelmTemplateMethod,
			Versions: []appskubermaticv1.ApplicationVersion{
				{
					Version: "v1.0.0",
					Template: appskubermaticv1.ApplicationTemplate{

						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								URL:          "https://charts.example.com",
								ChartName:    name,
								ChartVersion: "v1.0.0",
							},
						},
					},
				},
				{
					Version: "v1.1.0",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Git: &appskubermaticv1.GitSource{
								Remote: "https://git.example.com",
								Ref: appskubermaticv1.GitReference{
									Branch: "main",
									Tag:    "v1.1.0",
								},
							},
						},
					},
				},
			},
		},
	}
}
