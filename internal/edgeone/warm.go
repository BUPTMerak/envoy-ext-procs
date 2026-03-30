package edgeone

import (
	"context"
	"net/netip"
	"time"
)

const fallbackProbeIP = "1.1.1.1"

// StartWarm launches a background goroutine that periodically probes the
// EdgeOne API to keep the underlying HTTP connection warm. Does nothing if
// WarmInterval is zero or negative.
func (v *Validator) StartWarm(ctx context.Context) {
	if v.warmInterval <= 0 {
		return
	}
	go v.warmLoop(ctx)
}

func (v *Validator) warmLoop(ctx context.Context) {
	v.warmProbe(ctx)

	ticker := time.NewTicker(v.warmInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			v.warmProbe(ctx)
		}
	}
}

func (v *Validator) warmProbe(ctx context.Context) {
	ipStr := fallbackProbeIP
	if key, _, ok := v.cache.GetOldest(); ok {
		ipStr = key
	}

	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		v.log.Warn().Err(err).Str("ip", ipStr).Msg("warm probe skipped")
		return
	}

	if v.warmTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, v.warmTimeout)
		defer cancel()
	}

	start := time.Now()
	valid, err := v.fetchAndCache(ctx, ip)
	if err != nil {
		v.log.Warn().
			Dur("duration", time.Since(start)).
			Err(err).
			Str("probe_ip", ipStr).
			Msg("warm probe failed")
		return
	}

	v.log.Debug().
		Dur("duration", time.Since(start)).
		Str("probe_ip", ipStr).
		Bool("valid", valid).
		Msg("warm probe completed")
}
