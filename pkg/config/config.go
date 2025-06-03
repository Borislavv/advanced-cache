package config

import "time"

type Config struct {
	SeoUrl string `mapstructure:"SEO_URL"`
	// RevalidateBeta is a value which will be used for generate
	// random time point for refresh response (must be from 0.1 to 0.9).
	// Don't use absolute values like 0 and 1 due it will be leading to CPU peaks usage.
	//  - beta = 0.5 — regular, good for start value
	//  - beta = 1.0 — aggressive refreshing
	//  - beta = 0.0 — disables refreshing
	RevalidateBeta            float64       `mapstructure:"REVALIDATE_BETA"`
	RevalidateInterval        time.Duration `mapstructure:"REVALIDATE_INTERVAL"`
	InitStorageLengthPerShard int           `mapstructure:"INIT_STORAGE_LEN_PER_SHARD"`
	EvictionAlgo              string        `mapstructure:"EVICTION_ALGO"`
	MemoryFillThreshold       float64       `mapstructure:"MEMORY_FILL_THRESHOLD"`
	MemoryLimit               float64       `mapstructure:"MEMORY_LIMIT"`
}
