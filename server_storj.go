package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"storj.io/uplink/edge"
)

type serverStorj struct {
	opts   obsOptions
	logger *zap.SugaredLogger

	sc *storjAggegrateClient
}

func (s *serverStorj) Init(ctx context.Context, opts serverOptions) (err error) {
	s.opts = opts.GetOpts()
	s.logger = opts.Logger.Named(s.Name()).Sugar()
	{
		if s.sc, err = newObsStorjClient(ctx, opts.GetUplinkOpts()); err != nil {
			err = errors.Wrap(err, "obs uplink client")
			return
		}
	}
	return
}

func (s *serverStorj) Name() string {
	return "storj"
}

func (s *serverStorj) getLogger() *zap.SugaredLogger { return s.logger }
func (s *serverStorj) reportError(ctx *fasthttp.RequestCtx, errType string, err any) {
	reportError(s, ctx, errType, err)
}

var (
	ErrKind_StorjComposeShareURL = "STORJ_COMPOSE_SHARE_URL"
)

func (s *serverStorj) handle(ctx *fasthttp.RequestCtx) {
	isMethodGet := bytes.Equal(ctx.Method(), MethodGet)
	isMethodHead := bytes.Equal(ctx.Method(), MethodHead)
	if !(isMethodGet || isMethodHead) {
		ctx.SetStatusCode(http.StatusMethodNotAllowed)
		s.reportError(ctx, ErrKind_MethodNotAllowed, "")
		return
	}

	if isMethodHead {
		// Doc: https://www.rfc-editor.org/rfc/rfc9110.html#section-9.3.2-1
		ctx.Response.Header.Set("Content-Length", "0")
	}

	bucketName := s.opts.BucketName
	path := ctx.Path()
	_path := bytes.TrimLeft(path, "/")
	if s.opts.RemoveBucketName {
		if _, _pathWithoutBucketName, found := bytes.Cut(_path, []byte(`/`)); found {
			// no need to check `isVirtualHostStyle` since this is our own implementation of handling request URI
			_path = _pathWithoutBucketName
		}
	}
	objectName := unsafeByteSliceToString(_path)

	s.logger.Debugw("handle",
		"bucket", bucketName,
		"objectName", objectName)

	// use project
	if project := s.sc.getProject(); project != nil {
		// check if we had access to the object
		if meta, err := project.StatObject(ctx, bucketName, objectName); err != nil {
			ctx.SetStatusCode(http.StatusNotFound)
			s.reportError(ctx, ErrKind_ResourceNotFound, err)
			return
		} else {
			_ = meta
		}
	}

	shareURL, err := s.sc.JoinShareURL(bucketName, objectName, &edge.ShareURLOptions{
		Raw: true,
	})
	if err != nil {
		ctx.SetStatusCode(http.StatusInternalServerError)
		s.reportError(ctx, ErrKind_StorjComposeShareURL, err)
	}

	var statusCode = s.opts.RedirectCode

	if statusCode < 300 || statusCode >= 399 {
		// fallback of invalid redirect code
		statusCode = http.StatusTemporaryRedirect
	}

	expireAt := time.Now().UTC().Add(s.opts.URLExpiry)
	expireSeconds := int64(s.opts.URLExpiry / time.Second)
	// set redirect cache lifetime
	if statusCode == http.StatusTemporaryRedirect {
		ctx.Response.Header.Set("Cache-Control", fmt.Sprintf("max-age=%d", expireSeconds))
		ctx.Response.Header.Set("Expires", expireAt.Format("Mon, 02 Jan 2006 15:04:05 GMT"))
	}

	ctx.Redirect(shareURL, s.opts.RedirectCode)
}

func (s *serverStorj) GetHandler() fasthttp.RequestHandler {
	return s.handle
}
