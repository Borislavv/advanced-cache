package config

import "time"

const (
	EnvProd = "prod"
	EnvDev  = "dev"
	EnvTest = "test"
)

type Config struct {
	AppEnv   string `mapstructure:"APP_ENV"`
	AppDebug bool   `mapstructure:"APP_DEBUG"`
	SeoUrl   string `mapstructure:"SEO_URL"`
	// RevalidateBeta is a value which will be used for generate
	// random time point for refresh response (must be from 0.1 to 0.9).
	// Don't use absolute values like 0 and 1 due it will be leading to CPU peaks usage.
	//  - beta = 0.5 — regular, good for start value
	//  - beta = 1.0 — aggressive refreshing
	//  - beta = 0.0 — disables refreshing
	RevalidateBeta                   float64       `mapstructure:"REVALIDATE_BETA"`
	RevalidateInterval               time.Duration `mapstructure:"REVALIDATE_INTERVAL"`
	InitStorageLengthPerShard        int           `mapstructure:"INIT_STORAGE_LEN_PER_SHARD"`
	EvictionAlgo                     string        `mapstructure:"EVICTION_ALGO"`
	MemoryFillThreshold              float64       `mapstructure:"MEMORY_FILL_THRESHOLD"`
	MemoryLimit                      uint          `mapstructure:"MEMORY_LIMIT"`
	LivenessProbeTimeout             time.Duration `mapstructure:"LIVENESS_PROBE_FAILED_TIMEOUT"`
	RefreshEvictionDurationThreshold time.Duration // calculates on load
}

func (c *Config) IsProdEnv() bool {
	return c.AppEnv == EnvProd
}
func (c *Config) IsDevEnv() bool {
	return c.AppEnv == EnvDev
}
func (c *Config) IsTestEnv() bool {
	return c.AppEnv == EnvTest
}
func (c *Config) IsDebugOn() bool {
	return c.AppDebug
}
