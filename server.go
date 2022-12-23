package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/s3utils"
	"github.com/minio/minio-go/v7/pkg/signer"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

var (
	MethodGet  = []byte(http.MethodGet)
	MethodHead = []byte(http.MethodHead)
)

type serverOptions struct {
	Addr   string
	Logger *zap.Logger
	OBS    *obsOptions

	S3 *minio.Client
}

type server struct {
	opts   serverOptions
	logger *zap.SugaredLogger
}

var srv server

func (s *server) Init(opts serverOptions) {
	s.opts = opts
	s.logger = opts.Logger.Sugar()
}

func (s *server) reportError(ctx *fasthttp.RequestCtx, errType string, err any) {
	s.logger.Errorw("handler error",
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

var (
	ErrKind_ResourceNotFound = "OBS_RESOURCE_NOT_FOUND"
	ErrKind_MethodNotAllowed = "OBS_METHOD_NOT_ALLOWED"
	ErrKind_S3ComposeRequest = "S3_COMPOSE_REQUEST"
	ErrKind_S3CredsProvider  = "S3_CREDS_PROVIDER"
)

func (s *server) handle(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("server", "obs-access-signer")
	isMethodGet := bytes.Equal(ctx.Method(), MethodGet)
	isMethodHead := bytes.Equal(ctx.Method(), MethodHead)
	if !isMethodGet && !isMethodHead {
		ctx.SetStatusCode(http.StatusMethodNotAllowed)
		s.reportError(ctx, ErrKind_MethodNotAllowed, "")
		return
	}

	if isMethodHead {
		// Doc: https://www.rfc-editor.org/rfc/rfc9110.html#section-9.3.2-1
		ctx.Response.Header.Set("Content-Length", "0")
	}

	bucketName := s.opts.OBS.BucketName
	isVirtualHostStyle := isVirtualHostStyleRequest(s.opts.S3, *s.opts.S3.EndpointURL(), bucketName)

	path := ctx.Path()
	_path := bytes.TrimLeft(path, "/")
	objectName := unsafeByteSliceToString(_path)

	s.logger.Debugw("handle",
		"bucket", bucketName,
		"objectName", objectName)

	// check if we had access to the object
	if _, err := s.opts.S3.StatObject(ctx, bucketName, objectName, minio.GetObjectOptions{}); err != nil {
		ctx.SetStatusCode(http.StatusNotFound)
		s.reportError(ctx, ErrKind_ResourceNotFound, err)
		return
	}

	// compose initial request
	expireSeconds := int64(s.opts.OBS.URLExpiry / time.Second)
	req, err := newRequest(s.opts.S3, ctx, http.MethodGet, requestMetadata{
		presignURL:  true,
		bucketName:  bucketName,
		objectName:  objectName,
		expires:     expireSeconds, // to trigger presigned generator
		queryValues: url.Values{},
	})
	if err != nil {
		ctx.SetStatusCode(http.StatusInternalServerError)
		s.reportError(ctx, ErrKind_S3ComposeRequest, err)
		return
	}

	// grab creds from provider
	value, err := getCredsProvider(s.opts.S3).Get()
	if err != nil {
		ctx.SetStatusCode(http.StatusInternalServerError)
		s.reportError(ctx, ErrKind_S3CredsProvider, err)
		return
	}

	var statusCode = s.opts.OBS.RedirectCode

	// custom "expiry"
	var exp string
	if expiry := s.opts.OBS.URLExpiry; expiry == maxURLExpiry || expiry <= 0 {
		// clear given params, set max signed value for expire, and re-presign.
		exp = strconv.FormatInt(int64(^uint64(0)/2), 10) // ~250years
	} else {
		// we can't allow a permanent redirect here since we already have
		// expiry set, the redirected url needs to be updated.
		if statusCode == http.StatusMovedPermanently || (statusCode < 300 || statusCode > 399) {
			statusCode = http.StatusTemporaryRedirect
		}

		expireAt := time.Now().UTC().Add(s.opts.OBS.URLExpiry)
		exp = strconv.FormatInt(int64(expireAt.Unix()), 10)
		// set object cache lifetime
		if statusCode == http.StatusTemporaryRedirect {
			ctx.Response.Header.Set("Cache-Control", fmt.Sprintf("max-age=%d", expireSeconds))
			ctx.Response.Header.Set("Expires", expireAt.Format("Mon, 02 Jan 2006 15:04:05 GMT"))
		}
	}
	req.Header.Set("Expires", exp)
	req.URL.RawQuery = ""
	req = signer.PreSignV2(*req, value.AccessKeyID, value.SecretAccessKey, 0, isVirtualHostStyle)

	// re-encode query string with Expires hack.
	query := req.URL.Query()
	query.Set("Expires", exp)
	req.URL.RawQuery = s3utils.QueryEncode(query)

	if s.opts.OBS.RedirectSecure {
		req.URL.Scheme = "https"
	} else {
		req.URL.Scheme = "http"
	}

	if hostRedirect := s.opts.OBS.HostRedirect; hostRedirect != "" {
		req.URL.Host = hostRedirect
	}

	ctx.Redirect(req.URL.String(), statusCode)
}

func (s *server) Run() {
	s.logger.Infow("running server",
		"addr", s.opts.Addr)
	fasthttp.ListenAndServe(s.opts.Addr, s.handle)
}
