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

package resources

import (
	"context"
	"slices"
	"testing"

	"k8c.io/kubermatic/v2/pkg/resources"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/admission/plugin/cel"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/matchconditions"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/predicates/rules"
	celconfig "k8s.io/apiserver/pkg/apis/cel"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/cel/environment"
)

// desiredPolicies returns the policies the reconcilers create in an empty cluster.
func desiredPolicies(t *testing.T) []*admissionregistrationv1.ValidatingAdmissionPolicy {
	t.Helper()

	desired := []*admissionregistrationv1.ValidatingAdmissionPolicy{}
	for _, factory := range ValidatingAdmissionPolicyReconcilers() {
		name, reconcile := factory()
		policy, err := reconcile(&admissionregistrationv1.ValidatingAdmissionPolicy{ObjectMeta: metav1.ObjectMeta{Name: name}})
		if err != nil {
			t.Fatalf("failed to reconcile policy %s: %v", name, err)
		}
		desired = append(desired, policy)
	}

	return desired
}

func desiredBindings(t *testing.T) []*admissionregistrationv1.ValidatingAdmissionPolicyBinding {
	t.Helper()

	desired := []*admissionregistrationv1.ValidatingAdmissionPolicyBinding{}
	for _, factory := range ValidatingAdmissionPolicyBindingReconcilers() {
		name, reconcile := factory()
		binding, err := reconcile(&admissionregistrationv1.ValidatingAdmissionPolicyBinding{ObjectMeta: metav1.ObjectMeta{Name: name}})
		if err != nil {
			t.Fatalf("failed to reconcile binding %s: %v", name, err)
		}
		desired = append(desired, binding)
	}

	return desired
}

// compiledPolicy is a policy with its expressions compiled by the compiler of the kube-apiserver.
type compiledPolicy struct {
	policy          *admissionregistrationv1.ValidatingAdmissionPolicy
	matchConditions cel.ConditionEvaluator
	validations     cel.ConditionEvaluator
}

func compilePolicies(t *testing.T) []compiledPolicy {
	t.Helper()

	// what the kube-apiserver declares for the expressions of a policy without parameters
	variables := cel.OptionalVariableDeclarations{HasAuthorizer: true}
	compiled := []compiledPolicy{}
	for _, policy := range desiredPolicies(t) {
		compiler, err := cel.NewCompositedCompiler(environment.MustBaseEnvSet(environment.DefaultCompatibilityVersion()))
		if err != nil {
			t.Fatalf("failed to create the compiler: %v", err)
		}

		matchConditions := make([]cel.ExpressionAccessor, len(policy.Spec.MatchConditions))
		for i := range policy.Spec.MatchConditions {
			matchConditions[i] = (*matchconditions.MatchCondition)(&policy.Spec.MatchConditions[i])
		}
		// a validation is an expression that returns a boolean, like a match condition
		validations := make([]cel.ExpressionAccessor, len(policy.Spec.Validations))
		for i, validation := range policy.Spec.Validations {
			validations[i] = &matchconditions.MatchCondition{Name: validation.Message, Expression: validation.Expression}
		}

		compiled = append(compiled, compiledPolicy{
			policy:          policy,
			matchConditions: compiler.CompileCondition(matchConditions, variables, environment.StoredExpressions),
			validations:     compiler.CompileCondition(validations, variables, environment.StoredExpressions),
		})
	}

	return compiled
}

// testRequest is a request to the kube-apiserver of a user cluster, in kube-system.
type testRequest struct {
	user        user.Info
	operation   admission.Operation
	resource    string
	subresource string
	name        string
	kind        schema.GroupVersionKind
	object      runtime.Object
	oldObject   runtime.Object
}

