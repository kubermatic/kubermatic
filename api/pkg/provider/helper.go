package provider

import (
	"fmt"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
)

func ValidateSecretKeySelector(selector *providerconfig.GlobalSecretKeySelector, key string) error {
	if selector.Name == "" && selector.Namespace == "" && selector.Key == "" {
		return fmt.Errorf("%q cannot be empty", key)
	}
	return nil
}
