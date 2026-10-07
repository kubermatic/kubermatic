//go:build e2e && ee

/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package kyverno

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	userclusterresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/user-cluster"
	policybindingcontroller "k8c.io/kubermatic/v2/pkg/ee/policy-binding-controller"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	"k8c.io/kubermatic/v2/pkg/validation"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	waitInterval       = 3 * time.Second
	waitTimeout        = 5 * time.Minute
	requiredLabelKey   = "kyverno-e2e"
	requiredLabelValue = "enabled"
	policyDenyMessage  = "the required kyverno-e2e label is missing or invalid"
	testClusterLabel   = "e2e.kubermatic.k8c.io/kyverno"
)

var (
	credentials jig.AWSCredentials
	logOptions  = utils.DefaultLogOptions
)

func init() {
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

type templateOptions struct {
	namespaced, enforced, defaultPolicy bool
	target                              string
}

func TestPolicyTemplateFixtures(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		namespaced, enforced bool
	}{
		{name: "cluster-wide"},
		{name: "cluster-wide-enforced", enforced: true},
		{name: "namespaced", namespaced: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			template, err := newPolicyTemplate("kyverno-e2e-fixture", "kyverno-e2e-fixture", tc.namespaced, tc.enforced)
			require.NoError(t, err)
			require.Empty(t, validation.ValidatePolicyTemplate(template))
		})
	}
}

