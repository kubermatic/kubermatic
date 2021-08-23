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

package test

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	ver "github.com/Masterminds/semver/v3"
	constrainttemplatev1beta1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"
	gatekeeperconfigv1alpha1 "github.com/open-policy-agent/gatekeeper/apis/config/v1alpha1"
	prometheusapi "github.com/prometheus/client_golang/api"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	k8cuserclusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	kubermaticfakeclientset "k8c.io/kubermatic/v2/pkg/crd/client/clientset/versioned/fake"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/auth"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/serviceaccount"
	"k8c.io/kubermatic/v2/pkg/version"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/kubermatic/v2/pkg/watcher"
	kuberneteswatcher "k8c.io/kubermatic/v2/pkg/watcher/kubernetes"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sjson "k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/sets"
	kubernetesclientset "k8s.io/client-go/kubernetes"
	fakerestclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/reference"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	// We call this in init because even thought it is possible to register the same
	// scheme multiple times it is an unprotected concurrent map access and these tests
	// are very good at making that panic
	if err := clusterv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		kubermaticlog.Logger.Fatalw("failed to add cluster/v1alpha1 scheme to scheme.Scheme", "error", err)
	}
	if err := v1beta1.AddToScheme(scheme.Scheme); err != nil {
		kubermaticlog.Logger.Fatalw("failed to register scheme metrics/v1beta1", "error", err)
	}
	if err := apiextensionv1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		kubermaticlog.Logger.Fatalw("failed to register scheme apiextension/v1beta1", "error", err)
	}
	if err := gatekeeperconfigv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		kubermaticlog.Logger.Fatalw("failed to register scheme gatekeeperconfig/v1alpha1", "error", err)
	}
}

const (
	// UserID holds a test user ID
	UserID = "1233"
	// UserID2 holds a test user ID
	UserID2 = "1523"
	// UserName holds a test user name
	UserName = "user1"
	// UserName2 holds a test user name
	UserName2 = "user2"
	// UserEmail holds a test user email
	UserEmail = "john@acme.com"
	// UserEmail2 holds a test user email
	UserEmail2 = "bob@example.com"
	// ClusterID holds the test cluster ID
	ClusterID = "AbcClusterID"
	// DefaultClusterID holds the test default cluster ID
	DefaultClusterID = "defClusterID"
	// DefaultClusterName holds the test default cluster name
	DefaultClusterName = "defClusterName"
	// ProjectName holds the test project ID
	ProjectName = "my-first-project-ID"
	// TestDatacenter holds datacenter name
	TestSeedDatacenter = "us-central1"
	// TestServiceAccountHashKey authenticates the service account's token value using HMAC
	TestServiceAccountHashKey = "eyJhbGciOiJIUzI1NeyJhbGciOiJIUzI1N"
	// TestFakeToken signed JWT token with fake data
	TestFakeToken = "eyJhbGciOiJIUzI1NiJ9.eyJlbWFpbCI6IjEiLCJleHAiOjE2NDk3NDg4NTYsImlhdCI6MTU1NTA1NDQ1NiwibmJmIjoxNTU1MDU0NDU2LCJwcm9qZWN0X2lkIjoiMSIsInRva2VuX2lkIjoiMSJ9.Q4qxzOaCvUnWfXneY654YiQjUTd_Lsmw56rE17W2ouo"
	// TestOSdomain OpenStack domain
	TestOSdomain = "OSdomain"
	// TestOSuserPass OpenStack user password
	TestOSuserPass = "OSpass"
	// TestOSuserName OpenStack user name
	TestOSuserName = "OSuser"
	// TestFakeCredential Fake provider credential name
	TestFakeCredential = "fake"
	// RequiredEmailDomain required domain for predefined credentials
	RequiredEmailDomain = "acme.com"
	// DefaultKubernetesVersion kubernetes version
	DefaultKubernetesVersion = "1.17.9"
	// Kubermatic namespace
	KubermaticNamespace = "kubermatic"
)

// GetUser is a convenience function for generating apiv1.User
func GetUser(email, id, name string) apiv1.User {
	u := apiv1.User{
		ObjectMeta: apiv1.ObjectMeta{
			ID:   id,
			Name: name,
		},
		Email: email,
	}
	return u
}

// newRoutingFunc defines a func that knows how to create and set up routing required for testing
// this function is temporal until all types end up in their own packages.
// it is meant to be used by legacy handler.createTestEndpointAndGetClients function
type newRoutingFunc func(
	adminProvider provider.AdminProvider,
	settingsProvider provider.SettingsProvider,
	userInfoGetter provider.UserInfoGetter,
	seedsGetter provider.SeedsGetter,
	seedClientGetter provider.SeedClientGetter,
	clusterProviderGetter provider.ClusterProviderGetter,
	addonProviderGetter provider.AddonProviderGetter,
	addonConfigProvider provider.AddonConfigProvider,
	newSSHKeyProvider provider.SSHKeyProvider,
	privilegedSSHKeyProvider provider.PrivilegedSSHKeyProvider,
	userProvider provider.UserProvider,
	serviceAccountProvider provider.ServiceAccountProvider,
	privilegedServiceAccountProvider provider.PrivilegedServiceAccountProvider,
	serviceAccountTokenProvider provider.ServiceAccountTokenProvider,
	privilegedServiceAccountTokenProvider provider.PrivilegedServiceAccountTokenProvider,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	oidcIssuerVerifier auth.OIDCIssuerVerifier,
	tokenVerifiers auth.TokenVerifier,
	tokenExtractors auth.TokenExtractor,
	prometheusClient prometheusapi.Client,
	projectMemberProvider *kubernetes.ProjectMemberProvider,
	privilegedProjectMemberProvider provider.PrivilegedProjectMemberProvider,
	versions []*version.Version,
	updates []*version.Update,
	saTokenAuthenticator serviceaccount.TokenAuthenticator,
	saTokenGenerator serviceaccount.TokenGenerator,
	eventRecorderProvider provider.EventRecorderProvider,
	presetsProvider provider.PresetProvider,
	admissionPluginProvider provider.AdmissionPluginsProvider,
	settingsWatcher watcher.SettingsWatcher,
	userWatcher watcher.UserWatcher,
	externalClusterProvider provider.ExternalClusterProvider,
	privilegedExternalClusterProvider provider.PrivilegedExternalClusterProvider,
	constraintTemplateProvider provider.ConstraintTemplateProvider,
	constraintProviderGetter provider.ConstraintProviderGetter,
	alertmanagerProviderGetter provider.AlertmanagerProviderGetter,
	clusterTemplateProvider provider.ClusterTemplateProvider,
	clusterTemplateInstanceProviderGetter provider.ClusterTemplateInstanceProviderGetter,
	ruleGroupProviderGetter provider.RuleGroupProviderGetter,
	kubermaticVersions kubermatic.Versions,
	defaultConstraintProvider provider.DefaultConstraintProvider,
	privilegedAllowedRegistryProvider provider.PrivilegedAllowedRegistryProvider,
	etcdBackupConfigProviderGetter provider.EtcdBackupConfigProviderGetter,
	etcdRestoreProviderGetter provider.EtcdRestoreProviderGetter,
	etcdBackupConfigProjectProviderGetter provider.EtcdBackupConfigProjectProviderGetter,
	etcdRestoreProjectProviderGetter provider.EtcdRestoreProjectProviderGetter,
) http.Handler

