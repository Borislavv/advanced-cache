package lru

import (
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/consts"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/list"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
	"sync/atomic"
	"unsafe"
)

type shardNode struct {
	lruList     *list.List[*model.Request] // less used starts at the back
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

func (t *Balancer) register(shard *sharded.Shard[*model.Response]) {
	n := &shardNode{
		shard:   shard,
		lruList: list.New[*model.Request](true),
	}

	n.memListElem = t.memList.PushBack(n)
	t.shards[shard.ID()] = n
}

func (t *Balancer) set(resp *model.Response) *shardNode {
	node := t.shards[resp.Request().ShardKey()]
	atomic.AddInt64(&node.len, 1)
	resp.SetLruListElement(node.lruList.PushFront(resp.Request()))
	return node
}

// moves shards between neighbors (biggest in the front)
func (t *Balancer) rebalance(n *shardNode) {
	curr := n.memListElem
	if curr == nil {
		return
	}
	next := curr.Next()
	for next != nil && curr.Value.shard.Size() < next.Value.shard.Size() {
		t.memList.SwapValues(curr, next)
		curr = next
		next = curr.Next()
	}
}

func (t *Balancer) move(shardKey uint64, el *list.Element[*model.Request]) {
	t.shards[shardKey].lruList.MoveToFront(el)
}

func (t *Balancer) remove(key uint64, shardKey uint64) (freedMem uintptr, isHit bool) {
	freed, listElem, isHit := t.shardedMap.Release(key)
	if !isHit {
		return 0, false
	}

	node := t.shards[shardKey]
	node.lruList.Remove(listElem.(*list.Element[*model.Request]))
	atomic.AddInt64(&node.len, -1)
	t.rebalance(node)

	return freed, true
}

// mostLoaded returns the first non-empty shard node found in memList.
func (t *Balancer) mostLoaded(offset int) (*shardNode, bool) {
	for cur := t.memList.Front(); cur != nil; cur = cur.Next() {
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

func (t *Balancer) memory() uintptr {
	mem := unsafe.Sizeof(t) + uintptr(sharded.ShardCount*consts.PtrBytesWeight)
	for _, shard := range t.shards {
		mem += shard.memory()
	}
	return mem
}
