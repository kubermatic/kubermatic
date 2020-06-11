package flagopts

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/util/homedir"
)

// NewKubeconfig detect KUBECONFIG from ENV or default to $HOME/.kube/config
func NewKubeconfig() KubeconfigFlag {
	defaultKubeconfig, ok := os.LookupEnv("KUBECONFIG")
	if !ok {
		defaultKubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}

	return KubeconfigFlag(defaultKubeconfig)
}

// KubeconfigFlag flag, will detect possible defaults
type KubeconfigFlag string

// String is flag.Value implementation method
func (s KubeconfigFlag) String() string {
	return string(s)
}

// Set is flag.Value implementation method
func (s *KubeconfigFlag) Set(val string) error {
	if s == nil {
		s = new(KubeconfigFlag)
	}
	*s = KubeconfigFlag(val)
	return nil
}
