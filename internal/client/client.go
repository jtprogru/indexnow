// Package client implements the IndexNow protocol client.
//
// See https://www.indexnow.org/documentation for the protocol spec.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

// MaxBatchSize is the protocol limit on URLs per POST request.
const MaxBatchSize = 10000

const (
	defaultMaxRetries  = 3
	defaultBaseBackoff = 1 * time.Second
	defaultMaxBackoff  = 30 * time.Second
	defaultTimeout     = 30 * time.Second
)

// Config configures a Client.
type Config struct {
	Key         string
	Host        string
	KeyLocation string
	Endpoint    string
	UserAgent   string
	HTTPClient  *http.Client
	MaxRetries  int
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
}

// Client is an IndexNow API client. Safe for concurrent use.
type Client struct {
	cfg  Config
	http *http.Client
}

// Result reports the outcome of a single HTTP submission.
type Result struct {
	StatusCode int
	Attempts   int
	URLs       []string
	Err        error
}

var keyRe = regexp.MustCompile(`^[A-Za-z0-9-]+$`)

// ValidateKey reports whether key matches the IndexNow constraints
// (8..128 chars from [a-zA-Z0-9-]).
func ValidateKey(key string) error {
	if len(key) < 8 || len(key) > 128 {
		return fmt.Errorf("%w: length %d not in 8..128", ErrInvalidKey, len(key))
	}
	if !keyRe.MatchString(key) {
		return fmt.Errorf("%w: contains disallowed characters", ErrInvalidKey)
	}
	return nil
}

// New validates cfg and returns a ready-to-use Client.
func New(cfg Config) (*Client, error) {
	if err := ValidateKey(cfg.Key); err != nil {
		return nil, err
	}
	if cfg.Endpoint == "" {
		return nil, ErrMissingEndpoint
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: defaultTimeout}
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = defaultMaxRetries
	}
	if cfg.BaseBackoff <= 0 {
		cfg.BaseBackoff = defaultBaseBackoff
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = defaultMaxBackoff
	}
	return &Client{cfg: cfg, http: cfg.HTTPClient}, nil
}

// Submit notifies the endpoint about a single URL via GET.
func (c *Client) Submit(ctx context.Context, rawURL string) (*Result, error) {
	if rawURL == "" {
		return nil, ErrEmptyURLList
	}
	q := url.Values{}
	q.Set("url", rawURL)
	q.Set("key", c.cfg.Key)
	target := c.cfg.Endpoint + "?" + q.Encode()

	build := func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return nil, err
		}
		c.setUserAgent(req)
		return req, nil
	}
	res := c.executeWithRetry(ctx, build)
	res.URLs = []string{rawURL}
	return res, nil
}

// SubmitBatch sends urls via one or more POST requests, splitting at
// MaxBatchSize. Returns one Result per batch in submission order.
func (c *Client) SubmitBatch(ctx context.Context, urls []string) ([]*Result, error) {
	if len(urls) == 0 {
		return nil, ErrEmptyURLList
	}
	if c.cfg.Host == "" {
		return nil, ErrMissingHost
	}

	var results []*Result
	for start := 0; start < len(urls); start += MaxBatchSize {
		end := start + MaxBatchSize
		if end > len(urls) {
			end = len(urls)
		}
		chunk := urls[start:end]

		payload := struct {
			Host        string   `json:"host"`
			Key         string   `json:"key"`
			KeyLocation string   `json:"keyLocation,omitempty"`
			URLList     []string `json:"urlList"`
		}{
			Host:        c.cfg.Host,
			Key:         c.cfg.Key,
			KeyLocation: c.cfg.KeyLocation,
			URLList:     chunk,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return results, fmt.Errorf("marshal batch: %w", err)
		}

		build := func() (*http.Request, error) {
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint, bytes.NewReader(body))
			if err != nil {
				return nil, err
			}
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
			c.setUserAgent(req)
			return req, nil
		}
		res := c.executeWithRetry(ctx, build)
		res.URLs = chunk
		results = append(results, res)
	}
	return results, nil
}

func (c *Client) setUserAgent(req *http.Request) {
	if c.cfg.UserAgent != "" {
		req.Header.Set("User-Agent", c.cfg.UserAgent)
	}
}

// executeWithRetry runs build()+Do with retry logic. The build function
// must produce a fresh request on each call (so bodies can be re-read).
func (c *Client) executeWithRetry(ctx context.Context, build func() (*http.Request, error)) *Result {
	rng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0xdeadbeef)) //nolint:gosec // jitter, not security-sensitive
	res := &Result{}

	for {
		req, err := build()
		if err != nil {
			res.Err = err
			return res
		}
		res.Attempts++

		resp, err := c.http.Do(req)
		var status int
		var retryAfter time.Duration
		if err == nil {
			status = resp.StatusCode
			retryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			res.StatusCode = status
			if !retryableStatus(status) {
				if status < 200 || status >= 300 {
					res.Err = fmt.Errorf("indexnow: http %d", status)
				}
				return res
			}
		} else {
			// Transport error — treat as retryable unless ctx is done.
			if ctxErr := ctx.Err(); ctxErr != nil {
				res.Err = ctxErr
				return res
			}
			res.Err = err
		}

		// Retry exhausted?
		retriesUsed := res.Attempts - 1
		if retriesUsed >= c.cfg.MaxRetries {
			if res.Err == nil && status != 0 {
				res.Err = fmt.Errorf("indexnow: http %d after %d attempts", status, res.Attempts)
			}
			return res
		}

		delay := nextBackoff(retriesUsed+1, c.cfg.BaseBackoff, c.cfg.MaxBackoff, retryAfter, rng)
		select {
		case <-ctx.Done():
			res.Err = ctx.Err()
			return res
		case <-time.After(delay):
		}
	}
}

func parseRetryAfter(h string) time.Duration {
	if h == "" {
		return 0
	}
	if secs, err := strconv.Atoi(h); err == nil {
		if secs < 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(h); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}
