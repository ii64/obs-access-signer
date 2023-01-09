package main

import (
	"context"
	"net/http"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

var (
	ErrKind_ResourceNotFound = "OBS_RESOURCE_NOT_FOUND"
	ErrKind_MethodNotAllowed = "OBS_METHOD_NOT_ALLOWED"
)

var (
	MethodGet  = []byte(http.MethodGet)
	MethodHead = []byte(http.MethodHead)
)

type serverOptions struct {
	Addr   string
	Logger *zap.Logger

	Opts       *obsOptions
	S3Opts     *obsS3Options
	UplinkOpts *obsStorjOptions
}

func (s *serverOptions) GetOpts() obsOptions {
	if s.Opts == nil {
		return defaultObsOpts
	}
	return *s.Opts
}

func (s *serverOptions) GetS3Opts() obsS3Options {
	if s.S3Opts == nil {
		return defaultObsS3Opts
	}
	return *s.S3Opts
}

func (s *serverOptions) GetUplinkOpts() obsStorjOptions {
	if s.UplinkOpts == nil {
		return defaultObsUplinkOpts
	}
	return *s.UplinkOpts
}

func reportError(self interface {
	getLogger() *zap.SugaredLogger
}, ctx *fasthttp.RequestCtx, errType string, err any) {
	self.getLogger().Errorw("handler error",
		"kind", errType,
		"err", err)
	ctx.Response.Header.Set("x-error-code", errType)
	switch errVal := err.(type) {
	case []byte:
		ctx.Response.Header.Set("x-error-message", unsafeByteSliceToString(errVal))
	case string:
		ctx.Response.Header.Set("x-error-message", errVal)
	case error:
		ctx.Response.Header.Set("x-error-message", errVal.Error())
	default:
		ctx.Response.Header.Set("x-error-message", "unknown error")
	}
}

type Server interface {
	Init(ctx context.Context, opts serverOptions) (err error)
	Name() string
	GetHandler() fasthttp.RequestHandler
}

func RunServer(ctx context.Context, s Server, opts serverOptions) {
	sug := opts.Logger.Sugar()
	s.Init(ctx, opts)
	sug.Infow("running server",
		"addr", opts.Addr)
	handler := s.GetHandler()
	fasthttp.ListenAndServe(opts.Addr, func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.Set("server", "obs-access-signer")
		handler(ctx)
	})
}