// evaluate returns whether a rule of the policies matches the request, and the message the
// policies refuse it with. The message is empty if the request is admitted.
func (r testRequest) evaluate(t *testing.T, policies []compiledPolicy) (bool, string) {
	t.Helper()

	ctx := context.Background()
	resource := corev1.SchemeGroupVersion.WithResource(r.resource)
	attributes := admission.NewAttributesRecord(r.object, r.oldObject, r.kind, metav1.NamespaceSystem, r.name, resource, r.subresource, r.operation, nil, false, r.user)
	versioned := &admission.VersionedAttributes{Attributes: attributes, VersionedKind: r.kind}
	if r.object != nil {
		versioned.VersionedObject = admission.NewLazyObject(r.object)
	}
	if r.oldObject != nil {
		versioned.VersionedOldObject = admission.NewLazyObject(r.oldObject)
	}
	request := cel.CreateAdmissionRequest(attributes, metav1.GroupVersionResource(resource), metav1.GroupVersionKind(r.kind))
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name:   metav1.NamespaceSystem,
		Labels: map[string]string{corev1.LabelMetadataName: metav1.NamespaceSystem},
	}}
	// returns for every expression whether it is true; an expression that fails is a bug in the policy
	isTrue := func(policy string, evaluator cel.ConditionEvaluator) []bool {
		results, _, err := evaluator.ForInput(ctx, versioned, request, cel.OptionalVariableBindings{}, namespace, celconfig.RuntimeCELCostBudget)
		if err != nil {
			t.Fatalf("policy %s could not be evaluated: %v", policy, err)
		}

		values := make([]bool, len(results))
		for i, result := range results {
			if result.Error != nil {
				t.Fatalf("policy %s, expression %q: %v", policy, result.ExpressionAccessor.GetExpression(), result.Error)
			}
			values[i] = result.EvalResult.Value() == true
		}

		return values
	}

	matched, refusal := false, ""
	for _, compiled := range policies {
		if !slices.ContainsFunc(compiled.policy.Spec.MatchConstraints.ResourceRules, func(rule admissionregistrationv1.NamedRuleWithOperations) bool {
			return (&rules.Matcher{Rule: rule.RuleWithOperations, Attr: attributes}).Matches()
		}) {
			continue
		}
		matched = true

		// a policy is not evaluated for a request that does not meet all of its match conditions
		if slices.Contains(isTrue(compiled.policy.Name, compiled.matchConditions), false) {
			continue
		}
		for i, valid := range isTrue(compiled.policy.Name, compiled.validations) {
			if !valid {
				refusal = compiled.policy.Spec.Validations[i].Message
			}
		}
	}

	return matched, refusal
}

