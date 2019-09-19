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
	"syscall"
	"time"

	"go.uber.org/zap"

	httpproberapi "github.com/kubermatic/kubermatic/api/cmd/http-prober/api"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {
	var (
		endpoint      string
		insecure      bool
		retries       int
		retryWait     int
		timeout       int
		crdKind       string
		crdAPIVersion string
		commandRaw    string
	)
	flag.StringVar(&endpoint, "endpoint", "", "The endpoint which should be waited for")
	flag.BoolVar(&insecure, "insecure", false, "Disable certificate validation")
	flag.IntVar(&retries, "retries", 10, "Number of retries")
	flag.IntVar(&retryWait, "retry-wait", 1, "Wait interval in seconds between retries")
	flag.IntVar(&timeout, "timeout", 30, "Timeout in seconds")
	flag.StringVar(&crdKind, "wait-for-crd-kind", "", "If set, wait for this CRD to be present. Requires the KUBECONFIG env var to be set.")
	flag.StringVar(&crdAPIVersion, "wait-for-crd-apiversion", "", "If set, wait for this CRD to be present. Requires the KUBECONFIG env var to be set.")
	flag.StringVar(&commandRaw, "command", "", "If passed, the http prober will exec this command. Must be json encoded")
	flag.Parse()

	log := kubermaticlog.Logger.Named("http-prober")

	crdChecker, err := crdCheckerFactory(crdKind, crdAPIVersion)
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

	req, err := http.NewRequest("GET", e.String(), nil)
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

		if err := crdChecker(); err != nil {
			log.Infow("Check if crd is available was not successful", "crd", crdKind, zap.Error(err))
			continue
		}
		if crdKind != "" {
			log.Infow("CRD is available", "crd", crdKind)
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

func crdCheckerFactory(crdKind, crdAPIVersion string) (func() error, error) {
	if crdKind == "" && crdAPIVersion == "" {
		return func() error {
			return nil
		}, nil
	}
	if crdKind == "" {
		return nil, errors.New("--wait-for-crd-kind must bet set if --wait-for-crd-apiversion is set")
	}
	if crdAPIVersion == "" {
		return nil, errors.New("--wait-for-crd-apiversion must be set if --wait-for-crd-kind is set")
	}

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		return nil, errors.New("--wait-for-crd was set, but KUBECONFIG env var was not")
	}
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubeconfig: %v", err)
	}

	list := &unstructured.UnstructuredList{}
	list.SetKind(crdKind)
	list.SetAPIVersion(crdAPIVersion)

	listOpts := &ctrlruntimeclient.ListOptions{Raw: &metav1.ListOptions{Limit: 1}}

	return func() error {
		// Client creation does discovery calls, so do not attempt to do it initially
		// when the API may not be up yet.
		client, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{})
		if err != nil {
			return fmt.Errorf("failed to create kube client: %v", err)
		}

		if err := client.List(context.Background(), listOpts, list); err != nil {
			return fmt.Errorf("failed to list %q: %v", crdKind, err)
		}

		return nil
	}, nil
}
