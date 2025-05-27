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
	"encoding/json"
	"fmt"
	"io"
	"time"

	constrainttemplatev1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1"
	regoschema "github.com/open-policy-agent/frameworks/constraint/pkg/client/drivers/rego/schema"
	"github.com/open-policy-agent/frameworks/constraint/pkg/core/templates"
	gatekeeperconfigv1alpha1 "github.com/open-policy-agent/gatekeeper/v3/apis/config/v1alpha1"
	"go.uber.org/zap"

	apiv2 "k8c.io/kubermatic/sdk/v2/api/v2"
	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/cni"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"k8s.io/utils/ptr"
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
	if err := gatekeeperconfigv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		kubermaticlog.Logger.Fatalw("Failed to register scheme", zap.Stringer("api", gatekeeperconfigv1alpha1.GroupVersion), zap.Error(err))
	}
	if err := osmv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		kubermaticlog.Logger.Fatalw("Failed to register scheme", zap.Stringer("api", osmv1alpha1.SchemeGroupVersion), zap.Error(err))
	}
}

const (
	// UserID holds a test user ID.
	UserID = "1233"
	// UserName holds a test user name.
	UserName = "user1"
	// UserEmail holds a test user email.
	UserEmail = "john@acme.com"
	// ClusterID holds the test cluster ID.
	ClusterID = "AbcClusterID"
	// DefaultClusterID holds the test default cluster ID.
	DefaultClusterID = "defClusterID"
	// DefaultClusterName holds the test default cluster name.
	DefaultClusterName = "defClusterName"
	// ProjectName holds the test project ID.
	ProjectName = "my-first-project-ID"
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

func GenTestSeed(modifiers ...func(seed *kubermaticv1.Seed)) *kubermaticv1.Seed {
	seed := &kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "us-central1",
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.SeedSpec{
			Location: "us-central",
			Country:  "US",
			Datacenters: map[string]kubermaticv1.Datacenter{
				"private-do1": {
					Country:  "NL",
					Location: "US ",
					Spec: kubermaticv1.DatacenterSpec{
						Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{
							Region: "ams2",
						},
						EnforcePodSecurityPolicy: true,
					},
					Node: &kubermaticv1.NodeSettings{
						PauseImage: "image-pause",
					},
				},
				"regular-do1": {
					Country:  "NL",
					Location: "Amsterdam",
					Spec: kubermaticv1.DatacenterSpec{
						Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{
							Region: "ams2",
						},
					},
				},
				"restricted-fake-dc": {
					Country:  "NL",
					Location: "Amsterdam",
					Spec: kubermaticv1.DatacenterSpec{
						Fake:           &kubermaticv1.DatacenterSpecFake{},
						RequiredEmails: []string{"example.com"},
					},
				},
				"restricted-fake-dc2": {
					Country:  "NL",
					Location: "Amsterdam",
					Spec: kubermaticv1.DatacenterSpec{
						Fake:           &kubermaticv1.DatacenterSpecFake{},
						RequiredEmails: []string{"23f67weuc.com", "example.com", "12noifsdsd.org"},
					},
				},
				"fake-dc": {
					Location: "Henrik's basement",
					Country:  "Germany",
					Spec: kubermaticv1.DatacenterSpec{
						Fake: &kubermaticv1.DatacenterSpecFake{},
					},
				},
				"audited-dc": {
					Location: "Finanzamt Castle",
					Country:  "Germany",
					Spec: kubermaticv1.DatacenterSpec{
						Fake:                &kubermaticv1.DatacenterSpecFake{},
						EnforceAuditLogging: true,
					},
				},
				"psp-dc": {
					Location: "Alexandria",
					Country:  "Egypt",
					Spec: kubermaticv1.DatacenterSpec{
						Fake:                     &kubermaticv1.DatacenterSpecFake{},
						EnforcePodSecurityPolicy: true,
					},
				},
				"node-dc": {
					Location: "Santiago",
					Country:  "Chile",
					Spec: kubermaticv1.DatacenterSpec{
						Fake: &kubermaticv1.DatacenterSpecFake{},
					},
					Node: &kubermaticv1.NodeSettings{
						ProxySettings: kubermaticv1.ProxySettings{
							HTTPProxy: kubermaticv1.NewProxyValue("HTTPProxy"),
						},
						InsecureRegistries: []string{"incsecure-registry"},
						RegistryMirrors:    []string{"http://127.0.0.1:5001"},
						PauseImage:         "pause-image",
					},
				},
			},
		},
	}
	seed.Status.Versions.Kubermatic = kubermatic.GetFakeVersions().GitVersion
	for _, modifier := range modifiers {
		modifier(seed)
	}
	return seed
}

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

