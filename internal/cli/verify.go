package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jtprogru/indexnow/internal/client"
)

// Cap how much of the hosted key file we read. The protocol expects a
// short opaque string; anything pathological is almost certainly the
// wrong file, and we keep diagnostics from blowing up the heap.
const verifyMaxBody = 1 << 20

type VerifyOptions struct {
	Key         string
	Host        string
	KeyLocation string
	UserAgent   string
	Output      string
	Quiet       bool
	Verbose     bool
	Timeout     time.Duration
}

type VerifyResult struct {
	URL    string `json:"url"`
	OK     bool   `json:"ok"`
	Status int    `json:"status,omitempty"`
	Hosted string `json:"hosted,omitempty"` // truncated hosted body on mismatch
	Error  string `json:"error,omitempty"`
}

// RunVerify performs the same check that participating IndexNow endpoints
// perform: GET the hosted key file and compare its trimmed body to the
// expected key. Returns ExitOK only when the hosted value matches.
func RunVerify(ctx context.Context, opts VerifyOptions, stdout, stderr io.Writer, httpClient *http.Client) int {
	logger := newLogger(opts.Verbose, stderr)
	if err := validateOutput(opts.Output); err != nil {
		fmt.Fprintln(stderr, err)
		return ExitUsageError
	}
	if opts.Key == "" {
		fmt.Fprintln(stderr, "indexnow cli: --key (or INDEXNOW_KEY) is required")
		return ExitUsageError
	}
	target, err := resolveKeyURL(opts)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return ExitUsageError
	}
	if httpClient == nil {
		timeout := opts.Timeout
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}
	logger.Info("verify", "url", target)
	res := doVerify(ctx, httpClient, target, opts.Key, opts.UserAgent)
	if res.OK {
		logger.Info("verify ok", "url", target)
	} else {
		logger.Warn("verify failed", "url", target, "status", res.Status, "err", res.Error)
	}
	if !opts.Quiet {
		if err := renderVerifyResult(stdout, res, opts.Output); err != nil {
			fmt.Fprintf(stderr, "render: %v\n", err)
			return ExitFailed
		}
	}
	if !res.OK {
		return ExitFailed
	}
	return ExitOK
}

func resolveKeyURL(opts VerifyOptions) (string, error) {
	if opts.KeyLocation != "" {
		u, err := url.Parse(opts.KeyLocation)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return "", fmt.Errorf("--key-location must be an absolute URL: %q", opts.KeyLocation)
		}
		return opts.KeyLocation, nil
	}
	if opts.Host == "" {
		return "", errors.New("indexnow cli: need --key-location, or --host (to derive https://<host>/<key>.txt)")
	}
	if err := client.ValidateKey(opts.Key); err != nil {
		return "", fmt.Errorf("invalid --key: %w", err)
	}
	return "https://" + opts.Host + "/" + opts.Key + ".txt", nil
}

func doVerify(ctx context.Context, httpClient *http.Client, target, expectedKey, ua string) *VerifyResult {
	res := &VerifyResult{URL: target}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	if ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer func() { _ = resp.Body.Close() }()
	res.Status = resp.StatusCode
	body, err := io.ReadAll(io.LimitReader(resp.Body, verifyMaxBody))
	if err != nil {
		res.Error = err.Error()
		return res
	}
	if resp.StatusCode != http.StatusOK {
		res.Error = fmt.Sprintf("http %d", resp.StatusCode)
		return res
	}
	hosted := strings.TrimSpace(string(body))
	if hosted != expectedKey {
		if len(hosted) > 80 {
			hosted = hosted[:80] + "…"
		}
		res.Hosted = hosted
		res.Error = "hosted key does not match expected"
		return res
	}
	res.OK = true
	return res
}

func renderVerifyResult(w io.Writer, res *VerifyResult, output string) error {
	if output == "" {
		output = OutputText
	}
	if output == OutputJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}
	if res.OK {
		fmt.Fprintf(w, "OK: %s\n", res.URL)
		return nil
	}
	fmt.Fprintf(w, "FAIL: %s", res.URL)
	if res.Status != 0 {
		fmt.Fprintf(w, " status=%d", res.Status)
	}
	if res.Error != "" {
		fmt.Fprintf(w, " err=%s", res.Error)
	}
	fmt.Fprintln(w)
	return nil
}
