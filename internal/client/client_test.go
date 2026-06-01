package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testKey = "abcdef1234567890"

func fastConfig(endpoint string) Config {
	return Config{
		Key:         testKey,
		Host:        "example.com",
		Endpoint:    endpoint,
		MaxRetries:  3,
		BaseBackoff: 1 * time.Millisecond,
		MaxBackoff:  10 * time.Millisecond,
	}
}

func TestValidateKey(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		for _, k := range []string{
			"abcdef12",                // 8 chars min
			"ABCDEF12",                // upper allowed
			"a-b-c-d-e-f-g-h",         // dashes allowed
			strings.Repeat("a", 128),  // 128 chars max
			"deadbeefcafebabe-12345",  // mixed
		} {
			if err := ValidateKey(k); err != nil {
				t.Errorf("ValidateKey(%q) = %v, want nil", k, err)
			}
		}
	})
	t.Run("invalid", func(t *testing.T) {
		for _, k := range []string{
			"",                       // empty
			"abc",                    // too short
			"abcdef1",                // 7 chars
			strings.Repeat("a", 129), // too long
			"abcdef12!",              // bad char
			"abcdef 12",              // space
			"абвгдежз",               // non-ASCII
		} {
			if err := ValidateKey(k); !errors.Is(err, ErrInvalidKey) {
				t.Errorf("ValidateKey(%q) = %v, want ErrInvalidKey", k, err)
			}
		}
	})
}

