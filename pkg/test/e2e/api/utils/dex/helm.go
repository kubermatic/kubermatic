package dex

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type helmValues struct {
	Dex struct {
		Ingress struct {
			Scheme string `yaml:"scheme"`
			Host   string `yaml:"host"`
			Path   string `yaml:"path"`
		} `yaml:"ingress"`

		Clients []struct {
			ID           string   `yaml:"id"`
			RedirectURIs []string `yaml:"RedirectURIs"`
		} `yaml:"clients"`
	} `yaml:"dex"`
}

// NewClientFromHelmValues is a helper for e2e tests, reading the hack/ci/testdata/oauth_values.yaml
// to provide a matching OIDC client. We use this instead of spreading the client ID etc.
// in tons of shell scripts and env vars.
func NewClientFromHelmValues(valuesFile string, clientID string, log *zap.SugaredLogger) (*Client, error) {
	values := helmValues{}

	f, err := os.Open(valuesFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %v", valuesFile, err)
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(&values); err != nil {
		return nil, fmt.Errorf("failed to decode %s as YAML: %v", valuesFile, err)
	}

	redirectURI := ""

	for _, client := range values.Dex.Clients {
		if client.ID == clientID {
			// The actual redirect URI does not matter, as long as it's registered with
			// Dex. We will intercept the redirect anyway.
			redirectURI = client.RedirectURIs[0]
		}
	}

	if redirectURI == "" {
		return nil, fmt.Errorf("could not find a client with ID %q", clientID)
	}

	scheme := values.Dex.Ingress.Scheme
	if scheme == "" {
		scheme = "https"
	}

	providerURI := fmt.Sprintf("%s://%s%s/auth", scheme, values.Dex.Ingress.Host, values.Dex.Ingress.Path)

	return NewClient(clientID, redirectURI, providerURI, log)
}
