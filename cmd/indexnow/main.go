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
	f.StringVar(&opts.Endpoint, "endpoint", "api", "endpoint: api|bing|yandex|naver|seznam|yep or full URL (env: INDEXNOW_ENDPOINT)")
	f.StringVar(&opts.File, "file", "", "read URLs from file (one per line; # comments allowed)")
	f.BoolVar(&opts.Stdin, "stdin", false, "read URLs from stdin")
	f.BoolVar(&opts.DryRun, "dry-run", false, "print what would be sent and exit")
	f.StringVar(&opts.Output, "output", cli.OutputText, "output format: text|json")
	f.StringVar(&opts.FailOn, "fail-on", cli.FailOnAny, "exit non-zero on: any|4xx|5xx|never")
	f.IntVar(&opts.MaxRetries, "max-retries", 3, "max retries on 429/5xx/transport errors")
	f.DurationVar(&opts.BaseBackoff, "base-backoff", time.Second, "base retry backoff")
	f.DurationVar(&opts.MaxBackoff, "max-backoff", 30*time.Second, "max retry backoff")

	return cmd
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
}
