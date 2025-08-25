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

package main

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
	"k8c.io/kubermatic/sdk/v2/semver"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"

	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/metrics"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/runner"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/scenarios"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/cli"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var banner = `
 ██████╗ ██████╗ ███╗   ██╗███████╗ ██████╗ ██████╗ ███╗   ███╗ █████╗ ███╗   ██╗ ██████╗███████╗    ████████╗███████╗███████╗████████╗███████╗██████╗ 
██╔════╝██╔═══██╗████╗  ██║██╔════╝██╔═══██╗██╔══██╗████╗ ████║██╔══██╗████╗  ██║██╔════╝██╔════╝    ╚══██╔══╝██╔════╝██╔════╝╚══██╔══╝██╔════╝██╔══██╗
██║     ██║   ██║██╔██╗ ██║█████╗  ██║   ██║██████╔╝██╔████╔██║███████║██╔██╗ ██║██║     █████╗         ██║   █████╗  ███████╗   ██║   █████╗  ██████╔╝
██║     ██║   ██║██║╚██╗██║██╔══╝  ██║   ██║██╔══██╗██║╚██╔╝██║██╔══██║██║╚██╗██║██║     ██╔══╝         ██║   ██╔══╝  ╚════██║   ██║   ██╔══╝  ██╔══██╗
╚██████╗╚██████╔╝██║ ╚████║██║     ╚██████╔╝██║  ██║██║ ╚═╝ ██║██║  ██║██║ ╚████║╚██████╗███████╗       ██║   ███████╗███████║   ██║   ███████╗██║  ██║
 ╚═════╝ ╚═════╝ ╚═╝  ╚═══╝╚═╝      ╚═════╝ ╚═╝  ╚═╝╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝ ╚═════╝╚══════╝       ╚═╝   ╚══════╝╚══════╝   ╚═╝   ╚══════╝╚═╝  ╚═╝
`

type step struct {
	key      string
	example  string
	defaultV string
}

type model struct {
	width, height int
	stage         int
	steps         []step
	index         int
	textInput     textinput.Model
	values        map[string]string
	errorMsg      string
}

var validators = map[string]func(string, step) error{
	"PROVIDERS":     validateInExample,
	"DISTRIBUTIONS": validateInExample,
	"RELEASES":      validateInExample,
	"RUNTIMES":      validateInExample,
	"UPDATE":        validateBool,
	"PARALLEL":      validateInt,
	"EXCLUDE_TESTS": validateInExample,
}

// Example-based validator
func validateInExample(val string, s step) error {
	if s.example == "" {
		return nil
	}
	allowed := strings.Split(s.example, ",")
	vals := strings.Split(val, ",")
	seen := make(map[string]int)
	var duplicates []string

	for _, v := range vals {
		v = strings.TrimSpace(v)
		seen[v]++
		if seen[v] == 2 {
			duplicates = append(duplicates, v)
		}
		found := false
		for _, a := range allowed {
			if v == strings.TrimSpace(a) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid value: %s (allowed: %s)", v, s.example)
		}
	}
	if len(duplicates) > 0 {
		return fmt.Errorf("duplicate value(s): %s", strings.Join(duplicates, ", "))
	}
	return nil
}

func validateInt(val string, s step) error {
	if _, err := fmt.Sscanf(val, "%d", new(int)); err != nil {
		return fmt.Errorf("must be a number")
	}
	return nil
}

func validateBool(val string, s step) error {
	v := strings.ToLower(val)
	if v != "true" && v != "false" {
		return fmt.Errorf("must be true or false")
	}
	return nil
}

func getDefaultHostname() string {
	b, _ := exec.Command("hostname").Output()
	return strings.ToLower(strings.TrimSpace(string(b)))
}

func defaultOrEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func initialSteps() []step {
	hostname := getDefaultHostname()

	return []step{
		{"PROVIDERS", "alibaba, anexia, aws, azure, digitalocean, gcp, hetzner, kubevirt, openstack, vsphere", defaultOrEnv("PROVIDERS", "")},
		{"DISTRIBUTIONS", "", defaultOrEnv("DISTRIBUTIONS", "ubuntu,flatcar,rhel,rockylinux")},
		{"RELEASES", "", defaultOrEnv("RELEASES", "1.30,1.31,1.32,1.33")},
		{"RUNTIMES", "", defaultOrEnv("RUNTIMES", "containerd")},
		{"UPDATE", "", defaultOrEnv("UPDATE", "false")},
		{"PARALLEL", "", defaultOrEnv("PARALLEL", "2")},
		{"NAME_PREFIX", "", defaultOrEnv("NAME_PREFIX", hostname)},
		{"SEED", "", defaultOrEnv("SEED", "shared")},
		{"PRESET", "", defaultOrEnv("PRESET", "dev")},
		{"PROJECT", "", defaultOrEnv("PROJECT", "qs2f84cd67")},
		{"EXCLUDE_TESTS", "", defaultOrEnv("EXCLUDE_TESTS", "conformance,telemetry")},
	}
}

