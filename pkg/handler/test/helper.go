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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	semverlib "github.com/Masterminds/semver/v3"
	constrainttemplatev1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1"
	gatekeeperconfigv1alpha1 "github.com/open-policy-agent/gatekeeper/apis/config/v1alpha1"
	prometheusapi "github.com/prometheus/client_golang/api"
	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	k8cuserclusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/handler/auth"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v2/etcdbackupconfig"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/serviceaccount"
	"k8c.io/kubermatic/v2/pkg/version/cni"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/kubermatic/v2/pkg/watcher"
	kuberneteswatcher "k8c.io/kubermatic/v2/pkg/watcher/kubernetes"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sjson "k8s.io/apimachinery/pkg/util/json"
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

	middleware.Now = func() time.Time {
		return UserLastSeen
	}
}

const (
	// UserID holds a test user ID.
	UserID = "1233"
	// UserID2 holds a test user ID.
	UserID2 = "1523"
	// UserName holds a test user name.
	UserName = "user1"
	// UserName2 holds a test user name.
	UserName2 = "user2"
	// UserEmail holds a test user email.
	UserEmail = "john@acme.com"
	// UserEmail2 holds a test user email.
	UserEmail2 = "bob@example.com"
	// ClusterID holds the test cluster ID.
	ClusterID = "AbcClusterID"
	// DefaultClusterID holds the test default cluster ID.
	DefaultClusterID = "defClusterID"
	// DefaultClusterName holds the test default cluster name.
	DefaultClusterName = "defClusterName"
	// ProjectName holds the test project ID.
	ProjectName = "my-first-project-ID"
	// TestDatacenter holds datacenter name.
	TestSeedDatacenter = "us-central1"
	// TestServiceAccountHashKey authenticates the service account's token value using HMAC.
	TestServiceAccountHashKey = "eyJhbGciOiJIUzI1NeyJhbGciOiJIUzI1N"
	// TestFakeToken signed JWT token with fake data. It will expire after 3 years from 12-04-2022. To generate new token use kubermatic/pkg/serviceaccount/jwt_test.go.
	TestFakeToken = "eyJhbGciOiJIUzI1NiJ9.eyJlbWFpbCI6InRlc3RAZXhhbXBsZS5jb20iLCJleHAiOjE3NDQ0NjQ1OTYsImlhdCI6MTY0OTc3MDE5NiwibmJmIjoxNjQ5NzcwMTk2LCJwcm9qZWN0X2lkIjoidGVzdFByb2plY3QiLCJ0b2tlbl9pZCI6InRlc3RUb2tlbiJ9.IGcnVhrTGeemEZ_dOGCRE1JXwpSMWJEbrG8hylpTEUY"
	// TestOSdomain OpenStack domain.
	TestOSdomain = "OSdomain"
	// TestOSuserPass OpenStack user password.
	TestOSuserPass = "OSpass"
	// TestOSuserName OpenStack user name.
	TestOSuserName = "OSuser"
	// TestFakeCredential Fake provider credential name.
	TestFakeCredential = "fake"
	// RequiredEmailDomain required domain for predefined credentials.
	RequiredEmailDomain = "acme.com"
)

var (
	// UserLastSeen hold a time the user was last seen.
	UserLastSeen = time.Date(2020, time.December, 31, 23, 0, 0, 0, time.UTC)
)

