package main

import (
	"flag"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	httpAddr      string
	logLevel      string
	zapLogLevel   zapcore.Level
	postFlagParse = []func(){}
)

func init() {
	var err error
	_ = err

	// -- app
	flag.StringVar(&httpAddr, "addr", os.Getenv("HTTP_ADDR"), "Server address")

	// -- log
	flag.StringVar(&logLevel, "log-level", os.Getenv("LOG_LEVEL"), "Log level")
	qpostFlagParse(func() {
		err := zapLogLevel.UnmarshalText([]byte(logLevel))
		if err != nil {
			zapLogLevel = zapcore.InfoLevel
		}
	})

	// -- OBS
	flag.StringVar(&defaultObsOpts.Endpoint, "obs-endpoint", os.Getenv("OBS_ENDPOINT"), "OBS host")
	flag.StringVar(&defaultObsOpts.Region, "obs-region", os.Getenv("OBS_REGION"), "OBS region")
	flag.BoolVar(&defaultObsOpts.Secure, "obs-secure", ok1(strconv.ParseBool(os.Getenv("OBS_SECURE"))), "OBS secure transport")
	flag.StringVar(&defaultObsOpts.BucketName, "obs-bucket", os.Getenv("OBS_BUCKET_NAME"), "OBS bucket name")

	flag.BoolVar(&defaultObsOpts.RedirectSecure, "obs-redirect-secure", ok1(strconv.ParseBool(os.Getenv("OBS_REDIRECT_SECURE"))), "OBS redirect secure transport")
	flag.StringVar(&defaultObsOpts.HostRedirect, "obs-host-redirect", os.Getenv("OBS_HOST_REDIRECT"), "OBS host redirect")

	// redirect http code
	var obsRedirectCode = int64(defaultObsOpts.RedirectCode)
	if obsRedirectCodeStr := os.Getenv("OBS_REDIRECT_CODE"); obsRedirectCodeStr != "" {
		obsRedirectCode, err = strconv.ParseInt(obsRedirectCodeStr, 10, 64)
		if err != nil {
			obsRedirectCode = int64(defaultObsOpts.RedirectCode)
		}
	}
	flag.IntVar(&defaultObsOpts.RedirectCode, "obs-redirect-code", int(obsRedirectCode), "OBS redirect http code")

	// url expiry
	var obsUrlExpiry = defaultObsOpts.URLExpiry
	if obsUrlExpiryStr := os.Getenv("OBS_URL_EXPIRY"); obsUrlExpiryStr != "" {
		if obsUrlExpiry, err = time.ParseDuration(obsUrlExpiryStr); err != nil {
			obsUrlExpiry = defaultObsOpts.URLExpiry
		}
	}
	flag.DurationVar(&defaultObsOpts.URLExpiry, "obs-url-expiry", obsUrlExpiry, "OBS url expiry")

}

func qpostFlagParse(f func()) {
	postFlagParse = append(postFlagParse, f)
}
func qpostFlagParseInvoke() {
	for _, f := range postFlagParse {
		f()
	}
}

func main() {
	flag.Parse()
	qpostFlagParseInvoke()

	zcfg := zap.NewProductionConfig()
	zcfg.Level = zap.NewAtomicLevelAt(zapLogLevel)

	logger := unwrap1(zcfg.Build())
	defer logger.Sync()

	sug := logger.Named("main").Sugar()
	sug.Infow("starting",
		"log_level", zapLogLevel,
		"obs_bucket", defaultObsOpts.BucketName,
		"obs_endpoint", defaultObsOpts.Endpoint,
		"obs_redirect_secure", defaultObsOpts.RedirectSecure,
		"obs_host_redirect", defaultObsOpts.HostRedirect,
		"obs_redirect_code", defaultObsOpts.RedirectCode,
		"obs_url_expiry", defaultObsOpts.URLExpiry.String(),
	)

	client := unwrap1(newObsClient(defaultObsOpts))
	srv.Init(serverOptions{
		Addr:   httpAddr,
		Logger: logger.Named("server"),
		OBS:    &defaultObsOpts,
		S3:     client,
	})

	srv.Run()
}