func getRuntimeObjects(objs ...ctrlruntimeclient.Object) []runtime.Object {
	runtimeObjects := []runtime.Object{}
	for _, obj := range objs {
		runtimeObjects = append(runtimeObjects, obj.(runtime.Object))
	}

	return runtimeObjects
}

func initTestEndpoint(user apiv1.User, seedsGetter provider.SeedsGetter, kubeObjects, machineObjects, kubermaticObjects []ctrlruntimeclient.Object, versions []*version.Version, updates []*version.Update, routingFunc newRoutingFunc) (http.Handler, *ClientsSets, error) {
	ctx := context.Background()

	allObjects := kubeObjects
	allObjects = append(allObjects, machineObjects...)
	allObjects = append(allObjects, kubermaticObjects...)
	fakeClient := fakectrlruntimeclient.
		NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(allObjects...).
		Build()
	kubermaticClient := kubermaticfakeclientset.NewSimpleClientset(getRuntimeObjects(kubermaticObjects...)...)
	kubernetesClient := fakerestclient.NewSimpleClientset(getRuntimeObjects(kubeObjects...)...)
	fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
		return fakeClient, nil
	}

	sshKeyProvider := kubernetes.NewSSHKeyProvider(fakeImpersonationClient, fakeClient)
	privilegedSSHKeyProvider, err := kubernetes.NewPrivilegedSSHKeyProvider(fakeClient)
	if err != nil {
		return nil, nil, err
	}
	userProvider := kubernetes.NewUserProvider(fakeClient, kubernetes.IsProjectServiceAccount, kubermaticClient)
	adminProvider := kubernetes.NewAdminProvider(fakeClient)
	settingsProvider := kubernetes.NewSettingsProvider(ctx, kubermaticClient, fakeClient)
	addonConfigProvider := kubernetes.NewAddonConfigProvider(fakeClient)
	tokenGenerator, err := serviceaccount.JWTTokenGenerator([]byte(TestServiceAccountHashKey))
	if err != nil {
		return nil, nil, err
	}
	tokenAuth := serviceaccount.JWTTokenAuthenticator([]byte(TestServiceAccountHashKey))
	serviceAccountTokenProvider, err := kubernetes.NewServiceAccountTokenProvider(fakeImpersonationClient, fakeClient)
	if err != nil {
		return nil, nil, err
	}
	serviceAccountProvider := kubernetes.NewServiceAccountProvider(fakeImpersonationClient, fakeClient, "localhost")
	projectMemberProvider := kubernetes.NewProjectMemberProvider(fakeImpersonationClient, fakeClient, kubernetes.IsProjectServiceAccount)
	userInfoGetter, err := provider.UserInfoGetterFactory(projectMemberProvider)
	if err != nil {
		return nil, nil, err
	}
	var verifiers []auth.TokenVerifier
	var extractors []auth.TokenExtractor
	{
		// if the API users is actually a service account use JWTTokenAuthentication
		// that knows how to extract and verify the token
		if strings.HasPrefix(user.Email, "serviceaccount-") {
			saExtractorVerifier := auth.NewServiceAccountAuthClient(
				auth.NewHeaderBearerTokenExtractor("Authorization"),
				serviceaccount.JWTTokenAuthenticator([]byte(TestServiceAccountHashKey)),
				serviceAccountTokenProvider,
			)
			verifiers = append(verifiers, saExtractorVerifier)
			extractors = append(extractors, saExtractorVerifier)

			// for normal users we use OIDCClient which is broken at the moment
			// because the tests don't send a token in the Header instead
			// the client spits out a hardcoded value
		} else {
			fakeOIDCClient := NewFakeOIDCClient(user)
			verifiers = append(verifiers, fakeOIDCClient)
			extractors = append(extractors, fakeOIDCClient)
		}
	}
	tokenVerifiers := auth.NewTokenVerifierPlugins(verifiers)
	tokenExtractors := auth.NewTokenExtractorPlugins(extractors)
	fakeOIDCClient := NewFakeOIDCClient(user)

	projectProvider, err := kubernetes.NewProjectProvider(fakeImpersonationClient, fakeClient)
	if err != nil {
		return nil, nil, err
	}
	privilegedProjectProvider, err := kubernetes.NewPrivilegedProjectProvider(fakeClient)
	if err != nil {
		return nil, nil, err
	}

	kubermaticVersions := kubermatic.NewFakeVersions()
	fUserClusterConnection := &fakeUserClusterConnection{fakeClient}
	clusterProvider := kubernetes.NewClusterProvider(
		&restclient.Config{},
		fakeImpersonationClient,
		fUserClusterConnection,
		"",
		rbac.ExtractGroupPrefix,
		fakeClient,
		kubernetesClient,
		false,
		kubermaticVersions,
	)
	clusterProviders := map[string]provider.ClusterProvider{"us-central1": clusterProvider}
	clusterProviderGetter := func(seed *kubermaticv1.Seed) (provider.ClusterProvider, error) {
		if clusterProvider, exists := clusterProviders[seed.Name]; exists {
			return clusterProvider, nil
		}
		return nil, fmt.Errorf("can not find clusterprovider for cluster %q", seed.Name)
	}

	addonProvider := kubernetes.NewAddonProvider(
		fakeClient,
		fakeImpersonationClient,
		sets.NewString("addon1", "addon2"),
	)
	addonProviders := map[string]provider.AddonProvider{"us-central1": addonProvider}
	addonProviderGetter := func(seed *kubermaticv1.Seed) (provider.AddonProvider, error) {
		if addonProvider, exists := addonProviders[seed.Name]; exists {
			return addonProvider, nil
		}
		return nil, fmt.Errorf("can not find addonprovider for cluster %q", seed.Name)
	}

	credentialsManager, err := kubernetes.NewPresetsProvider(ctx, fakeClient, "", true)
	if err != nil {
		return nil, nil, err
	}
	admissionPluginProvider := kubernetes.NewAdmissionPluginsProvider(ctx, fakeClient)

	if seedsGetter == nil {
		seedsGetter = CreateTestSeedsGetter(ctx, fakeClient)
	}

	seedClientGetter := func(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error) {
		return fakeClient, nil
	}

	externalClusterProvider, err := kubernetes.NewExternalClusterProvider(fakeImpersonationClient, fakeClient)
	if err != nil {
		return nil, nil, err
	}
	fakeExternalClusterProvider := &FakeExternalClusterProvider{
		Provider:   externalClusterProvider,
		FakeClient: fakeClient,
	}

	constraintTemplateProvider, err := kubernetes.NewConstraintTemplateProvider(fakeImpersonationClient, fakeClient)
	if err != nil {
		return nil, nil, err
	}
	fakeConstraintTemplateProvider := &FakeConstraintTemplateProvider{
		Provider:   constraintTemplateProvider,
		FakeClient: fakeClient,
	}

	privilegedAllowedRegistryProvider, err := kubernetes.NewAllowedRegistryPrivilegedProvider(fakeClient)
	if err != nil {
		return nil, nil, err
	}
	fakePrivilegedAllowedRegistryProvider := &FakePrivilegedAllowedRegistryProvider{
		Provider:   privilegedAllowedRegistryProvider,
		FakeClient: fakeClient,
	}

	defaultConstraintProvider, err := kubernetes.NewDefaultConstraintProvider(fakeImpersonationClient, fakeClient, KubermaticNamespace)
	if err != nil {
		return nil, nil, err
	}
	fakeDefaultConstraintProvider := &FakeDefaultConstraintProvider{
		Provider:   defaultConstraintProvider,
		FakeClient: fakeClient,
	}

	constraintProvider, err := kubernetes.NewConstraintProvider(fakeImpersonationClient, fakeClient)
	if err != nil {
		return nil, nil, err
	}
	constraintProviders := map[string]provider.ConstraintProvider{"us-central1": constraintProvider}
	constraintProviderGetter := func(seed *kubermaticv1.Seed) (provider.ConstraintProvider, error) {
		if constraint, exists := constraintProviders[seed.Name]; exists {
			return constraint, nil
		}
		return nil, fmt.Errorf("can not find constraintprovider for cluster %q", seed.Name)
	}

	alertmanagerProvider := kubernetes.NewAlertmanagerProvider(fakeImpersonationClient, fakeClient)
	alertmanagerProviders := map[string]provider.AlertmanagerProvider{"us-central1": alertmanagerProvider}
	alertmanagerProviderGetter := func(seed *kubermaticv1.Seed) (provider.AlertmanagerProvider, error) {
		if alertmanager, exists := alertmanagerProviders[seed.Name]; exists {
			return alertmanager, nil
		}
		return nil, fmt.Errorf("can not find alertmanagerprovider for cluster %q", seed.Name)
	}

	clusterTemplateProvider, err := kubernetes.NewClusterTemplateProvider(fakeImpersonationClient, fakeClient)
	if err != nil {
		return nil, nil, err
	}

	ruleGroupProvider := kubernetes.NewRuleGroupProvider(fakeImpersonationClient, fakeClient)
	ruleGroupProviders := map[string]provider.RuleGroupProvider{"us-central1": ruleGroupProvider}
	ruleGroupProviderGetter := func(seed *kubermaticv1.Seed) (provider.RuleGroupProvider, error) {
		if ruleGroup, exists := ruleGroupProviders[seed.Name]; exists {
			return ruleGroup, nil
		}
		return nil, fmt.Errorf("can not find ruleGroupProvider for cluster %q", seed.Name)
	}

	clusterTemplateInstanceProvider := kubernetes.NewClusterTemplateInstanceProvider(fakeImpersonationClient, fakeClient)
	clusterTemplateInstanceProviders := map[string]provider.ClusterTemplateInstanceProvider{"us-central1": clusterTemplateInstanceProvider}
	clusterTemplateInstanceProviderGetter := func(seed *kubermaticv1.Seed) (provider.ClusterTemplateInstanceProvider, error) {
		if instances, exists := clusterTemplateInstanceProviders[seed.Name]; exists {
			return instances, nil
		}
		return nil, fmt.Errorf("can not find clusterTemplateInstanceProvider for seed %q", seed.Name)
	}

	etcdBackupConfigProvider := kubernetes.NewEtcdBackupConfigProvider(fakeImpersonationClient, fakeClient)
	etcdBackupConfigProviders := map[string]provider.EtcdBackupConfigProvider{"us-central1": etcdBackupConfigProvider}
	etcdBackupConfigProviderGetter := func(seed *kubermaticv1.Seed) (provider.EtcdBackupConfigProvider, error) {
		if etcdBackupConfig, exists := etcdBackupConfigProviders[seed.Name]; exists {
			return etcdBackupConfig, nil
		}
		return nil, fmt.Errorf("can not find etcdBackupConfigProvider for cluster %q", seed.Name)
	}

	etcdRestoreProvider := kubernetes.NewEtcdRestoreProvider(fakeImpersonationClient, fakeClient)
	etcdRestoreProviders := map[string]provider.EtcdRestoreProvider{"us-central1": etcdRestoreProvider}
	etcdRestoreProviderGetter := func(seed *kubermaticv1.Seed) (provider.EtcdRestoreProvider, error) {
		if etcdRestore, exists := etcdRestoreProviders[seed.Name]; exists {
			return etcdRestore, nil
		}
		return nil, fmt.Errorf("can not find etcdRestoreProvider for cluster %q", seed.Name)
	}

	etcdBackupConfigProjectProvider := kubernetes.NewEtcdBackupConfigProjectProvider(
		map[string]kubernetes.ImpersonationClient{"us-central1": fakeImpersonationClient},
		map[string]ctrlruntimeclient.Client{"us-central1": fakeClient})
	etcdBackupConfigProjectProviderGetter := func(seed map[string]*kubermaticv1.Seed) (provider.EtcdBackupConfigProjectProvider, error) {
		return etcdBackupConfigProjectProvider, nil
	}

	etcdRestoreProjectProvider := kubernetes.NewEtcdRestoreProjectProvider(
		map[string]kubernetes.ImpersonationClient{"us-central1": fakeImpersonationClient},
		map[string]ctrlruntimeclient.Client{"us-central1": fakeClient})
	etcdRestoreProjectProviderGetter := func(seed map[string]*kubermaticv1.Seed) (provider.EtcdRestoreProjectProvider, error) {
		return etcdRestoreProjectProvider, nil
	}

	eventRecorderProvider := kubernetes.NewEventRecorder()

	settingsWatcher, err := kuberneteswatcher.NewSettingsWatcher(settingsProvider)
	if err != nil {
		return nil, nil, err
	}

	userWatcher, err := kuberneteswatcher.NewUserWatcher(userProvider)
	if err != nil {
		return nil, nil, err
	}

	// Disable the metrics endpoint in tests
	var prometheusClient prometheusapi.Client

	mainRouter := routingFunc(
		adminProvider,
		settingsProvider,
		userInfoGetter,
		seedsGetter,
		seedClientGetter,
		clusterProviderGetter,
		addonProviderGetter,
		addonConfigProvider,
		sshKeyProvider,
		privilegedSSHKeyProvider,
		userProvider,
		serviceAccountProvider,
		serviceAccountProvider,
		serviceAccountTokenProvider,
		serviceAccountTokenProvider,
		projectProvider,
		privilegedProjectProvider,
		fakeOIDCClient,
		tokenVerifiers,
		tokenExtractors,
		prometheusClient,
		projectMemberProvider,
		projectMemberProvider,
		versions,
		updates,
		tokenAuth,
		tokenGenerator,
		eventRecorderProvider,
		credentialsManager,
		admissionPluginProvider,
		settingsWatcher,
		userWatcher,
		fakeExternalClusterProvider,
		externalClusterProvider,
		fakeConstraintTemplateProvider,
		constraintProviderGetter,
		alertmanagerProviderGetter,
		clusterTemplateProvider,
		clusterTemplateInstanceProviderGetter,
		ruleGroupProviderGetter,
		kubermaticVersions,
		fakeDefaultConstraintProvider,
		fakePrivilegedAllowedRegistryProvider,
		etcdBackupConfigProviderGetter,
		etcdRestoreProviderGetter,
		etcdBackupConfigProjectProviderGetter,
		etcdRestoreProjectProviderGetter,
	)

	return mainRouter, &ClientsSets{kubermaticClient, fakeClient, kubernetesClient, tokenAuth, tokenGenerator}, nil
}

