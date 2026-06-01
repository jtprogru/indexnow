package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func defaultVerifyOpts() VerifyOptions {
	return VerifyOptions{
		Key:    testKey,
		Output: OutputText,
	}
}

func TestRunVerify_OK_ExplicitKeyLocation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(testKey + "\n"))
	}))
	defer srv.Close()

	opts := defaultVerifyOpts()
	opts.KeyLocation = srv.URL + "/" + testKey + ".txt"
	var stdout, stderr bytes.Buffer
	code := RunVerify(context.Background(), opts, &stdout, &stderr, srv.Client())
	if code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "OK:") {
		t.Fatalf("expected OK output, got %q", stdout.String())
	}
}

func TestRunVerify_Mismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not-the-real-key"))
	}))
	defer srv.Close()

	opts := defaultVerifyOpts()
	opts.KeyLocation = srv.URL + "/k.txt"
	var stdout, stderr bytes.Buffer
	code := RunVerify(context.Background(), opts, &stdout, &stderr, srv.Client())
	if code != ExitFailed {
		t.Fatalf("expected ExitFailed; got %d", code)
	}
	if !strings.HasPrefix(stdout.String(), "FAIL:") {
		t.Fatalf("expected FAIL output, got %q", stdout.String())
	}
}

func TestRunVerify_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	opts := defaultVerifyOpts()
	opts.KeyLocation = srv.URL + "/missing.txt"
	var stdout, stderr bytes.Buffer
	code := RunVerify(context.Background(), opts, &stdout, &stderr, srv.Client())
	if code != ExitFailed {
		t.Fatalf("got %d", code)
	}
	if !strings.Contains(stdout.String(), "status=404") {
		t.Fatalf("expected status=404 in output, got %q", stdout.String())
	}
}

func TestRunVerify_TrimsTrailingWhitespace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("  " + testKey + "\r\n"))
	}))
	defer srv.Close()
	opts := defaultVerifyOpts()
	opts.KeyLocation = srv.URL
	var stdout, stderr bytes.Buffer
	if code := RunVerify(context.Background(), opts, &stdout, &stderr, srv.Client()); code != ExitOK {
		t.Fatalf("padded body should still match; code=%d stderr=%s", code, stderr.String())
	}
}

func TestRunVerify_DerivedURLFromHostAndKey(t *testing.T) {
	// Stand up the server, then route the derived path through the same
	// server's client by overriding Host in URL via the round-tripper.
	// Simpler: use httptest.NewServer and pass --key-location explicitly
	// pointing at the conventional /<key>.txt path; we already cover the
	// derived-URL formatting via TestResolveKeyURL below.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+testKey+".txt" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(testKey))
	}))
	defer srv.Close()
	opts := defaultVerifyOpts()
	opts.KeyLocation = srv.URL + "/" + testKey + ".txt"
	var stdout, stderr bytes.Buffer
	if code := RunVerify(context.Background(), opts, &stdout, &stderr, srv.Client()); code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
}

func TestRunVerify_OutputJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(testKey))
	}))
	defer srv.Close()
	opts := defaultVerifyOpts()
	opts.KeyLocation = srv.URL
	opts.Output = OutputJSON
	var stdout, stderr bytes.Buffer
	if code := RunVerify(context.Background(), opts, &stdout, &stderr, srv.Client()); code != ExitOK {
		t.Fatalf("code=%d", code)
	}
	var parsed map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Fatalf("not json: %v\n%s", err, stdout.String())
	}
	if ok, _ := parsed["ok"].(bool); !ok {
		t.Fatalf("ok=true expected, got %v", parsed["ok"])
	}
}

func TestRunVerify_Quiet_Silent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(testKey))
	}))
	defer srv.Close()
	opts := defaultVerifyOpts()
	opts.KeyLocation = srv.URL
	opts.Quiet = true
	var stdout, stderr bytes.Buffer
	if code := RunVerify(context.Background(), opts, &stdout, &stderr, srv.Client()); code != ExitOK {
		t.Fatalf("code=%d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("--quiet should produce no stdout; got %q", stdout.String())
	}
}

func TestRunVerify_Verbose_LogsToStderr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(testKey))
	}))
	defer srv.Close()
	opts := defaultVerifyOpts()
	opts.KeyLocation = srv.URL
	opts.Verbose = true
	var stdout, stderr bytes.Buffer
	if code := RunVerify(context.Background(), opts, &stdout, &stderr, srv.Client()); code != ExitOK {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stderr.String(), `msg=verify`) {
		t.Fatalf("expected verify log on stderr; got %q", stderr.String())
	}
}

func TestRunVerify_MissingKey(t *testing.T) {
	opts := VerifyOptions{Output: OutputText}
	var stdout, stderr bytes.Buffer
	code := RunVerify(context.Background(), opts, &stdout, &stderr, nil)
	if code != ExitUsageError {
		t.Fatalf("got %d, want ExitUsageError", code)
	}
}

func TestRunVerify_NeedHostOrKeyLocation(t *testing.T) {
	opts := defaultVerifyOpts()
	// no KeyLocation, no Host
	var stdout, stderr bytes.Buffer
	code := RunVerify(context.Background(), opts, &stdout, &stderr, nil)
	if code != ExitUsageError {
		t.Fatalf("got %d, want ExitUsageError", code)
	}
}

func TestRunVerify_InvalidKeyLocation(t *testing.T) {
	opts := defaultVerifyOpts()
	opts.KeyLocation = "not-a-url"
	var stdout, stderr bytes.Buffer
	code := RunVerify(context.Background(), opts, &stdout, &stderr, nil)
	if code != ExitUsageError {
		t.Fatalf("got %d", code)
	}
}

func TestResolveKeyURL_Derived(t *testing.T) {
	opts := VerifyOptions{Key: testKey, Host: "example.com"}
	got, err := resolveKeyURL(opts)
	if err != nil {
		t.Fatal(err)
	}
	want := "https://example.com/" + testKey + ".txt"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveKeyURL_RejectsInvalidKeyWhenDeriving(t *testing.T) {
	opts := VerifyOptions{Key: "bad!", Host: "example.com"}
	if _, err := resolveKeyURL(opts); err == nil {
		t.Fatal("expected validation error for invalid key while deriving URL")
	}
}

func TestRunVerify_UserAgentSent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(testKey))
	}))
	defer srv.Close()
	opts := defaultVerifyOpts()
	opts.KeyLocation = srv.URL
	opts.UserAgent = "verify-test/1.0"
	var stdout, stderr bytes.Buffer
	if code := RunVerify(context.Background(), opts, &stdout, &stderr, srv.Client()); code != ExitOK {
		t.Fatalf("code=%d", code)
	}
	if gotUA != "verify-test/1.0" {
		t.Fatalf("UA: got %q", gotUA)
	}
}
