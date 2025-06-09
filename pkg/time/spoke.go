package time

import (
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/list"
	"sync/atomic"
	"time"
)

type Spoke struct {
	key               uint64
	shardKey          uint64
	interval          int64 // duration:  UnixNano
	mustBeRefreshedAt int64 // timestamp: UnixNano
	element           *atomic.Pointer[list.Element[*Spoke]]
}

func (s *Spoke) IsReady() bool {
	return time.Unix(0, atomic.LoadInt64(&s.mustBeRefreshedAt)).After(time.Now())
}
