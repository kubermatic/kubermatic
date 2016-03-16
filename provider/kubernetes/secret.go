package kubernetes

import (
	"bufio"
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"
)

// KeyCert is a pair of key and cert in the secrets.yaml.
type KeyCert struct {
	Key  string `yaml:"key"`
	Cert string `yaml:"cert"`
}

// Secrets keeps cloud provider secrets, e.g. to create seed nodes.
type Secrets struct {
	Tokens       map[string]string  `yaml:"tokens"`
	RootCAs      map[string]KeyCert `yaml:"root-cas"`
	Certificates map[string]KeyCert `yaml:"certificates"`
	ApiserverSSH map[string]string  `yaml:"apiserverSSH"`
}

// LoadSecrets loads secrets from the given path.
func LoadSecrets(path string) (*Secrets, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(bufio.NewReader(f))
	if err != nil {
		return nil, err
	}

	secrets := Secrets{}
	err = yaml.Unmarshal(bytes, &secrets)
	if err != nil {
		return nil, err
	}

	if secrets.Tokens == nil {
		secrets.Tokens = map[string]string{}
	}
	if secrets.RootCAs == nil {
		secrets.RootCAs = map[string]KeyCert{}
	}

	return &secrets, nil
}
