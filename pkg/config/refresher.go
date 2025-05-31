package config

import "time"

type Refresher struct {
	RefreshInterval    time.Duration `mapstructure:"REFRESH_INTERVAL"`
	RefreshParallelism int           `mapstructure:"REFRESH_PARALLELISM"`
}
