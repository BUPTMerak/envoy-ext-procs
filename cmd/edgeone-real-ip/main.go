package main

import (
	"context"
	"os"

	"github.com/alecthomas/kong"
	"github.com/mnixry/envoy-ext-procs/internal/config"
	"github.com/mnixry/envoy-ext-procs/internal/edgeone"
	edgeoneproc "github.com/mnixry/envoy-ext-procs/internal/extproc/edgeone"
	"github.com/mnixry/envoy-ext-procs/internal/logger"
	"github.com/mnixry/envoy-ext-procs/internal/server"
)

func main() {
	var cli config.EdgeOneCLI
	kong.Parse(&cli,
		kong.Description("Envoy external processor that validates EdgeOne CDN requests and sets real client IP headers."),
		kong.UsageOnError(),
	)

	log := logger.New(cli.Log)

	validator, err := edgeone.New(edgeone.Config{
		SecretID:            cli.EdgeOne.SecretID,
		SecretKey:           cli.EdgeOne.SecretKey,
		APIEndpoint:         cli.EdgeOne.APIEndpoint,
		Region:              cli.EdgeOne.Region,
		CacheSize:           cli.EdgeOne.CacheSize,
		CacheTTL:            cli.EdgeOne.CacheTTL,
		Timeout:             cli.EdgeOne.Timeout,
		IdleConnTimeout:     cli.EdgeOne.IdleConnTimeout,
		MaxIdleConns:        cli.EdgeOne.MaxIdleConns,
		MaxIdleConnsPerHost: cli.EdgeOne.MaxIdleConnsPerHost,
		DialKeepAlive:       cli.EdgeOne.DialKeepAlive,
		WarmInterval:        cli.EdgeOne.WarmInterval,
		WarmTimeout:         cli.EdgeOne.WarmTimeout,
	}, log)
	if err != nil {
		log.Fatal().Err(err).Msg("edgeone validator init failed")
	}

	log.Info().
		Str("api_endpoint", cli.EdgeOne.APIEndpoint).
		Str("region", cli.EdgeOne.Region).
		Int("cache_size", cli.EdgeOne.CacheSize).
		Dur("cache_ttl", cli.EdgeOne.CacheTTL).
		Dur("timeout", cli.EdgeOne.Timeout).
		Dur("warm_interval", cli.EdgeOne.WarmInterval).
		Str("log_output", cli.Log.Output).
		Str("log_format", string(cli.Log.Format)).
		Msg("edgeone validator configured")

	validator.StartWarm(context.Background())

	factory := edgeoneproc.NewProcessorFactory(validator, log)

	if err := server.Run(server.Config{
		GRPCPort:       cli.GRPC.Port,
		CertPath:       cli.GRPC.CertPath,
		CAFile:         cli.GRPC.CAFile,
		HealthPort:     cli.Health.Port,
		DialServerName: cli.Health.DialServerName,
	}, factory, log); err != nil {
		log.Fatal().Err(err).Send()
		os.Exit(1)
	}
}
