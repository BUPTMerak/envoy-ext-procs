package edgeone

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	teo "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/teo/v20220901"
	"golang.org/x/sync/singleflight"
)

type Config struct {
	SecretID    string
	SecretKey   string
	APIEndpoint string
	Region      string
	CacheSize   int
	CacheTTL    time.Duration
	Timeout     time.Duration

	IdleConnTimeout     time.Duration
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	DialKeepAlive       time.Duration

	WarmInterval time.Duration
	WarmTimeout  time.Duration
}

type Validator struct {
	cache        *expirable.LRU[string, bool]
	client       *teo.Client
	sg           singleflight.Group
	log          zerolog.Logger
	warmInterval time.Duration
	warmTimeout  time.Duration
}

func New(cfg Config, log zerolog.Logger) (*Validator, error) {
	if strings.TrimSpace(cfg.SecretID) == "" || strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, oops.
			In("edgeone").
			Code("MISSING_CREDENTIALS").
			Errorf("missing SecretID or SecretKey")
	}

	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = cfg.APIEndpoint
	cpf.HttpProfile.ReqTimeout = int(cfg.Timeout.Seconds())
	cpf.UnsafeRetryOnConnectionFailure = true

	credential := common.NewCredential(cfg.SecretID, cfg.SecretKey)
	client, err := teo.NewClient(credential, cfg.Region, cpf)
	if err != nil {
		return nil, oops.
			In("edgeone").
			Code("CLIENT_INIT_FAILED").
			With("region", cfg.Region).
			With("endpoint", cfg.APIEndpoint).
			Wrapf(err, "failed to create tencent teo client")
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: cfg.DialKeepAlive,
		}).DialContext,
		MaxIdleConns:          cfg.MaxIdleConns,
		MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
		IdleConnTimeout:       cfg.IdleConnTimeout,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	client.WithHttpTransport(transport)

	return &Validator{
		cache:        expirable.NewLRU[string, bool](cfg.CacheSize, nil, cfg.CacheTTL),
		client:       client,
		log:          log.With().Str("component", "edgeone").Logger(),
		warmInterval: cfg.WarmInterval,
		warmTimeout:  cfg.WarmTimeout,
	}, nil
}

func (v *Validator) IsEdgeOneIP(ctx context.Context, ip netip.Addr) (bool, error) {
	ip = ip.Unmap()
	if cached, ok := v.cache.Get(ip.String()); ok {
		return cached, nil
	}
	start := time.Now()
	return v.fetchAndCache(ctx, ip, func(valid bool) {
		v.log.Info().
			Dur("duration", time.Since(start)).
			Str("ip", ip.String()).
			Bool("valid", valid).
			Msg("IP validation completed")
		v.cache.Add(ip.String(), valid)
	})
}

// fetchAndCache validates an IP through singleflight (deduplicating concurrent
// lookups for the same address) and writes the result into the cache.
func (v *Validator) fetchAndCache(ctx context.Context, ip netip.Addr, postValidate func(bool)) (bool, error) {
	// EdgeOne IPs are public; private/loopback can never be EdgeOne.
	if !ip.IsGlobalUnicast() || ip.IsPrivate() {
		return false, nil
	}
	ipStr := ip.String()
	val, err, _ := v.sg.Do(ipStr, func() (any, error) {
		valid, err := v.validateIP(ctx, ip)
		if err != nil {
			return false, err
		}
		postValidate(valid)
		return valid, nil
	})
	return val.(bool), err
}

func (v *Validator) validateIP(ctx context.Context, ip netip.Addr) (bool, error) {
	// EdgeOne IPs are public; private/loopback can never be EdgeOne.
	if !ip.IsGlobalUnicast() || ip.IsPrivate() {
		return false, nil
	}

	req := teo.NewDescribeIPRegionRequest()
	req.IPs = []*string{new(ip.String())}

	resp, err := v.client.DescribeIPRegionWithContext(ctx, req)
	if err != nil {
		return false, oops.
			In("edgeone").
			Code("API_REQUEST_FAILED").
			With("ip", ip.String()).
			Wrapf(err, "failed to describe IP region")
	}

	validated := slices.ContainsFunc(resp.Response.IPRegionInfo, func(info *teo.IPRegionInfo) bool {
		return strings.EqualFold(*info.IsEdgeOneIP, "yes")
	})
	v.log.Debug().
		Str("ip", ip.String()).
		Bool("valid", validated).
		Interface("request", req).
		Interface("response", resp).
		Msg("IP region validation result")
	return validated, nil
}
