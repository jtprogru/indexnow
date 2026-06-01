package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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

// multiEndpointFactory routes each factory call to a per-endpoint fake.
// Used by tests that exercise the parallel fan-out across endpoints.
type multiEndpointFactory struct {
	mu       sync.Mutex
	fakes    map[string]*fakeSubmitter
	initErrs map[string]error
	seen     []string
}

func (m *multiEndpointFactory) factory() SubmitterFactory {
	return func(c client.Config) (Submitter, error) {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.seen = append(m.seen, c.Endpoint)
		if err, ok := m.initErrs[c.Endpoint]; ok {
			return nil, err
		}
		f, ok := m.fakes[c.Endpoint]
		if !ok {
			f = &fakeSubmitter{}
			if m.fakes == nil {
				m.fakes = map[string]*fakeSubmitter{}
			}
			m.fakes[c.Endpoint] = f
		}
		return f, nil
	}
}

func (m *multiEndpointFactory) seenSorted() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := append([]string(nil), m.seen...)
	sort.Strings(out)
	return out
}

func TestRunSubmit_MultiEndpoint_FanOut(t *testing.T) {
	mf := &multiEndpointFactory{
		fakes: map[string]*fakeSubmitter{
			client.EndpointBing:   {results: []*client.Result{{StatusCode: 200, Attempts: 1, URLs: []string{"https://example.com/a"}}}},
			client.EndpointYandex: {results: []*client.Result{{StatusCode: 200, Attempts: 1, URLs: []string{"https://example.com/a"}}}},
		},
	}
	opts := defaultOpts()
	opts.Endpoint = "bing,yandex"
	opts.Args = []string{"https://example.com/a"}
	var stdout, stderr bytes.Buffer
	code := RunSubmit(context.Background(), opts, nil, &stdout, &stderr, mf.factory())
	if code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	got := mf.seenSorted()
	want := []string{client.EndpointBing, client.EndpointYandex}
	sort.Strings(want)
	if !equalStrings(got, want) {
		t.Fatalf("endpoints called: got %v want %v", got, want)
	}
	out := stdout.String()
	if !strings.Contains(out, "endpoint="+client.EndpointBing) || !strings.Contains(out, "endpoint="+client.EndpointYandex) {
		t.Fatalf("multi text output should prefix endpoint per line; got %q", out)
	}
}

func TestRunSubmit_MultiEndpoint_JSON(t *testing.T) {
	mf := &multiEndpointFactory{
		fakes: map[string]*fakeSubmitter{
			client.EndpointBing:   {results: []*client.Result{{StatusCode: 200, Attempts: 1, URLs: []string{"https://example.com/a"}}}},
			client.EndpointYandex: {results: []*client.Result{{StatusCode: 202, Attempts: 2, URLs: []string{"https://example.com/a"}}}},
		},
	}
	opts := defaultOpts()
	opts.Endpoint = "bing,yandex"
	opts.Output = OutputJSON
	opts.Args = []string{"https://example.com/a"}
	var stdout, stderr bytes.Buffer
	code := RunSubmit(context.Background(), opts, nil, &stdout, &stderr, mf.factory())
	if code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	var parsed []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Fatalf("not json: %v\n%s", err, stdout.String())
	}
	if len(parsed) != 2 {
		t.Fatalf("expected 2 entries (one per endpoint), got %d: %s", len(parsed), stdout.String())
	}
	gotEndpoints := []string{parsed[0]["endpoint"].(string), parsed[1]["endpoint"].(string)}
	sort.Strings(gotEndpoints)
	want := []string{client.EndpointBing, client.EndpointYandex}
	sort.Strings(want)
	if !equalStrings(gotEndpoints, want) {
		t.Fatalf("endpoints in json: got %v want %v", gotEndpoints, want)
	}
}

func TestRunSubmit_MultiEndpoint_PartialFailure_AggregatesExit(t *testing.T) {
	mf := &multiEndpointFactory{
		fakes: map[string]*fakeSubmitter{
			client.EndpointBing:   {results: []*client.Result{{StatusCode: 200, Attempts: 1, URLs: []string{"https://example.com/a"}}}},
			client.EndpointYandex: {results: []*client.Result{{StatusCode: 503, Attempts: 3, URLs: []string{"https://example.com/a"}, Err: errors.New("svc unavail")}}},
		},
	}
	opts := defaultOpts()
	opts.Endpoint = "bing,yandex"
	opts.FailOn = FailOnAny
	opts.Args = []string{"https://example.com/a"}
	var stdout, stderr bytes.Buffer
	code := RunSubmit(context.Background(), opts, nil, &stdout, &stderr, mf.factory())
	if code != ExitFailed {
		t.Fatalf("one endpoint failing should fail aggregate; got code=%d", code)
	}
	if !strings.Contains(stdout.String(), client.EndpointBing) || !strings.Contains(stdout.String(), client.EndpointYandex) {
		t.Fatalf("output should include both endpoints; got %q", stdout.String())
	}
}

