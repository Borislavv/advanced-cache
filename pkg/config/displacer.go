package config

import (
	"time"
)

type Displacer struct {
	Algorithm                    string        `mapstructure:"DISPLACE_ALGORITHM"` // implemented: LRU, MRU, not implemented: LFU, MFU, see service.Algorithm for more information
	DisplaceInterval             time.Duration `mapstructure:"DISPLACE_INTERVAL"`
	DisplaceThresholdMemoryBytes int           `mapstructure:"DISPLACE_THRESHOLD_MEMORY_BYTES"`
	// DisplaceThreshold is a value which means that displace will start displacing responses
	// if storage will be filled for threshold value (80% = 0.8).
	DisplaceThreshold   float32 `mapstructure:"DISPLACE_THRESHOLD"`   // must be value from 0 to 1, e.g. 0.7
	DisplaceParallelism int     `mapstructure:"DISPLACE_PARALLELISM"` // 25 displacers
}
