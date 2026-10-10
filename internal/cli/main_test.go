package cli

import (
	"os"
	"testing"

	"phasionary/internal/config"
)

// Every write journals into the state dir, so without this the tests append
// their throwaway ops to the real journal, which sync then pushes.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "phasionary-cli-test-")
	if err != nil {
		panic(err)
	}
	os.Setenv(config.EnvStatePath, dir+"/state")
	os.Setenv(config.EnvConfigPath, dir+"/config")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
