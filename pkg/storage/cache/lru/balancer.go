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

// ShardNode represents a single shard's LRU and accounting info.
// Each shard has its own LRU list and a pointer to its element in the balancer's memList.
type ShardNode struct {
	lruList     *list.List[*model.Response]     // Per-shard LRU list; less used responses at the back
	memListElem *list.Element[*ShardNode]       // Pointer to this node's position in Balance.memList
	shard       *sharded.Shard[*model.Response] // Reference to the actual shard (map + sync)
	len         int64                           // Number of items in the shard (atomically updated)
}

// memory returns an approximate memory usage of this ShardNode structure.
func (s *ShardNode) memory() uintptr {
	return unsafe.Sizeof(s) + uintptr(atomic.LoadInt64(&s.len)*consts.PtrBytesWeight)
}

type Balancer interface {
	Shards() [sharded.ShardCount]*ShardNode
	RandShardNode() *ShardNode
	Register(shard *sharded.Shard[*model.Response])
	Set(resp *model.Response) *ShardNode
	Rebalance(n *ShardNode)
	Move(shardKey uint64, el *list.Element[*model.Response])
	Remove(key uint64, shardKey uint64) (freedMem uintptr, isHit bool)
	MostLoaded(offset int) (*ShardNode, bool)
	Memory() uintptr
}

// Balance maintains per-shard LRU lists and provides efficient selection of loaded shards for eviction.
// - memList orders shardNodes by usage (most loaded in front).
// - shards is a flat array for O(1) access by shard index.
// - shardedMap is the underlying data storage (map of all entries).
type Balance struct {
	shards     [sharded.ShardCount]*ShardNode // Shard index → *ShardNode
	memList    *list.List[*ShardNode]         // Doubly-linked list of shards, ordered by Memory usage (most loaded at front)
	shardedMap *sharded.Map[*model.Response]  // Actual underlying storage of entries
}

// NewBalancer creates a new Balance instance and initializes memList.
func NewBalancer(shardedMap *sharded.Map[*model.Response]) *Balance {
	return &Balance{
		memList:    list.New[*ShardNode](true), // Sorted mode for easier rebalancing
		shardedMap: shardedMap,
	}
}

func (b *Balance) Shards() [sharded.ShardCount]*ShardNode {
	return b.shards
}

// RandShardNode returns a random ShardNode for sampling (e.g., for background refreshers).
func (b *Balance) RandShardNode() *ShardNode {
	return b.shards[rand.Uint64N(sharded.ShardCount)]
}

// Register inserts a new ShardNode for a given shard, creates its LRU, and adds it to memList and shards array.
func (b *Balance) Register(shard *sharded.Shard[*model.Response]) {
	n := &ShardNode{
		shard:   shard,
		lruList: list.New[*model.Response](true),
	}
	n.memListElem = b.memList.PushBack(n)
	b.shards[shard.ID()] = n
}

// Set inserts a response into the appropriate shard's LRU list and updates counters.
// Returns the affected ShardNode for further operations.
func (b *Balance) Set(resp *model.Response) *ShardNode {
	node := b.shards[resp.Request().ShardKey()]
	atomic.AddInt64(&node.len, 1)
	resp.SetLruListElement(node.lruList.PushFront(resp))
	return node
}

// Rebalance maintains the order of memList so that more loaded shards are kept in the front.
// Moves the given ShardNode forward until its position is correct.
func (b *Balance) Rebalance(n *ShardNode) {
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

// Move moves an element to the front of the per-shard LRU list.
// Used for touch/Set operations to mark entries as most recently used.
func (b *Balance) Move(shardKey uint64, el *list.Element[*model.Response]) {
	b.shards[shardKey].lruList.MoveToFront(el)
}

// Remove releases an entry from the shardedMap and removes it from the per-shard LRU list.
// Also updates counters and rebalances memList if necessary.
// Returns (memory_freed, was_found).
func (b *Balance) Remove(key uint64, shardKey uint64) (freedMem uintptr, isHit bool) {
	freed, listElem, isHit := b.shardedMap.Release(key)
	if !isHit {
		return 0, false
	}

	node := b.shards[shardKey]
	node.lruList.Remove(listElem.(*list.Element[*model.Response]))
	atomic.AddInt64(&node.len, -1)
	b.Rebalance(node)

	return freed, true
}

// MostLoaded returns the first non-empty shard node from the front of memList,
// optionally skipping a number of nodes by offset (for concurrent eviction fairness).
func (b *Balance) MostLoaded(offset int) (*ShardNode, bool) {
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

// Memory returns an approximate total Memory usage of all shards and the balancer itself.
func (b *Balance) Memory() uintptr {
	mem := unsafe.Sizeof(b) + uintptr(sharded.ShardCount*consts.PtrBytesWeight)
	for _, shard := range b.shards {
		mem += shard.memory()
	}
	return mem
}
