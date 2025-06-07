## HTTP cache service developed as a Service and the same time as a Traefik plugin.

### Parallel benchmarks
Each bench has 1000 ops. inside one b.N iteration due to have a more heavy job. Benchmark has a problems with tests with nanosecond executions. That is mean that results must be divided by 1000.

Get: ~80ns/op, 20 bytes and zero allocations. Set: ~15ns/op, 105 bytes and zero allocations too.

<img width="1269" alt="image" src="https://github.com/user-attachments/assets/6a28aa7d-bda2-4b40-ae34-5248bd60962a" />

### Wrk results:

RPS: ~140.000, AVG response duration: 7.1µs.

<img width="426" alt="image" src="https://github.com/user-attachments/assets/0a3901af-c536-445c-a2bb-dcf9266b3458" />


### Important constants:
1. sharded.ShardCount - number of map shards (used as a len for array type).
    Current value is 4096 (should be upped to 8192 or 16384 if necessary (this is depending on num of total keys due to each shard must contain from 2K to 20K items)).
2. synced.PreallocationBatchSize - each sync.Pool wrapped by synced.BatchPool, will be preallocating N={{this const}} items on call creator-func.
    Don't use big values because at the start number of preallocated items will be N={{this const}} * 10.
    Current value is 1000. That is number of preallocations at start will be 1000 * 10 and 1000  for further calls.
3. lur.evictItemsPerIter - number of items which will be evicted per iteration.
4. lru.maxEvictIterations - number of max eviction iterations per one eviction.
5. lru.topPercentageShards - percentage of shards which will be queried for evict items from them.
6. wheel.numOfRefreshesPerSec - number of revalidation per second (refresh rate limiter).
    Current value is 1000.
#### Note: Don't change them if you don't know what you do and be careful if you still decide. In most cases it does not necessary. 


#### Sorry, but at now you cannot use this code in traefik plugins due to this version has unsafe and other low-level libs, but you can use it as a service. 
