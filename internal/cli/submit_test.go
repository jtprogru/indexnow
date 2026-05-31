package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jtprogru/indexnow/internal/client"
)

const testKey = "abcdef1234567890"

// fakeSubmitter records calls and returns pre-canned results.
type fakeSubmitter struct {
	calls   [][]string
	results []*client.Result
	err     error
}

func (f *fakeSubmitter) SubmitBatch(_ context.Context, urls []string) ([]*client.Result, error) {
	f.calls = append(f.calls, append([]string(nil), urls...))
	if f.err != nil {
		return nil, f.err
	}
	if len(f.results) == 0 {
		return []*client.Result{{StatusCode: 200, Attempts: 1, URLs: urls}}, nil
	}
	return f.results, nil
}

func factoryFor(f *fakeSubmitter) SubmitterFactory {
	return func(_ client.Config) (Submitter, error) { return f, nil }
}

func defaultOpts() SubmitOptions {
	return SubmitOptions{
		Key:      testKey,
		Host:     "example.com",
		Endpoint: "api",
		Output:   OutputText,
		FailOn:   FailOnAny,
	}
}

func TestParseURLLines(t *testing.T) {
	in := strings.NewReader(`
https://example.com/a
# this is a comment
https://example.com/b

  https://example.com/c
`)
	got, err := parseURLLines(in)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://example.com/a", "https://example.com/b", "https://example.com/c"}
	if !equalStrings(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseURLLines_Empty(t *testing.T) {
	_, err := parseURLLines(strings.NewReader("\n# only comment\n\n"))
	if !errors.Is(err, ErrNoURLs) {
		t.Fatalf("got %v, want ErrNoURLs", err)
	}
}

func TestHostFromURL(t *testing.T) {
	cases := map[string]string{
		"https://example.com/page":         "example.com",
		"http://example.com:8080/p":        "example.com",
		"https://sub.example.com/x?q=1":    "sub.example.com",
		"https://www.bing.com/indexnow":    "www.bing.com",
	}
	for in, want := range cases {
		got, err := hostFromURL(in)
		if err != nil {
			t.Errorf("%q: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
	if _, err := hostFromURL("not a url"); err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestCollectURLs_Args(t *testing.T) {
	opts := defaultOpts()
	opts.Args = []string{"https://example.com/a", "https://example.com/b"}
	got, err := collectURLs(opts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !equalStrings(got, opts.Args) {
		t.Errorf("got %v, want %v", got, opts.Args)
	}
}

func TestCollectURLs_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "urls.txt")
	if err := os.WriteFile(path, []byte("https://example.com/a\nhttps://example.com/b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := defaultOpts()
	opts.File = path
	got, err := collectURLs(opts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("got %d urls, want 2", len(got))
	}
}

func TestCollectURLs_Stdin(t *testing.T) {
	opts := defaultOpts()
	opts.Stdin = true
	got, err := collectURLs(opts, strings.NewReader("https://example.com/a\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "https://example.com/a" {
		t.Errorf("got %v", got)
	}
}

func TestCollectURLs_Conflict(t *testing.T) {
	opts := defaultOpts()
	opts.Args = []string{"https://example.com/a"}
	opts.Stdin = true
	_, err := collectURLs(opts, strings.NewReader("x"))
	if !errors.Is(err, ErrSourceConflict) {
		t.Fatalf("got %v, want ErrSourceConflict", err)
	}
}

func TestCollectURLs_NoSource(t *testing.T) {
	_, err := collectURLs(defaultOpts(), nil)
	if !errors.Is(err, ErrNoSource) {
		t.Fatalf("got %v, want ErrNoSource", err)
	}
}

func TestRunSubmit_HappyPath_Args(t *testing.T) {
	fake := &fakeSubmitter{}
	opts := defaultOpts()
	opts.Args = []string{"https://example.com/a", "https://example.com/b"}
	var stdout, stderr bytes.Buffer
	code := RunSubmit(context.Background(), opts, nil, &stdout, &stderr, factoryFor(fake))
	if code != ExitOK {
		t.Errorf("exit code: got %d, want %d. stderr=%s", code, ExitOK, stderr.String())
	}
	if len(fake.calls) != 1 {
		t.Fatalf("submitter calls: got %d, want 1", len(fake.calls))
	}
	if len(fake.calls[0]) != 2 {
		t.Errorf("urls submitted: got %d, want 2", len(fake.calls[0]))
	}
}

func TestRunSubmit_DryRun_NoCall(t *testing.T) {
	fake := &fakeSubmitter{}
	opts := defaultOpts()
	opts.Args = []string{"https://example.com/a"}
	opts.DryRun = true
	var stdout, stderr bytes.Buffer
	code := RunSubmit(context.Background(), opts, nil, &stdout, &stderr, factoryFor(fake))
	if code != ExitOK {
		t.Errorf("exit code: got %d, want %d", code, ExitOK)
	}
	if len(fake.calls) != 0 {
		t.Errorf("--dry-run should not call submitter; got %d calls", len(fake.calls))
	}
	if !strings.Contains(stdout.String(), "https://example.com/a") {
		t.Errorf("dry-run output should mention URL; got %q", stdout.String())
	}
}

func TestRunSubmit_OutputJSON(t *testing.T) {
	fake := &fakeSubmitter{
		results: []*client.Result{{StatusCode: 202, Attempts: 1, URLs: []string{"https://example.com/a"}}},
	}
	opts := defaultOpts()
	opts.Args = []string{"https://example.com/a"}
	opts.Output = OutputJSON
	var stdout, stderr bytes.Buffer
	code := RunSubmit(context.Background(), opts, nil, &stdout, &stderr, factoryFor(fake))
	if code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	var parsed []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not JSON array: %v\n%s", err, stdout.String())
	}
	if len(parsed) != 1 {
		t.Fatalf("got %d entries", len(parsed))
	}
	if int(parsed[0]["status"].(float64)) != 202 {
		t.Errorf("status field missing/wrong: %v", parsed[0])
	}
}

func TestRunSubmit_FailOn_Any(t *testing.T) {
	fake := &fakeSubmitter{
		results: []*client.Result{{
			StatusCode: 400, Attempts: 1, URLs: []string{"https://example.com/a"},
			Err: errors.New("bad"),
		}},
	}
	opts := defaultOpts()
	opts.Args = []string{"https://example.com/a"}
	opts.FailOn = FailOnAny
	code := RunSubmit(context.Background(), opts, nil, new(bytes.Buffer), new(bytes.Buffer), factoryFor(fake))
	if code != ExitFailed {
		t.Errorf("got %d, want %d", code, ExitFailed)
	}
}

func TestRunSubmit_FailOn_5xx_Only(t *testing.T) {
	fake := &fakeSubmitter{
		results: []*client.Result{{
			StatusCode: 400, Attempts: 1, URLs: []string{"https://example.com/a"},
			Err: errors.New("bad"),
		}},
	}
	opts := defaultOpts()
	opts.Args = []string{"https://example.com/a"}
	opts.FailOn = FailOn5xx
	code := RunSubmit(context.Background(), opts, nil, new(bytes.Buffer), new(bytes.Buffer), factoryFor(fake))
	if code != ExitOK {
		t.Errorf("4xx with fail-on=5xx: got %d, want %d", code, ExitOK)
	}
}

func TestRunSubmit_FailOn_5xx_Triggered(t *testing.T) {
	fake := &fakeSubmitter{
		results: []*client.Result{{
			StatusCode: 503, Attempts: 3, URLs: []string{"https://example.com/a"},
			Err: errors.New("svc unavailable"),
		}},
	}
	opts := defaultOpts()
	opts.Args = []string{"https://example.com/a"}
	opts.FailOn = FailOn5xx
	code := RunSubmit(context.Background(), opts, nil, new(bytes.Buffer), new(bytes.Buffer), factoryFor(fake))
	if code != ExitFailed {
		t.Errorf("5xx with fail-on=5xx: got %d, want %d", code, ExitFailed)
	}
}

func TestRunSubmit_FailOn_Never(t *testing.T) {
	fake := &fakeSubmitter{
		results: []*client.Result{{
			StatusCode: 500, Attempts: 3, URLs: []string{"https://example.com/a"},
			Err: errors.New("boom"),
		}},
	}
	opts := defaultOpts()
	opts.Args = []string{"https://example.com/a"}
	opts.FailOn = FailOnNever
	code := RunSubmit(context.Background(), opts, nil, new(bytes.Buffer), new(bytes.Buffer), factoryFor(fake))
	if code != ExitOK {
		t.Errorf("fail-on=never: got %d, want %d", code, ExitOK)
	}
}

func TestRunSubmit_InvalidOutput(t *testing.T) {
	opts := defaultOpts()
	opts.Args = []string{"https://example.com/a"}
	opts.Output = "xml"
	code := RunSubmit(context.Background(), opts, nil, new(bytes.Buffer), new(bytes.Buffer), factoryFor(&fakeSubmitter{}))
	if code != ExitUsageError {
		t.Errorf("got %d, want %d", code, ExitUsageError)
	}
}

func TestRunSubmit_InvalidFailOn(t *testing.T) {
	opts := defaultOpts()
	opts.Args = []string{"https://example.com/a"}
	opts.FailOn = "bogus"
	code := RunSubmit(context.Background(), opts, nil, new(bytes.Buffer), new(bytes.Buffer), factoryFor(&fakeSubmitter{}))
	if code != ExitUsageError {
		t.Errorf("got %d, want %d", code, ExitUsageError)
	}
}

func TestRunSubmit_MissingKey(t *testing.T) {
	opts := defaultOpts()
	opts.Key = ""
	opts.Args = []string{"https://example.com/a"}
	code := RunSubmit(context.Background(), opts, nil, new(bytes.Buffer), new(bytes.Buffer), nil)
	if code != ExitUsageError {
		t.Errorf("got %d, want %d", code, ExitUsageError)
	}
}

func TestRunSubmit_HostInferredFromURL(t *testing.T) {
	fake := &fakeSubmitter{}
	captured := ""
	factory := SubmitterFactory(func(c client.Config) (Submitter, error) {
		captured = c.Host
		return fake, nil
	})
	opts := defaultOpts()
	opts.Host = "" // intentionally omit
	opts.Args = []string{"https://my-site.example.com/a"}
	code := RunSubmit(context.Background(), opts, nil, new(bytes.Buffer), new(bytes.Buffer), factory)
	if code != ExitOK {
		t.Fatalf("code=%d", code)
	}
	if captured != "my-site.example.com" {
		t.Errorf("host inferred: got %q, want %q", captured, "my-site.example.com")
	}
}

func TestRunSubmit_EndpointResolvedFromAlias(t *testing.T) {
	captured := ""
	factory := SubmitterFactory(func(c client.Config) (Submitter, error) {
		captured = c.Endpoint
		return &fakeSubmitter{}, nil
	})
	opts := defaultOpts()
	opts.Endpoint = "bing"
	opts.Args = []string{"https://example.com/a"}
	code := RunSubmit(context.Background(), opts, nil, new(bytes.Buffer), new(bytes.Buffer), factory)
	if code != ExitOK {
		t.Fatalf("code=%d", code)
	}
	if captured != client.EndpointBing {
		t.Errorf("endpoint resolved: got %q, want %q", captured, client.EndpointBing)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