// GenUserWithGroups generates a User resource
// note if the id is empty then it will be auto generated.
func GenUserWithGroups(id, name, email string, groups []string) *kubermaticv1.User {
	user := GenUser(id, name, email)
	user.Spec.Groups = groups
	return user
}

// DefaultCreationTimestamp returns default test timestamp.
func DefaultCreationTimestamp() time.Time {
	return time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)
}

// GenDefaultUser generates a default user.
func GenDefaultUser() *kubermaticv1.User {
	userEmail := "bob@acme.com"
	return GenUser("", "Bob", userEmail)
}

// GenProject generates new empty project.
func GenProject(name string, phase kubermaticv1.ProjectPhase, creationTime time.Time) *kubermaticv1.Project {
	return &kubermaticv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:              fmt.Sprintf("%s-%s", name, "ID"),
			CreationTimestamp: metav1.NewTime(creationTime),
		},
		Spec: kubermaticv1.ProjectSpec{
			Name: name,
		},
		Status: kubermaticv1.ProjectStatus{
			Phase: phase,
		},
	}
}

// GenDefaultProject generates a default project.
func GenDefaultProject() *kubermaticv1.Project {
	return GenProject("my-first-project", kubermaticv1.ProjectActive, DefaultCreationTimestamp())
}

// GenBinding generates a binding.
func GenBinding(projectID, email, group string) *kubermaticv1.UserProjectBinding {
	return &kubermaticv1.UserProjectBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s-%s", projectID, email, group),
		},
		Spec: kubermaticv1.UserProjectBindingSpec{
			UserEmail: email,
			ProjectID: projectID,
			Group:     fmt.Sprintf("%s-%s", group, projectID),
		},
	}
}

// GenGroupBinding generates a binding.
func GenGroupBinding(projectID, groupName, role string) *kubermaticv1.GroupProjectBinding {
	return &kubermaticv1.GroupProjectBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-xxxxxxxxxx", projectID),
			Labels: map[string]string{
				kubermaticv1.ProjectIDLabelKey: projectID,
			},
		},
		Spec: kubermaticv1.GroupProjectBindingSpec{
			Role:      role,
			ProjectID: projectID,
			Group:     groupName,
		},
	}
}

// GenDefaultOwnerBinding generates default owner binding.
func GenDefaultOwnerBinding() *kubermaticv1.UserProjectBinding {
	return GenBinding(GenDefaultProject().Name, GenDefaultUser().Spec.Email, "owners")
}

// GenDefaultKubermaticObjects generates default kubermatic object.
func GenDefaultKubermaticObjects(objs ...ctrlruntimeclient.Object) []ctrlruntimeclient.Object {
	defaultsObjs := []ctrlruntimeclient.Object{
		// add a project
		GenDefaultProject(),
		// add a user
		GenDefaultUser(),
		// make a user the owner of the default project
		GenDefaultOwnerBinding(),
		// add presets
		GenDefaultPreset(),
	}

	return append(defaultsObjs, objs...)
}

func GenCluster(id string, name string, projectID string, creationTime time.Time, modifiers ...func(*kubermaticv1.Cluster)) *kubermaticv1.Cluster {
	version := *semver.NewSemverOrDie("9.9.9") // initTestEndpoint() configures KKP to know 8.8.8 and 9.9.9
	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   id,
			Labels: map[string]string{"project-id": projectID},
			CreationTimestamp: func() metav1.Time {
				return metav1.NewTime(creationTime)
			}(),
		},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: "private-do1",
				ProviderName:   string(kubermaticv1.FakeCloudProvider),
				Fake:           &kubermaticv1.FakeCloudSpec{Token: "SecretToken"},
			},
			Version:               version,
			HumanReadableName:     name,
			EnableUserSSHKeyAgent: ptr.To(false),
			ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort,
			ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
				DNSDomain: "cluster.local",
				ProxyMode: "ipvs",
				IPVS: &kubermaticv1.IPVSConfiguration{
					StrictArp: ptr.To(true),
				},
				IPFamily: kubermaticv1.IPFamilyIPv4,
				Pods: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{"1.2.3.4/8"},
				},
				Services: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{"5.6.7.8/8"},
				},
				NodeCIDRMaskSizeIPv4: ptr.To[int32](24),
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
	return GenCluster(DefaultClusterID, DefaultClusterName, GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
}

