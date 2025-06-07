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

func (t *Balancer) mostLoaded() (shard *shardNode, found bool) {
	const (
		defaultShardSize            uintptr = 8
		checkNumOfNextNeighborhoods int     = 48
	)

	cur := t.memList.Front()
	if cur == nil {
		return nil, false
	}
	if cur.Value.shard.Size() > defaultShardSize {
		return cur.Value, true
	}

	i := 0
	cur = cur.Next()
	for i < checkNumOfNextNeighborhoods && cur != nil {
		if cur.Value.shard.Size() > defaultShardSize {
			return cur.Value, true
		}
		cur = cur.Next()
		i++
	}

	return nil, false
}

func (t *Balancer) mostLoadedList(percentage int) []*shardNode {
	shardsNum := (int(sharded.ShardCount) / 100) * percentage
	if shardsNum <= 0 {
		shardsNum = 1
	}

	i := 0
	shards := make([]*shardNode, 0, shardsNum)
	cur := t.memList.Front()
	for cur != nil && i < shardsNum {
		shards = append(shards, cur.Value)
		cur = cur.Next()
		i++
	}
	return shards
}
