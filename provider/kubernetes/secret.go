package kubernetes

import (
	"bufio"
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"
)

// Secrets keeps cloud provider secrets, e.g. to create seed nodes.
type Secrets struct {
	Tokens map[string]string `yaml:"tokens"`
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

	return &secrets, nil
}
