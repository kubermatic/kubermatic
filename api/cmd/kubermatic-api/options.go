package main

import (
	"flag"
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/features"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

type serverRunOptions struct {
	listenAddress   string
	kubeconfig      string
	internalAddr    string
	prometheusURL   string
	masterResources string
	dcFile          string
	workerName      string
	versionsFile    string
	updatesFile     string

	// OIDC configuration
	oidcIssuerURL            string
	oidcClientID             string
	oidcClientSecret         string
	oidcRedirectURI          string
	oidcSkipTLSVerify        bool
	oidcOfflineAccessAsScope bool

	featureGates features.FeatureGate
}

func newServerRunOptions() (serverRunOptions, error) {
	s := serverRunOptions{}
	var rawFeatureGates string

	flag.StringVar(&s.listenAddress, "address", ":8080", "The address to listen on")
	flag.StringVar(&s.kubeconfig, "kubeconfig", "", "Path to the kubeconfig.")
	flag.StringVar(&s.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the internal handler should be exposed")
	flag.StringVar(&s.prometheusURL, "prometheus-url", "http://prometheus-kubermatic.monitoring.svc.local:web", "The URL on which this API can talk to Prometheus")
	flag.StringVar(&s.masterResources, "master-resources", "", "The path to the master resources (Required).")
	flag.StringVar(&s.dcFile, "datacenters", "datacenters.yaml", "The datacenters.yaml file path")
	flag.StringVar(&s.workerName, "worker-name", "", "Create clusters only processed by worker-name cluster controller")
	flag.StringVar(&s.versionsFile, "versions", "versions.yaml", "The versions.yaml file path")
	flag.StringVar(&s.updatesFile, "updates", "updates.yaml", "The updates.yaml file path")
	flag.StringVar(&s.oidcIssuerURL, "oidc-issuer-url", "", "URL of the OpenID token issuer. Example: http://auth.int.kubermatic.io")
	flag.BoolVar(&s.oidcSkipTLSVerify, "oidc-skip-tls-verify", false, "Skip TLS verification for the token issuer")
	flag.StringVar(&s.oidcClientID, "oidc-client-id", "", "OpenID client ID")
	flag.StringVar(&s.oidcClientSecret, "oidc-client-secret", "", "OpenID client secret")
	flag.StringVar(&s.oidcRedirectURI, "oidc-redirect-uri", "", "Callback URL for OpenID responses.")
	flag.BoolVar(&s.oidcOfflineAccessAsScope, "oidc-offline-access-as-scope", true, "Set it to false if OIDC provider requires to set \"access_type=offline\" query param when accessing the refresh token")
	flag.StringVar(&rawFeatureGates, "feature-gates", "", "A set of key=value pairs that describe feature gates for various features.")
	flag.Parse()

	featureGates, err := features.NewFeatures(rawFeatureGates)
	if err != nil {
		return s, err
	}
	s.featureGates = featureGates
	return s, nil
}

func (o serverRunOptions) validate() error {
	if o.featureGates.Enabled(OIDCKubeCfgEndpoint) {
		if len(o.oidcClientSecret) == 0 {
			return fmt.Errorf("%s feature is enabled but \"oidc-client-secret\" flag was not specified", OIDCKubeCfgEndpoint)
		}
		if len(o.oidcRedirectURI) == 0 {
			return fmt.Errorf("%s feature is enabled but \"oidc-redirect-uri\" flag was not specified", OIDCKubeCfgEndpoint)
		}
	}
	return nil
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
