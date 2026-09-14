package main

import (
	"context"
	"os"

	"phasionary/internal/config"
	"phasionary/internal/data"
	"phasionary/internal/journal"
	"phasionary/internal/server"
	"phasionary/internal/syncclient"
)

func main() {
	if err := server.NewRootCmd(enrollLocal).Execute(); err != nil {
		os.Exit(1)
	}
}

func enrollLocal(ctx context.Context, serverURL string, mint func() (string, error)) (int, server.LocalState, error) {
	stateDir, err := config.ResolveStateDir()
	if err != nil {
		return 0, server.LocalSkipped, err
	}
	// An existing enrolment may name another server; never overwrite one.
	if dev, enrolled, err := journal.LoadDevice(stateDir); err != nil || enrolled {
		if err == nil && dev.ServerURL == serverURL {
			return 0, server.LocalAlreadyEnrolled, nil
		}
		return 0, server.LocalSkipped, err
	}
	dataDir, err := config.ResolveDataDir("")
	if err != nil {
		return 0, server.LocalSkipped, err
	}
	// Stat, not Ensure: a host with no TUI must not grow a data dir.
	if _, err := os.Stat(dataDir); err != nil {
		return 0, server.LocalSkipped, nil
	}
	code, err := mint()
	if err != nil {
		return 0, server.LocalSkipped, err
	}
	name, _ := os.Hostname()
	_, sum, err := syncclient.Login(ctx, stateDir, data.NewStore(dataDir), serverURL, code, name)
	if err != nil {
		return 0, server.LocalSkipped, err
	}
	return sum.Pushed, server.LocalEnrolled, nil
}
