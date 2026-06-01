package sitemap

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

const urlset = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://example.com/a</loc>
    <lastmod>2026-05-01T00:00:00Z</lastmod>
  </url>
  <url>
    <loc>https://example.com/b</loc>
    <lastmod>2026-06-01</lastmod>
  </url>
  <url>
    <loc>https://example.com/c</loc>
  </url>
</urlset>`

const indexBody = `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap><loc>%s</loc></sitemap>
  <sitemap><loc>%s</loc></sitemap>
</sitemapindex>`

func TestParse_URLSet(t *testing.T) {
	entries, nested, err := Parse(strings.NewReader(urlset))
	if err != nil {
		t.Fatal(err)
	}
	if len(nested) != 0 {
		t.Errorf("urlset must not yield nested sitemaps; got %v", nested)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}
	if !entries[0].HasLastMod || entries[0].LastMod.Year() != 2026 {
		t.Errorf("entry[0] lastmod parse failed: %+v", entries[0])
	}
	if !entries[1].HasLastMod || entries[1].LastMod.Month() != time.June {
		t.Errorf("entry[1] date-only lastmod parse failed: %+v", entries[1])
	}
	if entries[2].HasLastMod {
		t.Errorf("entry[2] should have no lastmod; got %+v", entries[2])
	}
}

func TestParse_NoNamespace(t *testing.T) {
	doc := `<urlset><url><loc>https://example.com/x</loc></url></urlset>`
	entries, _, err := Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Loc != "https://example.com/x" {
		t.Errorf("got %+v", entries)
	}
}

func TestParse_SitemapIndex(t *testing.T) {
	doc := fmt.Sprintf(indexBody, "https://example.com/s1.xml", "https://example.com/s2.xml")
	entries, nested, err := Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("sitemapindex must not yield url entries; got %v", entries)
	}
	if len(nested) != 2 {
		t.Fatalf("got %d nested, want 2", len(nested))
	}
}

func TestParse_BadXML(t *testing.T) {
	_, _, err := Parse(strings.NewReader("<urlset><url>"))
	if err == nil {
		t.Fatal("expected parse error on truncated xml")
	}
}

func TestParse_BadLastModTreatedAsAbsent(t *testing.T) {
	doc := `<urlset><url><loc>https://example.com/x</loc><lastmod>not-a-date</lastmod></url></urlset>`
	entries, _, err := Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d, want 1", len(entries))
	}
	if entries[0].HasLastMod {
		t.Errorf("unparseable lastmod must leave HasLastMod=false: %+v", entries[0])
	}
}

func openFromMap(m map[string]string) OpenFunc {
	return func(_ context.Context, src string) (io.ReadCloser, error) {
		body, ok := m[src]
		if !ok {
			return nil, fmt.Errorf("no entry for %s", src)
		}
		return io.NopCloser(strings.NewReader(body)), nil
	}
}

func TestCollect_FlattensIndex(t *testing.T) {
	leaf := `<urlset><url><loc>https://example.com/a</loc></url><url><loc>https://example.com/b</loc></url></urlset>`
	idx := fmt.Sprintf(indexBody, "https://example.com/leaf1.xml", "https://example.com/leaf2.xml")
	m := map[string]string{
		"https://example.com/index.xml": idx,
		"https://example.com/leaf1.xml": leaf,
		"https://example.com/leaf2.xml": `<urlset><url><loc>https://example.com/c</loc></url></urlset>`,
	}
	got, err := Collect(context.Background(), "https://example.com/index.xml", time.Time{}, openFromMap(m))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(got)
	want := []string{"https://example.com/a", "https://example.com/b", "https://example.com/c"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCollect_SinceFilter(t *testing.T) {
	doc := `<urlset>
		<url><loc>https://example.com/old</loc><lastmod>2026-01-01</lastmod></url>
		<url><loc>https://example.com/new</loc><lastmod>2026-06-01</lastmod></url>
		<url><loc>https://example.com/unknown</loc></url>
	</urlset>`
	m := map[string]string{"src": doc}
	since, _ := time.Parse("2006-01-02", "2026-05-01")
	got, err := Collect(context.Background(), "src", since, openFromMap(m))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(got)
	// "old" filtered out, "new" passes (lastmod >= since), "unknown" included
	// because absent-signal is treated as "may have changed".
	want := []string{"https://example.com/new", "https://example.com/unknown"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCollect_SelfReferenceTerminates(t *testing.T) {
	loop := `<sitemapindex><sitemap><loc>src</loc></sitemap></sitemapindex>`
	m := map[string]string{"src": loop}
	got, err := Collect(context.Background(), "src", time.Time{}, openFromMap(m))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want empty (loop deduped)", got)
	}
}

func TestCollect_DepthCap(t *testing.T) {
	// Build a chain longer than MaxDepth so recursion has to bail.
	m := map[string]string{}
	chain := MaxDepth + 3
	for i := range chain {
		next := fmt.Sprintf("s%d", i+1)
		m[fmt.Sprintf("s%d", i)] = fmt.Sprintf(`<sitemapindex><sitemap><loc>%s</loc></sitemap></sitemapindex>`, next)
	}
	m[fmt.Sprintf("s%d", chain)] = `<urlset><url><loc>https://example.com/x</loc></url></urlset>`
	_, err := Collect(context.Background(), "s0", time.Time{}, openFromMap(m))
	if err == nil {
		t.Fatal("expected depth-cap error")
	}
	if !strings.Contains(err.Error(), "recursion") {
		t.Errorf("error should mention recursion: %v", err)
	}
}

func TestCollect_FetchErrorPropagated(t *testing.T) {
	open := func(_ context.Context, _ string) (io.ReadCloser, error) {
		return nil, errors.New("nope")
	}
	_, err := Collect(context.Background(), "src", time.Time{}, open)
	if err == nil || !strings.Contains(err.Error(), "fetch src") {
		t.Errorf("got %v", err)
	}
}

func TestDefaultOpen_HTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != "test/1" {
			t.Errorf("UA: got %q, want test/1", got)
		}
		_, _ = w.Write([]byte(urlset))
	}))
	defer srv.Close()
	open := DefaultOpen(srv.Client(), "test/1")
	got, err := Collect(context.Background(), srv.URL+"/sitemap.xml", time.Time{}, open)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Errorf("got %d urls, want 3", len(got))
	}
}

func TestDefaultOpen_HTTPNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()
	open := DefaultOpen(srv.Client(), "")
	_, err := Collect(context.Background(), srv.URL, time.Time{}, open)
	if err == nil || !strings.Contains(err.Error(), "http 404") {
		t.Errorf("got %v", err)
	}
}

func TestDefaultOpen_LocalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sitemap.xml")
	if err := os.WriteFile(path, []byte(urlset), 0o644); err != nil {
		t.Fatal(err)
	}
	open := DefaultOpen(nil, "")
	got, err := Collect(context.Background(), path, time.Time{}, open)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Errorf("got %d, want 3", len(got))
	}
}

func TestDefaultOpen_GzipLocalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sitemap.xml.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	if _, err := gw.Write([]byte(urlset)); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	open := DefaultOpen(nil, "")
	got, err := Collect(context.Background(), path, time.Time{}, open)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Errorf("got %d, want 3", len(got))
	}
}