// CreateTestEndpointAndGetClients is a convenience function that instantiates fake providers and sets up routes for the tests
func CreateTestEndpointAndGetClients(user apiv1.User, seedsGetter provider.SeedsGetter, kubeObjects, machineObjects, kubermaticObjects []ctrlruntimeclient.Object, versions []*version.Version, updates []*version.Update, routingFunc newRoutingFunc) (http.Handler, *ClientsSets, error) {
	return initTestEndpoint(user, seedsGetter, kubeObjects, machineObjects, kubermaticObjects, versions, updates, routingFunc)
}

// CreateTestEndpoint does exactly the same as CreateTestEndpointAndGetClients except it omits ClientsSets when returning
func CreateTestEndpoint(user apiv1.User, kubeObjects, kubermaticObjects []ctrlruntimeclient.Object, versions []*version.Version, updates []*version.Update, routingFunc newRoutingFunc) (http.Handler, error) {
	router, _, err := CreateTestEndpointAndGetClients(user, nil, kubeObjects, nil, kubermaticObjects, versions, updates, routingFunc)
	return router, err
}

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
						Fake:                &kubermaticv1.DatacenterSpecFake{},
						RequiredEmailDomain: "example.com",
					},
				},
				"restricted-fake-dc2": {
					Country:  "NL",
					Location: "Amsterdam",
					Spec: kubermaticv1.DatacenterSpec{
						Fake:                 &kubermaticv1.DatacenterSpecFake{},
						RequiredEmailDomains: []string{"23f67weuc.com", "example.com", "12noifsdsd.org"},
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
						HyperkubeImage:     "hyperkube-image",
					},
				},
			},
		},
	}
	for _, modifier := range modifiers {
		modifier(seed)
	}
	return seed
}

