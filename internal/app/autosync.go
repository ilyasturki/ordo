package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"phasionary/internal/config"
	"phasionary/internal/data"
	"phasionary/internal/journal"
	"phasionary/internal/syncclient"
)

const autoSyncTimeout = 3 * time.Second

type launchSync struct {
	// Toast is the transient status message; cleared by the first keypress.
	Toast string
	// Offline stays set for the session: the server never answered.
	Offline bool
}

func syncAtLaunch(store *data.Store) launchSync {
	err := autoSyncRound(store)
	switch {
	case err == nil:
		return launchSync{}
	case unreachable(err):
		return launchSync{Toast: "Sync server unreachable; changes queue locally", Offline: true}
	default:
		return launchSync{Toast: "Sync failed: " + err.Error()}
	}
}

func syncAtExit(store *data.Store) {
	err := autoSyncRound(store)
	switch {
	case err == nil:
	case unreachable(err):
		fmt.Fprintln(os.Stderr, "Sync server unreachable; changes queued locally.\nRun `phasionary sync now` when it is back.")
	default:
		fmt.Fprintf(os.Stderr, "Sync failed: %v\nRun `phasionary sync now` when the server is back.\n", err)
	}
}

// A *url.Error means the round trip itself failed (dial, timeout, TLS); a
// server that answered yields *syncclient.HTTPError instead.
func unreachable(err error) bool {
	var ue *url.Error
	return errors.As(err, &ue)
}

func autoSyncRound(store *data.Store) error {
	stateDir, err := config.ResolveStateDir()
	if err != nil {
		return nil
	}
	if d, ok, err := journal.LoadDevice(stateDir); err != nil || !ok || d.Token == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), autoSyncTimeout)
	defer cancel()
	_, err = syncclient.Run(ctx, stateDir, store)
	return err
}
