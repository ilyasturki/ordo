package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"phasionary/internal/config"
	"phasionary/internal/version"
)

const (
	EnvHost         = "PHASIONARY_SERVER_HOST"
	EnvPort         = "PHASIONARY_SERVER_PORT"
	EnvAllowedHosts = "PHASIONARY_SERVER_ALLOWED_HOSTS"

	defaultHost = "0.0.0.0"
	defaultPort = "7777"
)

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

func envHosts(name string) []string {
	return strings.Fields(strings.ReplaceAll(os.Getenv(name), ",", " "))
}

type LocalState int

const (
	LocalSkipped LocalState = iota
	LocalEnrolled
	LocalAlreadyEnrolled
)

// Injected: syncclient's tests run this package, so the import points one way.
type LocalEnroll func(ctx context.Context, serverURL string, mint func() (string, error)) (pushed int, state LocalState, err error)

func NewRootCmd(local LocalEnroll) *cobra.Command {
	var (
		dataDir       string
		host          string
		port          string
		allowedHosts  []string
		noLocalEnroll bool
	)
	cmd := &cobra.Command{
		Use:   "phasionary-server",
		Short: "Self-hosted sync server for phasionary",
		Long: `Self-hosted sync server for phasionary.

The server exists to put the web app and other devices in front of your
projects; a machine that only ever runs the TUI has no use for it.

This machine's own phasionary is enrolled on startup, with no code. Every
other device joins once with a single-use code (see "phasionary-server pair"),
then exchanges its change journal with the server over HTTP. The server merges
every device's changes and keeps the result in a SQLite database.

It binds to 0.0.0.0:7777 so a phone on the same network can reach it. Keep
that network private (a LAN, Tailscale, WireGuard) or put the server behind a
TLS reverse proxy rather than exposing the plain-HTTP port to the internet:
the per-device bearer tokens travel in every request.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := config.ResolveServerDataDir(dataDir)
			if err != nil {
				return err
			}
			db, err := OpenDB(dir)
			if err != nil {
				return err
			}
			defer db.Close()
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			onReady := func(bound string) {
				if !noLocalEnroll && local != nil {
					// The bound address may be a wildcard, which cannot be dialed.
					_, boundPort, _ := net.SplitHostPort(bound)
					mint := func() (string, error) { return db.NewEnrollCode(ctx) }
					pushed, state, err := local(ctx, "http://127.0.0.1:"+boundPort, mint)
					switch {
					case err != nil:
						log.Printf("this machine was not enrolled automatically: %v", err)
					case state == LocalEnrolled:
						fmt.Printf("\nEnrolled this machine's phasionary and uploaded %d change(s).\n", pushed)
					}
					if state != LocalSkipped {
						fmt.Println("This machine's phasionary now syncs with this server at launch and at quit;")
						fmt.Println("while the server is down its TUI shows [offline]. Undo with `phasionary sync logout`.")
					}
				}
				// A code in a log file would outlive its ten minutes.
				if term.IsTerminal(os.Stdout.Fd()) {
					fmt.Println()
					if err := printPair(ctx, os.Stdout, db, host, port); err != nil {
						log.Printf("could not print a pairing code: %v", err)
					}
				}
			}
			return New(db, net.JoinHostPort(host, port), allowedHosts...).Run(ctx, onReady)
		},
	}
	cmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version.Version, version.Commit, version.BuildDate)

	cmd.PersistentFlags().StringVar(&dataDir, "data", "", "database directory (default: $"+config.EnvServerDataPath+" or ~/.local/share/phasionary-server)")
	cmd.Flags().StringVar(&host, "host", envOr(EnvHost, defaultHost), "listen host/IP ($"+EnvHost+")")
	cmd.Flags().StringVar(&port, "port", envOr(EnvPort, defaultPort), "listen port ($"+EnvPort+")")
	cmd.Flags().StringArrayVar(&allowedHosts, "allowed-host", envHosts(EnvAllowedHosts),
		"hostname browsers may reach the app by; repeatable, IPs and localhost always pass ($"+EnvAllowedHosts+", comma-separated)")
	cmd.Flags().BoolVar(&noLocalEnroll, "no-local-enroll", false,
		"do not enroll this machine's own phasionary on startup (for a server with no local TUI)")

	cmd.AddCommand(newPairCmd(&dataDir, &host, &port))
	return cmd
}

func newPairCmd(dataDir, host, port *string) *cobra.Command {
	return &cobra.Command{
		Use:     "pair",
		Aliases: []string{"enroll"},
		Short:   "Print a QR and a single-use code that let one device join",
		Long: `Print a single-use enrollment code, and a QR code carrying it.

The code is valid for ten minutes and admits exactly one device. It is stored
in the server's database, so run this against the same --data directory (and
as the same user) as the running server.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := config.ResolveServerDataDir(*dataDir)
			if err != nil {
				return err
			}
			db, err := OpenDB(dir)
			if err != nil {
				return err
			}
			defer db.Close()
			return printPair(cmd.Context(), cmd.OutOrStdout(), db, *host, *port)
		},
	}
}

func printPair(ctx context.Context, out io.Writer, db *DB, host, port string) error {
	code, err := db.NewEnrollCode(ctx)
	if err != nil {
		return err
	}
	hosts := advertisedHosts(host)
	url := "http://" + net.JoinHostPort(hosts[0], port)
	fmt.Fprintf(out, "Enrollment code: %s (single use, valid %s)\n\n", code, CodeTTL)
	// In the fragment: no proxy or server log ever sees it.
	if err := writeQR(out, url+"/#code="+code); err == nil {
		fmt.Fprintln(out)
	}
	fmt.Fprintf(out, "Scan it with the phone, or open %s and type the code.\n", url)
	if len(hosts) > 1 {
		fmt.Fprintf(out, "This server also answers on %s.\n", strings.Join(hosts[1:], ", "))
	}
	fmt.Fprintf(out, "\nOn another machine running the TUI:\n\n    phasionary sync login %s\n", url)
	return nil
}
