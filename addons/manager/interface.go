package manager

import (
	"github.com/kubermatic/api"
)

// AddonManager represents a addon manager
type AddonManager interface {
	ListReleases() error
	Install(addon *api.ClusterAddon) error
	Delete(rlsName string) error
	UpdateRelease(rlsName string) error
	RollbackRelease(rlsName string) error
}
