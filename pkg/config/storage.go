package config

type Storage struct {
	InitStorageLengthPerShard int     `mapstructure:"INIT_STORAGE_LEN_PER_SHARD"`
	EvictionAlgo              string  `mapstructure:"EVICTION_ALGO"`
	MemoryFillThreshold       float64 `mapstructure:"MEMORY_FILL_THRESHOLD"`
	MemoryLimit               float64 `mapstructure:"MEMORY_LIMIT"`
}