// TestPolicies evaluates the expressions of the policies for requests to the kube-apiserver, with
// the compiler and the evaluator the kube-apiserver uses for them.
func TestPolicies(t *testing.T) {
	policies := compilePolicies(t)

	// the names the KKP API gives the pod and the ServiceAccount of a session
	session := resources.WebTerminalSessionNamePrefix + "5d41402abc4b2a76b9719d911017c592"

	kkp := &user.DefaultInfo{Name: resources.InternalUserClusterAdminKubeconfigCertUsername, Groups: []string{user.SystemPrivilegedGroup, user.AllAuthenticated}}
	// the admin kubeconfig users download
	admin := &user.DefaultInfo{Name: "admin", Groups: []string{user.SystemPrivilegedGroup, user.AllAuthenticated}}
	eve := &user.DefaultInfo{Name: "eve@acme.com", Groups: []string{"oidc:devs", user.AllAuthenticated}}
	kubelet := &user.DefaultInfo{Name: "system:node:worker-1", Groups: []string{user.NodesGroup, user.AllAuthenticated}}
	controller := &user.DefaultInfo{Name: "system:serviceaccount:kube-system:replicaset-controller", Groups: []string{"system:serviceaccounts", user.AllAuthenticated}}

	podKind := corev1.SchemeGroupVersion.WithKind("Pod")
	secretKind := corev1.SchemeGroupVersion.WithKind("Secret")
	// the pod of a session: a shell container and the kubectl proxy that holds the token
	sessionPod := func(modify func(*corev1.Pod)) *corev1.Pod {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      session,
				Namespace: metav1.NamespaceSystem,
				Labels:    map[string]string{resources.AppLabelKey: resources.WebTerminalAppLabelValue},
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: session,
				Containers: []corev1.Container{
					{Name: session, Image: resources.WEBTerminalImage},
					{Name: "kubectl-proxy", Image: resources.WEBTerminalImage},
				},
			},
		}
		if modify != nil {
			modify(pod)
		}

		return pod
	}
	otherPod := func(modify func(*corev1.Pod)) *corev1.Pod {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "coredns-5d78c9869d-abcde", Namespace: metav1.NamespaceSystem},
			Spec: corev1.PodSpec{
				ServiceAccountName: "coredns",
				Containers:         []corev1.Container{{Name: "coredns", Image: "registry.k8s.io/coredns/coredns:v1.12.0"}},
			},
		}
		if modify != nil {
			modify(pod)
		}

		return pod
	}
	tokenSecret := func(secretType corev1.SecretType, serviceAccount string) *corev1.Secret {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "token", Namespace: metav1.NamespaceSystem},
			Type:       secretType,
		}
		if serviceAccount != "" {
			secret.Annotations = map[string]string{corev1.ServiceAccountNameKey: serviceAccount}
		}

		return secret
	}

	connect := func(who user.Info, subresource, pod string, options runtime.Object) testRequest {
		kind := corev1.SchemeGroupVersion.WithKind(map[string]string{"exec": "PodExecOptions", "attach": "PodAttachOptions", "portforward": "PodPortForwardOptions"}[subresource])
		return testRequest{user: who, operation: admission.Connect, resource: "pods", subresource: subresource, name: pod, kind: kind, object: options}
	}
	requestToken := func(who user.Info, serviceAccount string) testRequest {
		return testRequest{
			user: who, operation: admission.Create, resource: "serviceaccounts", subresource: "token", name: serviceAccount,
			kind: authenticationv1.SchemeGroupVersion.WithKind("TokenRequest"), object: &authenticationv1.TokenRequest{},
		}
	}
	createPod := func(who user.Info, pod *corev1.Pod) testRequest {
		return testRequest{user: who, operation: admission.Create, resource: "pods", name: pod.Name, kind: podKind, object: pod}
	}
	updatePod := func(who user.Info, subresource string, pod, oldPod *corev1.Pod) testRequest {
		return testRequest{user: who, operation: admission.Update, resource: "pods", subresource: subresource, name: pod.Name, kind: podKind, object: pod, oldObject: oldPod}
	}
	writeSecret := func(who user.Info, operation admission.Operation, secret *corev1.Secret) testRequest {
		request := testRequest{user: who, operation: operation, resource: "secrets", name: secret.Name, kind: secretKind, object: secret}
		if operation == admission.Update {
			request.oldObject = secret
		}

		return request
	}

	otherImage := func(pod *corev1.Pod) {
		pod.Spec.Containers[len(pod.Spec.Containers)-1].Image = "example.com/other:latest"
	}
	debugContainer := func(pod *corev1.Pod) {
		pod.Spec.EphemeralContainers = []corev1.EphemeralContainer{{EphemeralContainerCommon: corev1.EphemeralContainerCommon{Name: "debug", Image: "busybox"}}}
	}

	const (
		noSession  = "only KKP may open a session in a web terminal pod"
		noToken    = "only KKP may request tokens for the ServiceAccount of a web terminal session"
		noPod      = "only KKP may create pods that use the ServiceAccount of a web terminal session"
		noImage    = "only KKP may change the images of a web terminal pod"
		noSecret   = "only KKP may create token Secrets for the ServiceAccount of a web terminal session"
		isAdmitted = ""
	)

	testcases := []struct {
		name    string
		request testRequest
		refusal string
	}{
		// a session in the pod
		{"a user execs into the web terminal pod", connect(eve, "exec", session, &corev1.PodExecOptions{Command: []string{"bash"}}), noSession},
		{"a user attaches to the web terminal pod", connect(eve, "attach", session, &corev1.PodAttachOptions{}), noSession},
		{"a user forwards a port of the web terminal pod", connect(eve, "portforward", session, &corev1.PodPortForwardOptions{Ports: []int32{8001}}), noSession},
		{"the admin kubeconfig execs into the web terminal pod", connect(admin, "exec", session, &corev1.PodExecOptions{Command: []string{"bash"}}), noSession},
		{"a user adds a debug container to the web terminal pod", updatePod(eve, "ephemeralcontainers", sessionPod(debugContainer), sessionPod(nil)), noSession},
		{"KKP execs into the web terminal pod", connect(kkp, "exec", session, &corev1.PodExecOptions{Command: []string{"bash"}}), isAdmitted},
		{"a user execs into another pod", connect(eve, "exec", "coredns-5d78c9869d-abcde", &corev1.PodExecOptions{Command: []string{"sh"}}), isAdmitted},
		{"a user forwards a port of another pod", connect(eve, "portforward", "coredns-5d78c9869d-abcde", &corev1.PodPortForwardOptions{Ports: []int32{9153}}), isAdmitted},
		{"a user adds a debug container to another pod", updatePod(eve, "ephemeralcontainers", otherPod(debugContainer), otherPod(nil)), isAdmitted},

		// a token of the ServiceAccount
		{"a user requests a token of the session", requestToken(eve, session), noToken},
		{"the admin kubeconfig requests a token of the session", requestToken(admin, session), noToken},
		{"the kubelet requests the token the web terminal pod mounts", requestToken(kubelet, session), isAdmitted},
		{"KKP requests a token of the session", requestToken(kkp, session), isAdmitted},
		{"a user requests a token of another ServiceAccount", requestToken(eve, "default"), isAdmitted},
		{"a user creates a token Secret for the session", writeSecret(eve, admission.Create, tokenSecret(corev1.SecretTypeServiceAccountToken, session)), noSecret},
		{"a user turns a Secret into a token Secret for the session", writeSecret(eve, admission.Update, tokenSecret(corev1.SecretTypeServiceAccountToken, session)), noSecret},
		{"KKP creates a token Secret for the session", writeSecret(kkp, admission.Create, tokenSecret(corev1.SecretTypeServiceAccountToken, session)), isAdmitted},
		{"a user creates a token Secret for another ServiceAccount", writeSecret(eve, admission.Create, tokenSecret(corev1.SecretTypeServiceAccountToken, "default")), isAdmitted},
		{"a user creates a Secret without annotations", writeSecret(eve, admission.Create, tokenSecret(corev1.SecretTypeOpaque, "")), isAdmitted},
		// Kubernetes writes no token into a Secret of another type
		{"a user creates a Secret of another type that names the session", writeSecret(eve, admission.Create, tokenSecret(corev1.SecretTypeOpaque, session)), isAdmitted},

		// another program with the token
		{"a user creates a pod that runs as the session", createPod(eve, otherPod(func(pod *corev1.Pod) { pod.Spec.ServiceAccountName = session })), noPod},
		{"a controller creates a pod that runs as the session", createPod(controller, otherPod(func(pod *corev1.Pod) { pod.Spec.ServiceAccountName = session })), noPod},
		{"KKP creates the web terminal pod", createPod(kkp, sessionPod(nil)), isAdmitted},
		{"a user creates a pod that runs as another ServiceAccount", createPod(eve, otherPod(nil)), isAdmitted},
		{"a user creates a pod without a ServiceAccount", createPod(eve, otherPod(func(pod *corev1.Pod) { pod.Spec.ServiceAccountName = "" })), isAdmitted},
		{"a user changes an image of the web terminal pod", updatePod(eve, "", sessionPod(otherImage), sessionPod(nil)), noImage},
		{"the admin kubeconfig changes an image of the web terminal pod", updatePod(admin, "", sessionPod(otherImage), sessionPod(nil)), noImage},
		{"KKP changes an image of the web terminal pod", updatePod(kkp, "", sessionPod(otherImage), sessionPod(nil)), isAdmitted},
		{"a user labels the web terminal pod", updatePod(eve, "", sessionPod(func(pod *corev1.Pod) { pod.Labels["team"] = "a" }), sessionPod(nil)), isAdmitted},
		// deleting a pod in the foreground only works if the finalizer can be removed again
		{"the garbage collector removes a finalizer of the web terminal pod", updatePod(controller, "", sessionPod(nil), sessionPod(func(pod *corev1.Pod) { pod.Finalizers = []string{metav1.FinalizerDeleteDependents} })), isAdmitted},
		{"a user changes an image of another pod", updatePod(eve, "", otherPod(otherImage), otherPod(nil)), isAdmitted},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			matched, refusal := tc.request.evaluate(t, policies)

			if !matched {
				t.Error("no rule of the policies matches the request")
			}
			if refusal != tc.refusal {
				t.Errorf("expected the request to be refused with %q, got %q", tc.refusal, refusal)
			}
		})
	}
}

