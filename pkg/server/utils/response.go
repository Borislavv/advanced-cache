package serverutils

import (
	"errors"
	"github.com/valyala/fasthttp"
)

var WriteResponseMsg = "error occurred while writing data into *fasthttp.RequestCtx"

func Write(b []byte, ctx *fasthttp.RequestCtx) (int, error) {
	n, err := ctx.Write(b)
	if err != nil {
		return 0, errors.New(WriteResponseMsg + " (" + err.Error() + ")")
	}
	return n, nil
}

func WriteString(s string, ctx *fasthttp.RequestCtx) (int, error) {
	n, err := ctx.WriteString(s)
	if err != nil {
		return 0, errors.New(WriteResponseMsg + " (" + err.Error() + ")")
	}
	return n, nil
}
