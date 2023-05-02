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
	"bufio"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
)

var kkpEndpointTemplate = "%v.nip.io"

func QuickStartCommand(logger *logrus.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quickstart [environment]",
		Short: "Initialize environment for simplified quickstart KKP installation",
		Long:  "Prepares minimal Kubernetes environment (e.g. kind) and auto-configures a non-production KKP installation for evaluation and development purpose.",
	}

	cmd.AddCommand(quickStartKindCommand(logger))
	// TODO: expose when ready
	cmd.Hidden = true
	return cmd
}

func quickStartKindCommand(logger *logrus.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kind",
		Short: "Initialize kind environment for simplified quickstart KKP installation",
		Long:  "Prepares minimal kind environment and auto-configures a non-production KKP installation for evaluation and development purpose.",
		PreRun: func(cmd *cobra.Command, args []string) {
			_, err := exec.LookPath("kind")
			if err != nil {
				logger.Fatalf("failed to find 'kind' binary", err)
			}
		},
		RunE: quickStartKindFunc(logger),
	}
	return cmd
}

func quickStartKind(logger *logrus.Logger, dir string) (client.Client, context.CancelFunc) {
	kindConfig := filepath.Join(dir, "kind-config.yaml")
	if err := os.WriteFile(kindConfig, []byte(kindConfigContent), 0600); err != nil {
		logger.Fatalf("failed to create 'kind' config: %v", err)
	}
	logger.Info("Creating kind cluster ...") // TODO: prettify
	// TODO: make this idempotent
	out, err := exec.Command("kind", "create", "cluster", "-n", "kkp-cluster", "--config", kindConfig).CombinedOutput()
	if err != nil {
		logger.Fatalf("failed to create 'kind' cluster: %v\n%v", err, string(out))
	}

	logger.Info("Kind cluster ready, continuing configuration ...") // TODO: prettify
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

	flag.Set("kubeconfig", kindKubeConfigPath)
	ctrlConfig, err := ctrlruntimeconfig.GetConfig()
	if err != nil {
		logger.Fatalf("failed to initialize runtime config: %w", err)
	}

	mgr, err := manager.New(ctrlConfig, manager.Options{
		MetricsBindAddress:     "0",
		HealthProbeBindAddress: "0",
	})
	if err != nil {
		logger.Fatalf("failed to construct mgr: %w", err)
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

func writeFileFromURL(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("Failed to get example file %v: %v", url, err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("Failed to get example file %v: status code %v", url, resp.StatusCode)
	}
	defer resp.Body.Close()
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("Failed to create file %v: %v", filepath, err)
	}
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	if _, err := io.Copy(writer, resp.Body); err != nil {
		return fmt.Errorf("Failed to write example file %v: %v", filepath, err)
	}
	return nil
}

func ensureResource(kubeClient client.Client, logger *logrus.Logger, o client.Object) {
	if err := kubeClient.Create(context.Background(), o); err != nil && !errors.IsAlreadyExists(err) {
		logger.Fatal(err)
	}
}

func installKubevirt(logger *logrus.Logger, dir string, kubeClient client.Client) {
	logger.Info("Installing KubeVirt ...")
	// TODO: package this whole thing in KKP charts
	ko := "https://github.com/kubevirt/kubevirt/releases/download/v1.0.0-alpha.0/kubevirt-operator.yaml"
	kcr := "https://github.com/kubevirt/kubevirt/releases/download/v1.0.0-alpha.0/kubevirt-cr.yaml"
	cdio := "https://github.com/kubevirt/containerized-data-importer/releases/download/v1.56.0/cdi-operator.yaml"
	cdicr := "https://github.com/kubevirt/containerized-data-importer/releases/download/v1.56.0/cdi-cr.yaml"
	out, err := exec.Command("kubectl", "apply", "-f", ko).CombinedOutput()
	if err != nil {
		logger.Fatalf("Failed to install KubeVirt: %v\n%v", err, string(out))
	}
	out, err = exec.Command("kubectl", "apply", "-f", kcr).CombinedOutput()
	if err != nil {
		logger.Fatalf("Failed to install KubeVirt CR: %v\n%v", err, string(out))
	}
	out, err = exec.Command("kubectl", "apply", "-f", cdio).CombinedOutput()
	if err != nil {
		logger.Fatalf("Failed to install KubeVirt CDI: %v\n%v", err, string(out))
	}
	out, err = exec.Command("kubectl", "apply", "-f", cdicr).CombinedOutput()
	if err != nil {
		logger.Fatalf("Failed to install KubeVirt CDI CR: %v\n%v", err, string(out))
	}
}

func installKubermatic(logger *logrus.Logger, dir string, kubeClient client.Client) string {
	ip := getLocalIP(logger)
	kkpEndpoint := fmt.Sprintf(kkpEndpointTemplate, ip)
	logger.Infof("Installing KKP at %v ...", ip) // TODO: prettify
	valuesExamplePath := filepath.Join(dir, "values.example.yaml")
	kubermaticExamplePath := filepath.Join(dir, "kubermatic.example.yaml")
	valuesPath := filepath.Join(dir, "values.yaml")
	kubermaticPath := filepath.Join(dir, "kubermatic.yaml")

	_, uk, err := loadKubermaticConfiguration(kubermaticExamplePath)
	if err != nil {
		logger.Fatalf("failed to unmarshal example kubermatic.yaml: %v", err)
	}
	// TODO: make this idempotent (e.g. if generated yamls already exist, don't touch them)
	unstructured.SetNestedField(uk.Object, kkpEndpoint, "spec", "ingress", "domain")
	unstructured.SetNestedField(uk.Object, nil, "spec", "ingress", "certificateIssuer")
	unstructured.SetNestedField(uk.Object, fmt.Sprintf("http://%v/dex", kkpEndpoint), "spec", "auth", "tokenIssuer")
	unstructured.SetNestedField(uk.Object, randomString(32), "spec", "auth", "issuerClientSecret")
	unstructured.SetNestedField(uk.Object, randomString(32), "spec", "auth", "issuerCookieKey")
	unstructured.SetNestedField(uk.Object, randomString(32), "spec", "auth", "serviceAccountKey")
	// TODO: backport https://github.com/kubermatic/dashboard/pull/5867 or wait for new release?
	unstructured.SetNestedField(uk.Object, "wozniakjan/dashboard", "spec", "ui", "dockerRepository")
	unstructured.SetNestedField(uk.Object, "latest", "spec", "ui", "dockerTag")
	kout, err := yaml.Marshal(uk.Object)
	if err != nil {
		logger.Fatalf("failed to marshal kubermatic.yaml: %v", err)
	}
	if err := os.WriteFile(kubermaticPath, kout, 0600); err != nil {
		logger.Fatalf("failed to create kubermatic.yaml: %v", err)
	}

	vc, err := os.ReadFile(valuesExamplePath)
	if err != nil {
		logger.Fatalf("failed to read values.yaml example: %v", err)
	}
	uv := make(map[any]any)
	if err := yaml.Unmarshal(vc, uv); err != nil {
		logger.Fatalf("failed to decode example values.yaml: %v", err)
	}

	setNestedField(uv, "http", "dex", "ingress", "scheme")
	setNestedField(uv, kkpEndpoint, "dex", "ingress", "host")
	clients := uv["dex"].(map[any]any)["clients"].([]any)
	for i, _ := range clients {
		clientsMap := clients[i].(map[any]any)
		setNestedField(clientsMap, randomString(32), "secret")
		uris := clientsMap["RedirectURIs"].([]any)
		for uri, _ := range uris {
			u, err := url.Parse(uris[uri].(string))
			if err != nil {
				logger.Fatalf("failed to modify values.yaml: %v", err)
			}
			u.Scheme = "http"
			u.Host = kkpEndpoint
			uris[uri] = u.String()
		}
	}

	setNestedField(uv, uuid.NewString(), "telemetry", "uuid")
	setNestedField(uv, nil, "minio")
	vout, err := yaml.Marshal(uv)
	if err != nil {
		logger.Fatalf("failed to marshal values.yaml: %v", err)
	}
	if err := os.WriteFile(valuesPath, vout, 0600); err != nil {
		logger.Fatalf("failed to create values.yaml: %v", err)
	}

	ensureResource(kubeClient, logger, &kindIngressControllerNamespace)
	ensureResource(kubeClient, logger, &kindKubermaticNamespace)
	ensureResource(kubeClient, logger, &kindStorageClass)
	ensureResource(kubeClient, logger, &kindIngressControllerService)
	ensureResource(kubeClient, logger, &kindNodeportProxyService)

	// TODO: use cmd_deploy.go instead
	kubeconfig := filepath.Join(dir, "kube-config.yaml")
	executable, err := os.Executable()
	if err != nil {
		logger.Fatal(err)
	}
	cmd := exec.Command(executable, "deploy", "--config", kubermaticPath, "--helm-values", valuesPath, "--kubeconfig", kubeconfig)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Fatal(err)
	}

	internalKubeconfig := filepath.Join(dir, "kube-config-internal.yaml")
	kindSeedSecret := initKindSeedSecret(kubeClient, logger, kubeconfig, internalKubeconfig)
	ensureResource(kubeClient, logger, &kindSeedSecret)
	kindPreset := initKindPreset(logger, internalKubeconfig)
	ensureResource(kubeClient, logger, &kindPreset)
	ensureResource(kubeClient, logger, &kindLocalSeed)
	return kkpEndpoint
}

