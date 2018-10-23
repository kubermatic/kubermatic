package main

import (
	"flag"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

type serverRunOptions struct {
	listenAddress            string
	kubeconfig               string
	internalAddr             string
	prometheusURL            string
	prometheusEndpoint       bool
	masterResources          string
	dcFile                   string
	workerName               string
	versionsFile             string
	updatesFile              string
	tokenIssuer              string
	clientID                 string
	tokenIssuerSkipTLSVerify bool
}

func newServerRunOptions() serverRunOptions {
	s := serverRunOptions{}

	flag.StringVar(&s.listenAddress, "address", ":8080", "The address to listen on")
	flag.StringVar(&s.kubeconfig, "kubeconfig", "", "Path to the kubeconfig.")
	flag.StringVar(&s.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the internal handler should be exposed")
	flag.StringVar(&s.prometheusURL, "prometheus-url", "http://prometheus-kubermatic.monitoring.svc.local:web", "The URL on which this API can talk to Prometheus")
	flag.BoolVar(&s.prometheusEndpoint, "enable-prometheus-endpoint", false, "Activate the API endpoint to expose metrics")
	flag.StringVar(&s.masterResources, "master-resources", "", "The path to the master resources (Required).")
	flag.StringVar(&s.dcFile, "datacenters", "datacenters.yaml", "The datacenters.yaml file path")
	flag.StringVar(&s.workerName, "worker-name", "", "Create clusters only processed by worker-name cluster controller")
	flag.StringVar(&s.versionsFile, "versions", "versions.yaml", "The versions.yaml file path")
	flag.StringVar(&s.updatesFile, "updates", "updates.yaml", "The updates.yaml file path")
	flag.StringVar(&s.tokenIssuer, "token-issuer", "", "URL of the OpenID token issuer. Example: http://auth.int.kubermatic.io")
	flag.BoolVar(&s.tokenIssuerSkipTLSVerify, "token-issuer-skip-tls-verify", false, "SKip TLS verification for the token issuer")
	flag.StringVar(&s.clientID, "client-id", "", "OpenID client ID")
	flag.Parse()

	return s
}

type providers struct {
	sshKey        provider.SSHKeyProvider
	user          provider.UserProvider
	project       provider.ProjectProvider
	projectMember provider.ProjectMemberProvider
	memberMapper  provider.ProjectMemberMapper
	cloud         provider.CloudRegistry
	clusters      map[string]provider.ClusterProvider
	datacenters   map[string]provider.DatacenterMeta
}