// CreateTestSeedsGetter creates a SeedsGetter only useful for generic tests,
// as it does not follow the CE/EE conventions and always returns all Seeds.
func CreateTestSeedsGetter(ctx context.Context, client ctrlruntimeclient.Client) provider.SeedsGetter {
	listOpts := &ctrlruntimeclient.ListOptions{Namespace: "kubermatic"}

	return func() (map[string]*kubermaticv1.Seed, error) {
		seeds := &kubermaticv1.SeedList{}
		if err := client.List(ctx, seeds, listOpts); err != nil {
			return nil, fmt.Errorf("failed to list the seeds: %v", err)
		}
		seedMap := map[string]*kubermaticv1.Seed{}
		for idx, seed := range seeds.Items {
			seedMap[seed.Name] = &seeds.Items[idx]
		}
		return seedMap, nil
	}
}

type fakeUserClusterConnection struct {
	fakeDynamicClient ctrlruntimeclient.Client
}

func (f *fakeUserClusterConnection) GetClient(_ context.Context, _ *kubermaticv1.Cluster, _ ...k8cuserclusterclient.ConfigOption) (ctrlruntimeclient.Client, error) {
	return f.fakeDynamicClient, nil
}

// ClientsSets a simple wrapper that holds fake client sets
type ClientsSets struct {
	FakeKubermaticClient *kubermaticfakeclientset.Clientset
	FakeClient           ctrlruntimeclient.Client
	// this client is used for unprivileged methods where impersonated client is used
	FakeKubernetesCoreClient kubernetesclientset.Interface

	TokenAuthenticator serviceaccount.TokenAuthenticator
	TokenGenerator     serviceaccount.TokenGenerator
}

// GenerateTestKubeconfig returns test kubeconfig yaml structure
func GenerateTestKubeconfig(clusterID, token string) string {
	return fmt.Sprintf(`
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data:
    server: test.fake.io
  name: %s
contexts:
- context:
    cluster: %s
    user: default
  name: default
current-context: default
kind: Config
users:
- name: default
  user:
    token: %s`, clusterID, clusterID, token)
}

// APIUserToKubermaticUser simply converts apiv1.User to kubermaticv1.User type
func APIUserToKubermaticUser(user apiv1.User) *kubermaticv1.User {
	var deletionTimestamp *metav1.Time
	if user.DeletionTimestamp != nil {
		deletionTimestamp = &metav1.Time{Time: user.DeletionTimestamp.Time}
	}
	return &kubermaticv1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:              user.Name,
			CreationTimestamp: metav1.Time{Time: user.CreationTimestamp.Time},
			DeletionTimestamp: deletionTimestamp,
		},
		Spec: kubermaticv1.UserSpec{
			Name:    user.Name,
			Email:   user.Email,
			ID:      user.ID,
			IsAdmin: user.IsAdmin,
		},
	}
}

// CompareWithResult a convenience function for comparing http.Body content with response
func CompareWithResult(t *testing.T, res *httptest.ResponseRecorder, response string) {
	t.Helper()
	bBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("Unable to read response body")
	}

	r := strings.TrimSpace(response)
	b := strings.TrimSpace(string(bBytes))

	if r != b {
		t.Fatalf("Expected response body to be \n%s \ngot \n%s", r, b)
	}
}

// GenUser generates a User resource
// note if the id is empty then it will be auto generated
func GenUser(id, name, email string) *kubermaticv1.User {
	if len(id) == 0 {
		// the name of the object is derived from the email address and encoded as sha256
		id = fmt.Sprintf("%x", sha256.Sum256([]byte(email)))
	}

	specID := ""
	{
		h := sha512.New512_224()
		if _, err := io.WriteString(h, email); err != nil {
			// not nice, better to use t.Error
			panic("unable to generate a test user due to " + err.Error())
		}
		specID = fmt.Sprintf("%x_KUBE", h.Sum(nil))
	}

	return &kubermaticv1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: id,
			UID:  types.UID(fmt.Sprintf("fake-uid-%s", id)),
		},
		Spec: kubermaticv1.UserSpec{
			ID:    specID,
			Name:  name,
			Email: email,
		},
	}
}