func quickStartKindFunc(logger *logrus.Logger) cobraFuncE {
	return handleErrors(logger, func(cmd *cobra.Command, args []string) error {
		dir := "./examples"
		path, err := os.Executable()
		if err == nil {
			dir = filepath.Join(filepath.Dir(path), "examples")
		}
		kubeClient, cancel := quickStartKind(logger, dir)
		defer cancel()
		installKubevirt(logger, dir, kubeClient)
		endpoint := installKubermatic(logger, dir, kubeClient)
		logger.Infof("KKP installed successfully, login at http://%v.", endpoint)
		return nil
	})
}

func getLocalIP(logger *logrus.Logger) string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logger.Fatalf("unable to get interfaces: %v", err)
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	logger.Fatalf("no interface with public IP")
	return ""
}

func setNestedField(obj map[any]any, value any, fields ...string) error {
	m := obj
	for _, field := range fields[:len(fields)-1] {
		if val, ok := m[field]; ok {
			if valMap, ok := val.(map[any]any); ok {
				m = valMap
			} else {
				return fmt.Errorf("value cannot be set because %v is not a map[any]any: %T %v", field, val, val)
			}
		} else {
			newVal := make(map[any]any)
			m[field] = newVal
			m = newVal
		}
	}
	m[fields[len(fields)-1]] = value
	return nil
}

