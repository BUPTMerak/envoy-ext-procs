package config

import "time"

// EdgeOneCLI is the CLI configuration for the EdgeOne real IP processor.
type EdgeOneCLI struct {
	GRPC    GRPCConfig    `embed:"" prefix:"grpc-" envprefix:"GRPC_"`
	Health  HealthConfig  `embed:"" prefix:"health-" envprefix:"HEALTH_"`
	EdgeOne EdgeOneConfig `embed:"" prefix:"edgeone-" envprefix:"EDGEONE_"`
	Log     LogConfig     `embed:"" prefix:"log-" envprefix:"LOG_"`
}

// EdgeOneConfig holds EdgeOne API configuration.
type EdgeOneConfig struct {
	SecretID    string        `name:"secret-id" env:"SECRET_ID" required:"" help:"Tencent Cloud SecretId for TEO API."`
	SecretKey   string        `name:"secret-key" env:"SECRET_KEY" required:"" help:"Tencent Cloud SecretKey for TEO API."`
	APIEndpoint string        `name:"api-endpoint" env:"API_ENDPOINT" default:"teo.tencentcloudapi.com" help:"Tencent EdgeOne TEO API endpoint (hostname or URL)."`
	Region      string        `name:"region" env:"REGION" default:"" help:"Tencent Cloud region for TEO client (optional)."`
	CacheSize   int           `name:"cache-size" env:"CACHE_SIZE" default:"1000" help:"LRU cache size for IP validation results."`
	CacheTTL    time.Duration `name:"cache-ttl" env:"CACHE_TTL" default:"1h" help:"Cache TTL for IP validation results (e.g. 1h, 30m)."`
	Timeout     time.Duration `name:"timeout" env:"TIMEOUT" default:"5s" help:"Tencent API request timeout (e.g. 5s, 10s)."`

	IdleConnTimeout     time.Duration `name:"http-idle-conn-timeout" env:"HTTP_IDLE_CONN_TIMEOUT" default:"90s" help:"HTTP idle connection timeout for the TEO API client."`
	MaxIdleConns        int           `name:"http-max-idle-conns" env:"HTTP_MAX_IDLE_CONNS" default:"2" help:"Maximum number of idle HTTP connections to the TEO API."`
	MaxIdleConnsPerHost int           `name:"http-max-idle-conns-per-host" env:"HTTP_MAX_IDLE_CONNS_PER_HOST" default:"2" help:"Maximum idle HTTP connections per host for the TEO API."`
	DialKeepAlive       time.Duration `name:"http-dial-keepalive" env:"HTTP_DIAL_KEEPALIVE" default:"30s" help:"TCP keepalive interval for TEO API connections."`

	WarmInterval time.Duration `name:"warm-interval" env:"WARM_INTERVAL" default:"20s" help:"Interval between warm probe calls to keep TEO API connection hot (0 to disable)."`
	WarmTimeout  time.Duration `name:"warm-timeout" env:"WARM_TIMEOUT" default:"5s" help:"Per-probe request timeout for warm calls (0 uses the global timeout)."`
}
