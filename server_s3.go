package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/s3utils"
	"github.com/minio/minio-go/v7/pkg/signer"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

type serverS3 struct {
	opts   obsOptions
	s3opts obsS3Options

	logger *zap.SugaredLogger

	s3c *minio.Client
}

func (s *serverS3) Init(ctx context.Context, opts serverOptions) (err error) {
	s.opts = opts.GetOpts()
	s.s3opts = opts.GetS3Opts()

	s.logger = opts.Logger.Named(s.Name()).Sugar()

	if s.s3c, err = newObsS3Client(s.s3opts); err != nil {
		err = errors.Wrap(err, "obs s3 client")
		return
	}

	return
}

func (s *serverS3) Name() string {
	return "s3"
}

func (s *serverS3) getLogger() *zap.SugaredLogger { return s.logger }
func (s *serverS3) reportError(ctx *fasthttp.RequestCtx, errType string, err any) {
	reportError(s, ctx, errType, err)
}

var (
	ErrKind_S3ComposeRequest = "S3_COMPOSE_REQUEST"
	ErrKind_S3CredsProvider  = "S3_CREDS_PROVIDER"
)

func (s *serverS3) handle(ctx *fasthttp.RequestCtx) {
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

	bucketName := s.opts.BucketName
	isVirtualHostStyle := isVirtualHostStyleRequest(s.s3c, *s.s3c.EndpointURL(), bucketName)

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

	// check if we had access to the object
	if meta, err := s.s3c.StatObject(ctx, bucketName, objectName, minio.GetObjectOptions{}); err != nil {
		ctx.SetStatusCode(http.StatusNotFound)
		s.reportError(ctx, ErrKind_ResourceNotFound, err)
		return
	} else {
		_ = meta
	}

	// compose initial request
	expireSeconds := int64(s.opts.URLExpiry / time.Second)
	req, err := newRequest(s.s3c, ctx, http.MethodGet, requestMetadata{
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
	value, err := getCredsProvider(s.s3c).Get()
	if err != nil {
		ctx.SetStatusCode(http.StatusInternalServerError)
		s.reportError(ctx, ErrKind_S3CredsProvider, err)
		return
	}

	var statusCode = s.opts.RedirectCode

	// custom "expiry"
	var exp string
	if expiry := s.opts.URLExpiry; expiry == maxURLExpiry || expiry <= 0 {
		// clear given params, set max signed value for expire, and re-presign.
		exp = strconv.FormatInt(int64(^uint64(0)/2), 10) // ~250years
	} else {
		// we can't allow a permanent redirect here since we already have
		// expiry set, the redirected url needs to be updated.
		if statusCode == http.StatusMovedPermanently || (statusCode < 300 || statusCode > 399) {
			statusCode = http.StatusTemporaryRedirect
		}

		expireAt := time.Now().UTC().Add(s.opts.URLExpiry)
		exp = strconv.FormatInt(int64(expireAt.Unix()), 10)
		// set redirect cache lifetime
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

	if s.opts.RedirectSecure {
		req.URL.Scheme = "https"
	} else {
		req.URL.Scheme = "http"
	}

	if hostRedirect := s.opts.HostRedirect; hostRedirect != "" {
		req.URL.Host = hostRedirect
	}

	ctx.Redirect(req.URL.String(), statusCode)
}

func (s *serverS3) GetHandler() fasthttp.RequestHandler {
	return s.handle
}
