package informer

import "time"

const (
	// DefaultInformerResyncPeriod defines the default resync period for all informers
	DefaultInformerResyncPeriod = 30 * time.Minute
)
