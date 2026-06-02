package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var hexRe = regexp.MustCompile(`^[0-9a-f]+$`)

func TestRunKeygen_DefaultLengthAndFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := RunKeygen(KeygenOptions{}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit code: got %d want %d (stderr=%q)", code, ExitOK, stderr.String())
	}
	key := strings.TrimRight(stdout.String(), "\n")
	if len(key) != keygenDefaultLength {
		t.Fatalf("key length: got %d want %d", len(key), keygenDefaultLength)
	}
	if !hexRe.MatchString(key) {
		t.Fatalf("key not hex: %q", key)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr without --write; got %q", stderr.String())
	}
}

func TestRunKeygen_LengthBoundaries(t *testing.T) {
	for _, n := range []int{keygenMinLength, 16, 64, keygenMaxLength} {
		var stdout, stderr bytes.Buffer
		code := RunKeygen(KeygenOptions{Length: n}, &stdout, &stderr)
		if code != ExitOK {
			t.Fatalf("length=%d: exit %d (stderr=%q)", n, code, stderr.String())
		}
		key := strings.TrimRight(stdout.String(), "\n")
		if len(key) != n {
			t.Errorf("length=%d: got key of length %d", n, len(key))
		}
		if !hexRe.MatchString(key) {
			t.Errorf("length=%d: not hex: %q", n, key)
		}
	}
}

func TestRunKeygen_LengthOutOfRange(t *testing.T) {
	for _, n := range []int{1, keygenMinLength - 1, keygenMaxLength + 1, 1000} {
		var stdout, stderr bytes.Buffer
		code := RunKeygen(KeygenOptions{Length: n}, &stdout, &stderr)
		if code != ExitUsageError {
			t.Errorf("length=%d: got code %d, want ExitUsageError", n, code)
		}
		if !strings.Contains(stderr.String(), "8..128") {
			t.Errorf("length=%d: stderr should mention range, got %q", n, stderr.String())
		}
		if stdout.Len() != 0 {
			t.Errorf("length=%d: stdout should be empty, got %q", n, stdout.String())
		}
	}
}

func TestRunKeygen_WriteSucceeds(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := RunKeygen(KeygenOptions{Write: dir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit %d (stderr=%q)", code, stderr.String())
	}
	key := strings.TrimRight(stdout.String(), "\n")
	wantPath := filepath.Join(dir, key+".txt")
	body, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if got := strings.TrimRight(string(body), "\n"); got != key {
		t.Errorf("file body: got %q, want %q", got, key)
	}
	info, err := os.Stat(wantPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != keygenFileMode {
		t.Errorf("perm: got %o, want %o", perm, keygenFileMode)
	}
	if !strings.Contains(stderr.String(), "wrote "+wantPath) {
		t.Errorf("stderr should mention wrote-path, got %q", stderr.String())
	}
}

func TestRunKeygen_WriteFailsOnMissingDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does", "not", "exist")
	var stdout, stderr bytes.Buffer
	code := RunKeygen(KeygenOptions{Write: missing}, &stdout, &stderr)
	if code != ExitFailed {
		t.Fatalf("exit %d, want ExitFailed", code)
	}
	if !strings.Contains(stderr.String(), missing) {
		t.Errorf("stderr should mention path, got %q", stderr.String())
	}
}

func TestRunKeygen_WriteRefusesExisting(t *testing.T) {
	// Use a known key so the path is deterministic.
	dir := t.TempDir()

	// Generate once.
	var stdout1, stderr1 bytes.Buffer
	if code := RunKeygen(KeygenOptions{Write: dir}, &stdout1, &stderr1); code != ExitOK {
		t.Fatalf("first gen: %d (%q)", code, stderr1.String())
	}
	key := strings.TrimRight(stdout1.String(), "\n")

	// Stub randRead to return a deterministic value that yields the same key.
	withSeededRand(t, key, func() {
		var stdout2, stderr2 bytes.Buffer
		code := RunKeygen(KeygenOptions{Write: dir}, &stdout2, &stderr2)
		if code != ExitFailed {
			t.Fatalf("expected ExitFailed on existing file, got %d", code)
		}
		if !strings.Contains(stderr2.String(), "already exists") {
			t.Errorf("stderr should mention 'already exists', got %q", stderr2.String())
		}
		if !strings.Contains(stderr2.String(), "--force") {
			t.Errorf("stderr should mention --force, got %q", stderr2.String())
		}
	})
}

func TestRunKeygen_ForceOverwrites(t *testing.T) {
	dir := t.TempDir()

	// Pre-create a fake key file by writing one with a stub seed.
	var firstStdout, firstStderr bytes.Buffer
	if code := RunKeygen(KeygenOptions{Write: dir}, &firstStdout, &firstStderr); code != ExitOK {
		t.Fatalf("first gen: %d (%q)", code, firstStderr.String())
	}
	firstKey := strings.TrimRight(firstStdout.String(), "\n")
	firstPath := filepath.Join(dir, firstKey+".txt")

	// Overwrite with same key via deterministic seed + --force.
	withSeededRand(t, firstKey, func() {
		var stdout, stderr bytes.Buffer
		code := RunKeygen(KeygenOptions{Write: dir, Force: true}, &stdout, &stderr)
		if code != ExitOK {
			t.Fatalf("force gen: %d (%q)", code, stderr.String())
		}
	})

	// File should still exist and contain the (same) key.
	body, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatalf("reading file after --force: %v", err)
	}
	if got := strings.TrimRight(string(body), "\n"); got != firstKey {
		t.Errorf("body after --force: got %q want %q", got, firstKey)
	}
}