func GenTestMachine(name, rawProviderSpec string, labels map[string]string, ownerRef []metav1.OwnerReference) *clusterv1alpha1.Machine {
	return &clusterv1alpha1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			UID:             types.UID(name + "-machine"),
			Name:            name,
			Namespace:       metav1.NamespaceSystem,
			Labels:          labels,
			OwnerReferences: ownerRef,
		},
		TypeMeta: metav1.TypeMeta{
			Kind: "Machine",
		},
		Spec: clusterv1alpha1.MachineSpec{
			ProviderSpec: clusterv1alpha1.ProviderSpec{
				Value: &runtime.RawExtension{
					Raw: []byte(rawProviderSpec),
				},
			},
			Versions: clusterv1alpha1.MachineVersionInfo{
				Kubelet: "v9.9.9", // initTestEndpoint() configures KKP to know 8.8.8 and 9.9.9
			},
		},
	}
}

func GenDefaultPreset() *kubermaticv1.Preset {
	return &kubermaticv1.Preset{
		ObjectMeta: metav1.ObjectMeta{
			Name: TestFakeCredential,
		},
		Spec: kubermaticv1.PresetSpec{
			Openstack: &kubermaticv1.Openstack{
				Username: TestOSuserName, Password: TestOSuserPass, Domain: TestOSdomain,
			},
			Fake: &kubermaticv1.Fake{Token: "dummy_pluton_token"},
		},
	}
}

func GenBlacklistTokenSecret(name string, tokens []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: resources.KubermaticNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			resources.TokenBlacklist: tokens,
		},
	}
}

func GenAdminUser(name, email string, isAdmin bool) *kubermaticv1.User {
	user := GenUser("", name, email)
	user.Spec.IsAdmin = isAdmin
	return user
}

const requiredLabelsRegoScript = `package k8srequiredlabels

deny[{"msg": msg, "details": {"missing_labels": missing}}] {
  provided := {label | input.review.object.metadata.labels[label]}
  required := {label | label := input.parameters.labels[_]}
  missing := required - provided
  count(missing) > 0
  msg := sprintf("you must provide labels: %v", [missing])
}`

func GenConstraintTemplate(name string) *kubermaticv1.ConstraintTemplate {
	ct := &kubermaticv1.ConstraintTemplate{}
	ct.Name = name
	ct.Spec = kubermaticv1.ConstraintTemplateSpec{
		CRD: constrainttemplatev1.CRD{
			Spec: constrainttemplatev1.CRDSpec{
				Names: constrainttemplatev1.Names{
					Kind:       "labelconstraint",
					ShortNames: []string{"lc"},
				},
				Validation: &constrainttemplatev1.Validation{
					OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
						Properties: map[string]apiextensionsv1.JSONSchemaProps{
							"labels": {
								Type: "array",
								Items: &apiextensionsv1.JSONSchemaPropsOrArray{
									Schema: &apiextensionsv1.JSONSchemaProps{
										Type: "string",
									},
								},
							},
						},
						Required: []string{"labels"},
					},
				},
			},
		},
		Targets: []constrainttemplatev1.Target{
			{
				Target: "admission.k8s.gatekeeper.sh",
				Code: []constrainttemplatev1.Code{
					{
						Engine: regoschema.Name,
						Source: &templates.Anything{
							Value: (&regoschema.Source{
								Rego: requiredLabelsRegoScript,
							}).ToUnstructured(),
						},
					},
				},
			},
		},
		Selector: kubermaticv1.ConstraintTemplateSelector{
			Providers: []string{"aws", "gcp"},
			LabelSelector: metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "cluster",
						Operator: metav1.LabelSelectorOpExists,
					},
				},
				MatchLabels: map[string]string{
					"deployment": "prod",
					"domain":     "sales",
				},
			},
		},
	}

	return ct
}

func RegisterScheme(builder runtime.SchemeBuilder) error {
	return builder.AddToScheme(scheme.Scheme)
}

