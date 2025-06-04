/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-logr/zapr"
	"github.com/google/uuid"
	"github.com/jackpal/gateway"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	kubermaticmaster "k8c.io/kubermatic/v2/pkg/install/stack/kubermatic-master"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/util/yamled"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

const (
	nip                  = "nip.io"
	sslip                = "sslip.io"
	kkpDefaultLogin      = "kubermatic@example.com"
	kkpDefaultPassword   = "password"
	localKindTeardownCmd = "kind delete cluster -n kkp-cluster"
)

var (
	minSupportedKindVersion = semver.NewSemverOrDie("0.17.0")
)

type LocalOptions struct {
	Options

	HelmBinary     string
	HelmTimeout    time.Duration
	Endpoint       string
	KubeOVNEnabled bool
}

func LocalCommand(logger *logrus.Logger) *cobra.Command {
	opt := LocalOptions{
		HelmTimeout: 5 * time.Minute,
		HelmBinary:  "helm",
		Options: Options{
			ChartsDirectory: "./charts",
		},
	}
	cmd := &cobra.Command{
		Use:   "local [environment]",
		Short: "Initialize environment for simplified local KKP installation",
		Long:  "Prepares minimal Kubernetes environment (e.g. kind) and auto-configures a non-production KKP installation for evaluation and development purpose.",
		PreRun: func(cmd *cobra.Command, args []string) {
			options.CopyInto(&opt.Options)
			if opt.HelmBinary == "" {
				opt.HelmBinary = os.Getenv("HELM_BINARY")
			}
		},
	}
	cmd.PersistentFlags().DurationVar(&opt.HelmTimeout, "helm-timeout", opt.HelmTimeout, "time to wait for Helm operations to finish")
	cmd.PersistentFlags().StringVar(&opt.HelmBinary, "helm-binary", opt.HelmBinary, "full path to the Helm 3 binary to use")

	cmd.AddCommand(localKindCommand(logger, opt))
	// TODO: expose when ready
	cmd.Hidden = true
	return cmd
}

func localKindCommand(logger *logrus.Logger, opt LocalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kind",
		Short: "Initialize kind environment for simplified local KKP installation",
		Long:  "Prepares minimal kind environment and auto-configures a non-production KKP installation for evaluation and development purpose.",
		PreRun: func(cmd *cobra.Command, args []string) {
			_, err := exec.LookPath("kind")
			if err != nil {
				logger.Fatalf("failed to find 'kind' binary: %v", err)
			}
			out, err := exec.Command("kind", "version").CombinedOutput()
			if err != nil {
				logger.Fatalf("failed to determine 'kind' version, requires at least %v: %v\n%v", minSupportedKindVersion, err, string(out))
			}
			submatch := regexp.MustCompile(`.* v([^ ]*) .*`).FindStringSubmatch(string(out))
			if len(submatch) != 2 {
				logger.Fatalf("failed to parse 'kind' version, requires at least %v: %v", minSupportedKindVersion, string(out))
			}
			kindVersion, err := semver.NewSemver(submatch[1])
			if err != nil {
				logger.Fatalf("failed to process 'kind' semver %q, requires at least %v: %v", submatch[1], minSupportedKindVersion, string(out))
			}
			if kindVersion.LessThan(minSupportedKindVersion) {
				logger.Fatalf("please update your 'kind' %v, requires at least %v", kindVersion, minSupportedKindVersion)
			}

			_, err = exec.LookPath("helm")
			if err != nil {
				logger.Fatalf("failed to find 'helm' binary: %v", err)
			}
		},
		RunE: localKindFunc(logger, &opt),
	}
	cmd.PersistentFlags().StringVar(&opt.Endpoint, "endpoint", "", "endpoint address for KKP installation (e.g. 10.0.0.5.nip.io), if this flag is left empty, the installer does best effort in auto-configuring from available network interfaces")
	cmd.PersistentFlags().BoolVar(&opt.KubeOVNEnabled, "kube-ovn-enabled", false, "enables usage of kube-ovn instead of kindnet as the cni plugin")
	return cmd
}

