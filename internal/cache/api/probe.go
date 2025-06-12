package api

import (
	"github.com/fasthttp/router"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"gitlab.xbet.lan/v3group/backend/packages/go/liveness-prober"
)

var (
	successResponseBytes = []byte(`{
	  "status": 200,
      "message": "I'm fine :D'"
	}`)
	failedResponseBytes = []byte(`{
	  "status": 503,
      "message": "I'm tired :('"
	}`)
)

type LivenessController struct {
	probe liveness.Prober
}

func NewLivenessController(probe liveness.Prober) *LivenessController {
	return &LivenessController{probe: probe}
}

func (c *LivenessController) Probe(ctx *fasthttp.RequestCtx) {
	if c.probe.IsAlive() {
		ctx.SetStatusCode(fasthttp.StatusOK)
		if _, err := ctx.Write(successResponseBytes); err != nil {
			log.Err(err).Msg("[probe-controller] failed to write success response into *fasthttp.RequestCtx")
		}
		return
	}

	ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
	if _, err := ctx.Write(failedResponseBytes); err != nil {
		log.Err(err).Msg("[probe-controller] failed to write failed response into *fasthttp.RequestCtx")
	}
}

func (c *LivenessController) AddRoute(router *router.Router) {
	router.GET("/k8s/probe", c.Probe)
}
