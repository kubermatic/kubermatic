package options

import (
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type MigrationOptions struct {
	DatacentersFile    string
	DynamicDatacenters bool
	Kubeconfig         *clientcmdapi.Config
}

// MigrationEnabled returns true if at least one migration is enabled.
func (o MigrationOptions) MigrationEnabled() bool {
	return o.SeedMigrationEnabled()
}

// SeedMigrationEnabled returns true if the datacenters->seed migration is enabled.
func (o MigrationOptions) SeedMigrationEnabled() bool {
	return o.DatacentersFile != "" && o.DynamicDatacenters
}