func initialModel() model {
	ti := textinput.New()
	ti.Focus()
	return model{
		steps:     initialSteps(),
		textInput: ti,
		values:    make(map[string]string),
		stage:     0,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.stage {
		case 0:
			if msg.String() == "enter" {
				m.stage = 1
				cur := m.steps[0]
				m.textInput.Placeholder = cur.defaultV
				m.textInput.SetValue("")
				m.textInput.Focus()
			} else if msg.String() == "q" || msg.String() == "esc" {
				return m, tea.Quit
			}

		case 1:
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)

			if msg.Type == tea.KeyEnter {
				val := m.textInput.Value()
				if val == "" {
					val = m.steps[m.index].defaultV
				}
				// Validate only if not empty
				if val != "" {
					if validator, ok := validators[m.steps[m.index].key]; ok {
						if err := validator(val, m.steps[m.index]); err != nil {
							m.textInput.SetValue("")
							m.textInput.Placeholder = m.steps[m.index].defaultV
							m.errorMsg = err.Error()
							return m, textinput.Blink
						}
						m.errorMsg = ""
					}
				}
				m.values[m.steps[m.index].key] = val
				m.index++
				m.textInput.SetValue("")
				if m.index >= len(m.steps) {
					m.stage = 2
				}
			}

			return m, cmd

		case 2:
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder

	// Top separator
	sep := termenv.String(strings.Repeat("=", m.width)).Foreground(termenv.ANSIBlue).String() + "\n"
	b.WriteString(sep)

	// Trim the banner if it’s too tall for the terminal
	bannerLines := strings.Split(strings.Trim(banner, "\n"), "\n")
	maxLines := m.height - 15
	if maxLines > 0 && len(bannerLines) > maxLines {
		bannerLines = bannerLines[:maxLines]
	}

	for _, line := range bannerLines {
		trimmedLine := strings.TrimRight(line, " ")
		maxBannerWidth := m.width - 4
		visibleWidth := termenv.String(trimmedLine).Width()

		// Truncate by runes if too wide
		runes := []rune(trimmedLine)
		if visibleWidth > maxBannerWidth && maxBannerWidth > 0 {
			trimmedLine = string(runes[:maxBannerWidth])
			visibleWidth = termenv.String(trimmedLine).Width()
		}

		padTotal := maxBannerWidth - visibleWidth
		padLeft := padTotal / 2
		padRight := padTotal - padLeft
		if padLeft < 0 {
			padLeft = 0
			padRight = 0
		}

		colored := termenv.String("||" + strings.Repeat(" ", padLeft) + trimmedLine + strings.Repeat(" ", padRight) + "||").
			Foreground(termenv.ANSIBlue)
		b.WriteString(colored.String() + "\n")
	}

	b.WriteString("\n")
	b.WriteString(termenv.String(centerWithBars("Conformance Tester", m.width)).Foreground(termenv.ANSIBlue).String() + "\n")
	b.WriteString(termenv.String(centerWithBars("By Kubermatic · For Kubermatic", m.width)).Foreground(termenv.ANSIBlue).String() + "\n")
	// Bottom separator
	b.WriteString(sep)

	// Now the rest of the UI below the banner + texts
	switch m.stage {
	case 0:
		b.WriteString(center("Press Enter to begin setup", m.width) + "\n")
		b.WriteString(center("Press Q or Esc to quit", m.width))

	case 1:
		cur := m.steps[m.index]
		label := fmt.Sprintf("Enter %s", cur.key)
		if cur.example != "" {
			label += fmt.Sprintf(" (%s)", cur.example)
		}
		label += fmt.Sprintf(" [default: %s]:", cur.defaultV)

		b.WriteString(label + "\n")
		b.WriteString(m.textInput.View() + "\n")
		if m.errorMsg != "" {
			b.WriteString(termenv.String(m.errorMsg).Foreground(termenv.ANSIRed).String() + "\n")
		}

	case 2:
		b.WriteString("Configuration complete:\n\n")
		for _, s := range m.steps {
			b.WriteString(fmt.Sprintf("%-15s = %s\n", s.key, m.values[s.key]))
		}
		b.WriteString("\nStarting conformance tester...\n")
	}

	return b.String()
}

func centerWithBars(s string, w int) string {
	visibleWidth := termenv.String(s).Width()
	maxBannerWidth := w - 4
	padTotal := maxBannerWidth - visibleWidth
	padLeft := padTotal / 2
	padRight := padTotal - padLeft
	if padLeft < 0 {
		padLeft = 0
		padRight = 0
	}
	return "||" + strings.Repeat(" ", padLeft) + s + strings.Repeat(" ", padRight) + "||"
}

func center(s string, w int) string {
	p := (w - len(s)) / 2
	if p < 0 {
		p = 0
	}
	return strings.Repeat(" ", p) + s
}

func rightAlign(s string, w int) string {
	p := w - len(s)
	if p < 0 {
		p = 0
	}
	return strings.Repeat(" ", p) + s
}

func populateOptionsFromValues(values map[string]string, opts *types.Options, log *zap.SugaredLogger) {
	for key, val := range values {
		log.Infow("Processing option", "key", key, "value", val)

		switch key {
		case "PROVIDERS":
			providers := splitCSV(val)
			log.Infow("Setting providers", "providers", providers)
			opts.Providers = sets.New[string](providers...)

		case "DISTRIBUTIONS":
			distros := splitCSV(val)
			log.Infow("Setting distributions", "distributions", distros)
			opts.Distributions = sets.New[string](distros...)

		case "RELEASES":
			releases := splitCSV(val)
			log.Infow("Setting releases", "releases", releases)
			opts.Releases = sets.New[string](releases...)

		case "EXCLUDE_TESTS":
			exclude := splitCSV(val)
			log.Infow("Setting exclude tests", "excludeTests", exclude)
			opts.ExcludeTests = sets.New[string](exclude...)

		case "UPDATE":
			log.Infow("Setting test cluster update", "value", val)
			opts.TestClusterUpdate = strings.ToLower(val) == "true"

		case "SEED":
			log.Infow("Setting seed", "value", val)
			opts.KubermaticSeedName = val

		case "NAME_PREFIX":
			log.Infow("Setting name prefix", "value", val)
			opts.NamePrefix = val

		case "PROJECT":
			log.Infow("Setting project", "value", val)
			opts.KubermaticProject = val

		case "PARALLEL":
			if n, err := strconv.Atoi(val); err == nil {
				log.Infow("Setting parallel count", "value", n)
				opts.ClusterParallelCount = n
			}
		}

		opts.Versions = []*semver.Semver{
			semver.NewSemverOrDie("1.30.5"),
		}
	}
}

func populateOptionsFromSteps(steps []step, opts *types.Options, log *zap.SugaredLogger) {
	for _, s := range steps {
		log.Infow("Processing step", "key", s.key, "value", s.defaultV)
		switch s.key {
		case "PROVIDERS":
			providers := splitCSV(s.defaultV)
			log.Debugw("Setting providers", "providers", providers)
			opts.Providers = sets.New[string](providers...)
		case "DISTRIBUTIONS":
			distros := splitCSV(s.defaultV)
			log.Debugw("Setting distributions", "distributions", distros)
			opts.Distributions = sets.New[string](distros...)
		case "RELEASES":
			releases := splitCSV(s.defaultV)
			log.Debugw("Setting releases", "releases", releases)
			opts.Releases = sets.New[string](releases...)
		case "EXCLUDE_TESTS":
			excludeTests := splitCSV(s.defaultV)
			log.Debugw("Setting exclude tests", "excludeTests", excludeTests)
			opts.ExcludeTests = sets.New[string](excludeTests...)
		case "UPDATE":
			log.Debugw("Setting test cluster update", "value", s.defaultV)
			opts.TestClusterUpdate = s.defaultV == "true"
		case "SEED":
			log.Debugw("Setting seed", "value", s.defaultV)
			opts.KubermaticSeedName = s.defaultV
		case "NAME_PREFIX":
			log.Debugw("Setting name prefix", "value", s.defaultV)
			opts.NamePrefix = s.defaultV
		//case "PRESET":
		//	opts.Preset = s.defaultV
		case "PROJECT":
			log.Debugw("Setting project", "value", s.defaultV)
			opts.KubermaticProject = s.defaultV
		case "PARALLEL":
			log.Debugw("Setting parallel count", "value", s.defaultV)
			if n, err := strconv.Atoi(s.defaultV); err == nil {
				opts.ClusterParallelCount = n
			}
		}
	}
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	rootCtx := signals.SetupSignalHandler()

	// setup flags
	opts := types.NewDefaultOptions()
	opts.AddFlags()
	opts.Secrets = types.Secrets{
		AWS: struct {
			KKPDatacenter   string
			AccessKeyID     string
			SecretAccessKey string
		}{
			KKPDatacenter:   "aws-eu-central-1a",
			AccessKeyID:     "accesskey",
			SecretAccessKey: "secretskey",
		},
	}

	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)

	flag.Parse()

	// setup logging
	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	// parse our CLI flags
	//if err := opts.ParseFlags(log); err != nil {
	//	log.Fatalw("Invalid flags", zap.Error(err))
	//}

	if mm, ok := m.(model); ok {
		populateOptionsFromValues(mm.values, opts, log)
	}
	log.Infow("Options after population",
		"providers", opts.Providers,
		"distributions", opts.Distributions,
		"releases", opts.Releases,
		"versions", opts.Versions,
	)

	reconciling.Configure(log)

	// collect runtime metrics if there is a pushgateway URL configured
	// and these variables are set
	metrics.Setup(opts.PushgatewayEndpoint, os.Getenv("JOB_NAME"), os.Getenv("PROW_JOB_ID"))
	defer metrics.UpdateMetrics(log)

	// say hello
	cli.Hello(log, "Conformance Tests", nil)
	log.Infow("Runner configuration",
		"providers", sets.List(opts.Providers),
		"operatingsystems", sets.List(opts.Distributions),
		"versions", opts.Versions,
		"tests", sets.List(opts.Tests),
		"dualstack", opts.DualStackEnabled,
		"konnectivity", opts.KonnectivityEnabled,
		"updates", opts.TestClusterUpdate,
	)

	// setup kube client, ctrl-runtime client, clientgetter, seedgetter etc.
	if err := setupKubeClients(rootCtx, opts); err != nil {
		log.Fatalw("Failed to setup kube clients", zap.Error(err))
	}

	// create a temporary home directory and a fresh SSH key
	homeDir, dynamicSSHPublicKey, err := setupHomeDir(log)
	if err != nil {
		log.Fatalw("Failed to setup temporary home dir", zap.Error(err))
	}
	opts.PublicKeys = append(opts.PublicKeys, dynamicSSHPublicKey)
	opts.HomeDir = homeDir

	// setup runner and KKP clients
	log.Info("Preparing project...")
	testRunner := runner.NewKubeRunner(opts, log)
	if err := testRunner.Setup(rootCtx); err != nil {
		log.Fatalw("Failed to setup runner", zap.Error(err))
	}

	log.Infow("Generating test scenarios with parameters",
		"providers", opts.Providers,
		"distributions", opts.Distributions,
		"dualstack", opts.DualStackEnabled,
		"versions", opts.Versions,
	)

	// determine what's to do
	scenarios, err := scenarios.NewGenerator().
		WithCloudProviders(sets.List(opts.Providers)...).
		WithOperatingSystems(sets.List(opts.Distributions)...).
		WithDualstack(opts.DualStackEnabled).
		WithVersions(opts.Versions...).
		Scenarios(rootCtx, opts, log)
	if err != nil {
		log.Fatalw("Failed to determine test scenarios", zap.Error(err))
	}

	if len(scenarios) == 0 {
		// Fatalw() because Fatal() trips up the linter because of the previous defer.
		log.Fatalw("No scenarios match the given criteria.")
	}

	// optionally restrict the full set of scenarios to those that previously did not succeed
	var previousResults *runner.ResultsFile
	if opts.RetryFailedScenarios {
		previousResults, err = loadPreviousResults(opts)
		if err != nil {
			log.Fatalw("Failed to load previous test results", zap.Error(err))
		}

		scenarios = keepOnlyFailedScenarios(log, scenarios, previousResults, *opts)
	}

	if err := testRunner.SetupProject(rootCtx); err != nil {
		log.Fatalw("Failed to setup project", zap.Error(err))
	}

	log.Infow("Using project", "project", opts.KubermaticProject)

	// let the magic happen!
	log.Info("Running E2E tests...")
	start := time.Now()

	results, err := testRunner.Run(rootCtx, scenarios)

	// always print the test results
	if results != nil {
		results.PrintJUnitDetails()
		results.PrintSummary()

		if filename := opts.ResultsFile; filename != "" {
			log.Infow("Writing results file", "filename", filename)

			// Merge the previous tests with the new, current results; otherwise if we'd only
			// dump the new results, those would not contain skipped/successful scenarios from
			// the previous run, effectively shrinking the results file every time it is used.
			if previousResults != nil {
				results = runner.MergeResults(previousResults, results)
			}

			if err := results.WriteToFile(filename); err != nil {
				log.Warnw("Failed to write results file", zap.Error(err))
			}
		}
	}

	if err != nil {
		log.Fatalw("Failed to execute tests", zap.Error(err))
	}

	if results.HasFailures() {
		// Fatalw() because Fatal() trips up the linter because of the previous defer.
		log.Fatalw("Not all tests have passed")
	}

	log.Infow("Test suite has completed successfully", "runtime", time.Since(start))
}

