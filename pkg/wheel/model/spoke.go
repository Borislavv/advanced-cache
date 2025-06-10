package wheelmodel

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/list"
)

type Spoke interface {
	Touch()
	Key() uint64
	ToQuery() []byte
	ShardKey() uint64
	ShouldRefresh() bool
	NativeRevalidatedAt() int64
	NativeRevalidateInterval() int64
	Revalidate(ctx context.Context) error
	WheelListElement() *list.LockFreeDoublyLinkedListElement[Spoke]
	StoreWheelListElement(el *list.LockFreeDoublyLinkedListElement[Spoke])
}