// TestPoliciesScope checks what the test above takes for granted: that the policies only match in
// kube-system, skip KKP, refuse what they can't evaluate and are enforced by their bindings.
func TestPoliciesScope(t *testing.T) {
	policies := desiredPolicies(t)
	if len(policies) == 0 {
		t.Fatal("expected policies")
	}

	names := []string{}
	for _, policy := range policies {
		names = append(names, policy.Name)

		selector := policy.Spec.MatchConstraints.NamespaceSelector
		if selector == nil || len(selector.MatchExpressions) != 0 || len(selector.MatchLabels) != 1 || selector.MatchLabels[corev1.LabelMetadataName] != metav1.NamespaceSystem {
			t.Errorf("policy %s: expected to match in kube-system only, got %+v", policy.Name, selector)
		}
		if policy.Spec.FailurePolicy == nil || *policy.Spec.FailurePolicy != admissionregistrationv1.Fail {
			t.Errorf("policy %s: expected failurePolicy Fail", policy.Name)
		}
		if len(policy.Spec.MatchConditions) != 1 || policy.Spec.MatchConditions[0].Expression != "request.userInfo.username != 'kubermatic-controllers'" {
			t.Errorf("policy %s: expected to skip the requests of KKP only, got %+v", policy.Name, policy.Spec.MatchConditions)
		}
		for _, validation := range policy.Spec.Validations {
			if validation.Message == "" || validation.Reason == nil || *validation.Reason != metav1.StatusReasonForbidden {
				t.Errorf("policy %s: expected a message and the reason Forbidden for %q", policy.Name, validation.Expression)
			}
		}
	}
	if !slices.Equal(names, PolicyNames()) {
		t.Errorf("expected policies %v, got %v", PolicyNames(), names)
	}

	bindings := desiredBindings(t)
	if len(bindings) != len(policies) {
		t.Fatalf("expected %d bindings, got %d", len(policies), len(bindings))
	}
	for i, binding := range bindings {
		if binding.Name != policies[i].Name || binding.Spec.PolicyName != policies[i].Name {
			t.Errorf("expected a binding named %s for the policy of that name, got %s for %s", policies[i].Name, binding.Name, binding.Spec.PolicyName)
		}
		if !slices.Equal(binding.Spec.ValidationActions, []admissionregistrationv1.ValidationAction{admissionregistrationv1.Deny}) {
			t.Errorf("binding %s: expected the action Deny, got %v", binding.Name, binding.Spec.ValidationActions)
		}
		if binding.Spec.MatchResources != nil {
			t.Errorf("binding %s: must not narrow the policy", binding.Name)
		}
	}
}

func TestResourcesForDeletion(t *testing.T) {
	objects := ResourcesForDeletion()
	names := PolicyNames()
	if len(objects) != 2*len(names) {
		t.Fatalf("expected %d objects, got %d", 2*len(names), len(objects))
	}

	// bindings first, so that nothing is enforced that is about to lose its policy
	for i, object := range objects {
		_, isBinding := object.(*admissionregistrationv1.ValidatingAdmissionPolicyBinding)
		if isBinding != (i < len(names)) {
			t.Errorf("object %d: expected the bindings before the policies, got %T", i, object)
		}
		if object.GetName() != names[i%len(names)] {
			t.Errorf("object %d: expected name %s, got %s", i, names[i%len(names)], object.GetName())
		}
	}
}
