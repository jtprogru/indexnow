// Command indexnow is the CLI for the IndexNow protocol.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/jtprogru/indexnow/internal/cli"
	"github.com/jtprogru/indexnow/internal/config"
)

// Build-time variables, injected via -ldflags by goreleaser.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
	BuiltBy = "source"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return run(ctx, os.Args[1:])
}

func run(ctx context.Context, args []string) int {
	root := &cobra.Command{
		Use:   "indexnow",
		Short: "IndexNow protocol client for content pipelines",
		Long: `indexnow notifies participating search engines (Bing, Yandex, Naver,
Seznam, Yep, ...) about URL changes via the IndexNow protocol.

A submission to one endpoint is shared with every other participating
search engine — you only need to call one.`,
		Version:       fmt.Sprintf("%s (commit %s, built %s by %s)", Version, Commit, Date, BuiltBy),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetVersionTemplate("indexnow {{.Version}}\n")

	root.AddCommand(newSubmitCmd(ctx))
	root.AddCommand(newKeyCmd(ctx))
	root.AddCommand(newVerifyCmd(ctx))
	root.SetArgs(args)

	if err := root.Execute(); err != nil {
		var ec exitCodeError
		if errors.As(err, &ec) {
			return int(ec)
		}
		return cli.ExitUsageError
	}
	return cli.ExitOK
}

// exitCodeError carries a process exit code through cobra's error return.
type exitCodeError int

func (e exitCodeError) Error() string { return "" }

func newSubmitCmd(ctx context.Context) *cobra.Command {
	opts := cli.SubmitOptions{}
	var configPath string
	cmd := &cobra.Command{
		Use:   "submit [urls...]",
		Short: "Submit URLs to an IndexNow endpoint",
		Long: `submit sends one or more URLs to an IndexNow endpoint.

Sources (exactly one): positional args, --file, --stdin, or --sitemap.

Examples:
  indexnow submit https://example.com/post/1
  indexnow submit --file urls.txt --endpoint bing
  cat urls.txt | indexnow submit --stdin --output json
  indexnow submit --sitemap https://example.com/sitemap.xml
  indexnow submit --sitemap sitemap.xml.gz --sitemap-since 2026-05-01T00:00:00Z`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Args = args
			applyEnvDefaults(&opts)
			if err := applyConfig(&opts, configPath); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return exitCodeError(cli.ExitUsageError)
			}
			applyDefaults(&opts)
			code := cli.RunSubmit(ctx, opts, os.Stdin, cmd.OutOrStdout(), cmd.ErrOrStderr(), nil)
			if code != cli.ExitOK {
				return exitCodeError(code)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.Key, "key", "", "IndexNow key (env: INDEXNOW_KEY)")
	f.StringVar(&opts.Host, "host", "", "site host, e.g. example.com (env: INDEXNOW_HOST; inferred from first URL if empty)")
	f.StringVar(&opts.KeyLocation, "key-location", "", "absolute URL to the hosted key file (env: INDEXNOW_KEY_LOCATION)")
	f.StringVar(&opts.Endpoint, "endpoint", "api", "endpoint(s): comma-separated aliases (api|bing|yandex|naver|seznam|yep) or full URLs; multiple endpoints are submitted in parallel (env: INDEXNOW_ENDPOINT)")
	f.StringVar(&opts.File, "file", "", "read URLs from file (one per line; # comments allowed)")
	f.BoolVar(&opts.Stdin, "stdin", false, "read URLs from stdin")
	f.StringVar(&opts.Sitemap, "sitemap", "", "fetch URLs from a sitemap (URL or local path; sitemapindex is followed; .gz is gunzipped)")
	f.StringVar(&opts.SitemapSince, "sitemap-since", "", "filter sitemap entries by <lastmod> (RFC3339, e.g. 2026-05-01T00:00:00Z); entries without lastmod always pass")
	f.DurationVar(&opts.SitemapTimeout, "sitemap-timeout", 30*time.Second, "per-request HTTP timeout for sitemap fetches")
	f.BoolVar(&opts.DryRun, "dry-run", false, "print what would be sent and exit")
	f.BoolVarP(&opts.Quiet, "quiet", "q", false, "suppress stdout; rely on exit code (errors still go to stderr)")
	f.BoolVarP(&opts.Verbose, "verbose", "v", false, "log submit lifecycle and retry events to stderr (slog text format)")
	f.StringVar(&opts.Output, "output", cli.OutputText, "output format: text|json")
	f.StringVar(&opts.FailOn, "fail-on", cli.FailOnAny, "exit non-zero on: any|4xx|5xx|never")
	f.IntVar(&opts.MaxRetries, "max-retries", 3, "max retries on 429/5xx/transport errors")
	f.DurationVar(&opts.BaseBackoff, "base-backoff", time.Second, "base retry backoff")
	f.DurationVar(&opts.MaxBackoff, "max-backoff", 30*time.Second, "max retry backoff")
	f.StringVar(&configPath, "config", "", "path to yaml config (default: $XDG_CONFIG_HOME/indexnow/config.yaml)")
	f.StringVar(&opts.UserAgent, "user-agent", "", "HTTP User-Agent header (env: INDEXNOW_USER_AGENT; default: indexnow/<version>)")

	return cmd
}

// loadConfig resolves the yaml config path (explicit or XDG default) and
// reads it. An explicit path that points to a missing file is an error
// (--config typo'd); the default path silently returns a zero Config when
// absent, so users without a config file see no surprise and callers can
// merge unconditionally.
func loadConfig(explicitPath string) (*config.Config, error) {
	path := explicitPath
	explicit := path != ""
	if !explicit {
		path = config.DefaultPath()
		if path == "" {
			return &config.Config{}, nil
		}
	}
	cfg, err := config.Load(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !explicit {
			return &config.Config{}, nil
		}
		return nil, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}

// applyConfig fills any SubmitOptions fields still at their zero/default
// value from the yaml config file. Precedence: flag > env > config > default,
// so this runs after applyEnvDefaults.
func applyConfig(opts *cli.SubmitOptions, explicitPath string) error {
	cfg, err := loadConfig(explicitPath)
	if err != nil {
		return err
	}
	if opts.Key == "" {
		opts.Key = cfg.Key
	}
	if opts.Host == "" {
		opts.Host = cfg.Host
	}
	if opts.KeyLocation == "" {
		opts.KeyLocation = cfg.KeyLocation
	}
	if opts.Endpoint == "api" && cfg.Endpoint != "" {
		opts.Endpoint = cfg.Endpoint
	}
	if opts.UserAgent == "" {
		opts.UserAgent = cfg.UserAgent
	}
	return nil
}

func applyEnvDefaults(opts *cli.SubmitOptions) {
	if opts.Key == "" {
		opts.Key = os.Getenv("INDEXNOW_KEY")
	}
	if opts.Host == "" {
		opts.Host = os.Getenv("INDEXNOW_HOST")
	}
	if opts.KeyLocation == "" {
		opts.KeyLocation = os.Getenv("INDEXNOW_KEY_LOCATION")
	}
	if v := os.Getenv("INDEXNOW_ENDPOINT"); v != "" && opts.Endpoint == "api" {
		opts.Endpoint = v
	}
	if opts.UserAgent == "" {
		opts.UserAgent = os.Getenv("INDEXNOW_USER_AGENT")
	}
}

// applyDefaults plugs in built-in defaults for fields that no source
// (flag, env, config) filled. Currently only UserAgent benefits — the
// stdlib's "Go-http-client/1.1" is unhelpful for proxied setups and WAFs.
func applyDefaults(opts *cli.SubmitOptions) {
	if opts.UserAgent == "" {
		opts.UserAgent = "indexnow/" + Version
	}
}

// buildVerifyCmd wires the verify operation onto an empty cobra.Command, so it
// can be registered both as the top-level `indexnow verify` (backwards-compat
// alias) and as `indexnow key verify` (canonical). Behavior is identical; only
// the help text differs at the call sites.
func buildVerifyCmd(ctx context.Context, cmd *cobra.Command) *cobra.Command {
	opts := cli.VerifyOptions{}
	var configPath string

	cmd.Args = cobra.NoArgs
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		applyVerifyEnv(&opts)
		if err := applyVerifyConfig(&opts, configPath); err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), err)
			return exitCodeError(cli.ExitUsageError)
		}
		applyVerifyDefaults(&opts)
		code := cli.RunVerify(ctx, opts, cmd.OutOrStdout(), cmd.ErrOrStderr(), nil)
		if code != cli.ExitOK {
			return exitCodeError(code)
		}
		return nil
	}

	f := cmd.Flags()
	f.StringVar(&opts.Key, "key", "", "IndexNow key (env: INDEXNOW_KEY)")
	f.StringVar(&opts.Host, "host", "", "site host, e.g. example.com (env: INDEXNOW_HOST)")
	f.StringVar(&opts.KeyLocation, "key-location", "", "absolute URL to the hosted key file (env: INDEXNOW_KEY_LOCATION; default: https://<host>/<key>.txt)")
	f.StringVar(&opts.UserAgent, "user-agent", "", "HTTP User-Agent header (env: INDEXNOW_USER_AGENT; default: indexnow/<version>)")
	f.StringVar(&opts.Output, "output", cli.OutputText, "output format: text|json")
	f.BoolVarP(&opts.Quiet, "quiet", "q", false, "suppress stdout; rely on exit code")
	f.BoolVarP(&opts.Verbose, "verbose", "v", false, "log lifecycle events to stderr (slog text format)")
	f.DurationVar(&opts.Timeout, "timeout", 10*time.Second, "HTTP timeout for the key fetch")
	f.StringVar(&configPath, "config", "", "path to yaml config (default: $XDG_CONFIG_HOME/indexnow/config.yaml)")
	return cmd
}

