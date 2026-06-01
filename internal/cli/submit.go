// Package cli implements the indexnow CLI behavior in a testable form.
// Cobra wiring lives in cmd/indexnow; this package owns the logic.
package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"
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
	UserAgent   string
	File        string
	Stdin       bool
	Args        []string
	DryRun      bool
	Output      string
	FailOn      string
	Quiet       bool
	Verbose     bool
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

// newLogger returns a slog.Logger that writes lifecycle and retry events
// to stderr when --verbose is on, or a no-op handler otherwise. Output
// goes to stderr so it never collides with the stdout result stream
// (text batches or JSON) that scripts may capture.
func newLogger(verbose bool, stderr io.Writer) *slog.Logger {
	if !verbose {
		return slog.New(slog.DiscardHandler)
	}
	return slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func RunSubmit(ctx context.Context, opts SubmitOptions, stdin io.Reader, stdout, stderr io.Writer, factory SubmitterFactory) int {
	if factory == nil {
		factory = defaultFactory
	}
	logger := newLogger(opts.Verbose, stderr)
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

	endpoints, err := client.ResolveEndpoints(opts.Endpoint)
	if err != nil {
		fmt.Fprintf(stderr, "endpoint %q: %v\n", opts.Endpoint, err)
		return ExitUsageError
	}

	if opts.DryRun {
		if !opts.Quiet {
			key := "endpoint"
			if len(endpoints) > 1 {
				key = "endpoints"
			}
			fmt.Fprintf(stdout, "[dry-run] %s=%s host=%s urls=%d\n", key, strings.Join(endpoints, ","), host, len(urls))
			for _, u := range urls {
				fmt.Fprintf(stdout, "  %s\n", u)
			}
		}
		return ExitOK
	}

	logger.Info("submit", "host", host, "urls", len(urls), "endpoints", len(endpoints))
	batches := fanOut(ctx, opts, host, urls, endpoints, factory, logger)
	for _, eb := range batches {
		if eb.Err != nil {
			logger.Warn("endpoint failed", "endpoint", eb.Endpoint, "err", eb.Err)
			continue
		}
		for i, r := range eb.Results {
			level := slog.LevelInfo
			if r.Err != nil {
				level = slog.LevelWarn
			}
			logger.Log(ctx, level, "batch complete", "endpoint", eb.Endpoint, "batch", i+1, "status", r.StatusCode, "attempts", r.Attempts, "urls", len(r.URLs))
		}
	}

	if !opts.Quiet {
		if err := renderResults(stdout, batches, opts.Output); err != nil {
			fmt.Fprintf(stderr, "render: %v\n", err)
			return ExitFailed
		}
	}

	if shouldFail(batches, opts.FailOn) {
		return ExitFailed
	}
	return ExitOK
}

// endpointBatch is the per-endpoint outcome of a fan-out submission. Err
// captures factory/transport errors that abort that endpoint entirely; the
// per-batch errors live inside Results.
type endpointBatch struct {
	Endpoint string
	Results  []*client.Result
	Err      error
}

func fanOut(ctx context.Context, opts SubmitOptions, host string, urls, endpoints []string, factory SubmitterFactory, logger *slog.Logger) []endpointBatch {
	out := make([]endpointBatch, len(endpoints))
	var wg sync.WaitGroup
	for i, ep := range endpoints {
		wg.Add(1)
		go func(i int, ep string) {
			defer wg.Done()
			out[i].Endpoint = ep
			cfg := client.Config{
				Key:         opts.Key,
				Host:        host,
				KeyLocation: opts.KeyLocation,
				Endpoint:    ep,
				UserAgent:   opts.UserAgent,
				Logger:      logger,
				MaxRetries:  opts.MaxRetries,
				BaseBackoff: opts.BaseBackoff,
				MaxBackoff:  opts.MaxBackoff,
			}
			sub, err := factory(cfg)
			if err != nil {
				out[i].Err = fmt.Errorf("client init: %w", err)
				return
			}
			rs, err := sub.SubmitBatch(ctx, urls)
			if err != nil {
				out[i].Err = fmt.Errorf("submit: %w", err)
				return
			}
			out[i].Results = rs
		}(i, ep)
	}
	wg.Wait()
	return out
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

func renderResults(w io.Writer, batches []endpointBatch, output string) error {
	if output == "" {
		output = OutputText
	}
	if output == OutputJSON {
		var out []jsonResult
		for _, eb := range batches {
			if eb.Err != nil {
				out = append(out, jsonResult{Endpoint: eb.Endpoint, Error: eb.Err.Error()})
				continue
			}
			for _, r := range eb.Results {
				jr := jsonResult{
					Endpoint: eb.Endpoint,
					Status:   r.StatusCode,
					Attempts: r.Attempts,
					URLCount: len(r.URLs),
					URLs:     r.URLs,
				}
				if r.Err != nil {
					jr.Error = r.Err.Error()
				}
				out = append(out, jr)
			}
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}
	multi := len(batches) > 1
	for _, eb := range batches {
		prefix := ""
		if multi {
			prefix = fmt.Sprintf("endpoint=%s ", eb.Endpoint)
		}
		if eb.Err != nil {
			fmt.Fprintf(w, "%sERROR: %v\n", prefix, eb.Err)
			continue
		}
		for i, r := range eb.Results {
			status := "OK"
			if r.Err != nil {
				status = "FAIL"
			}
			fmt.Fprintf(w, "%sbatch %d: %s status=%d attempts=%d urls=%d", prefix, i+1, status, r.StatusCode, r.Attempts, len(r.URLs))
			if r.Err != nil {
				fmt.Fprintf(w, " err=%v", r.Err)
			}
			fmt.Fprintln(w)
		}
	}
	return nil
}

func shouldFail(batches []endpointBatch, failOn string) bool {
	if failOn == "" {
		failOn = FailOnAny
	}
	// Endpoint-level errors (factory init, transport failure that aborts the
	// whole batch) are system-level failures, not HTTP outcomes, so they
	// trigger a non-zero exit even under --fail-on=never.
	for _, eb := range batches {
		if eb.Err != nil {
			return true
		}
	}
	if failOn == FailOnNever {
		return false
	}
	for _, eb := range batches {
		for _, r := range eb.Results {
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
	}
	return false
}
