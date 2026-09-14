package app

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"phasionary/internal/config"
	"phasionary/internal/data"
	"phasionary/internal/journal"
)

func enrollAt(t *testing.T, serverURL string) *data.Store {
	t.Helper()
	stateDir := t.TempDir()
	t.Setenv(config.EnvStatePath, stateDir)
	require.NoError(t, journal.SaveDevice(stateDir, journal.Device{DeviceID: "dev", ServerURL: serverURL, Token: "tok"}))
	store := data.NewStore(t.TempDir())
	require.NoError(t, store.Ensure())
	return store
}

func TestSyncAtLaunch_Unenrolled(t *testing.T) {
	t.Setenv(config.EnvStatePath, t.TempDir())
	store := data.NewStore(t.TempDir())
	require.NoError(t, store.Ensure())
	assert.Equal(t, launchSync{}, syncAtLaunch(store))
}

func TestSyncAtLaunch_Unreachable(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())

	store := enrollAt(t, "http://"+addr)
	got := syncAtLaunch(store)
	assert.True(t, got.Offline)
	assert.Equal(t, "Sync server unreachable; changes queue locally", got.Toast)
}

func TestSyncAtLaunch_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"bad token"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	store := enrollAt(t, srv.URL)
	got := syncAtLaunch(store)
	assert.False(t, got.Offline)
	assert.Equal(t, "Sync failed: server answered 401: bad token", got.Toast)
}

func TestRenderProjectLine_OfflineTag(t *testing.T) {
	line := renderProjectLine("Default", true, true, false, false, true, "", 80)
	assert.Contains(t, line, "[offline]")
	line = renderProjectLine("Default", true, true, false, false, false, "", 80)
	assert.NotContains(t, line, "[offline]")
}