// GetUser is a convenience function for generating apiv1.User.
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
// it is meant to be used by legacy handler.createTestEndpointAndGetClients function.
type newRoutingFunc func(
	adminProvider provider.AdminProvider,
	settingsProvider provider.SettingsProvider,
	userInfoGetter provider.UserInfoGetter,
	seedsGetter provider.SeedsGetter,
	seedClientGetter provider.SeedClientGetter,
	configGetter provider.KubermaticConfigurationGetter,
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
	saTokenAuthenticator serviceaccount.TokenAuthenticator,
	saTokenGenerator serviceaccount.TokenGenerator,
	eventRecorderProvider provider.EventRecorderProvider,
	presetProvider provider.PresetProvider,
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
	backupCredentialsProviderGetter provider.BackupCredentialsProviderGetter,
	privilegedMLAAdminSettingProviderGetter provider.PrivilegedMLAAdminSettingProviderGetter,
	masterClient ctrlruntimeclient.Client,
	featureGatesProvider provider.FeatureGatesProvider,
	seedProvider provider.SeedProvider,
	resourceQuotaProvider provider.ResourceQuotaProvider,
	groupProjectBindingProvider provider.GroupProjectBindingProvider,
	applicationDefinitionProvider provider.ApplicationDefinitionProvider,
	privilegedIPAMPoolProviderGetter provider.PrivilegedIPAMPoolProviderGetter,
	privilegedOperatingSystemProfileProviderGetter provider.PrivilegedOperatingSystemProfileProviderGetter,
	features features.FeatureGate,
) http.Handler

func getRuntimeObjects(objs ...ctrlruntimeclient.Object) []runtime.Object {
	runtimeObjects := []runtime.Object{}
	for _, obj := range objs {
		runtimeObjects = append(runtimeObjects, obj.(runtime.Object))
	}

	return runtimeObjects
}

