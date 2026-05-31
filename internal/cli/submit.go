// Package cli implements the indexnow CLI behavior in a testable form.
// Cobra wiring lives in cmd/indexnow; this package owns the logic.
package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jtprogru/indexnow/internal/client"
)

const (
	OutputText = "text"
	OutputJSON = "json"
)

const (
	FailOnAny   = "any"
	FailOn4xx   = "4xx"
	FailOn5xx   = "5xx"
	FailOnNever = "never"
)

const (
	ExitOK         = 0
	ExitFailed     = 1
	ExitUsageError = 2
)

type SubmitOptions struct {
	Key         string
	Host        string
	KeyLocation string
	Endpoint    string
	File        string
	Stdin       bool
	Args        []string
	DryRun      bool
	Output      string
	FailOn      string
	MaxRetries  int
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
}

type Submitter interface {
	SubmitBatch(ctx context.Context, urls []string) ([]*client.Result, error)
}

type SubmitterFactory func(client.Config) (Submitter, error)

func defaultFactory(cfg client.Config) (Submitter, error) {
	return client.New(cfg)
}

func RunSubmit(ctx context.Context, opts SubmitOptions, stdin io.Reader, stdout, stderr io.Writer, factory SubmitterFactory) int {
	if factory == nil {
		factory = defaultFactory
	}
	if err := validateOutput(opts.Output); err != nil {
		fmt.Fprintln(stderr, err)
		return ExitUsageError
	}
	if err := validateFailOn(opts.FailOn); err != nil {
		fmt.Fprintln(stderr, err)
		return ExitUsageError
	}
	if opts.Key == "" {
		fmt.Fprintln(stderr, "indexnow cli: --key (or INDEXNOW_KEY) is required")
		return ExitUsageError
	}

	urls, err := collectURLs(opts, stdin)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return ExitUsageError
	}

	host := opts.Host
	if host == "" {
		h, err := hostFromURL(urls[0])
		if err != nil {
			fmt.Fprintf(stderr, "cannot infer host from %q: %v\n", urls[0], err)
			return ExitUsageError
		}
		host = h
	}

	endpoint, err := client.ResolveEndpoint(opts.Endpoint)
	if err != nil {
		fmt.Fprintf(stderr, "endpoint %q: %v\n", opts.Endpoint, err)
		return ExitUsageError
	}

	if opts.DryRun {
		fmt.Fprintf(stdout, "[dry-run] endpoint=%s host=%s urls=%d\n", endpoint, host, len(urls))
		for _, u := range urls {
			fmt.Fprintf(stdout, "  %s\n", u)
		}
		return ExitOK
	}

	cfg := client.Config{
		Key:         opts.Key,
		Host:        host,
		KeyLocation: opts.KeyLocation,
		Endpoint:    endpoint,
		MaxRetries:  opts.MaxRetries,
		BaseBackoff: opts.BaseBackoff,
		MaxBackoff:  opts.MaxBackoff,
	}
	sub, err := factory(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "client init: %v\n", err)
		return ExitFailed
	}

	results, err := sub.SubmitBatch(ctx, urls)
	if err != nil {
		fmt.Fprintf(stderr, "submit: %v\n", err)
		return ExitFailed
	}

	if err := renderResults(stdout, results, opts.Output, endpoint); err != nil {
		fmt.Fprintf(stderr, "render: %v\n", err)
		return ExitFailed
	}

	if shouldFail(results, opts.FailOn) {
		return ExitFailed
	}
	return ExitOK
}

func validateOutput(v string) error {
	switch v {
	case OutputText, OutputJSON, "":
		return nil
	default:
		return fmt.Errorf("%w: %q (allowed: text, json)", ErrInvalidOutput, v)
	}
}

func validateFailOn(v string) error {
	switch v {
	case FailOnAny, FailOn4xx, FailOn5xx, FailOnNever, "":
		return nil
	default:
		return fmt.Errorf("%w: %q (allowed: any, 4xx, 5xx, never)", ErrInvalidFailOn, v)
	}
}

func collectURLs(opts SubmitOptions, stdin io.Reader) ([]string, error) {
	sources := 0
	if len(opts.Args) > 0 {
		sources++
	}
	if opts.File != "" {
		sources++
	}
	if opts.Stdin {
		sources++
	}
	switch sources {
	case 0:
		return nil, ErrNoSource
	case 1:
		// ok
	default:
		return nil, ErrSourceConflict
	}

	switch {
	case len(opts.Args) > 0:
		return opts.Args, nil
	case opts.File != "":
		f, err := os.Open(opts.File)
		if err != nil {
			return nil, fmt.Errorf("open --file: %w", err)
		}
		defer f.Close()
		return parseURLLines(f)
	default:
		return parseURLLines(stdin)
	}
}

func parseURLLines(r io.Reader) ([]string, error) {
	if r == nil {
		return nil, ErrNoURLs
	}
	var urls []string
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(urls) == 0 {
		return nil, ErrNoURLs
	}
	return urls, nil
}

func hostFromURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("url %q has no scheme or host", rawURL)
	}
	return u.Hostname(), nil
}

type jsonResult struct {
	Endpoint string   `json:"endpoint"`
	Status   int      `json:"status"`
	Attempts int      `json:"attempts"`
	URLCount int      `json:"urlCount"`
	URLs     []string `json:"urls,omitempty"`
	Error    string   `json:"error,omitempty"`
}

func renderResults(w io.Writer, results []*client.Result, output, endpoint string) error {
	if output == "" {
		output = OutputText
	}
	if output == OutputJSON {
		out := make([]jsonResult, len(results))
		for i, r := range results {
			jr := jsonResult{
				Endpoint: endpoint,
				Status:   r.StatusCode,
				Attempts: r.Attempts,
				URLCount: len(r.URLs),
				URLs:     r.URLs,
			}
			if r.Err != nil {
				jr.Error = r.Err.Error()
			}
			out[i] = jr
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}
	for i, r := range results {
		status := "OK"
		if r.Err != nil {
			status = "FAIL"
		}
		fmt.Fprintf(w, "batch %d: %s status=%d attempts=%d urls=%d", i+1, status, r.StatusCode, r.Attempts, len(r.URLs))
		if r.Err != nil {
			fmt.Fprintf(w, " err=%v", r.Err)
		}
		fmt.Fprintln(w)
	}
	return nil
}

func shouldFail(results []*client.Result, failOn string) bool {
	if failOn == "" {
		failOn = FailOnAny
	}
	if failOn == FailOnNever {
		return false
	}
	for _, r := range results {
		switch failOn {
		case FailOnAny:
			if r.Err != nil || r.StatusCode < 200 || r.StatusCode >= 300 {
				return true
			}
		case FailOn4xx:
			if r.StatusCode >= 400 && r.StatusCode < 500 {
				return true
			}
		case FailOn5xx:
			if r.StatusCode >= 500 && r.StatusCode < 600 {
				return true
			}
		}
	}
	return false
}
