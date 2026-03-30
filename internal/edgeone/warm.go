package edgeone

import (
	"context"
	"net/netip"
	"time"
)

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
	var ipStr string
	if key, _, ok := v.cache.GetOldest(); ok {
		ipStr = key
	} else {
		v.log.Debug().Str("reason", "no oldest IP found").Msg("skipping warm probe")
		return
	}

	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		v.log.Warn().Err(err).Str("reason", "invalid IP").Str("ip", ipStr).Msg("skipping warm probe")
		return
	}

	if v.warmTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, v.warmTimeout)
		defer cancel()
	}

	start := time.Now()
	if _, err = v.fetchAndCache(ctx, ip, func(valid bool) {
		v.log.Debug().
			Dur("duration", time.Since(start)).
			Str("probe_ip", ipStr).
			Bool("valid", valid).
			Msg("warm probe completed")
		if valid {
			v.cache.Add(ipStr, valid)
		}
	}); err != nil {
		v.log.Warn().
			Dur("duration", time.Since(start)).
			Err(err).
			Str("probe_ip", ipStr).
			Msg("warm probe failed")
	}
}