func initTestEndpoint(user apiv1.User, seedsGetter provider.SeedsGetter, kubeObjects, machineObjects, kubermaticObjects []ctrlruntimeclient.Object, kubermaticConfiguration *kubermaticv1.KubermaticConfiguration, routingFunc newRoutingFunc) (http.Handler, *ClientsSets, error) {
	ctx := context.Background()

	allObjects := kubeObjects
	allObjects = append(allObjects, machineObjects...)
	allObjects = append(allObjects, kubermaticObjects...)

	// most tests don't actually use the KubermaticConfiguration, but since they handle the
	// configGetter, they can still fail if no config exists; to prevent this, we simply
	// create a dummy, empty config here by default, unless a test defines its own config
	if kubermaticConfiguration == nil {
		kubermaticConfiguration = &kubermaticv1.KubermaticConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubermatic",
				Namespace: resources.KubermaticNamespace,
			},
			Spec: kubermaticv1.KubermaticConfigurationSpec{
				API: kubermaticv1.KubermaticAPIConfiguration{
					AccessibleAddons: []string{"addon1", "addon2"},
				},
				Versions: kubermaticv1.KubermaticVersioningConfiguration{
					Versions: []semver.Semver{
						*semver.NewSemverOrDie("8.8.8"),
						*semver.NewSemverOrDie("9.9.9"),
						*semver.NewSemverOrDie("9.9.10"),
						*semver.NewSemverOrDie("9.11.3"),
					},
				},
			},
		}
	}

	allObjects = append(allObjects, kubermaticConfiguration)
	fakeClient := fakectrlruntimeclient.
		NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(allObjects...).
		Build()
	kubernetesClient := fakerestclient.NewSimpleClientset(getRuntimeObjects(kubeObjects...)...)
	fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
		return fakeClient, nil
	}

	sshKeyProvider := kubernetes.NewSSHKeyProvider(fakeImpersonationClient, fakeClient)
	privilegedSSHKeyProvider, err := kubernetes.NewPrivilegedSSHKeyProvider(fakeClient)
	if err != nil {
		return nil, nil, err
	}
	userProvider := kubernetes.NewUserProvider(fakeClient)
	adminProvider := kubernetes.NewAdminProvider(fakeClient)
	settingsProvider := kubernetes.NewSettingsProvider(fakeClient)
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
	projectMemberProvider := kubernetes.NewProjectMemberProvider(fakeImpersonationClient, fakeClient)
	userInfoGetter, err := provider.UserInfoGetterFactory(projectMemberProvider)
	resourceQuotaProvider := resourceQuotaProviderFactory(fakeImpersonationClient, fakeClient)
	groupProjectBindingProvider := groupProjectBindingProviderFactory(fakeImpersonationClient, fakeClient)
	if err != nil {
		return nil, nil, err
	}
	var verifiers []auth.TokenVerifier
	var extractors []auth.TokenExtractor
	{
		// if the API users is actually a service account use JWTTokenAuthentication
		// that knows how to extract and verify the token
		if kubermaticv1helper.IsProjectServiceAccount(user.Email) {
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
		GenTestSeed(),
	)
	clusterProviders := map[string]provider.ClusterProvider{"us-central1": clusterProvider}
	clusterProviderGetter := func(seed *kubermaticv1.Seed) (provider.ClusterProvider, error) {
		if clusterProvider, exists := clusterProviders[seed.Name]; exists {
			return clusterProvider, nil
		}
		return nil, fmt.Errorf("can not find clusterprovider for cluster %q", seed.Name)
	}

	credentialsManager, err := kubernetes.NewPresetProvider(fakeClient)
	if err != nil {
		return nil, nil, err
	}
	admissionPluginProvider := kubernetes.NewAdmissionPluginsProvider(fakeClient)

	if seedsGetter == nil {
		seedsGetter = CreateTestSeedsGetter(ctx, fakeClient)
	}

	seedClientGetter := func(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error) {
		return fakeClient, nil
	}

	// could also use a StaticKubermaticConfigurationGetterFactory, but this nicely tests
	// the more complex implementation on the side
	configGetter, err := kubernetes.DynamicKubermaticConfigurationGetterFactory(fakeClient, resources.KubermaticNamespace)
	if err != nil {
		return nil, nil, err
	}

	addonProvider := kubernetes.NewAddonProvider(
		fakeClient,
		fakeImpersonationClient,
		configGetter,
	)
	addonProviders := map[string]provider.AddonProvider{"us-central1": addonProvider}
	addonProviderGetter := func(seed *kubermaticv1.Seed) (provider.AddonProvider, error) {
		if addonProvider, exists := addonProviders[seed.Name]; exists {
			return addonProvider, nil
		}
		return nil, fmt.Errorf("can not find addonprovider for cluster %q", seed.Name)
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

	defaultConstraintProvider, err := kubernetes.NewDefaultConstraintProvider(fakeImpersonationClient, fakeClient, resources.KubermaticNamespace)
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

	backupCredentialsProvider := kubernetes.NewBackupCredentialsProvider(fakeClient)
	backupCredentialsProviders := map[string]provider.BackupCredentialsProvider{"us-central1": backupCredentialsProvider}
	backupCredentialsProviderGetter := func(seed *kubermaticv1.Seed) (provider.BackupCredentialsProvider, error) {
		if backupCredentials, exists := backupCredentialsProviders[seed.Name]; exists {
			return backupCredentials, nil
		}
		return nil, fmt.Errorf("can not find backupCredentialsProvider for cluster %q", seed.Name)
	}

	privilegedMLAAdminSettingProvider := kubernetes.NewPrivilegedMLAAdminSettingProvider(fakeClient)
	privilegedMLAAdminSettingProviders := map[string]provider.PrivilegedMLAAdminSettingProvider{"us-central1": privilegedMLAAdminSettingProvider}
	privilegedMLAAdminSettingProviderGetter := func(seed *kubermaticv1.Seed) (provider.PrivilegedMLAAdminSettingProvider, error) {
		if privilegedMLAAdminSetting, exists := privilegedMLAAdminSettingProviders[seed.Name]; exists {
			return privilegedMLAAdminSetting, nil
		}
		return nil, fmt.Errorf("can not find privilegedMLAAdminSettingProvider for cluster %q", seed.Name)
	}

	seedProvider := kubernetes.NewSeedProvider(fakeClient)
	if err != nil {
		return nil, nil, err
	}

	applicationDefinitionProvider := kubernetes.NewApplicationDefinitionProvider(fakeClient)

	eventRecorderProvider := kubernetes.NewEventRecorder()

	settingsWatcher, err := kuberneteswatcher.NewSettingsWatcher(ctx, zap.NewNop().Sugar())
	if err != nil {
		return nil, nil, err
	}

	userWatcher, err := kuberneteswatcher.NewUserWatcher(ctx, zap.NewNop().Sugar())
	if err != nil {
		return nil, nil, err
	}

	// Disable the metrics endpoint in tests
	var prometheusClient prometheusapi.Client

	featureGates, err := features.NewFeatures(common.StringifyFeatureGates(kubermaticConfiguration))
	if err != nil {
		return nil, nil, err
	}

	featureGatesProvider := kubernetes.NewFeatureGatesProvider(featureGates)

	privilegedIPAMPoolProvider := kubernetes.NewPrivilegedIPAMPoolProvider(fakeClient)
	privilegedIPAMPoolProviders := map[string]provider.PrivilegedIPAMPoolProvider{"us-central1": privilegedIPAMPoolProvider}
	privilegedIPAMPoolProviderGetter := func(seed *kubermaticv1.Seed) (provider.PrivilegedIPAMPoolProvider, error) {
		if privilegedIPAMPool, exists := privilegedIPAMPoolProviders[seed.Name]; exists {
			return privilegedIPAMPool, nil
		}
		return nil, fmt.Errorf("can not find privilegedIPAMPoolProvider for cluster %q", seed.Name)
	}

	privilegedOperatingSystemProfileProvider := kubernetes.NewPrivilegedOperatingSystemProfileProvider(fakeClient, "kubermatic")

	privilegedOperatingSystemProfileProviders := map[string]provider.PrivilegedOperatingSystemProfileProvider{"us-central1": privilegedOperatingSystemProfileProvider}
	privilegedOperatingSystemProfileProviderGetter := func(seed *kubermaticv1.Seed) (provider.PrivilegedOperatingSystemProfileProvider, error) {
		if operatingSystemProfiles, exists := privilegedOperatingSystemProfileProviders[seed.Name]; exists {
			return operatingSystemProfiles, nil
		}
		return nil, fmt.Errorf("can not find backupCredentialsProvider for cluster %q", seed.Name)
	}

	mainRouter := routingFunc(
		adminProvider,
		settingsProvider,
		userInfoGetter,
		seedsGetter,
		seedClientGetter,
		configGetter,
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
		backupCredentialsProviderGetter,
		privilegedMLAAdminSettingProviderGetter,
		fakeClient,
		featureGatesProvider,
		seedProvider,
		resourceQuotaProvider,
		groupProjectBindingProvider,
		applicationDefinitionProvider,
		privilegedIPAMPoolProviderGetter,
		privilegedOperatingSystemProfileProviderGetter,
		featureGates,
	)

	return mainRouter, &ClientsSets{fakeClient, kubernetesClient, tokenAuth, tokenGenerator}, nil
}

// CreateTestEndpointAndGetClients is a convenience function that instantiates fake providers and sets up routes for the tests.
func CreateTestEndpointAndGetClients(user apiv1.User, seedsGetter provider.SeedsGetter, kubeObjects, machineObjects, kubermaticObjects []ctrlruntimeclient.Object, config *kubermaticv1.KubermaticConfiguration, routingFunc newRoutingFunc) (http.Handler, *ClientsSets, error) {
	return initTestEndpoint(user, seedsGetter, kubeObjects, machineObjects, kubermaticObjects, config, routingFunc)
}

// CreateTestEndpoint does exactly the same as CreateTestEndpointAndGetClients except it omits ClientsSets when returning.
func CreateTestEndpoint(user apiv1.User, kubeObjects, kubermaticObjects []ctrlruntimeclient.Object, config *kubermaticv1.KubermaticConfiguration, routingFunc newRoutingFunc) (http.Handler, error) {
	router, _, err := CreateTestEndpointAndGetClients(user, nil, kubeObjects, nil, kubermaticObjects, config, routingFunc)
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
	seed.SetKubermaticVersion(kubermatic.NewFakeVersions())
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
			return nil, fmt.Errorf("failed to list the seeds: %w", err)
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

func (f *fakeUserClusterConnection) GetK8sClient(_ context.Context, _ *kubermaticv1.Cluster, _ ...k8cuserclusterclient.ConfigOption) (kubernetesclientset.Interface, error) {
	return nil, nil
}

func (f *fakeUserClusterConnection) GetClientConfig(_ context.Context, _ *kubermaticv1.Cluster, _ ...k8cuserclusterclient.ConfigOption) (*restclient.Config, error) {
	return nil, nil
}

func (f *fakeUserClusterConnection) GetClient(_ context.Context, _ *kubermaticv1.Cluster, _ ...k8cuserclusterclient.ConfigOption) (ctrlruntimeclient.Client, error) {
	return f.fakeDynamicClient, nil
}

// ClientsSets a simple wrapper that holds fake client sets.
type ClientsSets struct {
	FakeClient ctrlruntimeclient.Client
	// this client is used for unprivileged methods where impersonated client is used
	FakeKubernetesCoreClient kubernetesclientset.Interface

	TokenAuthenticator serviceaccount.TokenAuthenticator
	TokenGenerator     serviceaccount.TokenGenerator
}

// GenerateTestKubeconfig returns test kubeconfig yaml structure.
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

// APIUserToKubermaticUser simply converts apiv1.User to kubermaticv1.User type.
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
			IsAdmin: user.IsAdmin,
		},
	}
}

