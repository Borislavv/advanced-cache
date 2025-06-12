package middleware

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/prometheus/metrics"
	"github.com/valyala/fasthttp"
	"strconv"
)

type PrometheusMetrics struct {
	ctx     context.Context
	metrics metrics.Meter
}

func NewPrometheusMetrics(ctx context.Context, metrics metrics.Meter) *PrometheusMetrics {
	return &PrometheusMetrics{ctx: ctx, metrics: metrics}
}

func (m *PrometheusMetrics) Middleware(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		timer := m.metrics.NewResponseTimeTimer(string(ctx.Path()), string(ctx.Method()))

		m.metrics.IncTotal(string(ctx.Path()), string(ctx.Method()), "")

		next(ctx)

		m.metrics.IncStatus(string(ctx.Path()), string(ctx.Method()), strconv.Itoa(ctx.Response.StatusCode()))
		m.metrics.IncTotal(string(ctx.Path()), string(ctx.Method()), strconv.Itoa(ctx.Response.StatusCode()))

		m.metrics.FlushResponseTimeTimer(timer)
	}
}