// GenInactiveProjectServiceAccount generates a Service Account resource
func GenInactiveProjectServiceAccount(id, name, group, projectName string) *kubermaticv1.User {
	user := GenUser(id, name, fmt.Sprintf("serviceaccount-%s@sa.kubermatic.io", id))
	user.Labels = map[string]string{kubernetes.ServiceAccountLabelGroup: fmt.Sprintf("%s-%s", group, projectName)}
	user.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
			Kind:       kubermaticv1.ProjectKindName,
			Name:       projectName,
			UID:        types.UID(id),
		},
	}
	user.Spec.ID = id
	user.Name = fmt.Sprintf("serviceaccount-%s", id)
	user.UID = ""

	return user
}

func GenProjectServiceAccount(id, name, group, projectName string) *kubermaticv1.User {
	sa := GenInactiveProjectServiceAccount(id, name, group, projectName)
	sa.Labels = map[string]string{}
	return sa
}

func GenMainServiceAccount(id, name, group, ownerEmail string) *kubermaticv1.User {
	user := GenUser(id, name, fmt.Sprintf("main-serviceaccount-%s@sa.kubermatic.io", id))
	user.Labels = map[string]string{kubernetes.ServiceAccountLabelGroup: group}
	user.Annotations = map[string]string{kubernetes.ServiceAccountAnnotationOwner: ownerEmail}

	user.Spec.ID = id
	user.Name = fmt.Sprintf("main-serviceaccount-%s", id)
	user.UID = ""
	return user
}

// GenAPIUser generates a API user
func GenAPIUser(name, email string) *apiv1.User {
	usr := GenUser("", name, email)
	return &apiv1.User{
		ObjectMeta: apiv1.ObjectMeta{
			ID:   usr.Name,
			Name: usr.Spec.Name,
		},
		Email: usr.Spec.Email,
	}
}

// DefaultCreationTimestamp returns default test timestamp
func DefaultCreationTimestamp() time.Time {
	return time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)
}

// GenDefaultAPIUser generates a default API user
func GenDefaultAPIUser() *apiv1.User {
	return &apiv1.User{
		ObjectMeta: apiv1.ObjectMeta{
			ID:   GenDefaultUser().Name,
			Name: GenDefaultUser().Spec.Name,
		},
		Email: GenDefaultUser().Spec.Email,
	}
}

// GenDefaultAdminAPIUser generates a default admin API user
func GenDefaultAdminAPIUser() *apiv1.User {
	user := GenDefaultAPIUser()
	user.IsAdmin = true
	return user
}

// GenDefaultUser generates a default user
func GenDefaultUser() *kubermaticv1.User {
	userEmail := "bob@acme.com"
	return GenUser("", "Bob", userEmail)
}

// GenProject generates new empty project
func GenProject(name, phase string, creationTime time.Time, oRef ...metav1.OwnerReference) *kubermaticv1.Project {
	return &kubermaticv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:              fmt.Sprintf("%s-%s", name, "ID"),
			CreationTimestamp: metav1.NewTime(creationTime),
			OwnerReferences:   oRef,
		},
		Spec: kubermaticv1.ProjectSpec{Name: name},
		Status: kubermaticv1.ProjectStatus{
			Phase: phase,
		},
	}
}

// GenDefaultProject generates a default project
func GenDefaultProject() *kubermaticv1.Project {
	user := GenDefaultUser()
	oRef := metav1.OwnerReference{
		APIVersion: "kubermatic.io/v1",
		Kind:       "User",
		UID:        user.UID,
		Name:       user.Name,
	}
	return GenProject("my-first-project", kubermaticv1.ProjectActive, DefaultCreationTimestamp(), oRef)
}

// GenBinding generates a binding
func GenBinding(projectID, email, group string) *kubermaticv1.UserProjectBinding {
	return &kubermaticv1.UserProjectBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s-%s", projectID, email, group),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.ProjectKindName,
					Name:       projectID,
				},
			},
		},
		Spec: kubermaticv1.UserProjectBindingSpec{
			UserEmail: email,
			ProjectID: projectID,
			Group:     fmt.Sprintf("%s-%s", group, projectID),
		},
	}
}

// GenDefaultOwnerBinding generates default owner binding
func GenDefaultOwnerBinding() *kubermaticv1.UserProjectBinding {
	return GenBinding(GenDefaultProject().Name, GenDefaultUser().Spec.Email, "owners")
}

// GenDefaultKubermaticObjects generates default kubermatic object
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
				Fake:           &kubermaticv1.FakeCloudSpec{Token: "SecretToken"},
			},
			Version:               *semver.NewSemverOrDie("9.9.9"),
			HumanReadableName:     name,
			EnableUserSSHKeyAgent: pointer.BoolPtr(false),
		},
		Address: kubermaticv1.ClusterAddress{
			AdminToken:   "drphc2.g4kq82pnlfqjqt65",
			ExternalName: "w225mx4z66.asia-east1-a-1.cloud.kubermatic.io",
			IP:           "35.194.142.199",
			URL:          "https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885",
		},
		Status: kubermaticv1.ClusterStatus{
			ExtendedHealth: kubermaticv1.ExtendedClusterHealth{
				Apiserver:                    kubermaticv1.HealthStatusUp,
				Scheduler:                    kubermaticv1.HealthStatusUp,
				Controller:                   kubermaticv1.HealthStatusUp,
				MachineController:            kubermaticv1.HealthStatusUp,
				Etcd:                         kubermaticv1.HealthStatusUp,
				UserClusterControllerManager: kubermaticv1.HealthStatusUp,
				CloudProviderInfrastructure:  kubermaticv1.HealthStatusUp,
			},
			NamespaceName: "cluster-" + id,
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
				Kubelet: "v9.9.9",
			},
		},
	}
}

func GenTestMachineDeployment(name, rawProviderSpec string, selector map[string]string, dynamicConfig bool) *clusterv1alpha1.MachineDeployment {
	var replicas int32 = 1

	var configSource *corev1.NodeConfigSource
	if dynamicConfig {
		configSource = &corev1.NodeConfigSource{
			ConfigMap: &corev1.ConfigMapNodeConfigSource{
				Namespace:        "kube-system",
				KubeletConfigKey: "kubelet",
				Name:             "config-kubelet-9.9",
			},
		}
	}
	return &clusterv1alpha1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceSystem,
		},
		TypeMeta: metav1.TypeMeta{
			Kind: "MachineDeployment",
		},
		Spec: clusterv1alpha1.MachineDeploymentSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: selector,
			},
			Replicas: &replicas,
			Template: clusterv1alpha1.MachineTemplateSpec{
				Spec: clusterv1alpha1.MachineSpec{
					ProviderSpec: clusterv1alpha1.ProviderSpec{
						Value: &runtime.RawExtension{
							Raw: []byte(rawProviderSpec),
						},
					},
					Versions: clusterv1alpha1.MachineVersionInfo{
						Kubelet: "v9.9.9",
					},
					ConfigSource: configSource,
				},
			},
		},
	}
}

