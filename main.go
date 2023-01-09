package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	_ "github.com/joho/godotenv/autoload"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	httpAddr      = ":9003"
	logLevel      = "INFO"
	serverMode    = "s3"
	zapLogLevel   zapcore.Level
	postFlagParse = []func(){}

	registeredServers = []Server{
		&serverS3{},
		&serverStorj{},
	}
	mappedServers = map[string]Server{}
)

func init() {
	var err error
	_ = err

	/* --- [preload] --- */
	var availableServerNames []string
	for _, s := range registeredServers {
		serverName := s.Name()
		if old, exist := mappedServers[serverName]; exist {
			panic(fmt.Sprintf("duplicate server name old: %T, new: %T", old, s))
		}
		mappedServers[serverName] = s
		availableServerNames = append(availableServerNames, serverName)
	}

	/* --- app --- */
	var vHttpAddr = httpAddr
	if sHttpAddr := os.Getenv("HTTP_ADDR"); sHttpAddr != "" {
		vHttpAddr = sHttpAddr
	}
	flag.StringVar(&httpAddr, "addr", vHttpAddr, "Server address")

	var vServerMode = serverMode
	if sServerMode := os.Getenv("SERVER_MODE"); sServerMode != "" {
		vServerMode = sServerMode
	}
	flag.StringVar(&serverMode, "server", vServerMode,
		fmt.Sprintf("Server mode (available [%s])", strings.Join(availableServerNames, ", ")))
	qpostFlagParse(func() {
		if httpAddr == "" {
			httpAddr = ":9003"
		}
		if serverMode == "" {
			serverMode = "s3"
		}
	})

	/* --- log --- */
	var vLogLevel = logLevel
	if sLogLevel := os.Getenv("LOG_LEVEL"); sLogLevel != "" {
		vLogLevel = sLogLevel
	}
	flag.StringVar(&logLevel, "log-level", vLogLevel, "Log level")
	qpostFlagParse(func() {
		err := zapLogLevel.UnmarshalText([]byte(logLevel))
		if err != nil {
			zapLogLevel = zapcore.InfoLevel
		}
	})

	/* --- OBS --- */
	if err = defaultObsOpts.Bind(flag.CommandLine); err != nil {
		panic(err)
	}

	/* --- OBS S3 --- */
	if err = defaultObsS3Opts.Bind(flag.CommandLine); err != nil {
		panic(err)
	}

	/* --- OBS Storj (via LibUplink) --- */
	if err = defaultObsUplinkOpts.Bind(flag.CommandLine); err != nil {
		panic(err)
	}
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
		"server_mode", serverMode,
		// Generic OBS
		"obs_bucket", defaultObsOpts.BucketName,
		"obs_redirect_secure", defaultObsOpts.RedirectSecure,
		"obs_host_redirect", defaultObsOpts.HostRedirect,
		"obs_redirect_code", defaultObsOpts.RedirectCode,
		"obs_url_expiry", defaultObsOpts.URLExpiry.String(),
		// S3
		"obs_s3_endpoint", defaultObsS3Opts.Endpoint,
		// Storj (via LibUplink)
		"obs_storj_satellite_addr", defaultObsUplinkOpts.SatelliteAddress,
	)

	// lookup server mode handler
	srv, exist := mappedServers[serverMode]
	if !exist || srv == nil {
		sug.Fatalw("unknown server handler",
			"server_mode", serverMode)
	}

	// run http server
	RunServer(context.Background(),
		srv,
		serverOptions{
			Addr:       httpAddr,
			Logger:     logger.Named("server"),
			Opts:       &defaultObsOpts,
			S3Opts:     &defaultObsS3Opts,
			UplinkOpts: &defaultObsUplinkOpts,
		})
}