func initKindSeedSecret(kubeClient client.Client, logger *logrus.Logger, kubeconfigPath, internalKubeconfigPath string) corev1.Secret {
	cpPod := corev1.Pod{}
	key := client.ObjectKey{Namespace: "kube-system", Name: "kube-apiserver-kkp-cluster-control-plane"}
	if err := kubeClient.Get(context.Background(), key, &cpPod); err != nil {
		logger.Fatalf("Failed to get IP for kind control-plane pod: %v", err)
	}
	ip := cpPod.Status.PodIP
	k, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		logger.Fatalf("Failed to initialize seed secret: %v", err)
	}
	addrRe := regexp.MustCompile(`([ ]*server:) https://127.0.0.1:[0-9]*`)
	internalKubeconfig := addrRe.ReplaceAllString(string(k), fmt.Sprintf(`$1 https://%v:6443`, ip))

	if err := os.WriteFile(internalKubeconfigPath, []byte(internalKubeconfig), 0600); err != nil {
		logger.Fatalf("failed to write internal kubeconfig: %v", err)
	}
	kindKubeconfigSeedSecret.StringData["kubeconfig"] = internalKubeconfig
	return kindKubeconfigSeedSecret
}

func initKindPreset(logger *logrus.Logger, internalKubeconfigPath string) kubermaticv1.Preset {
	k, err := os.ReadFile(internalKubeconfigPath)
	if err != nil {
		logger.Fatalf("Failed to initialize preset: %v", err)
	}
	kindLocalPreset.Spec.Kubevirt.Kubeconfig = base64.StdEncoding.EncodeToString(k)
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
