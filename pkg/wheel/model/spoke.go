package wheelmodel

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/list"
)

type Spoke interface {
	Key() uint64
	ToQuery() []byte
	ShardKey() uint64
	ShouldRefresh() bool
	NativeRevalidatedAt() int64
	NativeRevalidateInterval() int64
	Revalidate(ctx context.Context) error
	WheelListElement() *list.Element[Spoke]
	StoreWheelListElement(el *list.Element[Spoke])
}
