package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rsc.io/qr"

	"github.com/ilyasturki/ordo/internal/web"
)

func TestServePair_PrintsTheLinkOnTheGivenAddress(t *testing.T) {
	out, err := runCLI(t, t.TempDir(), "serve", "pair", "--base-url", "https://ordo.example.com/")

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(out, "https://ordo.example.com/pair/"), out)
	assert.Contains(t, out, "in the Ordo app")
	assert.NotContains(t, out, "localhost")
}

func TestServePair_UsesTheAddressServeRemembers(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, web.NewAuth(dir).SetBaseURL("https://ordo.example.com"))

	out, err := runCLI(t, dir, "serve", "pair")

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(out, "https://ordo.example.com/pair/"), out)
}

func TestServePair_WarnsWhenItCanOnlyNameLocalhost(t *testing.T) {
	out, err := runCLI(t, t.TempDir(), "serve", "pair")

	require.NoError(t, err)
	assert.Contains(t, out, "http://localhost:7319/pair/")
	assert.Contains(t, out, "--base-url")
}

func TestServePair_HelpSaysToScanOrPasteInTheApp(t *testing.T) {
	out, err := runCLI(t, t.TempDir(), "serve", "pair", "--help")

	require.NoError(t, err)
	assert.Contains(t, out, "Scan the code or paste the link")
	assert.Contains(t, out, "--base-url")
}

func TestWriteQR_DrawsEveryModuleWithItsQuietZone(t *testing.T) {
	const link = "https://ordo.example.com/pair/ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	code, err := qr.Encode(link, qr.M)
	require.NoError(t, err)
	var buf bytes.Buffer

	require.NoError(t, writeQR(&buf, link))

	side := code.Size + 2*quietZone
	rows := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	assert.Len(t, rows, (side+1)/2)
	for _, row := range rows {
		assert.Equal(t, side, strings.Count(row, "▀"))
	}
}