// The subtests share one AWS cluster and form an ordered lifecycle. Stop when a
// prerequisite fails, while always cleaning up templates, cluster and project.
func TestKyvernoIntegration(t *testing.T) {
	ctx := context.Background()
	rawLogger := log.NewFromOptions(logOptions)
	logger := rawLogger.Sugar()
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))
	require.NoError(t, credentials.Parse(), "get AWS credentials")

	// The CI runner installs a combined master/seed. Templates are authored on
	// that cluster, and bindings live in the user cluster's seed namespace.
	seedClient, _, err := utils.GetClients()
	require.NoError(t, err, "create seed client")
	suffix := rand.String(6)
	testJig := jig.NewAWSCluster(seedClient, logger, credentials, 1, nil)
	testJig.ClusterJig.WithTestName("kyverno").WithLabels(map[string]string{testClusterLabel: suffix}).
		WithPatch(func(spec *kubermaticv1.ClusterSpec) *kubermaticv1.ClusterSpec {
			spec.Kyverno = &kubermaticv1.KyvernoSettings{Enabled: true}
			return spec
		})

	var templates []*kubermaticv1.PolicyTemplate
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		// Request every template deletion even if an earlier resource is stuck.
		// Then delete the cluster so its cleanup fallback can release bindings.
		for _, template := range templates {
			if err := seedClient.Delete(cleanupCtx, template); err != nil && !apierrors.IsNotFound(err) {
				t.Errorf("delete PolicyTemplate %s: %v", template.Name, err)
			}
		}
		testJig.Cleanup(cleanupCtx, t, true)
		for _, template := range templates {
			if err := waitForObjectDeleted(cleanupCtx, seedClient, logger, template); err != nil {
				t.Errorf("wait for PolicyTemplate %s cleanup: %v", template.Name, err)
			}
		}
	})

	createTemplate := func(t *testing.T, name, namespace string, options templateOptions) *kubermaticv1.PolicyTemplate {
		t.Helper()
		template, err := newPolicyTemplate(name+"-"+suffix, namespace, options.namespaced, options.enforced)
		require.NoError(t, err)
		template.Spec.Default = options.defaultPolicy
		target := options.target
		if target == "" {
			target = suffix
		}
		template.Spec.Target = &kubermaticv1.PolicyTemplateTarget{
			ClusterSelector: &metav1.LabelSelector{MatchLabels: map[string]string{testClusterLabel: target}},
		}
		template.Spec.Category = "E2E"
		template.Spec.Severity = "medium"
		template.Annotations = map[string]string{"e2e.kubermatic.k8c.io/fixture": suffix}
		require.Empty(t, validation.ValidatePolicyTemplate(template))
		require.NoError(t, seedClient.Create(ctx, template), "create PolicyTemplate %s", template.Name)
		templates = append(templates, template)
		return template
	}

	defaultNamespace := "kyverno-e2e-default-" + suffix
	clusterNamespace := "kyverno-e2e-cluster-" + suffix
	policyNamespace := "kyverno-e2e-policy-" + suffix
	movedNamespace := "kyverno-e2e-moved-" + suffix
	// Default bindings are initialized once, so these templates must exist
	// before the cluster becomes healthy for the first time.
	defaultTemplate := createTemplate(t, "kyverno-e2e-default", defaultNamespace, templateOptions{defaultPolicy: true})
	excludedTemplate := createTemplate(t, "kyverno-e2e-excluded", defaultNamespace, templateOptions{enforced: true, target: "excluded-" + suffix})
	_, cluster, err := testJig.Setup(ctx, jig.WaitForReadyPods)
	require.NoError(t, err, "set up AWS user cluster")
	userClient, err := testJig.ClusterClient(ctx)
	require.NoError(t, err, "create user cluster client")
	require.NoError(t, kyvernov1.Install(userClient.Scheme()))
	require.NoError(t, apiextensionsv1.AddToScheme(userClient.Scheme()))

	assertInstalled := func(t *testing.T) {
		t.Helper()
		require.NoError(t, testJig.WaitForKyvernoHealthy(ctx, 10*time.Minute))
		// Health can still report the previous installation as Up after re-enable.
		require.NoError(t, waitForSeedKyvernoControllersReady(ctx, seedClient, logger, cluster.Status.NamespaceName))
		require.NoError(t, waitForKyvernoUserClusterResources(ctx, userClient, logger, cluster.Status.NamespaceName))
		require.NoError(t, waitForClusterKyvernoFinalizer(ctx, seedClient, logger, cluster.Name, true))
	}
	assertActive := func(t *testing.T, template *kubermaticv1.PolicyTemplate, binding *kubermaticv1.PolicyBinding, namespace, labelValue string) {
		t.Helper()
		require.NoError(t, waitForActivePolicyBinding(ctx, seedClient, logger, ctrlruntimeclient.ObjectKeyFromObject(binding), template.Spec.Enforced))
		var policy kyvernoPolicy
		if template.Spec.NamespacedPolicy {
			require.NoError(t, waitForPolicyReady(ctx, userClient, logger, namespace, template.Name))
			policy = &kyvernov1.Policy{ObjectMeta: metav1.ObjectMeta{Name: template.Name, Namespace: namespace}}
		} else {
			require.NoError(t, waitForClusterPolicyReady(ctx, userClient, logger, template.Name))
			policy = &kyvernov1.ClusterPolicy{ObjectMeta: metav1.ObjectMeta{Name: template.Name}}
		}
		require.NoError(t, waitForPolicyContent(ctx, userClient, logger, policy, template, binding.Name))
		require.NoError(t, verifyPolicyAdmission(ctx, userClient, logger, namespace, template.Name, labelValue))
	}
	defaultBinding := newPolicyBinding(cluster.Status.NamespaceName, defaultTemplate.Name, "", false)
	excludedBinding := newPolicyBinding(cluster.Status.NamespaceName, excludedTemplate.Name, "", false)

	if !t.Run("installation-and-default-binding", func(t *testing.T) {
		assertInstalled(t)
		for _, namespace := range []string{defaultNamespace, clusterNamespace} {
			require.NoError(t, userClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}))
		}
		assertActive(t, defaultTemplate, defaultBinding, defaultNamespace, requiredLabelValue)
		require.NoError(t, seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(defaultBinding), defaultBinding))
		require.Equal(t, "true", defaultBinding.Annotations[kubermaticv1.AnnotationPolicyDefault])
		require.NoError(t, wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
			current := &kubermaticv1.Cluster{}
			if err := seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(cluster), current); err != nil {
				return err, nil
			}
			if !current.Status.HasConditionValue(kubermaticv1.ClusterConditionDefaultPolicyBindingsControllerCreatedSuccessfully, corev1.ConditionTrue) {
				return fmt.Errorf("default policy initialization is not complete"), nil
			}
			return nil, nil
		}))
		require.NoError(t, waitForObjectDeleted(ctx, seedClient, logger, excludedBinding))
		require.NoError(t, seedClient.Delete(ctx, defaultBinding))
		require.NoError(t, waitForObjectDeleted(ctx, seedClient, logger, defaultBinding))
		require.NoError(t, waitForObjectDeleted(ctx, userClient, logger, &kyvernov1.ClusterPolicy{ObjectMeta: metav1.ObjectMeta{Name: defaultTemplate.Name}}))
		require.NoError(t, waitForAdmissionAllowed(ctx, userClient, logger, defaultNamespace))
	}) {
		return
	}

	var clusterTemplate *kubermaticv1.PolicyTemplate
	var clusterBinding *kubermaticv1.PolicyBinding
	if !t.Run("enforced-binding-recreation-and-template-update", func(t *testing.T) {
		clusterTemplate = createTemplate(t, "kyverno-e2e-enforced", clusterNamespace, templateOptions{enforced: true})
		clusterBinding = newPolicyBinding(cluster.Status.NamespaceName, clusterTemplate.Name, "", false)
		assertActive(t, clusterTemplate, clusterBinding, clusterNamespace, requiredLabelValue)
		require.NoError(t, seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(clusterBinding), clusterBinding))
		oldUID := clusterBinding.UID
		require.NoError(t, seedClient.Delete(ctx, clusterBinding))
		require.NoError(t, wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
			current := &kubermaticv1.PolicyBinding{}
			if err := seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(clusterBinding), current); err != nil {
				return err, nil
			}
			if current.UID == oldUID || current.DeletionTimestamp != nil {
				return fmt.Errorf("enforced binding has not been recreated"), nil
			}
			clusterBinding = current
			return nil, nil
		}))
		assertActive(t, clusterTemplate, clusterBinding, clusterNamespace, requiredLabelValue)
		updateTemplateLabel(t, ctx, seedClient, clusterTemplate, "updated")
		assertActive(t, clusterTemplate, clusterBinding, clusterNamespace, "updated")
		// The enforced template event reconciled this cluster again. Defaults must
		// remain opted out, and a selector mismatch must never create a binding.
		require.NoError(t, waitForObjectDeleted(ctx, seedClient, logger, defaultBinding))
		require.NoError(t, waitForObjectDeleted(ctx, seedClient, logger, excludedBinding))
		require.NoError(t, waitForAdmissionAllowed(ctx, userClient, logger, defaultNamespace))
	}) {
		return
	}

	var namespacedTemplate *kubermaticv1.PolicyTemplate
	var namespacedBinding *kubermaticv1.PolicyBinding
	if !t.Run("namespaced-binding-activation-and-retargeting", func(t *testing.T) {
		namespacedTemplate = createTemplate(t, "kyverno-e2e-namespaced", "", templateOptions{namespaced: true})
		// Deliberately use a different binding name to check ownership mapping.
		namespacedBinding = newPolicyBinding(cluster.Status.NamespaceName, namespacedTemplate.Name, "", false)
		namespacedBinding.Name = "kyverno-e2e-binding-" + suffix
		require.NoError(t, seedClient.Create(ctx, namespacedBinding))
		require.NoError(t, waitForPolicyBindingState(ctx, seedClient, logger, ctrlruntimeclient.ObjectKeyFromObject(namespacedBinding), false, metav1.ConditionFalse, kubermaticv1.PolicyBindingReasonPolicyNamespaceMissing, true, false))
		setBindingNamespace(t, ctx, seedClient, namespacedBinding, policyNamespace)
		assertActive(t, namespacedTemplate, namespacedBinding, policyNamespace, requiredLabelValue)
		require.NoError(t, waitForAdmissionAllowed(ctx, userClient, logger, defaultNamespace), "namespaced policy must not apply outside its namespace")

		setBindingNamespace(t, ctx, seedClient, namespacedBinding, movedNamespace)
		assertActive(t, namespacedTemplate, namespacedBinding, movedNamespace, requiredLabelValue)
		require.NoError(t, waitForObjectDeleted(ctx, userClient, logger, &kyvernov1.Policy{ObjectMeta: metav1.ObjectMeta{Name: namespacedTemplate.Name, Namespace: policyNamespace}}))
		require.NoError(t, waitForAdmissionAllowed(ctx, userClient, logger, policyNamespace))

		setBindingNamespace(t, ctx, seedClient, namespacedBinding, "")
		require.NoError(t, waitForPolicyBindingState(ctx, seedClient, logger, ctrlruntimeclient.ObjectKeyFromObject(namespacedBinding), false, metav1.ConditionFalse, kubermaticv1.PolicyBindingReasonPolicyNamespaceMissing, true, false))
		require.NoError(t, waitForObjectDeleted(ctx, userClient, logger, &kyvernov1.Policy{ObjectMeta: metav1.ObjectMeta{Name: namespacedTemplate.Name, Namespace: movedNamespace}}))
		require.NoError(t, waitForAdmissionAllowed(ctx, userClient, logger, movedNamespace))
		setBindingNamespace(t, ctx, seedClient, namespacedBinding, policyNamespace)
		assertActive(t, namespacedTemplate, namespacedBinding, policyNamespace, requiredLabelValue)
	}) {
		return
	}

	if !t.Run("binding-and-template-deletion", func(t *testing.T) {
		require.NoError(t, seedClient.Delete(ctx, namespacedBinding))
		require.NoError(t, waitForObjectDeleted(ctx, seedClient, logger, namespacedBinding))
		require.NoError(t, waitForObjectDeleted(ctx, userClient, logger, &kyvernov1.Policy{ObjectMeta: metav1.ObjectMeta{Name: namespacedTemplate.Name, Namespace: policyNamespace}}))
		require.NoError(t, waitForAdmissionAllowed(ctx, userClient, logger, policyNamespace))
		require.NoError(t, waitForPolicyTemplateFinalizers(ctx, seedClient, logger, clusterTemplate.Name))
		require.NoError(t, seedClient.Delete(ctx, clusterTemplate))
		require.NoError(t, waitForObjectDeleted(ctx, seedClient, logger, clusterBinding))
		require.NoError(t, waitForObjectDeleted(ctx, userClient, logger, &kyvernov1.ClusterPolicy{ObjectMeta: metav1.ObjectMeta{Name: clusterTemplate.Name}}))
		require.NoError(t, waitForObjectDeleted(ctx, seedClient, logger, clusterTemplate))
		require.NoError(t, waitForAdmissionAllowed(ctx, userClient, logger, clusterNamespace))
	}) {
		return
	}

	if !t.Run("disable-with-active-bindings", func(t *testing.T) {
		clusterTemplate = createTemplate(t, "kyverno-e2e-manual", clusterNamespace, templateOptions{})
		clusterBinding = newPolicyBinding(cluster.Status.NamespaceName, clusterTemplate.Name, "", false)
		require.NoError(t, seedClient.Create(ctx, clusterBinding))
		namespacedBinding = newPolicyBinding(cluster.Status.NamespaceName, namespacedTemplate.Name, policyNamespace, true)
		namespacedBinding.Name = "kyverno-e2e-binding-" + suffix
		require.NoError(t, seedClient.Create(ctx, namespacedBinding))
		assertActive(t, clusterTemplate, clusterBinding, clusterNamespace, requiredLabelValue)
		assertActive(t, namespacedTemplate, namespacedBinding, policyNamespace, requiredLabelValue)
		require.NoError(t, setKyvernoEnabled(ctx, seedClient, cluster.Name, false))
		for _, binding := range []*kubermaticv1.PolicyBinding{clusterBinding, namespacedBinding} {
			require.NoError(t, waitForInactivePolicyBinding(ctx, seedClient, logger, ctrlruntimeclient.ObjectKeyFromObject(binding)))
		}
		for _, webhook := range userclusterresources.WebhooksForDeletion() {
			require.NoError(t, waitForObjectDeleted(ctx, userClient, logger, webhook))
		}
		require.NoError(t, waitForKyvernoCRDsRemoved(ctx, userClient, logger))
		require.NoError(t, waitForObjectDeleted(ctx, userClient, logger, &kyvernov1.ClusterPolicy{ObjectMeta: metav1.ObjectMeta{Name: clusterTemplate.Name}}))
		require.NoError(t, waitForObjectDeleted(ctx, userClient, logger, &kyvernov1.Policy{ObjectMeta: metav1.ObjectMeta{Name: namespacedTemplate.Name, Namespace: policyNamespace}}))
		require.NoError(t, waitForObjectDeleted(ctx, userClient, logger, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: cluster.Status.NamespaceName}}))
		require.NoError(t, waitForSeedKyvernoControllersRemoved(ctx, seedClient, logger, cluster.Status.NamespaceName))
		require.NoError(t, waitForClusterKyvernoFinalizer(ctx, seedClient, logger, cluster.Name, false))
		for _, namespace := range []string{clusterNamespace, policyNamespace} {
			require.NoError(t, waitForAdmissionAllowed(ctx, userClient, logger, namespace))
		}
	}) {
		return
	}

	if !t.Run("re-enable-existing-bindings", func(t *testing.T) {
		require.NoError(t, setKyvernoEnabled(ctx, seedClient, cluster.Name, true))
		assertInstalled(t)
		assertActive(t, clusterTemplate, clusterBinding, clusterNamespace, requiredLabelValue)
		assertActive(t, namespacedTemplate, namespacedBinding, policyNamespace, requiredLabelValue)
		require.NoError(t, waitForObjectDeleted(ctx, seedClient, logger, defaultBinding))
		require.NoError(t, waitForObjectDeleted(ctx, seedClient, logger, excludedBinding))
		require.NoError(t, waitForAdmissionAllowed(ctx, userClient, logger, defaultNamespace))
	}) {
		return
	}

	t.Run("cluster-deletion-with-active-bindings", func(t *testing.T) {
		require.NoError(t, testJig.ClusterJig.Delete(ctx, true), "Kyverno must not block cluster deletion")
		require.NoError(t, waitForObjectDeleted(ctx, seedClient, logger, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: cluster.Status.NamespaceName}}))
	})
}

