package test

import (
	"fmt"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
)

// mock that raises an error when try to read secret
func ShouldNotBeCalled(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error) {
	return "", fmt.Errorf("'GetGlobalSecretKeySelectorValue' should not be called")
}

// mock that returns default value when reading secret or value specify in generator map.
// Default value is key + "-value"
// generator is map of key (of GlobalSecretKeySelector) and value to return for this key. Value can be an error or a string
func DefaultOrOverride(generator map[string]interface{}) func(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error) {
	return func(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error) {
		if val, ok := generator[key]; ok {
			if err, ok := val.(error); ok {
				return "", err
			}
			return val.(string), nil
		}
		return key + "-value", nil
	}
}

// return an error with message: secret "default/the-secret" has no key "<key>"
func MissingKeyErr(key string) error {
	return fmt.Errorf("secret \"default/the-secret\" has no key \"%v\"", key)
}