func localKind(logger *logrus.Logger, dir string, opt *LocalOptions) (ctrlruntimeclient.Client, context.CancelFunc) {
	kindConfig := filepath.Join(dir, "kind-config.yaml")
	configContent := kindConfigContent
	if opt.KubeOVNEnabled {
		configContent += kindConfigKubeOVNContent
		logger.Info("Disabling kindnet to deploy kube-ovn cni plugin")
	}
	if err := os.WriteFile(kindConfig, []byte(configContent), 0600); err != nil {
		logger.Fatalf("failed to create 'kind' config: %v", err)
	}

	logger.Info("Creating kind cluster…")
	// TODO: make this idempotent
	out, err := exec.Command("kind", "create", "cluster", "-n", "kkp-cluster", "--config", kindConfig).CombinedOutput()
	if err != nil {
		logger.Fatalf("failed to create 'kind' cluster: %v\n%v", err, string(out))
	}

	logger.Info("Kind cluster ready, continuing configuration…")
	kubeconfigCmd := exec.Command("kubectl", "config", "view", "--minify", "--flatten")
	kindKubeConfigPath := filepath.Join(dir, "kube-config.yaml")
	kindKubeConfig, err := os.Create(kindKubeConfigPath)
	if err != nil {
		logger.Fatalf("failed to create 'kind' cluster kubeconfig: %v", err)
	}
	kubeconfigCmd.Stdout = kindKubeConfig
	if err = kubeconfigCmd.Run(); err != nil {
		logger.Fatalf("failed to write 'kind' cluster kubeconfig: %v", err)
	}
	if err := kindKubeConfig.Close(); err != nil {
		logger.Fatalf("failed to close 'kind' cluster kubeconfig: %v", err)
	}

	if err := flag.Set("kubeconfig", kindKubeConfigPath); err != nil {
		logger.Fatalf("failed to close set kubeconfig path: %v", err)
	}
	ctrlConfig, err := ctrlruntimeconfig.GetConfig()
	if err != nil {
		logger.Fatalf("failed to initialize runtime config: %v", err)
	}

	ctrlruntimelog.SetLogger(zapr.NewLogger(zap.NewNop()))
	mgr, err := manager.New(ctrlConfig, manager.Options{
		Metrics:                metricsserver.Options{BindAddress: "0"},
		HealthProbeBindAddress: "0",
	})
	if err != nil {
		logger.Fatalf("failed to construct mgr: %v", err)
	}

	// start the manager in its own goroutine
	appContext := context.Background()

	go func() {
		if err := mgr.Start(appContext); err != nil {
			logger.Fatalf("Failed to start Kubernetes client manager: %v", err)
		}
	}()

	// wait for caches to be synced
	mgrSyncCtx, cancel := context.WithTimeout(appContext, 30*time.Second)
	defer cancel()
	if synced := mgr.GetCache().WaitForCacheSync(mgrSyncCtx); !synced {
		logger.Fatal("Timed out while waiting for Kubernetes client caches to synchronize.")
	}
	return mgr.GetClient(), cancel
}

func ensureResource(kubeClient ctrlruntimeclient.Client, logger *logrus.Logger, o ctrlruntimeclient.Object) {
	if err := kubeClient.Create(context.Background(), o); err != nil && !apierrors.IsAlreadyExists(err) {
		logger.Fatal(err)
	}
}

func installKubevirt(logger *logrus.Logger, dir string, helmClient helm.Client, opt LocalOptions) {
	logger.Info("Installing KubeVirt…")
	err := helmClient.InstallChart("kubevirt", "kubevirt", filepath.Join(opt.ChartsDirectory, "local-kubevirt"), "", nil, []string{"--create-namespace"})
	if err != nil {
		logger.Fatalf("Failed to install KubeVirt Helm client: %v", err)
	}
}

