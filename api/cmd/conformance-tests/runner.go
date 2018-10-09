package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"
	"text/template"
	"time"

	"github.com/Masterminds/semver"
	"github.com/golang/glog"
	"github.com/onsi/ginkgo/reporters"

	kubermaticapiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
	clusterclient "github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machine"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type testScenario interface {
	Name() string
	Cluster(secrets secrets) *kubermaticv1.Cluster
	Nodes(num int) []*kubermaticapiv2.Node
}

func newRunner(scenarios []testScenario, opts *Opts) *testRunner {
	return &testRunner{
		scenarios:                    scenarios,
		clusterLister:                opts.clusterLister,
		controlPlaneReadyWaitTimeout: opts.controlPlaneReadyWaitTimeout,
		deleteClusterAfterTests:      opts.deleteClusterAfterTests,
		kubermaticClient:             opts.kubermaticClient,
		secrets:                      opts.secrets,
		namePrefix:                   opts.namePrefix,
		clusterClientProvider:        opts.clusterClientProvider,
		nodesReadyWaitTimeout:        opts.nodeReadyWaitTimeout,
		dcs:                          opts.dcs,
		nodeCount:                    opts.nodeCount,
		testBinPath:                  opts.testBinPath,
		reportsRoot:                  opts.reportsRoot,
		clusterParallelCount:         opts.clusterParallelCount,
		nodeSSHKeyData:               opts.nodeSSHKeyData,
	}
}

type testRunner struct {
	scenarios      []testScenario
	secrets        secrets
	namePrefix     string
	testBinPath    string
	reportsRoot    string
	nodeSSHKeyData []byte

	controlPlaneReadyWaitTimeout time.Duration
	deleteClusterAfterTests      bool
	nodesReadyWaitTimeout        time.Duration
	nodeCount                    int
	clusterParallelCount         int

	kubermaticClient      kubermaticclientset.Interface
	clusterLister         kubermaticv1lister.ClusterLister
	clusterClientProvider *clusterclient.Provider
	dcs                   map[string]provider.DatacenterMeta
}

type testResult struct {
	report   *reporters.JUnitTestSuite
	err      error
	scenario testScenario
}

func (t *testResult) Passed() bool {
	if t.err != nil {
		return false
	}

	if t.report == nil {
		return false
	}

	if len(t.report.TestCases) == 0 {
		return false
	}

	if t.report.Errors > 0 || t.report.Failures > 0 {
		return false
	}

	return true
}

func (r *testRunner) worker(id int, scenarios <-chan testScenario, results chan<- testResult) {
	for s := range scenarios {
		r.logInfo(s, "Starting...")
		report, err := r.testScenario(s)
		res := testResult{
			report:   report,
			scenario: s,
			err:      err,
		}
		if err != nil {
			r.logInfo(s, "Finished with error: %v", err)
		} else {
			r.logInfo(s, "Finished")
		}

		results <- res
	}
}

func (r *testRunner) Run() error {
	scenariosCh := make(chan testScenario, len(r.scenarios))
	resultsCh := make(chan testResult, len(r.scenarios))

	glog.V(2).Infoln("Test suite:")
	for _, scenario := range r.scenarios {
		glog.V(2).Infoln(scenario.Name())
		scenariosCh <- scenario
	}
	glog.V(2).Infoln(fmt.Sprintf("Total: %d tests", len(r.scenarios)))

	for i := 1; i <= r.clusterParallelCount; i++ {
		go r.worker(i, scenariosCh, resultsCh)
	}

	close(scenariosCh)

	var results []testResult
	for range r.scenarios {
		results = append(results, <-resultsCh)
		glog.V(2).Infof("Finished %d/%d test cases", len(results), len(r.scenarios))
	}

	overallResultBuf := &bytes.Buffer{}
	hadFailure := false
	for _, result := range results {
		prefix := "PASS"
		if !result.Passed() {
			prefix = "FAIL"
			hadFailure = true
		}
		scenarioResultMsg := fmt.Sprintf("[%s] - %s", prefix, result.scenario.Name())
		if result.err != nil {
			scenarioResultMsg = fmt.Sprintf("%s : %v", scenarioResultMsg, result.err)
		}

		fmt.Fprintln(overallResultBuf, scenarioResultMsg)
		if result.report != nil {
			printDetailedReport(result.report)
		}
	}

	fmt.Println("========================== RESULT ===========================")
	fmt.Println(overallResultBuf.String())

	if hadFailure {
		return errors.New("some tests failed")
	}
	return nil
}

