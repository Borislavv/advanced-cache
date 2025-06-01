package config

import "time"

type Response struct {
	// RevalidateBeta is a value which will be used for generate
	// random time point for refresh response (must be from 0.1 to 0.9).
	// Don't use absolute values like 0 and 1 due it will be leading to CPU peaks usage.
	RevalidateBeta     float64       `mapstructure:"REVALIDATE_BETA"`
	RevalidateInterval time.Duration `mapstructure:"REVALIDATE_INTERVAL"`
	// Revalidate Parallelism - the maximum number of validators at the moment.
	RevalidateParallelism int `mapstructure:"REVALIDATE_PARALLELISM"`
}
