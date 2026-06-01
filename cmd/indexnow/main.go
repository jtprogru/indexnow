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

Sources (exactly one): positional args, --file, or --stdin.

Examples:
  indexnow submit https://example.com/post/1
  indexnow submit --file urls.txt --endpoint bing
  cat urls.txt | indexnow submit --stdin --output json`,
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

// applyConfig fills any SubmitOptions fields still at their zero/default
// value from the yaml config file. Precedence: flag > env > config > default,
// so this runs after applyEnvDefaults. An explicit --config that points to
// a missing file is a usage error; the default XDG path is silently skipped
// when absent.
func applyConfig(opts *cli.SubmitOptions, explicitPath string) error {
	path := explicitPath
	explicit := path != ""
	if !explicit {
		path = config.DefaultPath()
		if path == "" {
			return nil
		}
	}
	cfg, err := config.Load(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !explicit {
			return nil
		}
		return fmt.Errorf("config: %w", err)
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