func (r *testRunner) testScenario(scenario testScenario) (*reporters.JUnitTestSuite, error) {
	// Always generate a random name
	cluster := scenario.Cluster(r.secrets)
	cluster.Name = rand.String(8)
	if r.namePrefix != "" {
		cluster.Name = fmt.Sprintf("%s-%s", r.namePrefix, cluster.Name)
	}

	dc, found := r.dcs[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("invalid cloud datacenter specified '%s'. Not found in datacenters list", cluster.Spec.Cloud.DatacenterName)
	}

	r.logInfo(scenario, "Creating cluster: %s...", cluster.Name)
	cluster, err := r.kubermaticClient.KubermaticV1().Clusters().Create(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to create the cluster: %v", err)
	}
	if r.deleteClusterAfterTests {
		defer r.tryToDeleteCluster(cluster.Name, scenario)
	}
	r.logInfo(scenario, "Successfully created cluster!")

	r.logInfo(scenario, "Waiting for control plane to become ready...")
	cluster, err = r.waitForControlPlane(scenario, cluster.Name)
	if err != nil {
		return nil, fmt.Errorf("failed waiting for control plane to become ready: %v", err)
	}
	r.logInfo(scenario, "Control plane became ready!")

	r.logInfo(scenario, "Getting kubeconfig...")
	kubeconfig, err := r.clusterClientProvider.GetAdminKubeconfig(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig for cluster: %v", err)
	}
	filename := fmt.Sprintf("/tmp/%s-kubeconfig", cluster.Name)
	if err := ioutil.WriteFile(filename, kubeconfig, 0644); err != nil {
		return nil, fmt.Errorf("failed to write kubeconfig to %s: %v", filename, err)
	}
	r.logInfo(scenario, "Wrote kubeconfig to %s", filename)

	r.logInfo(scenario, "Creating machines...")
	kubeMachineClient, err := r.clusterClientProvider.GetMachineClient(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get the machine client for the cluster: %v", err)
	}

	apiNodes := scenario.Nodes(r.nodeCount)
	for i, node := range apiNodes {
		var keys []*kubermaticv1.UserSSHKey
		if r.nodeSSHKeyData != nil {
			keys = append(keys, &kubermaticv1.UserSSHKey{
				Spec: kubermaticv1.SSHKeySpec{
					PublicKey: string(r.nodeSSHKeyData),
				},
			})
		}
		m, err := machine.Machine(cluster, node, dc, keys)
		if err != nil {
			return nil, fmt.Errorf("failed to create Machine from scenario node '%s': %v", node.Metadata.Name, err)
		}
		// Make sure all nodes have different names across all scenarios - otherwise the Kubelet might come up (OpenStack has this...)
		m.Name = fmt.Sprintf("%s-machine-%d", scenario.Name(), i)
		m.Spec.Name = fmt.Sprintf("%s-node-%d", scenario.Name(), i)

		if _, err := kubeMachineClient.ClusterV1alpha1().Machines(metav1.NamespaceSystem).Create(m); err != nil {
			return nil, fmt.Errorf("failed to create machine %s: %v", m.Name, err)
		}
	}
	r.logInfo(scenario, "Successfully created %d machine(s)!", len(apiNodes))

	clusterKubeClient, err := r.clusterClientProvider.GetClient(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get the client for the cluster: %v", err)
	}

	r.logInfo(scenario, "Waiting for nodes to become ready...")
	if err := r.waitForReadyNodes(scenario, clusterKubeClient, len(apiNodes)); err != nil {
		return nil, fmt.Errorf("failed waiting for nodes to become ready: %v", err)
	}
	r.logInfo(scenario, "All nodes became ready!")

	var report *reporters.JUnitTestSuite
	for attempt := 1; attempt <= 3; attempt++ {
		startedE2E := time.Now()
		reportsDir := path.Join(r.reportsRoot, scenario.Name(), fmt.Sprintf("attempt-%d", attempt))

		r.logInfo(scenario, "[Attempt: %d] Starting E2E tests...", attempt)
		if err := r.runE2E(scenario, cluster, filename, reportsDir); err != nil {
			r.logInfo(scenario, "[Attempt: %d] failed to run E2E tests: %v", attempt, err)
			continue
		}
		r.logInfo(scenario, "[Attempt: %d] E2E tests finished after %.2f seconds", attempt, time.Since(startedE2E).Seconds())

		report, err = collectReports(scenario.Name(), reportsDir, time.Since(startedE2E))
		if err != nil {
			r.logInfo(scenario, "[Attempt: %d] failed to combine reports: %v. Restarting...", attempt, err)
			continue
		}

		if report.Errors > 0 || report.Failures > 0 {
			r.logInfo(scenario, "[Attempt: %d] e2e tests had some failures (See %s for details). Restarting...", attempt, reportsDir)
			continue
		}
		return report, nil
	}

	if report == nil {
		return nil, errors.New("no report generated")
	}
	return report, nil
}

func (r *testRunner) logInfo(scenario testScenario, msg string, args ...interface{}) {
	scenarioName := scenario.Name()
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	glog.Infof("[%s] %s", scenarioName, msg)
}

func (r *testRunner) tryToDeleteCluster(clusterName string, scenario testScenario) {
	if err := r.kubermaticClient.KubermaticV1().Clusters().Delete(clusterName, &metav1.DeleteOptions{}); err != nil {
		r.logInfo(scenario, "Failed to delete cluster %s: %v. Make sure to delete it manually afterwards", clusterName, err)
	}
}