func TestNew_Validation(t *testing.T) {
	t.Run("missing key", func(t *testing.T) {
		_, err := New(Config{Endpoint: EndpointAPI, Host: "example.com"})
		if !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("got %v, want ErrInvalidKey", err)
		}
	})
	t.Run("invalid key", func(t *testing.T) {
		_, err := New(Config{Key: "short", Endpoint: EndpointAPI, Host: "example.com"})
		if !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("got %v, want ErrInvalidKey", err)
		}
	})
	t.Run("missing endpoint", func(t *testing.T) {
		_, err := New(Config{Key: testKey, Host: "example.com"})
		if !errors.Is(err, ErrMissingEndpoint) {
			t.Fatalf("got %v, want ErrMissingEndpoint", err)
		}
	})
	t.Run("ok defaults applied", func(t *testing.T) {
		c, err := New(Config{Key: testKey, Endpoint: EndpointAPI, Host: "example.com"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c == nil {
			t.Fatal("nil client")
		}
	})
}

func TestSubmit_Success(t *testing.T) {
	var gotURL, gotKey, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotURL = r.URL.Query().Get("url")
		gotKey = r.URL.Query().Get("key")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, err := New(fastConfig(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	res, err := c.Submit(context.Background(), "https://example.com/page")
	if err != nil {
		t.Fatalf("Submit returned err: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("status: got %d", res.StatusCode)
	}
	if res.Err != nil {
		t.Errorf("result err: %v", res.Err)
	}
	if res.Attempts != 1 {
		t.Errorf("attempts: got %d, want 1", res.Attempts)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method: got %q, want GET", gotMethod)
	}
	if gotURL != "https://example.com/page" {
		t.Errorf("url param: got %q", gotURL)
	}
	if gotKey != testKey {
		t.Errorf("key param: got %q", gotKey)
	}
}

func TestSubmit_UserAgent(t *testing.T) {
	t.Run("custom UA sent on GET", func(t *testing.T) {
		var gotUA string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotUA = r.Header.Get("User-Agent")
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		cfg := fastConfig(srv.URL)
		cfg.UserAgent = "indexnow-test/0.0"
		c, _ := New(cfg)
		if _, err := c.Submit(context.Background(), "https://example.com/x"); err != nil {
			t.Fatal(err)
		}
		if gotUA != "indexnow-test/0.0" {
			t.Errorf("UA: got %q, want %q", gotUA, "indexnow-test/0.0")
		}
	})
	t.Run("custom UA sent on POST batch", func(t *testing.T) {
		var gotUA string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotUA = r.Header.Get("User-Agent")
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		cfg := fastConfig(srv.URL)
		cfg.UserAgent = "indexnow-test/0.0"
		c, _ := New(cfg)
		if _, err := c.SubmitBatch(context.Background(), []string{"https://example.com/x"}); err != nil {
			t.Fatal(err)
		}
		if gotUA != "indexnow-test/0.0" {
			t.Errorf("UA: got %q, want %q", gotUA, "indexnow-test/0.0")
		}
	})
	t.Run("empty UA leaves Go default", func(t *testing.T) {
		var gotUA string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotUA = r.Header.Get("User-Agent")
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		c, _ := New(fastConfig(srv.URL))
		if _, err := c.Submit(context.Background(), "https://example.com/x"); err != nil {
			t.Fatal(err)
		}
		// Go's net/http picks a default like "Go-http-client/1.1" when no
		// explicit UA is set; the exact string isn't part of our contract,
		// but it must not be empty (which would mean we accidentally stripped it).
		if gotUA == "" {
			t.Errorf("expected stdlib default UA, got empty")
		}
	})
}

func TestSubmit_Accepted202(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	c, _ := New(fastConfig(srv.URL))
	res, err := c.Submit(context.Background(), "https://example.com/x")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusAccepted || res.Err != nil {
		t.Errorf("got %+v, want 202 ok", res)
	}
}

func TestSubmit_NonRetryable4xx(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	c, _ := New(fastConfig(srv.URL))
	res, err := c.Submit(context.Background(), "https://example.com/x")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("status: got %d", res.StatusCode)
	}
	if res.Err == nil {
		t.Error("expected res.Err for 4xx")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("calls: got %d, want 1 (no retry on 400)", got)
	}
}

func TestSubmit_RetryThenSuccess(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, _ := New(fastConfig(srv.URL))
	res, err := c.Submit(context.Background(), "https://example.com/x")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK || res.Err != nil {
		t.Errorf("result: %+v", res)
	}
	if res.Attempts != 3 {
		t.Errorf("attempts: got %d, want 3", res.Attempts)
	}
}

func TestSubmit_Logger_RetryEmitsWarn(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := calls.Add(1)
		if n < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cfg := fastConfig(srv.URL)
	cfg.Logger = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	c, _ := New(cfg)
	if _, err := c.Submit(context.Background(), "https://example.com/x"); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "level=WARN") || !strings.Contains(got, "msg=retry") {
		t.Fatalf("expected retry WARN log, got %q", got)
	}
	if !strings.Contains(got, "http 503") {
		t.Fatalf("retry log should record status reason; got %q", got)
	}
}

func TestSubmit_Logger_NilDefaultsToDiscard(t *testing.T) {
	// Ensures the existing default-nil-Logger behavior keeps working — no
	// panic on log calls and nothing leaks to anywhere visible.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c, _ := New(fastConfig(srv.URL)) // no Logger set
	if _, err := c.Submit(context.Background(), "https://example.com/x"); err != nil {
		t.Fatal(err)
	}
}

func TestSubmit_RetriesExhausted(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	cfg := fastConfig(srv.URL)
	cfg.MaxRetries = 2
	c, _ := New(cfg)
	res, err := c.Submit(context.Background(), "https://example.com/x")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusServiceUnavailable || res.Err == nil {
		t.Errorf("result: %+v, want 503 with err", res)
	}
	// initial + 2 retries
	if got := calls.Load(); got != 3 {
		t.Errorf("calls: got %d, want 3 (1 + 2 retries)", got)
	}
}

func TestSubmit_RetryAfter429(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, _ := New(fastConfig(srv.URL))
	res, err := c.Submit(context.Background(), "https://example.com/x")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("status: got %d", res.StatusCode)
	}
	if res.Attempts != 2 {
		t.Errorf("attempts: got %d, want 2", res.Attempts)
	}
}

func TestSubmit_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	cfg := fastConfig(srv.URL)
	cfg.BaseBackoff = 50 * time.Millisecond
	cfg.MaxBackoff = 500 * time.Millisecond
	cfg.MaxRetries = 5
	c, _ := New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	res, err := c.Submit(ctx, "https://example.com/x")
	if err == nil && (res == nil || res.Err == nil) {
		t.Fatal("expected error from cancelled context")
	}
}