func setupKubeClients(ctx context.Context, opts *types.Options) error {
	_, config, err := utils.GetClients()
	if err != nil {
		return fmt.Errorf("failed to get client config: %w", err)
	}
	opts.SeedRestConfig = config

	if err := clusterv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		return fmt.Errorf("failed to add clusterv1alpha1 to scheme: %w", err)
	}

	if err := metricsv1beta1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		return fmt.Errorf("failed to add metrics v1beta1 to scheme: %w", err)
	}

	seedClusterClient, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		return err
	}
	opts.SeedClusterClient = seedClusterClient

	seedGeneratedClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	opts.SeedGeneratedClient = seedGeneratedClient

	seedGetter, err := kubernetesprovider.SeedGetterFactory(ctx, seedClusterClient, opts.KubermaticSeedName, opts.KubermaticNamespace)
	if err != nil {
		return fmt.Errorf("failed to construct seedGetter: %w", err)
	}
	opts.Seed, err = seedGetter()
	if err != nil {
		return fmt.Errorf("failed to get seed: %w", err)
	}

	configGetter, err := kubernetesprovider.DynamicKubermaticConfigurationGetterFactory(opts.SeedClusterClient, opts.KubermaticNamespace)
	if err != nil {
		return fmt.Errorf("failed to construct configGetter: %w", err)
	}

	opts.KubermaticConfiguration, err = configGetter(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Kubermatic config: %w", err)
	}

	clusterClientProvider, err := clusterclient.NewExternalWithProxy(seedClusterClient, opts.Seed.GetManagementProxyURL())
	if err != nil {
		return fmt.Errorf("failed to get clusterClientProvider: %w", err)
	}
	opts.ClusterClientProvider = clusterClientProvider

	return nil
}

