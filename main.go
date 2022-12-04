package main

import (
	"flag"
	"os"
	"strconv"

	_ "github.com/joho/godotenv/autoload"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	httpAddr string
	logLevel string
	// obsSignedUrlExpiry time.Duration
	zapLogLevel   zapcore.Level
	postFlagParse = []func(){}
)

func init() {
	var err error
	_ = err

	// app
	flag.StringVar(&httpAddr, "addr", os.Getenv("HTTP_ADDR"), "Server address")

	// log
	flag.StringVar(&logLevel, "log-level", os.Getenv("LOG_LEVEL"), "Log level")
	qpostFlagParse(func() {
		err := zapLogLevel.UnmarshalText([]byte(logLevel))
		if err != nil {
			zapLogLevel = zapcore.InfoLevel
		}
	})

	// OBS
	flag.StringVar(&defaultObsOpts.Endpoint, "obs-endpoint", os.Getenv("OBS_ENDPOINT"), "OBS host")
	flag.StringVar(&defaultObsOpts.Region, "obs-region", os.Getenv("OBS_REGION"), "OBS region")
	flag.BoolVar(&defaultObsOpts.Secure, "obs-secure", ok1(strconv.ParseBool(os.Getenv("OBS_SECURE"))), "OBS secure transport")
	flag.StringVar(&defaultObsOpts.BucketName, "obs-bucket", os.Getenv("OBS_BUCKET_NAME"), "OBS bucket name")

	flag.StringVar(&defaultObsOpts.HostRedirect, "obs-host-redirect", os.Getenv("OBS_HOST_REDIRECT"), "OBS host redirect")

	// obsSignedUrlExpiry, err = time.ParseDuration(os.Getenv("OBS_SIGNED_URL_EXPIRY"))
	// if err != nil {
	// 	// max signed value
	// 	obsSignedUrlExpiry = time.Duration(^uint64(0) / 2)
	// }
	// flag.DurationVar(&obsSignedUrlExpiry, "obs-signed-url-expiry", obsSignedUrlExpiry, "OBS ")
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
		"obs_endpoint", defaultObsOpts.Endpoint,
		"obs_host_redirect", defaultObsOpts.HostRedirect,
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