func setBindingNamespace(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, binding *kubermaticv1.PolicyBinding, namespace string) {
	t.Helper()
	require.NoError(t, client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(binding), binding))
	before := binding.DeepCopy()
	binding.Spec.KyvernoPolicyNamespace = nil
	if namespace != "" {
		binding.Spec.KyvernoPolicyNamespace = &kubermaticv1.KyvernoPolicyNamespace{Name: namespace}
	}
	require.NoError(t, client.Patch(ctx, binding, ctrlruntimeclient.MergeFrom(before)))
}

func updateTemplateLabel(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, template *kubermaticv1.PolicyTemplate, value string) {
	t.Helper()
	require.NoError(t, client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(template), template))
	before := template.DeepCopy()
	var spec kyvernov1.Spec
	require.NoError(t, json.Unmarshal(template.Spec.PolicySpec.Raw, &spec))
	pattern, err := json.Marshal(map[string]any{"metadata": map[string]any{"labels": map[string]string{requiredLabelKey: value}}})
	require.NoError(t, err)
	spec.Rules[0].Validation.RawPattern = &apiextensionsv1.JSON{Raw: pattern}
	raw, err := json.Marshal(spec)
	require.NoError(t, err)
	template.Spec.PolicySpec = runtime.RawExtension{Raw: raw}
	template.Spec.Title = "Updated Kyverno e2e required label"
	template.Annotations["e2e.kubermatic.k8c.io/fixture"] = "updated"
	require.NoError(t, client.Patch(ctx, template, ctrlruntimeclient.MergeFrom(before)))
}

