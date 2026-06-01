package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_AllFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	body := "host: example.com\nkey: abc123\nkey_location: https://example.com/abc123.txt\nendpoint: bing\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := &Config{
		Host:        "example.com",
		Key:         "abc123",
		KeyLocation: "https://example.com/abc123.txt",
		Endpoint:    "bing",
	}
	if *got != *want {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestLoad_PartialFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("key: only-key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Key != "only-key" || got.Host != "" || got.Endpoint != "" {
		t.Fatalf("got %+v", got)
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if (*got != Config{}) {
		t.Fatalf("expected zero Config, got %+v", got)
	}
}

func TestLoad_Missing(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want ErrNotExist, got %v", err)
	}
}

func TestLoad_Malformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("host: [unbalanced\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLoad_UnknownField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("host: example.com\nbogus: 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected error on unknown field")
	}
}

func TestDefaultPath_XDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	got := DefaultPath()
	want := filepath.Join("/tmp/xdg", "indexnow", "config.yaml")
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestDefaultPath_HomeFallback(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/tmp/home")
	got := DefaultPath()
	want := filepath.Join("/tmp/home", ".config", "indexnow", "config.yaml")
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}
