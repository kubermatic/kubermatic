package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"
)

func testConformanceParallel(ctx *TestContext, t *testing.T) {
	testConformance(ctx, t, ctx.nodeCount*7, "parallel", `\[Conformance\]`, `\[Serial\]`)
}

func testConformanceSerial(ctx *TestContext, t *testing.T) {
	// Broken on aws. See https://github.com/kubernetes/kubernetes/pull/72965
	skip := `should not cause race condition when used for configmap`
	testConformance(ctx, t, 1, "serial", `\[Serial\].*\[Conformance\]`, skip)
}

func testConformance(ctx *TestContext, t *testing.T, workerCount int, name, ginkgoFocus, ginkgoSkip string) {
	MajorMinor := fmt.Sprintf("%d.%d", ctx.cluster.Spec.Version.Major(), ctx.cluster.Spec.Version.Minor())
	kubernetesDir := path.Join(ctx.testBinRoot, MajorMinor)
	binDir := path.Join(kubernetesDir, "/platforms/linux/amd64")

	args := []string{
		"-progress",
		"-randomizeAllSpecs=false",
		"-randomizeSuites=false",
		fmt.Sprintf("-nodes=%d", workerCount),
		"-noColor=true",
		// We'll use 5 flake attempts to avoid restarting Ginkgo
		"-flakeAttempts=5",
		fmt.Sprintf(`-focus=%s`, ginkgoFocus),
		fmt.Sprintf(`-skip=%s`, ginkgoSkip),
		path.Join(binDir, "e2e.test"),
		"--",
		"--disable-log-dump",
		fmt.Sprintf("--repo-root=%s", kubernetesDir),
		fmt.Sprintf("--report-dir=%s", ctx.workingDir),
		fmt.Sprintf("--report-prefix=%s", name+"_"),
		fmt.Sprintf("--kubectl-path=%s", path.Join(binDir, "kubectl")),
		fmt.Sprintf("--kubeconfig=%s", ctx.clusterContext.kubeconfig),
		fmt.Sprintf("--num-nodes=%d", ctx.nodeCount),
		"--provider=local",
	}

	cmdCtx, cancel := context.WithTimeout(ctx.ctx, 30*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, path.Join(binDir, "ginkgo"), args...)

	// Pipe stdout and stderr to a file to avoid chewing up all memory
	logFile := path.Join(ctx.workingDir, fmt.Sprintf("ginkgo_%s.log", name))
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		t.Fatalf("failed to create ginkgo logfile: %v", err)
	}
	defer f.Close()
	defer f.Sync()

	cmd.Stdout = f
	cmd.Stderr = f

	// Convenience function to flush the logs to the file more often. Enables watching the ginkgo logs
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	go func() {
		<-ticker.C
		f.Sync()
	}()

	if err := cmd.Run(); err != nil {
		t.Errorf("failed to execute Ginkgo: %v\nMore details in the ginkgo log %s", err, f.Name())
	}

	// Now parse all JUnit reports and create test cases from them. That way we can pass through potential test failures
	analyzeReport(name, ctx, t)
}
