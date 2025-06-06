# traefik-http-cache-plugin
HTTP cache service developed as a Traefik plugin. 

### Parallel benchmarks

### Important constants:
1. sharded.ShardCount - number of map shards (used as a len for array type).
    Current value is 4096 (should be upped to 8192 or 16384 if necessary (this is depending on num of total keys due to each shard must contain from 2K to 20K items)).
2. synced.PreallocationBatchSize - each sync.Pool wrapped by synced.BatchPool, will be preallocating N={{this const}} items on call creator-func.
    Don't use big values because at the start number of preallocated items will be N={{this const}}*10.
    Current value is 1000. That is number of preallocations at start will be 1000*10 and 1000  for further calls.
3. lur.evictItemsPerIter - number of items which will be evicted per iteration.
4. lru.maxEvictIterations - number of max eviction iterations per one eviction.
5. lru.topPercentageShards - percentage of shards which will be queried for evict items from them.
#### Note: Don't change them if you don't know what you do and be careful if you still decide. In most cases it does not necessary. 