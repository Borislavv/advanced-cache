package controller

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/prometheus/metrics/adapter"
	"github.com/fasthttp/router"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp"
)

const PrometheusMetricsPath = "/metrics"

type PrometheusMetrics struct {
	ctx context.Context
}

func NewPrometheusMetrics(ctx context.Context) *PrometheusMetrics {
	return &PrometheusMetrics{ctx: ctx}
}

func (m *PrometheusMetrics) Get(ctx *fasthttp.RequestCtx) {
	adapter.NewFastHTTPHandlerFunc(promhttp.Handler())(ctx)
}

func (m *PrometheusMetrics) AddRoute(router *router.Router) {
	router.GET(PrometheusMetricsPath, m.Get)
}