type batchBody struct {
	Host        string   `json:"host"`
	Key         string   `json:"key"`
	KeyLocation string   `json:"keyLocation,omitempty"`
	URLList     []string `json:"urlList"`
}

func TestSubmitBatch_SingleBatch(t *testing.T) {
	var got batchBody
	var gotMethod, gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotCT = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, _ := New(fastConfig(srv.URL))
	urls := []string{
		"https://example.com/a",
		"https://example.com/b",
		"https://example.com/c",
	}
	results, err := c.SubmitBatch(context.Background(), urls)
	if err != nil {
		t.Fatalf("SubmitBatch err: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].StatusCode != http.StatusOK || results[0].Err != nil {
		t.Errorf("result: %+v", results[0])
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method: %q", gotMethod)
	}
	if !strings.HasPrefix(gotCT, "application/json") {
		t.Errorf("content-type: %q", gotCT)
	}
	if got.Host != "example.com" || got.Key != testKey {
		t.Errorf("body: %+v", got)
	}
	if len(got.URLList) != 3 {
		t.Errorf("urlList len: %d", len(got.URLList))
	}
}

func TestSubmitBatch_KeyLocationIncluded(t *testing.T) {
	var got batchBody
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := fastConfig(srv.URL)
	cfg.KeyLocation = "https://example.com/keys/" + testKey + ".txt"
	c, _ := New(cfg)
	_, err := c.SubmitBatch(context.Background(), []string{"https://example.com/a"})
	if err != nil {
		t.Fatal(err)
	}
	if got.KeyLocation != cfg.KeyLocation {
		t.Errorf("keyLocation: got %q, want %q", got.KeyLocation, cfg.KeyLocation)
	}
}

func TestSubmitBatch_Splits(t *testing.T) {
	var batches atomic.Int32
	var totalURLs atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		batches.Add(1)
		var b batchBody
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &b)
		totalURLs.Add(int32(len(b.URLList)))
		if len(b.URLList) > MaxBatchSize {
			t.Errorf("batch size %d exceeds MaxBatchSize %d", len(b.URLList), MaxBatchSize)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, _ := New(fastConfig(srv.URL))
	urls := make([]string, MaxBatchSize*2+5)
	for i := range urls {
		urls[i] = "https://example.com/" + strconv.Itoa(i)
	}
	results, err := c.SubmitBatch(context.Background(), urls)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Errorf("results: got %d, want 3", len(results))
	}
	if got := batches.Load(); got != 3 {
		t.Errorf("HTTP calls: got %d, want 3", got)
	}
	if got := totalURLs.Load(); int(got) != len(urls) {
		t.Errorf("total urls submitted: got %d, want %d", got, len(urls))
	}
}

func TestSubmitBatch_Empty(t *testing.T) {
	c, _ := New(fastConfig("http://unused"))
	_, err := c.SubmitBatch(context.Background(), nil)
	if !errors.Is(err, ErrEmptyURLList) {
		t.Fatalf("got %v, want ErrEmptyURLList", err)
	}
}

func TestSubmitBatch_MissingHost(t *testing.T) {
	cfg := fastConfig("http://unused")
	cfg.Host = ""
	c, err := New(cfg)
	// New might allow empty host; SubmitBatch must reject.
	if err != nil {
		// If New rejects, that's also acceptable as long as it's ErrMissingHost.
		if !errors.Is(err, ErrMissingHost) {
			t.Fatalf("New err: %v", err)
		}
		return
	}
	_, err = c.SubmitBatch(context.Background(), []string{"https://example.com/a"})
	if !errors.Is(err, ErrMissingHost) {
		t.Fatalf("got %v, want ErrMissingHost", err)
	}
}