func GenTestAddon(name string, variables *runtime.RawExtension, cluster *kubermaticv1.Cluster, creationTime time.Time) *kubermaticv1.Addon {
	if variables == nil {
		variables = &runtime.RawExtension{}
	}
	return &kubermaticv1.Addon{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Status.NamespaceName,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.ClusterKindName,
					Name:       cluster.Name,
					UID:        cluster.UID,
				},
			},
			CreationTimestamp: metav1.NewTime(creationTime),
		},
		Spec: kubermaticv1.AddonSpec{
			Name:      name,
			Variables: *variables,
			Cluster: corev1.ObjectReference{
				APIVersion: kubermaticv1.SchemeGroupVersion.String(),
				Kind:       kubermaticv1.ClusterKindName,
				Name:       cluster.Name,
				UID:        cluster.UID,
			},
		},
	}
}

func CheckStatusCode(wantStatusCode int, recorder *httptest.ResponseRecorder, t *testing.T) {
	t.Helper()
	if recorder.Code != wantStatusCode {
		t.Errorf("Expected status code to be %d, got: %d", wantStatusCode, recorder.Code)
		t.Error(recorder.Body.String())
		return
	}
}

func GenDefaultSaToken(projectID, saID, name, id string) *corev1.Secret {
	secret := &corev1.Secret{}
	secret.Name = fmt.Sprintf("sa-token-%s", id)
	secret.Type = "Opaque"
	secret.Namespace = "kubermatic"
	secret.Data = map[string][]byte{}
	secret.Data["token"] = []byte(TestFakeToken)
	secret.Labels = map[string]string{
		kubermaticv1.ProjectIDLabelKey: projectID,
		"name":                         name,
	}
	secret.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
			Kind:       kubermaticv1.UserKindName,
			UID:        "",
			Name:       saID,
		},
	}

	return secret
}

func GenDefaultExpiry() (apiv1.Time, error) {
	authenticator := serviceaccount.JWTTokenAuthenticator([]byte(TestServiceAccountHashKey))
	claim, _, err := authenticator.Authenticate(TestFakeToken)
	if err != nil {
		return apiv1.Time{}, err
	}
	return apiv1.NewTime(claim.Expiry.Time()), nil
}

func GenTestEvent(eventName, eventType, eventReason, eventMessage, kind, uid string) *corev1.Event {
	return &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      eventName,
			Namespace: metav1.NamespaceSystem,
		},
		InvolvedObject: corev1.ObjectReference{
			UID:       types.UID(uid),
			Name:      "testMachine",
			Namespace: metav1.NamespaceSystem,
			Kind:      kind,
		},
		Reason:  eventReason,
		Message: eventMessage,
		Source:  corev1.EventSource{Component: "eventTest"},
		Count:   1,
		Type:    eventType,
	}
}

func sortVersion(versions []*apiv1.MasterVersion) {
	sort.SliceStable(versions, func(i, j int) bool {
		mi, mj := versions[i], versions[j]
		return mi.Version.LessThan(mj.Version)
	})
}