// CompareWithResult a convenience function for comparing http.Body content with response.
func CompareWithResult(t *testing.T, res *httptest.ResponseRecorder, response string) {
	t.Helper()
	bBytes, err := io.ReadAll(res.Body)
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

// GenInactiveProjectServiceAccount generates a Service Account resource.
func GenInactiveProjectServiceAccount(id, name, group, projectName string) *kubermaticv1.User {
	userName := kubermaticv1helper.EnsureProjectServiceAccountPrefix(id)

	user := GenUser(id, name, fmt.Sprintf("%s@sa.kubermatic.io", userName))
	user.Name = userName
	user.UID = ""
	user.Labels = map[string]string{kubernetes.ServiceAccountLabelGroup: fmt.Sprintf("%s-%s", group, projectName)}
	user.Spec.Project = projectName

	return user
}

func GenProjectServiceAccount(id, name, group, projectName string) *kubermaticv1.User {
	sa := GenInactiveProjectServiceAccount(id, name, group, projectName)
	sa.Labels = map[string]string{}
	return sa
}

// GenAPIUser generates a API user.
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

// GenAPIAdminUser generates an admin API user.
func GenAPIAdminUser(name, email string, isAdmin bool) *apiv1.User {
	user := GenAPIUser(name, email)
	user.IsAdmin = isAdmin
	return user
}

// DefaultCreationTimestamp returns default test timestamp.
func DefaultCreationTimestamp() time.Time {
	return time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)
}

