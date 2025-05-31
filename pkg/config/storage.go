package config

type Storage struct {
	EvictionAlgo               string  `mapstructure:"EVICTION_ALGO"`
	MemoryFillThreshold        float64 `mapstructure:"MEMORY_FILL_THRESHOLD"`
	MemoryLimit                float64 `mapstructure:"MEMORY_LIMIT"`
	ParallelEvictionsAvailable int     `mapstructure:"PARALLEL_EVICTIONS_AVAILABLE"`
}