// setupHomeDir set up a temporary home dir (because the e2e tests have some filenames hardcoded,
// which might conflict with the user files).
func setupHomeDir(log *zap.SugaredLogger) (string, []byte, error) {
	// We'll set the env-var $HOME to this directory when executing the tests
	homeDir, err := os.MkdirTemp("/tmp", "e2e-home-")
	if err != nil {
		return "", nil, fmt.Errorf("failed to setup temporary home dir: %w", err)
	}
	log.Infof("Setting up temporary home directory with SSH keys at %s...", homeDir)

	if err := os.MkdirAll(path.Join(homeDir, ".ssh"), os.ModePerm); err != nil {
		return "", nil, err
	}

	// Setup temporary home dir with filepath.Join(os.Getenv("HOME"), ".ssh")
	// Make sure to create relevant ssh keys (because they are hardcoded in the e2e tests...). They must not be password protected
	log.Debug("Generating SSH keys...")
	// Private Key generation
	privateKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return "", nil, err
	}

	// Validate Private Key
	err = privateKey.Validate()
	if err != nil {
		return "", nil, err
	}

	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	privatePEM := pem.EncodeToMemory(&privBlock)
	// Needs to be google_compute_engine as its hardcoded in the kubernetes e2e tests
	if err := os.WriteFile(path.Join(homeDir, ".ssh", "google_compute_engine"), privatePEM, 0400); err != nil {
		return "", nil, err
	}

	publicRsaKey, err := ssh.NewPublicKey(privateKey.Public())
	if err != nil {
		return "", nil, err
	}

	pubKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)
	if err := os.WriteFile(path.Join(homeDir, ".ssh", "google_compute_engine.pub"), pubKeyBytes, 0400); err != nil {
		return "", nil, err
	}

	log.Infof("Finished setting up temporary home dir %s", homeDir)
	return homeDir, pubKeyBytes, nil
}

