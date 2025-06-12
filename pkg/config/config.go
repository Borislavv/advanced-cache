package config

import "time"

// Environment constants for application mode.
const (
	EnvProd = "prod"
	EnvDev  = "dev"
	EnvTest = "test"
)

// Cache is the main application configuration struct.
// All fields are loaded via mapstructure for compatibility with envconfig/viper/unmarshalers.
type Cache struct {
	AppEnv   string `mapstructure:"APP_ENV"`   // Application environment: "prod", "dev", or "test"
	AppDebug bool   `mapstructure:"APP_DEBUG"` // Enable debug logging and features
	SeoUrl   string `mapstructure:"SEO_URL"`   // Upstream SEO service base URL

	// RevalidateBeta controls the probability distribution for background refresh of cached items.
	//   - Must be in the range [0.1, 0.9] (0 disables, 1 is always refresh, 0.5 is typical).
	//   - Used to randomize refresh times and avoid stampedes.
	//   - 0.5 is a reasonable starting point; 1.0 is aggressive; 0.0 disables background refreshing.
	RevalidateBeta float64 `mapstructure:"REVALIDATE_BETA"`

	// RevalidateInterval is the base interval for periodic revalidation (e.g., "5m", "10s").
	RevalidateInterval time.Duration `mapstructure:"REVALIDATE_INTERVAL"`

	// InitStorageLengthPerShard controls initial allocation for storage shard maps.
	InitStorageLengthPerShard int `mapstructure:"INIT_STORAGE_LEN_PER_SHARD"`

	// EvictionAlgo defines the eviction algorithm to use ("lru", "lfu", etc).
	EvictionAlgo string `mapstructure:"EVICTION_ALGO"`

	// MemoryFillThreshold is a ratio [0.0, 1.0] of allowed memory before aggressive eviction starts.
	MemoryFillThreshold float64 `mapstructure:"MEMORY_FILL_THRESHOLD"`

	// MemoryLimit is the upper bound (in bytes) for the cache's total memory usage.
	MemoryLimit uint `mapstructure:"MEMORY_LIMIT"`

	// LivenessProbeTimeout is the duration after which a failed liveness probe triggers error state.
	LivenessProbeTimeout time.Duration `mapstructure:"LIVENESS_PROBE_FAILED_TIMEOUT"`

	// RefreshDurationThreshold is calculated at startup and used to determine max staleness before revalidation.
	RefreshDurationThreshold time.Duration
}

// IsProdEnv returns true if the app is running in production mode.
func (c *Cache) IsProdEnv() bool {
	return c.AppEnv == EnvProd
}

// IsDevEnv returns true if the app is running in development mode.
func (c *Cache) IsDevEnv() bool {
	return c.AppEnv == EnvDev
}

// IsTestEnv returns true if the app is running in test mode.
func (c *Cache) IsTestEnv() bool {
	return c.AppEnv == EnvTest
}

// IsDebugOn returns true if debug mode is enabled.
func (c *Cache) IsDebugOn() bool {
	return c.AppDebug
}