func CompareVersions(t *testing.T, versions, expected []*apiv1.MasterVersion) {
	if len(versions) != len(expected) {
		t.Fatalf("got different lengths, got %d expected %d", len(versions), len(expected))
	}

	sortVersion(versions)
	sortVersion(expected)

	for i, v := range versions {
		if !v.Version.Equal(expected[i].Version) {
			t.Fatalf("expected version %v got %v", expected[i].Version, v.Version)
		}
		if v.Default != expected[i].Default {
			t.Fatalf("expected flag %v got %v", expected[i].Default, v.Default)
		}
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

func GenDefaultSettings() *kubermaticv1.KubermaticSetting {
	return &kubermaticv1.KubermaticSetting{
		ObjectMeta: metav1.ObjectMeta{
			Name: kubermaticv1.GlobalSettingsName,
		},
		Spec: kubermaticv1.SettingSpec{
			CustomLinks: []kubermaticv1.CustomLink{},
			CleanupOptions: kubermaticv1.CleanupOptions{
				Enabled:  false,
				Enforced: false,
			},
			DefaultNodeCount:      10,
			ClusterTypeOptions:    kubermaticv1.ClusterTypeAll,
			DisplayDemoInfo:       false,
			DisplayAPIDocs:        false,
			DisplayTermsOfService: false,
			EnableDashboard:       true,
			EnableOIDCKubeconfig:  false,
		},
	}
}

func GenDefaultVersions() []*version.Version {
	return []*version.Version{
		{
			Version: ver.MustParse("1.15.0"),
			Default: false,
			Type:    apiv1.KubernetesClusterType,
		},
		{
			Version: ver.MustParse("1.15.1"),
			Default: false,
			Type:    apiv1.KubernetesClusterType,
		},
		{
			Version: ver.MustParse("1.17.0"),
			Default: false,
			Type:    apiv1.KubernetesClusterType,
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

func GenDefaultGlobalSettings() *kubermaticv1.KubermaticSetting {
	return &kubermaticv1.KubermaticSetting{
		ObjectMeta: metav1.ObjectMeta{
			Name: kubermaticv1.GlobalSettingsName,
		},
		Spec: kubermaticv1.SettingSpec{
			CustomLinks: []kubermaticv1.CustomLink{
				{
					Label:    "label",
					URL:      "url:label",
					Icon:     "icon",
					Location: "EU",
				},
			},
			CleanupOptions: kubermaticv1.CleanupOptions{
				Enabled:  true,
				Enforced: true,
			},
			DefaultNodeCount:            5,
			ClusterTypeOptions:          5,
			DisplayDemoInfo:             true,
			DisplayAPIDocs:              true,
			DisplayTermsOfService:       true,
			EnableExternalClusterImport: true,
			OpaOptions: kubermaticv1.OpaOptions{
				Enabled:  true,
				Enforced: true,
			},
			MlaOptions: kubermaticv1.MlaOptions{
				LoggingEnabled:     true,
				LoggingEnforced:    true,
				MonitoringEnabled:  true,
				MonitoringEnforced: true,
			},
		},
	}
}

func GenClusterWithOpenstack(cluster *kubermaticv1.Cluster) *kubermaticv1.Cluster {
	cluster.Spec.Cloud = kubermaticv1.CloudSpec{
		DatacenterName: "OpenstackDatacenter",
		Openstack: &kubermaticv1.OpenstackCloudSpec{
			Username:       "username",
			Password:       "password",
			SubnetID:       "subnetID",
			Domain:         "domain",
			FloatingIPPool: "floatingIPPool",
			Network:        "network",
			RouterID:       "routerID",
			SecurityGroups: "securityGroups",
			Tenant:         "tenant",
		},
	}
	return cluster
}

func GenDefaultExternalClusterNodes() ([]ctrlruntimeclient.Object, error) {
	cpuQuantity, err := resource.ParseQuantity("290")
	if err != nil {
		return nil, err
	}
	memoryQuantity, err := resource.ParseQuantity("687202304")
	if err != nil {
		return nil, err
	}
	return []ctrlruntimeclient.Object{
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "node1"},
			Spec: corev1.NodeSpec{
				PodCIDR:       "abc",
				ProviderID:    "abc",
				Unschedulable: false,
			},
			Status: corev1.NodeStatus{
				Capacity:        nil,
				Allocatable:     corev1.ResourceList{"cpu": cpuQuantity, "memory": memoryQuantity},
				Phase:           "init",
				DaemonEndpoints: corev1.NodeDaemonEndpoints{},
				NodeInfo: corev1.NodeSystemInfo{
					MachineID:               "abc",
					SystemUUID:              "abc",
					BootID:                  "190ee9ec-75b7-43f3-9c39-0ebf361d64f0",
					KernelVersion:           "4.14",
					OSImage:                 "Container-Optimized OS from Google",
					ContainerRuntimeVersion: "containerd://1.2.8",
					KubeletVersion:          "v1.15.12-gke.2",
					KubeProxyVersion:        "v1.15.12-gke.2",
					OperatingSystem:         "linux",
					Architecture:            "amd64",
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "node2"},
			Spec: corev1.NodeSpec{
				PodCIDR:       "def",
				ProviderID:    "def",
				Unschedulable: false,
			},
			Status: corev1.NodeStatus{
				Capacity:        nil,
				Allocatable:     corev1.ResourceList{"cpu": cpuQuantity, "memory": memoryQuantity},
				Phase:           "init",
				DaemonEndpoints: corev1.NodeDaemonEndpoints{},
				NodeInfo: corev1.NodeSystemInfo{
					MachineID:               "def",
					SystemUUID:              "def",
					BootID:                  "190ee9ec-75b7-43f3-9c39-0ebf361d64f0",
					KernelVersion:           "4.14",
					OSImage:                 "Container-Optimized OS from Google",
					ContainerRuntimeVersion: "containerd://1.2.8",
					KubeletVersion:          "v1.15.12-gke.2",
					KubeProxyVersion:        "v1.15.12-gke.2",
					OperatingSystem:         "linux",
					Architecture:            "amd64",
				},
			},
		},
	}, nil
}

func GenDefaultExternalClusterNode() (*corev1.Node, error) {
	cpuQuantity, err := resource.ParseQuantity("290")
	if err != nil {
		return nil, err
	}
	memoryQuantity, err := resource.ParseQuantity("687202304")
	if err != nil {
		return nil, err
	}
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node1"},
		Spec: corev1.NodeSpec{
			PodCIDR:       "abc",
			ProviderID:    "abc",
			Unschedulable: false,
		},
		Status: corev1.NodeStatus{
			Capacity:        nil,
			Allocatable:     corev1.ResourceList{"cpu": cpuQuantity, "memory": memoryQuantity},
			Phase:           "init",
			DaemonEndpoints: corev1.NodeDaemonEndpoints{},
			NodeInfo: corev1.NodeSystemInfo{
				MachineID:               "abc",
				SystemUUID:              "abc",
				BootID:                  "190ee9ec-75b7-43f3-9c39-0ebf361d64f0",
				KernelVersion:           "4.14",
				OSImage:                 "Container-Optimized OS from Google",
				ContainerRuntimeVersion: "containerd://1.2.8",
				KubeletVersion:          "v1.15.12-gke.2",
				KubeProxyVersion:        "v1.15.12-gke.2",
				OperatingSystem:         "linux",
				Architecture:            "amd64",
			},
		},
	}, nil
}

func GenDefaultConstraintTemplate(name string) apiv2.ConstraintTemplate {
	return apiv2.ConstraintTemplate{
		Name: name,
		Spec: kubermaticv1.ConstraintTemplateSpec{
			CRD: constrainttemplatev1beta1.CRD{
				Spec: constrainttemplatev1beta1.CRDSpec{
					Names: constrainttemplatev1beta1.Names{
						Kind:       "labelconstraint",
						ShortNames: []string{"lc"},
					},
					Validation: &constrainttemplatev1beta1.Validation{
						OpenAPIV3Schema: &apiextensionv1.JSONSchemaProps{
							Properties: map[string]apiextensionv1.JSONSchemaProps{
								"labels": {
									Type: "array",
									Items: &apiextensionv1.JSONSchemaPropsOrArray{
										Schema: &apiextensionv1.JSONSchemaProps{
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
			Targets: []constrainttemplatev1beta1.Target{
				{
					Target: "admission.k8s.gatekeeper.sh",
					Rego: `
		package k8srequiredlabels

        deny[{"msg": msg, "details": {"missing_labels": missing}}] {
          provided := {label | input.review.object.metadata.labels[label]}
          required := {label | label := input.parameters.labels[_]}
          missing := required - provided
          count(missing) > 0
          msg := sprintf("you must provide labels: %v", [missing])
        }`,
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
		},
	}
}

func GenAdminUser(name, email string, isAdmin bool) *kubermaticv1.User {
	user := GenUser("", name, email)
	user.Spec.IsAdmin = isAdmin
	return user
}

func GenConstraintTemplate(name string) *kubermaticv1.ConstraintTemplate {
	ct := &kubermaticv1.ConstraintTemplate{}
	ct.Name = name
	ct.Spec = kubermaticv1.ConstraintTemplateSpec{
		CRD: constrainttemplatev1beta1.CRD{
			Spec: constrainttemplatev1beta1.CRDSpec{
				Names: constrainttemplatev1beta1.Names{
					Kind:       "labelconstraint",
					ShortNames: []string{"lc"},
				},
				Validation: &constrainttemplatev1beta1.Validation{
					OpenAPIV3Schema: &apiextensionv1.JSONSchemaProps{
						Properties: map[string]apiextensionv1.JSONSchemaProps{
							"labels": {
								Type: "array",
								Items: &apiextensionv1.JSONSchemaPropsOrArray{
									Schema: &apiextensionv1.JSONSchemaProps{
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
		Targets: []constrainttemplatev1beta1.Target{
			{
				Target: "admission.k8s.gatekeeper.sh",
				Rego: `
		package k8srequiredlabels

        deny[{"msg": msg, "details": {"missing_labels": missing}}] {
          provided := {label | input.review.object.metadata.labels[label]}
          required := {label | label := input.parameters.labels[_]}
          missing := required - provided
          count(missing) > 0
          msg := sprintf("you must provide labels: %v", [missing])
        }`,
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

func GenDefaultRole(name, namespace string) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Labels:    map[string]string{handlercommon.UserClusterComponentKey: handlercommon.UserClusterRoleComponentValue},
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get"},
				APIGroups: []string{""},
				Resources: []string{"pod"},
			},
		},
	}
}

func GenDefaultClusterRole(name string) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{handlercommon.UserClusterComponentKey: handlercommon.UserClusterRoleComponentValue},
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "list"},
				APIGroups: []string{""},
				Resources: []string{"pod"},
			},
		},
	}
}

func GenDefaultRoleBinding(name, namespace, roleID, userEmail string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Labels:    map[string]string{handlercommon.UserClusterComponentKey: handlercommon.UserClusterBindingComponentValue},
			Namespace: namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "User",
				Name: userEmail,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Name: roleID,
		},
	}
}

func GenDefaultGroupRoleBinding(name, namespace, roleID, group string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Labels:    map[string]string{handlercommon.UserClusterComponentKey: handlercommon.UserClusterBindingComponentValue},
			Namespace: namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "Group",
				Name: group,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Name: roleID,
		},
	}
}

func GenDefaultClusterRoleBinding(name, roleID, userEmail string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{handlercommon.UserClusterComponentKey: handlercommon.UserClusterBindingComponentValue},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "User",
				Name: userEmail,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Name: roleID,
		},
	}
}

func GenDefaultGroupClusterRoleBinding(name, roleID, group string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{handlercommon.UserClusterComponentKey: handlercommon.UserClusterBindingComponentValue},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "Group",
				Name: group,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Name: roleID,
		},
	}
}

func RegisterScheme(builder runtime.SchemeBuilder) error {
	return builder.AddToScheme(scheme.Scheme)
}

func CreateRawVariables(t *testing.T, in map[string]interface{}) *runtime.RawExtension {
	result := &runtime.RawExtension{}
	raw, err := k8sjson.Marshal(in)
	if err != nil {
		t.Fatalf("failed to marshal external Variables: %v", err)
	}
	result.Raw = raw
	return result
}

func GenConstraint(name, namespace, kind string) *kubermaticv1.Constraint {
	ct := &kubermaticv1.Constraint{}
	ct.Kind = kubermaticv1.ConstraintKind
	ct.APIVersion = kubermaticv1.SchemeGroupVersion.String()
	ct.Name = name
	ct.Namespace = namespace
	ct.Spec = kubermaticv1.ConstraintSpec{
		ConstraintType: kind,
		Match: kubermaticv1.Match{
			Kinds: []kubermaticv1.Kind{
				{Kinds: []string{"namespace"}, APIGroups: []string{""}},
			},
		},
		Parameters: kubermaticv1.Parameters{
			"labels": []interface{}{"gatekeeper", "opa"},
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
			Parameters: kubermaticv1.Parameters{
				"labels": []interface{}{"gatekeeper", "opa"},
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
			Synced: pointer.BoolPtr(true),
		},
	}
}

func GenAlertmanager(namespace, configSecretName string) *kubermaticv1.Alertmanager {
	return &kubermaticv1.Alertmanager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.AlertmanagerName,
			Namespace: namespace,
		},
		Spec: kubermaticv1.AlertmanagerSpec{
			ConfigSecret: corev1.LocalObjectReference{
				Name: configSecretName,
			},
		},
	}
}

func GenAlertmanagerConfigSecret(name, namespace string, config []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			resources.AlertmanagerConfigSecretKey: config,
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

func GenClusterTemplateInstance(projectID, templateID string, replicas int64) *kubermaticv1.ClusterTemplateInstance {
	return &kubermaticv1.ClusterTemplateInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%s-%s", projectID, templateID),
			Labels: map[string]string{kubernetes.ClusterTemplateLabelKey: templateID, kubermaticv1.ProjectIDLabelKey: projectID},
		},
		Spec: kubermaticv1.ClusterTemplateInstanceSpec{
			ProjectID:         projectID,
			ClusterTemplateID: templateID,
			Replicas:          replicas,
		},
	}
}

func GenRuleGroup(name, clusterName string, ruleGroupType kubermaticv1.RuleGroupType) *kubermaticv1.RuleGroup {
	return &kubermaticv1.RuleGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "cluster-" + clusterName,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       kubermaticv1.RuleGroupKindName,
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		},
		Spec: kubermaticv1.RuleGroupSpec{
			RuleGroupType: ruleGroupType,
			Cluster: corev1.ObjectReference{
				Kind:       kubermaticv1.ClusterKindName,
				Namespace:  "",
				Name:       clusterName,
				APIVersion: kubermaticv1.SchemeGroupVersion.String(),
			},
			Data: GenerateTestRuleGroupData(name),
		},
	}
}

func GenAPIRuleGroup(name string, ruleGroupType kubermaticv1.RuleGroupType) *apiv2.RuleGroup {
	return &apiv2.RuleGroup{
		Data: GenerateTestRuleGroupData(name),
		Type: ruleGroupType,
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

func GenDefaultAPIAllowedRegistry(name, registryPrefix string) apiv2.AllowedRegistry {
	return apiv2.AllowedRegistry{
		Name: name,
		Spec: kubermaticv1.AllowedRegistrySpec{
			RegistryPrefix: registryPrefix,
		},
	}
}

func GenAllowedRegistry(name, registryPrefix string) *kubermaticv1.AllowedRegistry {
	wr := &kubermaticv1.AllowedRegistry{}
	wr.Name = name
	wr.Spec = kubermaticv1.AllowedRegistrySpec{
		RegistryPrefix: registryPrefix,
	}
	return wr
}

func GenAPIEtcdBackupConfig(name, clusterID string) *apiv2.EtcdBackupConfig {
	keep := 5
	return &apiv2.EtcdBackupConfig{
		Name: name,
		Spec: apiv2.EtcdBackupConfigSpec{
			ClusterID: clusterID,
			Schedule:  "5 * * * * *",
			Keep:      &keep,
		},
	}
}

func GenEtcdBackupConfig(name string, cluster *kubermaticv1.Cluster) *kubermaticv1.EtcdBackupConfig {
	keep := 5
	clusterObjectRef, _ := reference.GetReference(scheme.Scheme, cluster)

	return &kubermaticv1.EtcdBackupConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Status.NamespaceName,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cluster, kubermaticv1.SchemeGroupVersion.WithKind("Cluster")),
			},
		},
		Spec: kubermaticv1.EtcdBackupConfigSpec{
			Name:     name,
			Cluster:  *clusterObjectRef,
			Schedule: "5 * * * * *",
			Keep:     &keep,
		},
	}
}

func GenAPIEtcdRestore(name, clusterID string) *apiv2.EtcdRestore {
	return &apiv2.EtcdRestore{
		Name: name,
		Spec: apiv2.EtcdRestoreSpec{
			ClusterID:                       clusterID,
			BackupName:                      "backup-1",
			BackupDownloadCredentialsSecret: "secret",
		},
	}
}

func GenEtcdRestore(name string, cluster *kubermaticv1.Cluster) *kubermaticv1.EtcdRestore {
	clusterObjectRef, _ := reference.GetReference(scheme.Scheme, cluster)

	return &kubermaticv1.EtcdRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Status.NamespaceName,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cluster, kubermaticv1.SchemeGroupVersion.WithKind("Cluster")),
			},
		},
		Spec: kubermaticv1.EtcdRestoreSpec{
			Name:                            name,
			Cluster:                         *clusterObjectRef,
			BackupName:                      "backup-1",
			BackupDownloadCredentialsSecret: "secret",
		},
	}
}