// GenDefaultAPIUser generates a default API user.
func GenDefaultAPIUser() *apiv1.User {
	return &apiv1.User{
		ObjectMeta: apiv1.ObjectMeta{
			ID:   GenDefaultUser().Name,
			Name: GenDefaultUser().Spec.Name,
		},
		Email:    GenDefaultUser().Spec.Email,
		LastSeen: &[]apiv1.Time{apiv1.NewTime(UserLastSeen)}[0],
	}
}

// GenDefaultAdminAPIUser generates a default admin API user.
func GenDefaultAdminAPIUser() *apiv1.User {
	user := GenDefaultAPIUser()
	user.IsAdmin = true
	return user
}

// GenDefaultUser generates a default user.
func GenDefaultUser() *kubermaticv1.User {
	userEmail := "bob@acme.com"
	return GenUser("", "Bob", userEmail)
}

// GenDefaultAdminUser generates a default admin user.
func GenDefaultAdminUser() *kubermaticv1.User {
	user := GenDefaultUser()
	user.Spec.IsAdmin = true
	return user
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
			EnableUserSSHKeyAgent: pointer.BoolPtr(false),
			ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort,
			ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
				DNSDomain: "cluster.local",
				ProxyMode: "ipvs",
				IPVS: &kubermaticv1.IPVSConfiguration{
					StrictArp: pointer.BoolPtr(true),
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
						Kubelet: "v9.9.9", // initTestEndpoint() configures KKP to know 8.8.8 and 9.9.9
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
			Name:              name,
			Namespace:         cluster.Status.NamespaceName,
			CreationTimestamp: metav1.NewTime(creationTime),
		},
		Spec: kubermaticv1.AddonSpec{
			Name:      name,
			Variables: variables,
			// in reality, the addon webhook would ensure this objectRef
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
			DisplayDemoInfo:       false,
			DisplayAPIDocs:        false,
			DisplayTermsOfService: false,
			EnableDashboard:       true,
			EnableOIDCKubeconfig:  false,
		},
	}
}

