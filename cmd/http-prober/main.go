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

package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

	httpproberapi "k8c.io/kubermatic/v2/cmd/http-prober/api"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

type multiValFlag []string

func (mvf *multiValFlag) Set(value string) error {
	*mvf = append(*mvf, value)
	return nil
}

func (mvf *multiValFlag) String() string {
	return fmt.Sprintf("%v", *mvf)
}

func main() {
	var (
		endpoint      string
		insecure      bool
		retries       int
		retryWait     int
		timeout       int
		crdsToWaitFor multiValFlag
		commandRaw    string
	)
	flag.StringVar(&endpoint, "endpoint", "", "The endpoint which should be waited for")
	flag.BoolVar(&insecure, "insecure", false, "Disable certificate validation")
	flag.IntVar(&retries, "retries", 10, "Number of retries")
	flag.IntVar(&retryWait, "retry-wait", 1, "Wait interval in seconds between retries")
	flag.IntVar(&timeout, "timeout", 30, "Timeout in seconds")
	flag.Var(&crdsToWaitFor, "crd-to-wait-for", "Wait for these crds to exist. Must contain kind and apiVersion comma separated, e.G `machines,cluster.k8s.io/v1alpha1`. Can be passed multiple times. Requires path to valid kubeconfig to work which can be passed via `PROBER_KUBECONFIG` or `KUBECONFIG` env var. If env var are missing then it will try to load in-cluster config.")
	flag.StringVar(&commandRaw, "command", "", "If passed, the http prober will exec this command. Must be json encoded")
	flag.Parse()

	log := kubermaticlog.Logger.Named("http-prober")

	crdCheckers, err := crdCheckersFactory(crdsToWaitFor)
	if err != nil {
		log.Fatal(err.Error())
	}

	var command *httpproberapi.Command
	if commandRaw != "" {
		command = &httpproberapi.Command{}
		if err := json.Unmarshal([]byte(commandRaw), command); err != nil {
			log.Fatalw("Failed to deserialize command", zap.Error(err))
		}
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure,
		},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   time.Duration(timeout) * time.Second,
	}

	e, err := url.Parse(endpoint)
	if err != nil {
		log.Fatalw("Invalid endpoint specified", zap.Error(err))
	}

	ctx := signals.SetupSignalHandler()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.String(), nil)
	if err != nil {
		log.Fatalw("Failed to build request", zap.Error(err))
	}

	trace := &httptrace.ClientTrace{
		GotConn: func(connInfo httptrace.GotConnInfo) {
			log.Infow("Hostname resolved", "hostname", e.Hostname(), "address", connInfo.Conn.RemoteAddr())
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	for i := 1; i <= retries; i++ {
		if i > 1 {
			time.Sleep(time.Duration(retryWait) * time.Second)
		}

		log.Infow("Probing", "attempt", i, "max-attempts", retries, "target", req.URL.String())
		resp, err := client.Do(req)
		if err != nil {
			log.Infow("Request failed", zap.Error(err))
			continue
		}
		resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			log.Infow("Response did not have a 2xx status code", "statuscode", resp.StatusCode)
			continue
		}

		log.Info("Endpoint is available")

		if err := executeCheckers(crdCheckers); err != nil {
			log.Infow("Check if crd is available was not successful", zap.Error(err))
			continue
		}
		if len(crdCheckers) > 0 {
			log.Info("All CRDs became available")
		}

		if command != nil {
			commandFullPath, err := exec.LookPath(command.Command)
			if err != nil {
				log.Fatalf("failed to look up full path for command %q: %v", command.Command, err)
			}
			// First arg should be the filename of the command being executed, quote from execve(2):
			// `By convention, the first of these strings (i.e., argv[0]) should contain the filename associated with the file being executed`
			args := append([]string{command.Command}, command.Args...)
			if err := syscall.Exec(commandFullPath, args, os.Environ()); err != nil {
				log.Fatalf("failed to execute command: %v", err)
			}
		}
		os.Exit(0)
	}

	log.Fatal("Failed: Reached retry limit!")
}

func crdCheckersFactory(mvf multiValFlag) ([]func() error, error) {
	if len(mvf) == 0 {
		return nil, nil
	}

	cfg, err := getConfig()
	if err != nil {
		return nil, errors.New("--crd-to-wait-for was set but couldn't load a valid kubeconfig.")
	}

	var checkers []func() error
	for _, val := range mvf {
		checker, err := crdCheckerFromFlag(val, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to construct crd checker: %w", err)
		}
		checkers = append(checkers, checker)
	}

	return checkers, nil
}

func crdCheckerFromFlag(flag string, cfg *rest.Config) (func() error, error) {
	splitVal := strings.Split(flag, ",")
	if n := len(splitVal); n != 2 {
		return nil, fmt.Errorf("comma-separating the flag value did not yield exactly two results, but %d", n)
	}
	kind := splitVal[0]
	apiVersion := splitVal[1]

	list := &unstructured.UnstructuredList{}
	list.SetKind(kind)
	list.SetAPIVersion(apiVersion)
	listOpts := &ctrlruntimeclient.ListOptions{Raw: &metav1.ListOptions{Limit: 1}}

	return func() error {
		// Client creation does discovery calls, so do not attempt to do it initially
		// when the API may not be up yet.
		client, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{})
		if err != nil {
			return fmt.Errorf("failed to create kube client: %w", err)
		}

		if err := client.List(context.Background(), list, listOpts); err != nil {
			return fmt.Errorf("failed to list %s.%s: %w", kind, apiVersion, err)
		}

		return nil
	}, nil
}

func executeCheckers(checkers []func() error) error {
	for _, checker := range checkers {
		if err := checker(); err != nil {
			return err
		}
	}
	return nil
}

// getConfig creates a *rest.Config for interactions with kubernetes API
// The precedence for loading configurations is as follows:
//
// 1. PROBER_KUBECONFIG env variable
// 2. KUBECONFIG env variable
// 3. -kubeconfig flag
// 4. In-cluster config if running inside a cluster
// 5. $HOME/.kube/config if exists.
//
// PROBER_KUBECONFIG is added specifically to facilitate scenarios where we are using multiple kubeconfigs within the same environment/container.
func getConfig() (*rest.Config, error) {
	kubeconfig := os.Getenv("PROBER_KUBECONFIG")
	if len(kubeconfig) > 0 {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	kubeconfig = os.Getenv("KUBECONFIG")
	if len(kubeconfig) > 0 {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	return config.GetConfig()
}
