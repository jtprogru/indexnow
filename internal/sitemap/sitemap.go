// Package sitemap parses sitemap.org documents (urlset + sitemapindex) and
// resolves them recursively to a flat list of page URLs, optionally filtered
// by <lastmod>.
//
// The package is namespace-agnostic: it matches element local names, so it
// handles documents that omit the xmlns declaration as well as the canonical
// form. It streams the input through encoding/xml's tokenizer so large
// sitemaps (50 MB / 50 000 entries — the sitemap.org limit) parse without
// loading the whole document into memory.
package sitemap

import (
	"compress/gzip"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// MaxDepth caps sitemapindex recursion to guard against pathological loops
// (an index that references itself transitively).
const MaxDepth = 5

// Entry is a single <url> child of a <urlset>.
type Entry struct {
	Loc        string
	LastMod    time.Time
	HasLastMod bool
}

// Parse streams a single sitemap document. It returns URL entries from a
// <urlset> document and child-sitemap locations from a <sitemapindex>
// document. For well-formed input exactly one of the two will be non-empty.
func Parse(r io.Reader) ([]Entry, []string, error) {
	dec := xml.NewDecoder(r)
	var (
		entries []Entry
		nested  []string
	)
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "url":
			var u struct {
				Loc     string `xml:"loc"`
				LastMod string `xml:"lastmod"`
			}
			if err := dec.DecodeElement(&u, &se); err != nil {
				return nil, nil, err
			}
			loc := strings.TrimSpace(u.Loc)
			if loc == "" {
				continue
			}
			e := Entry{Loc: loc}
			if s := strings.TrimSpace(u.LastMod); s != "" {
				if t, perr := parseLastMod(s); perr == nil {
					e.LastMod = t
					e.HasLastMod = true
				}
			}
			entries = append(entries, e)
		case "sitemap":
			var s struct {
				Loc string `xml:"loc"`
			}
			if err := dec.DecodeElement(&s, &se); err != nil {
				return nil, nil, err
			}
			if loc := strings.TrimSpace(s.Loc); loc != "" {
				nested = append(nested, loc)
			}
		}
	}
	return entries, nested, nil
}

// parseLastMod accepts the W3C Datetime forms and the date-only form allowed
// by the sitemap.org spec. Unrecognized strings are reported as errors so
// callers can decide whether to treat them as "include unconditionally".
func parseLastMod(s string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z07:00", "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized lastmod %q", s)
}

// OpenFunc opens a sitemap source for reading. src is either an absolute
// http/https URL or a local filesystem path. Implementations are responsible
// for transparently decompressing gzipped sources.
type OpenFunc func(ctx context.Context, src string) (io.ReadCloser, error)

// Collect fetches the sitemap at src and returns all URLs reachable through
// nested <sitemapindex> entries. If since is non-zero, entries whose <lastmod>
// is strictly before since are excluded. Entries without a <lastmod>, or with
// an unparseable one, are always included — "no signal" is treated as "may
// have changed", which is the safe default for a notification protocol.
//
// Recursion is capped at MaxDepth and already-visited sources are skipped to
// keep the call bounded on broken or self-referential indexes.
func Collect(ctx context.Context, src string, since time.Time, open OpenFunc) ([]string, error) {
	if open == nil {
		return nil, errors.New("sitemap: nil OpenFunc")
	}
	seen := map[string]bool{}
	var collected []string
	var visit func(s string, depth int) error
	visit = func(s string, depth int) error {
		if depth > MaxDepth {
			return fmt.Errorf("sitemap: recursion depth %d exceeded at %s", MaxDepth, s)
		}
		if seen[s] {
			return nil
		}
		seen[s] = true
		rc, err := open(ctx, s)
		if err != nil {
			return fmt.Errorf("fetch %s: %w", s, err)
		}
		entries, nested, perr := Parse(rc)
		_ = rc.Close()
		if perr != nil {
			return fmt.Errorf("parse %s: %w", s, perr)
		}
		for _, e := range entries {
			if !since.IsZero() && e.HasLastMod && e.LastMod.Before(since) {
				continue
			}
			collected = append(collected, e.Loc)
		}
		for _, n := range nested {
			if err := visit(n, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(src, 0); err != nil {
		return nil, err
	}
	return collected, nil
}

// DefaultOpen returns an OpenFunc that fetches http/https URLs with the given
// HTTP client and User-Agent, or reads local filesystem paths. Sources whose
// path/URL ends in `.gz` are gunzipped transparently; gzip-encoded HTTP
// responses without a `.gz` suffix are handled by the net/http stack itself.
func DefaultOpen(client *http.Client, userAgent string) OpenFunc {
	if client == nil {
		client = http.DefaultClient
	}
	return func(ctx context.Context, src string) (io.ReadCloser, error) {
		rc, err := openRaw(ctx, src, client, userAgent)
		if err != nil {
			return nil, err
		}
		if strings.HasSuffix(strings.ToLower(src), ".gz") {
			gz, gerr := gzip.NewReader(rc)
			if gerr != nil {
				_ = rc.Close()
				return nil, fmt.Errorf("gunzip %s: %w", src, gerr)
			}
			return gunzipReadCloser{Reader: gz, underlying: rc}, nil
		}
		return rc, nil
	}
}

func openRaw(ctx context.Context, src string, client *http.Client, userAgent string) (io.ReadCloser, error) {
	if u, err := url.Parse(src); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		req, rerr := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
		if rerr != nil {
			return nil, rerr
		}
		if userAgent != "" {
			req.Header.Set("User-Agent", userAgent)
		}
		resp, derr := client.Do(req)
		if derr != nil {
			return nil, derr
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("http %d", resp.StatusCode)
		}
		return resp.Body, nil
	}
	f, ferr := os.Open(src)
	if ferr != nil {
		return nil, ferr
	}
	return f, nil
}

type gunzipReadCloser struct {
	*gzip.Reader
	underlying io.Closer
}

func (g gunzipReadCloser) Close() error {
	gzErr := g.Reader.Close()
	rcErr := g.underlying.Close()
	if gzErr != nil {
		return gzErr
	}
	return rcErr
}