func (r *testRunner) waitForReadyNodes(scenario testScenario, client kubernetes.Interface, num int) error {
	started := time.Now()

	nodeIsReady := func(n corev1.Node) bool {
		for _, condition := range n.Status.Conditions {
			if condition.Type == corev1.NodeReady {
				if condition.Status == corev1.ConditionTrue {
					return true
				}
			}
		}
		return false
	}

	err := wait.Poll(nodesReadyPollPeriod, r.nodesReadyWaitTimeout, func() (done bool, err error) {
		nodeList, err := client.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to list nodes: %v", err)
		}

		if len(nodeList.Items) != num {
			return false, nil
		}

		for _, node := range nodeList.Items {
			if !nodeIsReady(node) {
				return false, nil
			}
		}

		return true, nil
	})

	if err != nil {
		return err
	}

	r.logInfo(scenario, "Nodes got ready after %.2f seconds", time.Since(started).Seconds())
	return nil
}

func (r *testRunner) waitForControlPlane(scenario testScenario, clusterName string) (*kubermaticv1.Cluster, error) {
	started := time.Now()

	err := wait.Poll(controlPlaneReadyPollPeriod, r.controlPlaneReadyWaitTimeout, func() (done bool, err error) {
		cluster, err := r.clusterLister.Get(clusterName)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, fmt.Errorf("failed to get the cluster %s from the lister: %v", clusterName, err)
		}

		return cluster.Status.Health.AllHealthy() && cluster.Status.Health.OpenVPN, nil
	})
	// Timeout or other error
	if err != nil {
		return nil, err
	}

	// Get copy of latest version
	cluster, err := r.clusterLister.Get(clusterName)
	if err != nil {
		return nil, err
	}

	r.logInfo(scenario, "Control plane got ready after %.2f seconds", time.Since(started).Seconds())
	return cluster.DeepCopy(), nil
}

func (r *testRunner) runE2E(scenario testScenario, cluster *kubermaticv1.Cluster, kubeconfigFilename, reportsDir string) error {
	kubeconfigFilename = path.Clean(kubeconfigFilename)
	testBinDir := path.Clean(r.testBinPath)

	version, err := semver.NewVersion(cluster.Spec.Version)
	if err != nil {
		return fmt.Errorf("invalid cluster version '%s': %v", cluster.Spec.Version, err)
	}

	data := struct {
		// Only main + minor
		Version            string
		TestBinDir         string
		KubeconfigFilename string
		ReportsDir         string
	}{
		Version:            fmt.Sprintf("%d.%d", version.Major(), version.Minor()),
		TestBinDir:         testBinDir,
		KubeconfigFilename: kubeconfigFilename,
		ReportsDir:         reportsDir,
	}

	tpl, err := template.New("script").Parse(testScriptTpl)
	if err != nil {
		return fmt.Errorf("failed to parse test script template: %v", err)
	}

	scriptBuffer := &bytes.Buffer{}
	err = tpl.Execute(scriptBuffer, data)
	if err != nil {
		return fmt.Errorf("failed to execute test script template: %v", err)
	}

	script := scriptBuffer.String()
	cmd := exec.Command("bash", "-c", script)

	logFile := path.Join(reportsDir, "e2e.log")
	out, err := cmd.CombinedOutput()
	defer func() {
		if err := ioutil.WriteFile(logFile, out, 0644); err != nil {
			r.logInfo(scenario, "failed to write e2e logs: %v", err)
		}
	}()
	if err != nil {
		r.logInfo(scenario, script)
		return fmt.Errorf("failed to run ginkgo script (logs: %s): %v", logFile, err)
	}

	return nil
}

const (
	testScriptTpl = `#!/usr/bin/env bash
{{ .TestBinDir }}/{{ .Version }}/ginkgo \
    -progress \
    -nodes=10 \
    -noColor=true \
    -flakeAttempts=10 \
    -focus="\[Conformance\]" \
    -skip="\[Serial\]" \
    {{ .TestBinDir }}/{{ .Version }}/e2e.test -- \
    --disable-log-dump \
    --repo-root={{ .TestBinDir }}/{{ .Version }}/ \
    --provider="local" \
    --report-dir="{{ .ReportsDir }}" \
    --report-prefix="parallel" \
    --kubectl-path={{ .TestBinDir }}/{{ .Version }}/kubectl \
    --kubeconfig="{{ .KubeconfigFilename }}"

{{ .TestBinDir }}/{{ .Version }}/ginkgo \
    -progress \
    -nodes=1 \
    -noColor=true \
    -flakeAttempts=10 \
    -focus="\[Serial\].*\[Conformance\]" \
    {{ .TestBinDir }}/{{ .Version }}/e2e.test -- \
    --disable-log-dump \
    --repo-root={{ .TestBinDir }}/{{ .Version }}/ \
    --provider="local" \
    --report-dir="{{ .ReportsDir }}" \
    --report-prefix="serial" \
    --kubectl-path={{ .TestBinDir }}/{{ .Version }}/kubectl \
    --kubeconfig="{{ .KubeconfigFilename }}"
`
)
