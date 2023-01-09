package main

import (
	"flag"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

type obsOptions struct {
	BucketName     string
	RedirectSecure bool
	RedirectCode   int // HTTP redirect status code
	URLExpiry      time.Duration
	HostRedirect   string
}

var defaultObsOpts = obsOptions{
	URLExpiry:    maxURLExpiry,
	RedirectCode: http.StatusMovedPermanently, // 301
}

func (opts *obsOptions) Bind(fs *flag.FlagSet) (err error) {

	var vBucketName = opts.BucketName
	if sBucketName := os.Getenv("OBS_BUCKET_NAME"); sBucketName != "" {
		vBucketName = sBucketName
	}
	fs.StringVar(&opts.BucketName, "obs-bucket", vBucketName, "OBS Bucket name")

	var vRedirectSecure = opts.RedirectSecure
	if sRedirectSecure := os.Getenv("OBS_REDIRECT_SECURE"); sRedirectSecure != "" {
		vRedirectSecure, _ = strconv.ParseBool(sRedirectSecure)
	}
	fs.BoolVar(&opts.RedirectSecure, "obs-redirect-secure", vRedirectSecure, "OBS Redirect secure")

	var vObsHostRedirect = opts.HostRedirect
	if sObsHostRedirect := os.Getenv("OBS_HOST_REDIRECT"); sObsHostRedirect != "" {
		vObsHostRedirect = sObsHostRedirect
	}
	fs.StringVar(&opts.HostRedirect, "obs-host-redirect", vObsHostRedirect, "OBS Host redirect")

	var vObsRedirectCode = opts.RedirectCode
	if sObsRedirectCode := os.Getenv("OBS_REDIRECT_CODE"); sObsRedirectCode != "" {
		var obsRedirectCode int64
		if obsRedirectCode, err = strconv.ParseInt(sObsRedirectCode, 10, 64); err != nil {
			err = errors.Wrap(err, "obs redirect code")
			return
		}
		vObsRedirectCode = int(obsRedirectCode)
	}
	fs.IntVar(&opts.RedirectCode, "obs-redirect-code", vObsRedirectCode, "OBS Redirect code")

	var vObsUrlExpiry = opts.URLExpiry
	if sObsUrlExpiry := os.Getenv("OBS_URL_EXPIRY"); sObsUrlExpiry != "" {
		var obsUrlExpiry time.Duration
		if obsUrlExpiry, err = time.ParseDuration(sObsUrlExpiry); err != nil {
			err = errors.Wrap(err, "obs url expiry")
			return
		}
		vObsUrlExpiry = obsUrlExpiry
	}
	fs.DurationVar(&opts.URLExpiry, "obs-url-expiry", vObsUrlExpiry, "OBS Redirection URL expiry")
	return
}