func GenDefaultVersions() []semver.Semver {
	return []semver.Semver{
		*semver.NewSemverOrDie("1.22.12"),
		*semver.NewSemverOrDie("1.23.9"),
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
		ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
		Openstack: &kubermaticv1.OpenstackCloudSpec{
			Username:       "username",
			Password:       "password",
			SubnetID:       "subnetID",
			Domain:         "domain",
			FloatingIPPool: "floatingIPPool",
			Network:        "network",
			RouterID:       "routerID",
			SecurityGroups: "securityGroups",
			Project:        "project",
		},
	}
	return cluster
}

func GenClusterWithOpenstackProjectAuth(cluster *kubermaticv1.Cluster) *kubermaticv1.Cluster {
	cluster.Spec.Cloud = kubermaticv1.CloudSpec{
		DatacenterName: "OpenstackDatacenter",
		ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
		Openstack: &kubermaticv1.OpenstackCloudSpec{
			Username:       "username",
			Password:       "password",
			SubnetID:       "subnetID",
			Domain:         "domain",
			FloatingIPPool: "floatingIPPool",
			Network:        "network",
			RouterID:       "routerID",
			SecurityGroups: "securityGroups",
			Project:        "project",
			ProjectID:      "projectID",
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

func GenServiceAccountRoleBinding(name string, namespace string, roleID string, subjects []rbacv1.Subject) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Labels:    map[string]string{handlercommon.UserClusterComponentKey: handlercommon.UserClusterBindingComponentValue},
			Namespace: namespace,
		},
		Subjects: subjects,
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

func GenServiceAccountClusterRoleBinding(name string, roleID string, subjects []rbacv1.Subject) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{handlercommon.UserClusterComponentKey: handlercommon.UserClusterBindingComponentValue},
		},
		Subjects: subjects,
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

func GenClusterTemplateInstance(projectID, templateID, owner string, replicas int64) *kubermaticv1.ClusterTemplateInstance {
	return &kubermaticv1.ClusterTemplateInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-%s", projectID, templateID),
			Labels:      map[string]string{kubernetes.ClusterTemplateLabelKey: templateID, kubermaticv1.ProjectIDLabelKey: projectID},
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

func GenAdminRuleGroup(name, namespace string, ruleGroupType kubermaticv1.RuleGroupType) *kubermaticv1.RuleGroup {
	return &kubermaticv1.RuleGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       kubermaticv1.RuleGroupKindName,
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		},
		Spec: kubermaticv1.RuleGroupSpec{
			RuleGroupType: ruleGroupType,
			Cluster:       corev1.ObjectReference{},
			Data:          GenerateTestRuleGroupData(name),
		},
	}
}

func GenAPIRuleGroup(name string, ruleGroupType kubermaticv1.RuleGroupType, isDefault bool) *apiv2.RuleGroup {
	return &apiv2.RuleGroup{
		Name:      name,
		IsDefault: isDefault,
		Data:      GenerateTestRuleGroupData(name),
		Type:      ruleGroupType,
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
		ObjectMeta: apiv1.ObjectMeta{
			Name:              name,
			ID:                etcdbackupconfig.GenEtcdBackupConfigID(name, clusterID),
			Annotations:       nil,
			CreationTimestamp: apiv1.Date(0001, 01, 01, 00, 00, 0, 0, time.UTC),
		},
		Spec: apiv2.EtcdBackupConfigSpec{
			ClusterID:   clusterID,
			Schedule:    "5 * * * * *",
			Keep:        &keep,
			Destination: "s3",
		},
	}
}

