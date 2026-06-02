package cli

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Bounds dictated by the IndexNow protocol: 8..128 chars from [A-Za-z0-9-].
// We generate hex, which is a subset.
const (
	keygenMinLength     = 8
	keygenMaxLength     = 128
	keygenDefaultLength = 32
	keygenFileMode      = 0o644
)

var (
	ErrKeygenLength = errors.New("indexnow cli: --length must be 8..128")

	// randRead is the source of entropy. It's package-private and
	// replaceable in tests; users get crypto/rand only.
	randRead = rand.Read //nolint:gochecknoglobals // package-private test seam
)

type KeygenOptions struct {
	Length int    // 8..128; 0 means default (32).
	Write  string // empty = print only; otherwise directory to write <key>.txt into
	Force  bool   // overwrite existing key file
	Output string // OutputText | OutputJSON; empty defaults to text
	Quiet  bool
}

type keygenJSON struct {
	Key  string `json:"key"`
	Path string `json:"path,omitempty"`
}

// RunKeygen generates an IndexNow key and optionally writes the hosted key
// file at <Write>/<key>.txt. stdout receives the key (or JSON) unless Quiet;
// stderr receives a `wrote <path>` notice on --write. Returns ExitOK, ExitFailed
// (I/O / entropy error), or ExitUsageError (invalid flags).
func RunKeygen(opts KeygenOptions, stdout, stderr io.Writer) int {
	if err := validateKeygen(&opts); err != nil {
		fmt.Fprintln(stderr, err)
		return ExitUsageError
	}

	key, err := generateHexKey(opts.Length)
	if err != nil {
		fmt.Fprintln(stderr, "indexnow cli: generating key:", err)
		return ExitFailed
	}

	var path string
	if opts.Write != "" {
		path = filepath.Join(opts.Write, key+".txt")
		if err := writeKeyFile(path, key, opts.Force); err != nil {
			fmt.Fprintln(stderr, "indexnow cli:", err)
			return ExitFailed
		}
		fmt.Fprintf(stderr, "wrote %s\n", path)
	}

	if opts.Quiet {
		return ExitOK
	}

	switch opts.Output {
	case OutputJSON:
		if err := json.NewEncoder(stdout).Encode(keygenJSON{Key: key, Path: path}); err != nil {
			fmt.Fprintln(stderr, "indexnow cli: encoding json:", err)
			return ExitFailed
		}
	default:
		fmt.Fprintln(stdout, key)
	}
	return ExitOK
}

func validateKeygen(opts *KeygenOptions) error {
	if opts.Length == 0 {
		opts.Length = keygenDefaultLength
	}
	if opts.Length < keygenMinLength || opts.Length > keygenMaxLength {
		return fmt.Errorf("%w: got %d", ErrKeygenLength, opts.Length)
	}
	if err := validateOutput(opts.Output); err != nil {
		return err
	}
	return nil
}

// generateHexKey returns a key of exactly n hex characters, sourced from
// randRead. We request ceil(n/2) bytes and slice the hex string to n, so odd
// lengths are honored without padding artifacts.
func generateHexKey(n int) (string, error) {
	buf := make([]byte, (n+1)/2)
	if _, err := randRead(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf)[:n], nil
}

func writeKeyFile(path, key string, force bool) error {
	flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if !force {
		flag = os.O_WRONLY | os.O_CREATE | os.O_EXCL
	}
	f, err := os.OpenFile(path, flag, keygenFileMode)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%s already exists; use --force to overwrite", path)
		}
		return fmt.Errorf("writing %s: %w", path, err)
	}
	defer f.Close()
	if _, err := f.WriteString(key + "\n"); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