func TestRunKeygen_OutputJSON(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := RunKeygen(KeygenOptions{Output: OutputJSON, Write: dir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit %d (%q)", code, stderr.String())
	}
	var got keygenJSON
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decoding json: %v (stdout=%q)", err, stdout.String())
	}
	if !hexRe.MatchString(got.Key) || len(got.Key) != keygenDefaultLength {
		t.Errorf("key shape: %q", got.Key)
	}
	if got.Path != filepath.Join(dir, got.Key+".txt") {
		t.Errorf("path: got %q", got.Path)
	}
}

func TestRunKeygen_OutputJSONNoWrite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := RunKeygen(KeygenOptions{Output: OutputJSON}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit %d (%q)", code, stderr.String())
	}
	var got keygenJSON
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decoding json: %v", err)
	}
	if got.Path != "" {
		t.Errorf("path should be empty without --write, got %q", got.Path)
	}
}

func TestRunKeygen_InvalidOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := RunKeygen(KeygenOptions{Output: "yaml"}, &stdout, &stderr)
	if code != ExitUsageError {
		t.Fatalf("got %d want ExitUsageError", code)
	}
	if !strings.Contains(stderr.String(), "yaml") {
		t.Errorf("stderr should mention 'yaml', got %q", stderr.String())
	}
}

func TestRunKeygen_QuietSuppressesStdout(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := RunKeygen(KeygenOptions{Write: dir, Quiet: true}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit %d (%q)", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout should be empty under --quiet, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "wrote ") {
		t.Errorf("stderr should still contain wrote-notice, got %q", stderr.String())
	}
	// The file should still exist (it's the whole point of --write under --quiet).
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("expected one file in %s, got %d", dir, len(entries))
	}
}

func TestRunKeygen_EntropyDistinct(t *testing.T) {
	const n = 16
	seen := make(map[string]struct{}, n)
	for i := range n {
		var stdout, stderr bytes.Buffer
		if code := RunKeygen(KeygenOptions{}, &stdout, &stderr); code != ExitOK {
			t.Fatalf("iter %d: code %d", i, code)
		}
		key := strings.TrimRight(stdout.String(), "\n")
		if _, dup := seen[key]; dup {
			t.Fatalf("collision in %d generations (key=%q)", n, key)
		}
		seen[key] = struct{}{}
	}
}

func TestRunKeygen_WriteIODirIsAFile(t *testing.T) {
	// --write target exists but is not a directory.
	tmp := t.TempDir()
	notDir := filepath.Join(tmp, "regular-file")
	if err := os.WriteFile(notDir, []byte("hi"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := RunKeygen(KeygenOptions{Write: notDir}, &stdout, &stderr)
	if code != ExitFailed {
		t.Fatalf("got %d want ExitFailed", code)
	}
	// On Linux/macOS, os.OpenFile under a non-directory parent fails with
	// ENOTDIR; we don't assert the exact wording, just that we surface it.
	if stderr.Len() == 0 {
		t.Error("expected stderr error message, got empty")
	}
}

// withSeededRand replaces randRead with a deterministic source that produces
// exactly the bytes whose hex encoding matches `want`. Restores the original
// at test exit. Lets us collide-on-purpose for --force tests.
func withSeededRand(t *testing.T, want string, body func()) {
	t.Helper()
	prev := randRead
	t.Cleanup(func() { randRead = prev })

	raw := make([]byte, (len(want)+1)/2)
	if _, err := decodeHex(want, raw); err != nil {
		t.Fatalf("seeding rand from %q: %v", want, err)
	}
	randRead = func(p []byte) (int, error) {
		if len(p) != len(raw) {
			t.Fatalf("stub asked for %d bytes, prepared %d", len(p), len(raw))
		}
		copy(p, raw)
		return len(p), nil
	}
	body()
}

// decodeHex parses an even-or-odd-length hex string into raw bytes.
// Odd lengths use 0 as the implicit low nibble of the final byte, which the
// generator will then truncate via slicing.
func decodeHex(s string, dst []byte) (int, error) {
	if len(s)%2 == 1 {
		s += "0"
	}
	for i := 0; i < len(s); i += 2 {
		hi, ok1 := hexNibble(s[i])
		lo, ok2 := hexNibble(s[i+1])
		if !ok1 || !ok2 {
			return 0, errors.New("not hex")
		}
		dst[i/2] = (hi << 4) | lo
	}
	return len(s) / 2, nil
}

func hexNibble(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	default:
		return 0, false
	}
}