func installKubeOVN(logger *logrus.Logger, helmClient helm.Client, opt LocalOptions) {
	logger.Info("Installing KubeOVN...")
	if err := helmClient.BuildChartDependencies(filepath.Join(opt.ChartsDirectory, "local-kube-ovn"), nil); err != nil {
		logger.Fatalf("Failed to fetch KubeOVN Helm chart: %v", err)
	}
	err := helmClient.InstallChart("kube-system", "kube-ovn", filepath.Join(opt.ChartsDirectory, "local-kube-ovn"), "", nil, []string{"--wait"})
	if err != nil {
		logger.Fatalf("Failed to install KubeOVN Helm release: %v", err)
	}
}

func prepareYAMLFile(dir, basename string, modifier func(*yamled.Document) error) (string, error) {
	inputFile := filepath.Join(dir, fmt.Sprintf("%s.example.yaml", basename))
	outputFile := filepath.Join(dir, fmt.Sprintf("%s.yaml", basename))

	f, err := os.Open(inputFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	doc, err := yamled.Load(f)
	if err != nil {
		return "", err
	}

	if err := modifier(doc); err != nil {
		return "", err
	}

	data, err := doc.MarshalYAML()
	if err != nil {
		return "", err
	}

	out, err := os.Create(outputFile)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if err := yaml.NewEncoder(out).Encode(data); err != nil {
		return "", err
	}

	return outputFile, nil
}

func prepareKubermaticConfiguration(dir, kkpEndpoint string) (string, error) {
	return prepareYAMLFile(dir, "kubermatic", func(doc *yamled.Document) error {
		doc.Set(yamled.Path{"spec", "ingress", "domain"}, kkpEndpoint)
		doc.Remove(yamled.Path{"spec", "ingress", "certificateIssuer"})
		doc.Set(yamled.Path{"spec", "auth", "tokenIssuer"}, fmt.Sprintf("http://%v/dex", kkpEndpoint))
		doc.Set(yamled.Path{"spec", "auth", "issuerClientSecret"}, randomString(32))
		doc.Set(yamled.Path{"spec", "auth", "issuerCookieKey"}, randomString(32))
		doc.Set(yamled.Path{"spec", "auth", "serviceAccountKey"}, randomString(32))

		return nil
	})
}

func prepareHelmValues(dir, kkpEndpoint string) (string, error) {
	return prepareYAMLFile(dir, "values", func(doc *yamled.Document) error {
		doc.Set(yamled.Path{"dex", "ingress", "scheme"}, "http")
		doc.Set(yamled.Path{"dex", "ingress", "host"}, kkpEndpoint)
		doc.Set(yamled.Path{"telemetry", "uuid"}, uuid.NewString())
		doc.Remove(yamled.Path{"minio"})

		clients, ok := doc.GetArray(yamled.Path{"dex", "clients"})
		if !ok {
			return errors.New("expected to find Dex clients, but got none")
		}

		for i := range clients {
			doc.Set(yamled.Path{"dex", "clients", i, "secret"}, randomString(32))

			redirectURIs, _ := doc.GetArray(yamled.Path{"dex", "clients", i, "RedirectURIs"})
			for j, redirectURI := range redirectURIs {
				if stringURI, ok := redirectURI.(string); ok {
					u, err := url.Parse(stringURI)
					if err != nil {
						return fmt.Errorf("failed to parse %q as URL: %w", stringURI, err)
					}

					u.Scheme = "http"
					u.Host = kkpEndpoint

					doc.Set(yamled.Path{"dex", "clients", i, "RedirectURIs", j}, u.String())
				}
			}
		}

		return nil
	})
}

func installKubermatic(logger *logrus.Logger, dir string, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opts LocalOptions) string {
	kkpEndpoint := getLocalEndpoint(logger, opts)
	logger.Infof("Installing KKP at %v…", kkpEndpoint) // TODO: prettify

	kubermaticPath, err := prepareKubermaticConfiguration(dir, kkpEndpoint)
	if err != nil {
		logger.Fatalf("failed to prepare Kubermatic configuration: %v", err)
	}

	valuesPath, err := prepareHelmValues(dir, kkpEndpoint)
	if err != nil {
		logger.Fatalf("failed to prepare Helm values: %v", err)
	}

	ensureResource(kubeClient, logger, &kindIngressControllerNamespace)
	ensureResource(kubeClient, logger, &kindKubermaticNamespace)
	ensureResource(kubeClient, logger, &kindStorageClass)
	ensureResource(kubeClient, logger, &kindIngressControllerService)
	ensureResource(kubeClient, logger, &kindNodeportProxyService)

	ms := kubermaticmaster.NewStack(false)
	k, uk, err := loadKubermaticConfiguration(kubermaticPath)
	if err != nil {
		logger.Panicf("Failed to load %v after autoconfiguration: %v", kubermaticPath, err)
	}
	v, err := loadHelmValues(valuesPath)
	if err != nil {
		logger.Panicf("Failed to load %v after autoconfiguration: %v", valuesPath, err)
	}
	msOpts := stack.DeployOptions{
		ChartsDirectory:            opts.ChartsDirectory,
		KubeClient:                 kubeClient,
		HelmClient:                 helmClient,
		Logger:                     log.Prefix(logrus.NewEntry(logger), "   "),
		MLASkipMinio:               true,
		MLASkipMinioLifecycleMgr:   true,
		KubermaticConfiguration:    k,
		RawKubermaticConfiguration: uk,
		HelmValues:                 v,
	}

	if err := ms.Deploy(context.Background(), msOpts); err != nil {
		logger.Fatalf("Failed to deploy KKP stack: %v", err)
	}

	kubeconfig := filepath.Join(dir, "kube-config.yaml")
	internalKubeconfig := filepath.Join(dir, "kube-config-internal.yaml")
	kindSeedSecret := initKindSeedSecret(kubeClient, logger, kubeconfig, internalKubeconfig)
	ensureResource(kubeClient, logger, &kindSeedSecret)
	kindPreset := initKindPreset(logger, internalKubeconfig, opts.KubeOVNEnabled)
	ensureResource(kubeClient, logger, &kindPreset)
	ensureResource(kubeClient, logger, &kindLocalSeed)
	return kkpEndpoint
}

func localKindFunc(logger *logrus.Logger, opt *LocalOptions) cobraFuncE {
	return handleErrors(logger, func(cmd *cobra.Command, args []string) error {
		exampleDir := "./examples"

		// make path relative to installer binary
		if path, err := os.Executable(); err == nil {
			exampleDir = filepath.Join(filepath.Dir(path), "examples")
		}

		if _, err := os.Stat(exampleDir); err != nil {
			logger.Fatal("Failed to find examples directory, please ensure it and the charts directory from the KKP download archive remain together with the kubermatic-installer.")
		}

		kubeClient, cancel := localKind(logger, exampleDir, opt)
		defer cancel()

		kubeconfig := filepath.Join(exampleDir, "kube-config.yaml")
		helmClient, err := helm.NewCLI(opt.HelmBinary, kubeconfig, "", opt.HelmTimeout, logger)
		if err != nil {
			logger.Fatalf("Failed to create helm client: %v", err)
		}
		helmVersion, err := helmClient.Version()
		if err != nil {
			logger.Fatalf("Failed to check Helm version: %v", err)
		}
		if helmVersion.LessThan(MinHelmVersion) {
			logger.Fatalf(
				"the installer requires Helm >= %s, but detected %q as %s (use --helm-binary or $HELM_BINARY to override)",
				MinHelmVersion,
				opt.HelmBinary,
				helmVersion,
			)
		}
		if opt.KubeOVNEnabled {
			installKubeOVN(logger, helmClient, *opt)
		}
		// installCertManager(logger, exampleDir, helmClient, *opt)
		installKubevirt(logger, exampleDir, helmClient, *opt)
		endpoint := installKubermatic(logger, exampleDir, kubeClient, helmClient, *opt)
		logger.Infoln()
		logger.Infof("KKP installed successfully, login at http://%v", endpoint)
		logger.Infof("  Default login:    %v", kkpDefaultLogin)
		logger.Infof("  Default password: %v\n", kkpDefaultPassword)
		logger.Infof("You can tear down the environment by %q", localKindTeardownCmd)
		return nil
	})
}

func getLocalEndpoint(logger *logrus.Logger, opts LocalOptions) string {
	if opts.Endpoint != "" {
		if ip := net.ParseIP(opts.Endpoint); ip != nil {
			return ipToNip(ip)
		}
		return opts.Endpoint
	}
	if gwip, err := gateway.DiscoverGateway(); err != nil {
		logger.Fatalf("Failed to determine default gateway IP, please use --endpoint flag: %v", err)
	} else {
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			logger.Fatalf("Failed to list interface addresses, please use --endpoint flag: %v", err)
		}
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.Contains(gwip) {
				return ipToNip(ipnet.IP)
			}
		}
	}
	logger.Fatalf("Failed to determine local IP, please use --endpoint flag")
	return ""
}

