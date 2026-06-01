package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jtprogru/indexnow/internal/cli"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// Isolate from any real config file the developer may have at the XDG
// default path while still letting DefaultPath() produce a stable answer.
func isolateXDG(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func TestApplyConfig_FillsEmptyFromExplicitFile(t *testing.T) {
	isolateXDG(t)
	path := writeConfig(t, "host: cfg.example\nkey: cfg-key\nkey_location: https://cfg.example/k.txt\nendpoint: bing\n")
	opts := cli.SubmitOptions{Endpoint: "api"}
	if err := applyConfig(&opts, path); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	if opts.Host != "cfg.example" || opts.Key != "cfg-key" || opts.KeyLocation != "https://cfg.example/k.txt" || opts.Endpoint != "bing" {
		t.Fatalf("got %+v", opts)
	}
}

func TestApplyConfig_PreservesPrecedence(t *testing.T) {
	// Flag/env already populated opts — config must not overwrite.
	isolateXDG(t)
	path := writeConfig(t, "host: cfg.example\nkey: cfg-key\nendpoint: bing\n")
	opts := cli.SubmitOptions{
		Host:     "flag.example",
		Key:      "flag-key",
		Endpoint: "yandex",
	}
	if err := applyConfig(&opts, path); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	if opts.Host != "flag.example" || opts.Key != "flag-key" || opts.Endpoint != "yandex" {
		t.Fatalf("config should not override set fields; got %+v", opts)
	}
}

func TestApplyConfig_EndpointDefaultIsOverridable(t *testing.T) {
	// Endpoint's "api" is the built-in default, so config wins over it.
	isolateXDG(t)
	path := writeConfig(t, "endpoint: bing\n")
	opts := cli.SubmitOptions{Endpoint: "api"}
	if err := applyConfig(&opts, path); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	if opts.Endpoint != "bing" {
		t.Fatalf("want bing, got %s", opts.Endpoint)
	}
}

func TestApplyConfig_ExplicitMissingFileIsError(t *testing.T) {
	opts := cli.SubmitOptions{}
	err := applyConfig(&opts, filepath.Join(t.TempDir(), "absent.yaml"))
	if err == nil {
		t.Fatal("expected error for explicit missing file")
	}
	if !strings.Contains(err.Error(), "config:") {
		t.Fatalf("error should mention config prefix, got %v", err)
	}
}

func TestApplyConfig_DefaultMissingFileIsSilent(t *testing.T) {
	// Point XDG at an empty dir — no config.yaml there.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	opts := cli.SubmitOptions{Endpoint: "api"}
	if err := applyConfig(&opts, ""); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	// No-op: defaults unchanged.
	if opts.Endpoint != "api" || opts.Key != "" {
		t.Fatalf("got %+v", opts)
	}
}

func TestApplyConfig_DefaultPathIsRespected(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	if err := os.MkdirAll(filepath.Join(xdg, "indexnow"), 0o700); err != nil {
		t.Fatal(err)
	}
	body := "host: from-xdg.example\nendpoint: bing\n"
	if err := os.WriteFile(filepath.Join(xdg, "indexnow", "config.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	opts := cli.SubmitOptions{Endpoint: "api"}
	if err := applyConfig(&opts, ""); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	if opts.Host != "from-xdg.example" || opts.Endpoint != "bing" {
		t.Fatalf("got %+v", opts)
	}
}