func GenConstraint(name, namespace, kind string) *kubermaticv1.Constraint {
	ct := &kubermaticv1.Constraint{}
	ct.Name = name
	ct.Namespace = namespace
	ct.Spec = kubermaticv1.ConstraintSpec{
		ConstraintType: kind,
		Match: kubermaticv1.Match{
			Kinds: []kubermaticv1.Kind{
				{Kinds: []string{"namespace"}, APIGroups: []string{""}},
			},
		},
		Parameters: map[string]json.RawMessage{
			"labels": []byte(`["gatekeeper","opa"]`),
		},
		Selector: kubermaticv1.ConstraintSelector{
			Providers: []string{"aws", "gcp"},
			LabelSelector: metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "cluster",
						Operator: metav1.LabelSelectorOpExists,
					},
				},
				MatchLabels: map[string]string{
					"deployment": "prod",
					"domain":     "sales",
				},
			},
		},
	}

	return ct
}

func GenDefaultAPIConstraint(name, kind string) apiv2.Constraint {
	return apiv2.Constraint{
		Name: name,
		Spec: kubermaticv1.ConstraintSpec{
			ConstraintType: kind,
			Match: kubermaticv1.Match{
				Kinds: []kubermaticv1.Kind{
					{Kinds: []string{"namespace"}, APIGroups: []string{""}},
				},
			},
			Parameters: map[string]json.RawMessage{
				"labels": []byte(`["gatekeeper","opa"]`),
			},
			Selector: kubermaticv1.ConstraintSelector{
				Providers: []string{"aws", "gcp"},
				LabelSelector: metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "cluster",
							Operator: metav1.LabelSelectorOpExists,
						},
					},
					MatchLabels: map[string]string{
						"deployment": "prod",
						"domain":     "sales",
					},
				},
			},
		},
		Status: &apiv2.ConstraintStatus{
			Enforcement:    "true",
			AuditTimestamp: "2019-05-11T01:46:13Z",
			Violations: []apiv2.Violation{
				{
					EnforcementAction: "deny",
					Kind:              "Namespace",
					Message:           "'you must provide labels: {\"gatekeeper\"}'",
					Name:              "default",
				},
				{
					EnforcementAction: "deny",
					Kind:              "Namespace",
					Message:           "'you must provide labels: {\"gatekeeper\"}'",
					Name:              "gatekeeper",
				},
			},
			Synced: ptr.To(true),
		},
	}
}

func GenClusterTemplate(name, id, projectID, scope, userEmail string) *kubermaticv1.ClusterTemplate {
	return &kubermaticv1.ClusterTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:        id,
			Labels:      map[string]string{kubermaticv1.ClusterTemplateScopeLabelKey: scope, kubermaticv1.ProjectIDLabelKey: projectID, kubermaticv1.ClusterTemplateHumanReadableNameLabelKey: name},
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

func GenClusterTemplateInstance(projectID, templateID, owner string, replicas int64) *kubermaticv1.ClusterTemplateInstance {
	return &kubermaticv1.ClusterTemplateInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-%s", projectID, templateID),
			Labels:      map[string]string{kubermaticv1.ClusterTemplateLabelKey: templateID, kubermaticv1.ProjectIDLabelKey: projectID},
			Annotations: map[string]string{kubermaticv1.ClusterTemplateInstanceOwnerAnnotationKey: owner},
		},
		Spec: kubermaticv1.ClusterTemplateInstanceSpec{
			ProjectID:         projectID,
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
			Kind:       kubermaticv1.RuleGroupKindName,
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

func GenMLAAdminSetting(name, clusterName string, value int32) *kubermaticv1.MLAAdminSetting {
	return &kubermaticv1.MLAAdminSetting{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: kubernetes.NamespaceName(clusterName),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       kubermaticv1.MLAAdminSettingKindName,
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		},
		Spec: kubermaticv1.MLAAdminSettingSpec{
			ClusterName: clusterName,
			MonitoringRateLimits: &kubermaticv1.MonitoringRateLimitSettings{
				IngestionRate:      value,
				IngestionBurstSize: value,
				MaxSeriesPerMetric: value,
				MaxSeriesTotal:     value,
				QueryRate:          value,
				QueryBurstSize:     value,
				MaxSamplesPerQuery: value,
				MaxSeriesPerQuery:  value,
			},
			LoggingRateLimits: &kubermaticv1.LoggingRateLimitSettings{
				IngestionRate:      value,
				IngestionBurstSize: value,
				QueryRate:          value,
				QueryBurstSize:     value,
			},
		},
	}
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
