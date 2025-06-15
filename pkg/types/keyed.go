package types

// Keyed defines a unique key and a precomputed shard key for the value.
type Keyed interface {
	Key() uint64
	ShardKey() uint64
}