func TestRunSubmit_MultiEndpoint_FactoryErrorDoesNotKillOthers(t *testing.T) {
	mf := &multiEndpointFactory{
		fakes: map[string]*fakeSubmitter{
			client.EndpointBing: {results: []*client.Result{{StatusCode: 200, Attempts: 1, URLs: []string{"https://example.com/a"}}}},
		},
		initErrs: map[string]error{
			client.EndpointYandex: errors.New("dial tcp: timeout"),
		},
	}
	opts := defaultOpts()
	opts.Endpoint = "bing,yandex"
	opts.FailOn = FailOnNever // ignore HTTP-status failures so we isolate factory-error path
	opts.Args = []string{"https://example.com/a"}
	var stdout, stderr bytes.Buffer
	code := RunSubmit(context.Background(), opts, nil, &stdout, &stderr, mf.factory())
	// Factory failure always counts as failure regardless of FailOn (it is a system-level error, not an HTTP outcome).
	if code != ExitFailed {
		t.Fatalf("factory failure should produce ExitFailed; got %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "ERROR") || !strings.Contains(out, client.EndpointYandex) {
		t.Fatalf("output should include yandex factory error; got %q", out)
	}
	if !strings.Contains(out, "status=200") {
		t.Fatalf("bing should still report success; got %q", out)
	}
}

func TestRunSubmit_MultiEndpoint_DryRunListsAll(t *testing.T) {
	opts := defaultOpts()
	opts.Endpoint = "bing,yandex"
	opts.DryRun = true
	opts.Args = []string{"https://example.com/a"}
	var stdout, stderr bytes.Buffer
	code := RunSubmit(context.Background(), opts, nil, &stdout, &stderr, nil)
	if code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "endpoints=") {
		t.Fatalf("dry-run with multi should use endpoints= key; got %q", out)
	}
	if !strings.Contains(out, client.EndpointBing) || !strings.Contains(out, client.EndpointYandex) {
		t.Fatalf("dry-run should list both endpoints; got %q", out)
	}
}

func TestRunSubmit_Quiet_SuppressesStdoutOnSuccess(t *testing.T) {
	fake := &fakeSubmitter{
		results: []*client.Result{{StatusCode: 200, Attempts: 1, URLs: []string{"https://example.com/a"}}},
	}
	opts := defaultOpts()
	opts.Args = []string{"https://example.com/a"}
	opts.Quiet = true
	var stdout, stderr bytes.Buffer
	code := RunSubmit(context.Background(), opts, nil, &stdout, &stderr, factoryFor(fake))
	if code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("--quiet should produce no stdout on success; got %q", stdout.String())
	}
	if len(fake.calls) != 1 {
		t.Fatalf("submitter must still be called under --quiet; got %d calls", len(fake.calls))
	}
}

func TestRunSubmit_Quiet_SuppressesStdoutOnFailure(t *testing.T) {
	fake := &fakeSubmitter{
		results: []*client.Result{{StatusCode: 500, Attempts: 3, URLs: []string{"https://example.com/a"}, Err: errors.New("boom")}},
	}
	opts := defaultOpts()
	opts.Args = []string{"https://example.com/a"}
	opts.Quiet = true
	opts.FailOn = FailOnAny
	var stdout, stderr bytes.Buffer
	code := RunSubmit(context.Background(), opts, nil, &stdout, &stderr, factoryFor(fake))
	if code != ExitFailed {
		t.Fatalf("expected ExitFailed; got %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("--quiet should produce no stdout on failure either; got %q", stdout.String())
	}
}

func TestRunSubmit_Quiet_DryRunSilent(t *testing.T) {
	opts := defaultOpts()
	opts.Args = []string{"https://example.com/a"}
	opts.Quiet = true
	opts.DryRun = true
	var stdout, stderr bytes.Buffer
	code := RunSubmit(context.Background(), opts, nil, &stdout, &stderr, factoryFor(&fakeSubmitter{}))
	if code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("--quiet --dry-run should produce no stdout; got %q", stdout.String())
	}
}

func TestRunSubmit_Verbose_EmitsLogsToStderr(t *testing.T) {
	fake := &fakeSubmitter{
		results: []*client.Result{{StatusCode: 200, Attempts: 1, URLs: []string{"https://example.com/a"}}},
	}
	opts := defaultOpts()
	opts.Args = []string{"https://example.com/a"}
	opts.Verbose = true
	var stdout, stderr bytes.Buffer
	code := RunSubmit(context.Background(), opts, nil, &stdout, &stderr, factoryFor(fake))
	if code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	out := stderr.String()
	if !strings.Contains(out, `msg=submit`) {
		t.Fatalf("expected lifecycle log on stderr; got %q", out)
	}
	if !strings.Contains(out, `msg="batch complete"`) {
		t.Fatalf("expected per-batch log on stderr; got %q", out)
	}
}

func TestRunSubmit_NotVerbose_NoLogs(t *testing.T) {
	fake := &fakeSubmitter{
		results: []*client.Result{{StatusCode: 200, Attempts: 1, URLs: []string{"https://example.com/a"}}},
	}
	opts := defaultOpts()
	opts.Args = []string{"https://example.com/a"}
	var stdout, stderr bytes.Buffer
	code := RunSubmit(context.Background(), opts, nil, &stdout, &stderr, factoryFor(fake))
	if code != ExitOK {
		t.Fatal(code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("default mode must not emit logs; got %q", stderr.String())
	}
}

func TestRunSubmit_Quiet_ValidationErrorStillReachesStderr(t *testing.T) {
	opts := defaultOpts()
	opts.Args = []string{"https://example.com/a"}
	opts.Quiet = true
	opts.Key = "" // trigger usage error
	var stdout, stderr bytes.Buffer
	code := RunSubmit(context.Background(), opts, nil, &stdout, &stderr, factoryFor(&fakeSubmitter{}))
	if code != ExitUsageError {
		t.Fatalf("expected ExitUsageError; got %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("--quiet should not touch stdout for validation; got %q", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Fatalf("--quiet must not silence stderr errors")
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