func loadPreviousResults(opts *types.Options) (*runner.ResultsFile, error) {
	if opts.ResultsFile == "" {
		return nil, nil
	}

	// non-existing or empty files are okay
	stat, err := os.Stat(opts.ResultsFile)
	if err != nil || stat.Size() == 0 {
		return nil, nil
	}

	return runner.LoadResultsFile(opts.ResultsFile)
}

func keepOnlyFailedScenarios(log *zap.SugaredLogger, allScenarios []scenarios.Scenario, previousResults *runner.ResultsFile, opts types.Options) []scenarios.Scenario {
	if optionsChanged(previousResults.Configuration, opts) {
		log.Warn("Disregarding previous test results as current options do not match previous options.")
		return allScenarios
	}

	filtered := []scenarios.Scenario{}
	for i, scenario := range allScenarios {
		hasSuccess := false

		for _, previous := range previousResults.Results {
			if previous.MatchesScenario(scenario) && previous.Status == runner.ScenarioPassed {
				hasSuccess = true
				break
			}
		}

		if hasSuccess {
			scenario.Log(log).Info("Skipping because scenario succeeded in a previous run.")
			continue
		}

		filtered = append(filtered, allScenarios[i])
	}

	return filtered
}

func optionsChanged(previous runner.TestConfiguration, current types.Options) bool {
	return false ||
		previous.KonnectivityEnabled != current.KonnectivityEnabled ||
		previous.DualstackEnabled != current.DualStackEnabled ||
		previous.TestClusterUpdate != current.TestClusterUpdate ||
		!sets.New(previous.Tests...).IsSuperset(current.Tests)
}