func ipToNip(ip net.IP) string {
	if ip.To4() != nil {
		return fmt.Sprintf("%v.%v", ip.String(), nip)
	}
	// nip.io is much faster and more reliable but doesn't support IPv6
	processedIP := strings.ReplaceAll(ip.String(), ":", "-")
	return fmt.Sprintf("%v.%v", processedIP, sslip)
}

func initKindSeedSecret(kubeClient ctrlruntimeclient.Client, logger *logrus.Logger, kubeconfigPath, internalKubeconfigPath string) corev1.Secret {
	cpPod := corev1.Pod{}
	key := ctrlruntimeclient.ObjectKey{Namespace: "kube-system", Name: "kube-apiserver-kkp-cluster-control-plane"}
	if err := kubeClient.Get(context.Background(), key, &cpPod); err != nil {
		logger.Fatalf("Failed to get IP for kind control-plane pod: %v", err)
	}
	ip := cpPod.Status.PodIP
	k, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		logger.Fatalf("Failed to initialize seed secret: %v", err)
	}
	addrRe := regexp.MustCompile(`([ ]*server:) https://127.0.0.1:[0-9]*`)
	internalKubeconfig := addrRe.ReplaceAllString(string(k), fmt.Sprintf(`$1 https://%s`, net.JoinHostPort(ip, "6443")))

	if err := os.WriteFile(internalKubeconfigPath, []byte(internalKubeconfig), 0600); err != nil {
		logger.Fatalf("failed to write internal kubeconfig: %v", err)
	}
	kindKubeconfigSeedSecret.StringData["kubeconfig"] = internalKubeconfig
	return kindKubeconfigSeedSecret
}

func initKindPreset(logger *logrus.Logger, internalKubeconfigPath string, kubeOVNEnabled bool) kubermaticv1.Preset {
	k, err := os.ReadFile(internalKubeconfigPath)
	if err != nil {
		logger.Fatalf("Failed to initialize preset: %v", err)
	}
	kindLocalPreset.Spec.Kubevirt.Kubeconfig = base64.StdEncoding.EncodeToString(k)
	if kubeOVNEnabled {
		kindLocalPreset.Spec.Kubevirt.VPCName = "ovn-cluster"
	}
	return kindLocalPreset
}

func randomString(n int) string {
	var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0987654321")
	str := make([]rune, n)
	for i := range str {
		str[i] = chars[rand.Intn(len(chars))]
	}
	return string(str)
}
