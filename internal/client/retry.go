package client

import (
	"math/rand/v2"
	"time"
)

// retryableStatus reports whether an HTTP status code should trigger a retry.
// Per the IndexNow spec: 429 (rate limited) and 5xx are transient.
func retryableStatus(code int) bool {
	return code == 429 || (code >= 500 && code <= 599)
}

// nextBackoff returns the delay before the next attempt.
//
// attempt is 1-indexed (1 = first retry). If retryAfter > 0 it is honored
// (clamped to max). Otherwise exponential backoff with full jitter is used:
// rand in [0, min(max, base * 2^(attempt-1))].
func nextBackoff(attempt int, base, maxDur, retryAfter time.Duration, rng *rand.Rand) time.Duration {
	if retryAfter > 0 {
		if retryAfter > maxDur {
			return maxDur
		}
		return retryAfter
	}
	if attempt < 1 {
		attempt = 1
	}
	// base * 2^(attempt-1), guarding overflow by capping shift.
	shift := attempt - 1
	if shift > 30 {
		shift = 30
	}
	ceil := base << shift
	if ceil <= 0 || ceil > maxDur {
		ceil = maxDur
	}
	return time.Duration(rng.Int64N(int64(ceil) + 1))
}
