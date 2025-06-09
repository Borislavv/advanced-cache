package time

import (
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage/list"
	"time"
	"unsafe"
)

type spoke struct {
	key               uint64
	shardKey          uint64
	element           *list.Element[*spoke]
	interval          time.Duration
	mustBeRefreshedAt time.Time
}

func (s *spoke) IsReady() bool {
	return s.mustBeRefreshedAt.After(time.Now())
}

func (s *spoke) Memory() uintptr {
	return unsafe.Sizeof(s)
}
