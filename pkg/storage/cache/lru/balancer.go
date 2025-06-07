package lru

import (
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/list"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
)

type shardNode struct {
	lruList     *list.List[*model.Request] // less used starts at the back
	memListElem *list.Element[*shardNode]
	shard       *sharded.Shard[*model.Response]
}

type Balancer struct {
	shards     [sharded.ShardCount]*shardNode
	memList    *list.List[*shardNode] // more loaded starts at the front
	shardedMap *sharded.Map[*model.Response]
}

func NewBalancer(shardedMap *sharded.Map[*model.Response]) *Balancer {
	const isListShouldByAThreadSafe bool = true

	return &Balancer{
		memList:    list.New[*shardNode](isListShouldByAThreadSafe),
		shardedMap: shardedMap,
	}
}

func (t *Balancer) register(shard *sharded.Shard[*model.Response]) {
	const isThreadSafeListMustBe bool = true

	n := &shardNode{
		shard:   shard,
		lruList: list.New[*model.Request](isThreadSafeListMustBe),
	}

	n.memListElem = t.memList.PushBack(n)
	t.shards[shard.ID()] = n
}

func (t *Balancer) set(resp *model.Response) *list.Element[*model.Request] {
	n := t.shards[resp.GetShardKey()]
	if n == nil {
		return nil
	}

	el := t.push(n, resp)
	t.rebalance(n)

	return el
}

func (t *Balancer) push(n *shardNode, resp *model.Response) *list.Element[*model.Request] {
	el := n.lruList.PushFront(resp.GetRequest())
	resp.SetListElement(el)
	return el
}

func (t *Balancer) rebalance(n *shardNode) {
	curr := n.memListElem
	if curr == nil {
		return
	}
	next := curr.Next()
	for next != nil && curr.Value.shard.Size() < next.Value.shard.Size() {
		t.swap(curr, next)
		curr = next
		next = curr.Next()
	}
}

func (t *Balancer) move(shardKey uint64, el *list.Element[*model.Request]) {
	n := t.shards[shardKey]
	if n == nil {
		return
	}

	n.lruList.MoveToFront(el)
}

func (t *Balancer) swap(a, b *list.Element[*shardNode]) {
	t.memList.SwapValues(a, b)
}

func (t *Balancer) remove(key uint64, shardKey uint64) (*model.Response, bool) {
	n := t.shards[shardKey]

	resp, found := t.shardedMap.Del(key)
	if !found {
		return nil, false
	}

	t.del(n, resp)
	t.rebalance(n)

	return resp, true
}

func (t *Balancer) del(n *shardNode, resp *model.Response) {
	n.lruList.Remove(resp.GetListElement())
}

// mostLoaded returns the first non-empty shard node found in memList.
func (t *Balancer) mostLoaded() (*shardNode, bool) {
	for cur := t.memList.Front(); cur != nil; cur = cur.Next() {
		if cur.Value != nil && cur.Value.shard.Len.Load() > 0 {
			return cur.Value, true
		}
	}
	return nil, false
}
