package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"rsc.io/qr"

	"github.com/ilyasturki/ordo/internal/attachments"
	"github.com/ilyasturki/ordo/internal/config"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/web"
)

const defaultListen = "127.0.0.1:7319"

func webAuth() (*web.Auth, string, error) {
	dataDir, err := config.ResolveDataDir(viper.GetString("data"))
	if err != nil {
		return nil, "", err
	}
	return web.NewAuth(filepath.Dir(dataDir)), dataDir, nil
}

func newServeCmd() *cobra.Command {
	var listen, baseURL string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the phone web app",
		Long: "Serve Ordo's web app, made for a phone, over the same project files the terminal app uses.\n\n" +
			"Listen on loopback (or a unix socket with unix:/path) behind a TLS reverse proxy, then\n" +
			"pair each device once with `ordo serve pair`. Devices are kept in web.json beside state.json.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			auth, dataDir, err := webAuth()
			if err != nil {
				return err
			}
			if baseURL != "" {
				if err := auth.SetBaseURL(strings.TrimRight(baseURL, "/")); err != nil {
					return err
				}
			}
			store := data.NewStore(dataDir)
			if err := store.Ensure(); err != nil {
				return err
			}
			logger := log.New(cmd.ErrOrStderr(), "ordo serve: ", log.LstdFlags)
			srv, err := web.New(store, attachments.NewStore(dataDir), auth, logger)
			if err != nil {
				return err
			}
			ln, err := listenOn(listen)
			if err != nil {
				return err
			}
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			go func() {
				if err := srv.Watch(ctx); err != nil {
					logger.Printf("live refresh off: %v", err)
				}
			}()
			httpSrv := &http.Server{
				Handler:           srv.Handler(),
				ReadHeaderTimeout: 10 * time.Second,
				IdleTimeout:       2 * time.Minute,
				BaseContext:       func(net.Listener) context.Context { return ctx },
			}
			go func() {
				<-ctx.Done()
				shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = httpSrv.Shutdown(shutdown)
			}()
			fmt.Fprintf(cmd.OutOrStdout(), "Serving %s on %s\n", dataDir, ln.Addr())
			if auth.BaseURL() == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "Set --base-url to the address your phone opens, so `ordo serve pair` prints a working link.")
			}
			if err := httpSrv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&listen, "listen", defaultListen, "address to listen on, host:port or unix:/path/to/socket")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "public address of the app, e.g. https://ordo.example.com (remembered)")
	cmd.AddCommand(newServePairCmd(), newServePassphraseCmd(), newServeDevicesCmd(), newServeRevokeCmd())
	return cmd
}

func listenOn(addr string) (net.Listener, error) {
	path, ok := strings.CutPrefix(addr, "unix:")
	if !ok {
		return net.Listen("tcp", addr)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	// The proxy runs as another user and reaches the socket through the group; nobody else may.
	old := syscall.Umask(0o117)
	ln, err := net.Listen("unix", path)
	syscall.Umask(old)
	return ln, err
}

func newServePairCmd() *cobra.Command {
	var baseURL string
	cmd := &cobra.Command{
		Use:   "pair",
		Short: "Print a one-time link and QR code that pairs a device",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			auth, _, err := webAuth()
			if err != nil {
				return err
			}
			base := strings.TrimRight(baseURL, "/")
			if base == "" {
				base = auth.BaseURL()
			}
			if base == "" {
				base = "http://localhost:" + strings.TrimPrefix(defaultListen, "127.0.0.1:")
			}
			token, err := auth.NewPairing()
			if err != nil {
				return err
			}
			link := base + "/pair/" + token
			out := cmd.OutOrStdout()
			if f, ok := out.(*os.File); ok && term.IsTerminal(f.Fd()) {
				if err := writeQR(out, link); err != nil {
					return err
				}
			}
			fmt.Fprintf(out, "%s\n\nOpen it on the device within %d minutes; it works once.\n", link, int(web.PairTTL.Minutes()))
			return nil
		},
	}
	cmd.Flags().StringVar(&baseURL, "base-url", "", "address the device opens (default: the one serve remembers)")
	return cmd
}

// The spec's quiet zone; a scanner reads nothing without it.
const quietZone = 4

func writeQR(w io.Writer, text string) error {
	code, err := qr.Encode(text, qr.M)
	if err != nil {
		return err
	}
	side := code.Size + 2*quietZone
	dark := func(x, y int) bool {
		x, y = x-quietZone, y-quietZone
		return x >= 0 && y >= 0 && x < code.Size && y < code.Size && code.Black(x, y)
	}
	var b strings.Builder
	for y := 0; y < side; y += 2 {
		for x := 0; x < side; x++ {
			// Explicit colours: a camera reads dark-on-light whatever the terminal theme.
			fg, bg := "97", "107"
			if dark(x, y) {
				fg = "30"
			}
			if dark(x, y+1) {
				bg = "40"
			}
			b.WriteString("\x1b[" + fg + ";" + bg + "m▀")
		}
		b.WriteString("\x1b[0m\n")
	}
	_, err = io.WriteString(w, b.String()+"\n")
	return err
}

func newServePassphraseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "passphrase",
		Short: "Set the passphrase that signs a device in without a pairing link",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			auth, _, err := webAuth()
			if err != nil {
				return err
			}
			phrase, err := readPassphrase(cmd)
			if err != nil {
				return err
			}
			if len([]rune(phrase)) < 12 {
				return errors.New("use at least 12 characters: the app may face the internet")
			}
			if err := auth.SetPassphrase(phrase); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Passphrase set.")
			return nil
		},
	}
}

func readPassphrase(cmd *cobra.Command) (string, error) {
	in := os.Stdin
	if !term.IsTerminal(in.Fd()) {
		line, err := bufio.NewReader(in).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		return strings.TrimRight(line, "\r\n"), nil
	}
	fmt.Fprint(cmd.ErrOrStderr(), "New passphrase: ")
	first, err := term.ReadPassword(in.Fd())
	fmt.Fprintln(cmd.ErrOrStderr())
	if err != nil {
		return "", err
	}
	fmt.Fprint(cmd.ErrOrStderr(), "Repeat it: ")
	second, err := term.ReadPassword(in.Fd())
	fmt.Fprintln(cmd.ErrOrStderr())
	if err != nil {
		return "", err
	}
	if string(first) != string(second) {
		return "", errors.New("the two passphrases differ")
	}
	return string(first), nil
}

func newServeDevicesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "devices",
		Short: "List the paired devices",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			auth, _, err := webAuth()
			if err != nil {
				return err
			}
			devs, err := auth.Devices()
			if err != nil {
				return err
			}
			if len(devs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No paired devices. Pair one with `ordo serve pair`.")
				return nil
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tNAME\tPAIRED\tLAST USED")
			for _, d := range devs {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", d.ID, d.Name, d.CreatedAt.Local().Format("2006-01-02"), d.LastSeen.Local().Format("2006-01-02"))
			}
			return tw.Flush()
		},
	}
}

func newServeRevokeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revoke <device>",
		Short: "Sign a paired device out",
		Args:  exactArgs("device"),
		RunE: func(cmd *cobra.Command, args []string) error {
			auth, _, err := webAuth()
			if err != nil {
				return err
			}
			found, err := auth.Revoke(args[0])
			if err != nil {
				return err
			}
			if !found {
				return fmt.Errorf("no device %q: see `ordo serve devices`", args[0])
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Revoked.")
			return nil
		},
	}
}