type kyvernoPolicy interface {
	ctrlruntimeclient.Object
	kyvernov1.PolicyInterface
}

func waitForPolicyContent(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, policy kyvernoPolicy, template *kubermaticv1.PolicyTemplate, bindingName string) error {
	var expected kyvernov1.Spec
	if err := json.Unmarshal(template.Spec.PolicySpec.Raw, &expected); err != nil {
		return err
	}
	var expectedPattern any
	if err := json.Unmarshal(expected.Rules[0].Validation.RawPattern.Raw, &expectedPattern); err != nil {
		return err
	}
	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(policy), policy); err != nil {
			return err, nil
		}
		if policy.GetLabels()[policybindingcontroller.LabelPolicyBinding] != bindingName || policy.GetLabels()[policybindingcontroller.LabelPolicyTemplate] != template.Name {
			return fmt.Errorf("policy %s has unexpected ownership labels: %v", template.Name, policy.GetLabels()), nil
		}
		expectedAnnotations := map[string]string{
			policybindingcontroller.AnnotationTitle:       template.Spec.Title,
			policybindingcontroller.AnnotationDescription: template.Spec.Description,
			policybindingcontroller.AnnotationCategory:    template.Spec.Category,
			policybindingcontroller.AnnotationSeverity:    template.Spec.Severity,
		}
		for key, value := range template.Annotations {
			expectedAnnotations[key] = value
		}
		for key, value := range expectedAnnotations {
			if policy.GetAnnotations()[key] != value {
				return fmt.Errorf("policy %s annotation %s is %q, want %q", template.Name, key, policy.GetAnnotations()[key], value), nil
			}
		}
		// Kyverno defaults other spec fields. Compare the authored rule pattern and
		// then exercise admission, rather than equating a defaulted and raw spec.
		if len(policy.GetSpec().Rules) != 1 || policy.GetSpec().Rules[0].Validation == nil {
			return fmt.Errorf("policy %s does not contain the expected validation rule", template.Name), nil
		}
		rawPattern := policy.GetSpec().Rules[0].Validation.RawPattern
		if rawPattern == nil {
			return fmt.Errorf("policy %s has no validation pattern", template.Name), nil
		}
		var actualPattern any
		if err := json.Unmarshal(rawPattern.Raw, &actualPattern); err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(actualPattern, expectedPattern) {
			return fmt.Errorf("policy %s has not reconciled the template's validation pattern", template.Name), nil
		}
		return nil, nil
	})
}