// newVerifyCmd is the top-level `indexnow verify` — kept as a backwards-compat
// alias for `indexnow key verify`.
func newVerifyCmd(ctx context.Context) *cobra.Command {
	return buildVerifyCmd(ctx, &cobra.Command{
		Use:   "verify",
		Short: "Alias for `indexnow key verify` (kept for backwards compatibility)",
		Long: `verify is the legacy top-level form of "indexnow key verify".

It is kept for backwards compatibility with scripts written against
indexnow v0.3.0..v0.6.x. Behavior is identical to "indexnow key verify".`,
	})
}

// newKeyCmd is the parent of "indexnow key {gen,verify}".
func newKeyCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "Manage IndexNow keys (generate, verify)",
		Long: `key groups operations that produce or check IndexNow keys.

Use "indexnow key gen" to produce a new key (optionally writing the
hosted key file), and "indexnow key verify" to check that the hosted
file matches an expected key.`,
	}
	cmd.AddCommand(newKeyGenCmd())
	cmd.AddCommand(newKeyVerifyCmd(ctx))
	return cmd
}

func newKeyGenCmd() *cobra.Command {
	opts := cli.KeygenOptions{}
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate a new IndexNow key (optionally writing the hosted file)",
		Long: `gen produces a random hex-encoded IndexNow key.

With --write <dir>, the hosted key file <dir>/<key>.txt is created with
mode 0644 and content "<key>\n". The default key length is 32 hex chars
(128 bits of entropy); --length accepts values in 8..128.

The key is printed to stdout (one line) unless --quiet. Status notices
("wrote …") go to stderr.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			code := cli.RunKeygen(opts, cmd.OutOrStdout(), cmd.ErrOrStderr())
			if code != cli.ExitOK {
				return exitCodeError(code)
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.IntVar(&opts.Length, "length", 0, "key length in hex chars (8..128, default 32)")
	f.StringVar(&opts.Write, "write", "", "directory to write hosted key file <key>.txt into")
	f.BoolVar(&opts.Force, "force", false, "overwrite existing key file")
	f.StringVar(&opts.Output, "output", cli.OutputText, "output format: text|json")
	f.BoolVarP(&opts.Quiet, "quiet", "q", false, "suppress stdout; rely on exit code")
	return cmd
}

func newKeyVerifyCmd(ctx context.Context) *cobra.Command {
	return buildVerifyCmd(ctx, &cobra.Command{
		Use:   "verify",
		Short: "Verify that the hosted IndexNow key file matches the expected key",
		Long: `verify performs the same check participating endpoints do:
fetch the hosted key file via HTTP GET and confirm its trimmed body equals
the expected key.

If --key-location is set, that URL is fetched. Otherwise the conventional
location https://<host>/<key>.txt is used (requires --host and a valid --key).

Exit 0 if the hosted key matches; non-zero otherwise.`,
	})
}

func applyVerifyEnv(opts *cli.VerifyOptions) {
	if opts.Key == "" {
		opts.Key = os.Getenv("INDEXNOW_KEY")
	}
	if opts.Host == "" {
		opts.Host = os.Getenv("INDEXNOW_HOST")
	}
	if opts.KeyLocation == "" {
		opts.KeyLocation = os.Getenv("INDEXNOW_KEY_LOCATION")
	}
	if opts.UserAgent == "" {
		opts.UserAgent = os.Getenv("INDEXNOW_USER_AGENT")
	}
}

func applyVerifyConfig(opts *cli.VerifyOptions, explicitPath string) error {
	cfg, err := loadConfig(explicitPath)
	if err != nil {
		return err
	}
	if opts.Key == "" {
		opts.Key = cfg.Key
	}
	if opts.Host == "" {
		opts.Host = cfg.Host
	}
	if opts.KeyLocation == "" {
		opts.KeyLocation = cfg.KeyLocation
	}
	if opts.UserAgent == "" {
		opts.UserAgent = cfg.UserAgent
	}
	return nil
}

func applyVerifyDefaults(opts *cli.VerifyOptions) {
	if opts.UserAgent == "" {
		opts.UserAgent = "indexnow/" + Version
	}
}
