package liveness

import (
	"context"
	"encoding/json"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"gitlab.xbet.lan/v3group/backend/packages/go/logger/pkg/logger"
	"net/http"
)

const K8SProbeGetPath = "/k8s/probe"

type Controller struct {
	ctx    context.Context
	logger logger.Logger
	prober Prober
}

func NewController(ctx context.Context, logger logger.Logger, prober Prober) *Controller {
	return &Controller{ctx: ctx, logger: logger, prober: prober}
}

func (c *Controller) Probe(ctx *fasthttp.RequestCtx) {
	reqCtx, ok := ctx.UserValue(CtxKey).(context.Context)
	if !ok {
		c.logger.ErrorMsg(c.ctx, "context.Context is not exists into the fasthttp.RequestCtx "+
			"(unable to handle request)", nil)

		ctx.SetStatusCode(fasthttp.StatusInternalServerError)

		resp := make(map[string]map[string]string, 1)
		resp["error"] = make(map[string]string, 1)
		resp["error"]["message"] = "Internal server error"

		b, err := json.Marshal(resp)
		if err != nil {
			c.logger.ErrorMsg(reqCtx, "unable to handle request,"+
				" error occurred while marshaling data into []byte", nil)
			return
		}
		if _, err = ctx.Write(b); err != nil {
			c.logger.ErrorMsg(reqCtx, "unable to handle request,"+
				" error occurred while writing data into *fasthttp.RequestCtx", nil)
			return
		}

		return
	}

	isAlive := c.prober.IsAlive()

	resp := make(map[string]map[string]bool, 1)
	resp["data"] = make(map[string]bool, 1)
	resp["data"]["success"] = isAlive

	b, err := json.Marshal(resp)
	if err != nil {
		c.logger.ErrorMsg(reqCtx, "unable to handle request,"+
			" error occurred while marshaling data into []byte", nil)
		return
	}

	if _, err = ctx.Write(b); err != nil {
		c.logger.ErrorMsg(reqCtx, "unable to handle request,"+
			" error occurred while writing data into *fasthttp.RequestCtx", nil)
		return
	}

	if !isAlive {
		ctx.Response.SetStatusCode(http.StatusInternalServerError)
	}
}

func (c *Controller) AddRoute(router *router.Router) {
	router.GET(K8SProbeGetPath, c.Probe)
}
