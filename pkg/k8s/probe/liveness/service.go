package liveness

import "context"

type Service interface {
	IsAlive(ctx context.Context) bool
}