func GenEtcdBackupConfig(name string, cluster *kubermaticv1.Cluster, projectID string) *kubermaticv1.EtcdBackupConfig {
	keep := 5
	clusterObjectRef, _ := reference.GetReference(scheme.Scheme, cluster)

	return &kubermaticv1.EtcdBackupConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Status.NamespaceName,
			Labels: map[string]string{
				kubermaticv1.ProjectIDLabelKey: projectID,
			},
		},
		Spec: kubermaticv1.EtcdBackupConfigSpec{
			Name:        name,
			Cluster:     *clusterObjectRef,
			Schedule:    "5 * * * * *",
			Keep:        &keep,
			Destination: "s3",
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
			Destination:                     "s3",
		},
	}
}

func GenEtcdRestore(name string, cluster *kubermaticv1.Cluster, projectID string) *kubermaticv1.EtcdRestore {
	clusterObjectRef, _ := reference.GetReference(scheme.Scheme, cluster)

	return &kubermaticv1.EtcdRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Status.NamespaceName,
			Labels: map[string]string{
				kubermaticv1.ProjectIDLabelKey: projectID,
			},
		},
		Spec: kubermaticv1.EtcdRestoreSpec{
			Name:                            name,
			Cluster:                         *clusterObjectRef,
			BackupName:                      "backup-1",
			BackupDownloadCredentialsSecret: "secret",
			Destination:                     "s3",
		},
	}
}

func GenDefaultAPIBackupCredentials() *apiv2.BackupCredentials {
	return &apiv2.BackupCredentials{
		S3BackupCredentials: apiv2.S3BackupCredentials{
			AccessKeyID:     "accessKeyId",
			SecretAccessKey: "secretAccessKey",
		},
		Destination: "s3",
	}
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

func GenAPIMLAAdminSetting(value int32) *apiv2.MLAAdminSetting {
	return &apiv2.MLAAdminSetting{
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
	}
}

func GenApplicationInstallation(name, clusterName, targetnamespace string) *appskubermaticv1.ApplicationInstallation {
	return &appskubermaticv1.ApplicationInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: targetnamespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       appskubermaticv1.ApplicationInstallationKindName,
			APIVersion: appskubermaticv1.SchemeGroupVersion.String(),
		},
		Spec: appskubermaticv1.ApplicationInstallationSpec{
			Namespace: appskubermaticv1.AppNamespaceSpec{
				Name:   targetnamespace,
				Create: true,
			},
			ApplicationRef: appskubermaticv1.ApplicationRef{
				Name: "sample-app",
				Version: appskubermaticv1.Version{
					Version: *semverlib.MustParse("v1.0.0"),
				},
			},
		},
	}
}

func GenApiApplicationInstallation(name, clusterName, targetnamespace string) *apiv2.ApplicationInstallation {
	return &apiv2.ApplicationInstallation{
		ObjectMeta: apiv1.ObjectMeta{
			Name: name,
			ID:   name,
		},
		Namespace: targetnamespace,
		Spec: &apiv2.ApplicationInstallationSpec{
			Namespace: apiv2.NamespaceSpec{
				Name:   targetnamespace,
				Create: true,
			},
			ApplicationRef: appskubermaticv1.ApplicationRef{
				Name: "sample-app",
				Version: appskubermaticv1.Version{
					Version: *semverlib.MustParse("v1.0.0"),
				},
			},
		},
		Status: &apiv2.ApplicationInstallationStatus{},
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

func GenApiApplicationDefinition(name string) apiv2.ApplicationDefinition {
	return apiv2.ApplicationDefinition{
		ObjectMeta: apiv1.ObjectMeta{
			Name: name,
		},
		Spec: &appskubermaticv1.ApplicationDefinitionSpec{
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

func GenApiApplicationDefinitionListItem(name string) apiv2.ApplicationDefinitionListItem {
	return apiv2.ApplicationDefinitionListItem{
		Name: name,
		Spec: apiv2.ApplicationDefinitionListItemSpec{},
	}
}
