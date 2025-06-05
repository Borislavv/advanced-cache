package lru

import (
	"container/list"
	"context"
	"sync"

	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	sharded "github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/map"
)

type node struct {
	mu          sync.Mutex
	lruList     *list.List // array of items lists with sort "from newest to oldest"
	memListElem *list.Element
	shard       *sharded.Shard[uint64, *model.Response]
}

type Balancer struct {
	mu         sync.RWMutex
	memList    *list.List // shards sorted from most loaded  by memory to lower loaded
	shards     [sharded.ShardCount]*node
	shardedMap *sharded.Map[uint64, *model.Response]
}

func NewBalancer() *Balancer {
	return &Balancer{
		mu:      sync.RWMutex{},
		memList: list.New(),
	}
}

func (t *Balancer) register(shard *sharded.Shard[uint64, *model.Response]) {
	t.mu.Lock()
	defer t.mu.Unlock()

	n := &node{
		shard:   shard,
		lruList: list.New(),
	}

	el := t.memList.PushBack(n)
	t.shards[shard.ID()] = n

	n.mu.Lock()
	n.memListElem = el
	n.mu.Unlock()
}

func (t *Balancer) set(resp *model.Response) *list.Element {
	n := t.shards[resp.GetShardKey()]
	if n == nil {
		return nil
	}

	el := t.push(n, resp)
	t.rebalance(n)

	return el
}

func (t *Balancer) push(n *node, resp *model.Response) *list.Element {
	n.mu.Lock()
	defer n.mu.Unlock()
	el := n.lruList.PushFront(resp.GetRequest())
	resp.SetListElement(el)
	return el
}

func (t *Balancer) rebalance(n *node) {
	t.mu.Lock()
	defer t.mu.Unlock()

	n.mu.Lock()
	curr := n.memListElem
	n.mu.Unlock()
	if curr == nil {
		return
	}
	curSize := curr.Value.(*node).shard.Size()
	for prev := curr.Prev(); prev != nil; prev = curr.Prev() {
		if curSize <= prev.Value.(*node).shard.Size() {
			break
		}
		t.swap(curr, prev)
		curr = prev
	}
	for next := curr.Next(); next != nil; next = curr.Next() {
		if curSize >= next.Value.(*node).shard.Size() {
			break
		}
		t.swap(curr, next)
		curr = next
	}
}

func (t *Balancer) move(shardID uint, el *list.Element) {
	n := t.shards[shardID]
	if n == nil {
		return
	}

	n.mu.Lock()
	n.lruList.MoveToFront(el)
	n.mu.Unlock()
}

func (t *Balancer) swap(a, b *list.Element) {
	t.memList.MoveBefore(a, b)
	aNode := a.Value.(*node)
	bNode := b.Value.(*node)

	aNode.mu.Lock()
	aNode.memListElem = a
	aNode.mu.Unlock()

	bNode.mu.Lock()
	bNode.memListElem = b
	bNode.mu.Unlock()
}

func (t *Balancer) remove(key uint64, sharedKey uint) {
	n := t.shards[sharedKey]

	resp, found := t.shardedMap.Del(key)
	if !found {
		return
	}

	t.del(n, resp)
	t.rebalance(n)
}

func (t *Balancer) del(n *node, resp *model.Response) {
	n.mu.Lock()
	n.lruList.Remove(resp.GetListElement())
	n.mu.Unlock()
}

func (t *Balancer) mostLoaded() *sharded.Shard[uint64, *model.Response] {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if front := t.memList.Front(); front != nil {
		return front.Value.(*node).shard
	}
	return nil
}

func (t *Balancer) mostLoadedList(n uintptr) []*node {
	t.mu.RLock()
	defer t.mu.RUnlock()

	shards := make([]*node, 0, n)

	i := 0
	for front := t.memList.Front(); front != nil; front = front.Next() {
		if i > 100 {
			panic("")
		}
		curNode := front.Value.(*node)
		shards = append(shards, curNode)
		i++
	}

	return shards
}

func (t *Balancer) evictBatch(ctx context.Context, n *node, num uintptr) uintptr {
	var evictionsNum uintptr
	for {
		select {
		case <-ctx.Done():
			return evictionsNum
		default:
			n.mu.Lock()
			back := n.lruList.Back()
			if back == nil || evictionsNum > num {
				n.mu.Unlock()
				return evictionsNum
			}
			n.lruList.Remove(back)
			n.mu.Unlock()

			if resp, found := n.shard.Del(back.Value.(*model.Request).UniqueKey()); found {
				model.RequestsPool.Put(resp.GetRequest())
				model.ResponsePool.Put(resp)
				evictionsNum++
			}
			continue
		}
	}
}
