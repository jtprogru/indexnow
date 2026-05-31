package client

import (
	"math/rand/v2"
	"testing"
	"time"
)

func TestRetryableStatus(t *testing.T) {
	retryable := []int{429, 500, 502, 503, 504}
	notRetryable := []int{200, 201, 202, 204, 301, 400, 401, 403, 404, 422}
	for _, code := range retryable {
		if !retryableStatus(code) {
			t.Errorf("status %d: expected retryable", code)
		}
	}
	for _, code := range notRetryable {
		if retryableStatus(code) {
			t.Errorf("status %d: expected non-retryable", code)
		}
	}
}

func newTestRand() *rand.Rand {
	return rand.New(rand.NewPCG(1, 2))
}

func TestNextBackoff_ExponentialWithCap(t *testing.T) {
	base := 100 * time.Millisecond
	maxDur := 1 * time.Second
	rng := newTestRand()

	// For full-jitter exp backoff: result in [0, min(maxDur, base*2^(n-1))].
	cases := []struct {
		attempt int
		ceil    time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{4, 800 * time.Millisecond},
		{5, 1 * time.Second}, // capped
		{10, 1 * time.Second},
	}
	for _, tc := range cases {
		// Sample multiple times to ensure bounds hold.
		for i := range 50 {
			got := nextBackoff(tc.attempt, base, maxDur, 0, rng)
			if got < 0 || got > tc.ceil {
				t.Fatalf("attempt=%d sample=%d: got %v, want in [0, %v]", tc.attempt, i, got, tc.ceil)
			}
		}
	}
}

func TestNextBackoff_RetryAfterHonored(t *testing.T) {
	base := 100 * time.Millisecond
	maxDur := 5 * time.Second
	rng := newTestRand()

	got := nextBackoff(1, base, maxDur, 2*time.Second, rng)
	if got != 2*time.Second {
		t.Errorf("retry-after 2s: got %v, want 2s", got)
	}

	// retry-after exceeding maxDur should be clamped.
	got = nextBackoff(1, base, maxDur, 10*time.Second, rng)
	if got != maxDur {
		t.Errorf("retry-after 10s with maxDur 5s: got %v, want %v", got, maxDur)
	}
}
