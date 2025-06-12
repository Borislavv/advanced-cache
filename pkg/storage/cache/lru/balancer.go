package lru

import (
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/consts"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/list"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	"math/rand/v2"
	"sync/atomic"
	"unsafe"
)

// shardNode represents a single shard's LRU and accounting info.
// Each shard has its own LRU list and a pointer to its element in the balancer's memList.
type shardNode struct {
	lruList     *list.List[*model.Response]     // Per-shard LRU list; less used responses at the back
	memListElem *list.Element[*shardNode]       // Pointer to this node's position in Balancer.memList
	shard       *sharded.Shard[*model.Response] // Reference to the actual shard (map + sync)
	len         int64                           // Number of items in the shard (atomically updated)
}

// memory returns an approximate memory usage of this shardNode structure.
func (s *shardNode) memory() uintptr {
	return unsafe.Sizeof(s) + uintptr(atomic.LoadInt64(&s.len)*consts.PtrBytesWeight)
}

// Balancer maintains per-shard LRU lists and provides efficient selection of loaded shards for eviction.
// - memList orders shardNodes by usage (most loaded in front).
// - shards is a flat array for O(1) access by shard index.
// - shardedMap is the underlying data storage (map of all entries).
type Balancer struct {
	shards     [sharded.ShardCount]*shardNode // Shard index → *shardNode
	memList    *list.List[*shardNode]         // Doubly-linked list of shards, ordered by memory usage (most loaded at front)
	shardedMap *sharded.Map[*model.Response]  // Actual underlying storage of entries
}

// NewBalancer creates a new Balancer instance and initializes memList.
func NewBalancer(shardedMap *sharded.Map[*model.Response]) *Balancer {
	return &Balancer{
		memList:    list.New[*shardNode](true), // Sorted mode for easier rebalancing
		shardedMap: shardedMap,
	}
}

// randShardNode returns a random shardNode for sampling (e.g., for background refreshers).
func (b *Balancer) randShardNode() *shardNode {
	return b.shards[rand.Uint64N(sharded.ShardCount)]
}

// register inserts a new shardNode for a given shard, creates its LRU, and adds it to memList and shards array.
func (b *Balancer) register(shard *sharded.Shard[*model.Response]) {
	n := &shardNode{
		shard:   shard,
		lruList: list.New[*model.Response](true),
	}
	n.memListElem = b.memList.PushBack(n)
	b.shards[shard.ID()] = n
}

// set inserts a response into the appropriate shard's LRU list and updates counters.
// Returns the affected shardNode for further operations.
func (b *Balancer) set(resp *model.Response) *shardNode {
	node := b.shards[resp.Request().ShardKey()]
	atomic.AddInt64(&node.len, 1)
	resp.SetLruListElement(node.lruList.PushFront(resp))
	return node
}

// rebalance maintains the order of memList so that more loaded shards are kept in the front.
// Moves the given shardNode forward until its position is correct.
func (b *Balancer) rebalance(n *shardNode) {
	curr := n.memListElem
	if curr == nil {
		return
	}
	next := curr.Next()
	for next != nil && curr.Value.shard.Size() < next.Value.shard.Size() {
		b.memList.SwapValues(curr, next)
		curr = next
		next = curr.Next()
	}
}

// move moves an element to the front of the per-shard LRU list.
// Used for touch/set operations to mark entries as most recently used.
func (b *Balancer) move(shardKey uint64, el *list.Element[*model.Response]) {
	b.shards[shardKey].lruList.MoveToFront(el)
}

// remove releases an entry from the shardedMap and removes it from the per-shard LRU list.
// Also updates counters and rebalances memList if necessary.
// Returns (memory_freed, was_found).
func (b *Balancer) remove(key uint64, shardKey uint64) (freedMem uintptr, isHit bool) {
	freed, listElem, isHit := b.shardedMap.Release(key)
	if !isHit {
		return 0, false
	}

	node := b.shards[shardKey]
	node.lruList.Remove(listElem.(*list.Element[*model.Response]))
	atomic.AddInt64(&node.len, -1)
	b.rebalance(node)

	return freed, true
}

// mostLoaded returns the first non-empty shard node from the front of memList,
// optionally skipping a number of nodes by offset (for concurrent eviction fairness).
func (b *Balancer) mostLoaded(offset int) (*shardNode, bool) {
	for cur := b.memList.Front(); cur != nil; cur = cur.Next() {
		if offset > 0 {
			offset--
			continue
		}
		if cur.Value != nil && cur.Value.shard.Len() > 0 {
			return cur.Value, true
		}
	}
	return nil, false
}

// memory returns an approximate total memory usage of all shards and the balancer itself.
func (b *Balancer) memory() uintptr {
	mem := unsafe.Sizeof(b) + uintptr(sharded.ShardCount*consts.PtrBytesWeight)
	for _, shard := range b.shards {
		mem += shard.memory()
	}
	return mem
}
