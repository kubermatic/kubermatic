package ubuntu

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"
)

// Config contains specific configuration for Ubuntu.
type Config struct {
	DistUpgradeOnBoot bool `json:"distUpgradeOnBoot"`
}

// LoadConfig retrieves the Ubuntu configuration from raw data.
func LoadConfig(r runtime.RawExtension) (*Config, error) {
	cfg := Config{}
	if len(r.Raw) == 0 {
		return &cfg, nil
	}
	if err := json.Unmarshal(r.Raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Spec return the configuration as raw data.
func (cfg *Config) Spec() (*runtime.RawExtension, error) {
	ext := &runtime.RawExtension{}
	b, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}
