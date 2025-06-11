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

type shardNode struct {
	lruList     *list.List[*model.Response] // less used starts at the back
	memListElem *list.Element[*shardNode]
	shard       *sharded.Shard[*model.Response]
	len         int64
}

func (s *shardNode) memory() uintptr {
	return unsafe.Sizeof(s) + uintptr(atomic.LoadInt64(&s.len)*consts.PtrBytesWeight)
}

type Balancer struct {
	shards     [sharded.ShardCount]*shardNode
	memList    *list.List[*shardNode] // more loaded starts at the front
	shardedMap *sharded.Map[*model.Response]
}

func NewBalancer(shardedMap *sharded.Map[*model.Response]) *Balancer {
	return &Balancer{
		memList:    list.New[*shardNode](true),
		shardedMap: shardedMap,
	}
}

func (b *Balancer) randShardNode() *shardNode {
	return b.shards[rand.Uint64N(sharded.ShardCount)]
}

func (b *Balancer) register(shard *sharded.Shard[*model.Response]) {
	n := &shardNode{
		shard:   shard,
		lruList: list.New[*model.Response](true),
	}

	n.memListElem = b.memList.PushBack(n)
	b.shards[shard.ID()] = n
}

func (b *Balancer) set(resp *model.Response) *shardNode {
	node := b.shards[resp.Request().ShardKey()]
	atomic.AddInt64(&node.len, 1)
	resp.SetLruListElement(node.lruList.PushFront(resp))
	return node
}

// moves shards between neighbors (biggest in the front)
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

func (b *Balancer) move(shardKey uint64, el *list.Element[*model.Response]) {
	b.shards[shardKey].lruList.MoveToFront(el)
}

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

// mostLoaded returns the first non-empty shard node found in memList.
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

func (b *Balancer) memory() uintptr {
	mem := unsafe.Sizeof(b) + uintptr(sharded.ShardCount*consts.PtrBytesWeight)
	for _, shard := range b.shards {
		mem += shard.memory()
	}
	return mem
}
