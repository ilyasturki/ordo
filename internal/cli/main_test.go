package cli

import (
	"os"
	"testing"

	"github.com/ilyasturki/ordo/internal/config"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "ordo-cli-test-")
	if err != nil {
		panic(err)
	}
	os.Setenv(config.EnvConfigPath, dir+"/config")
	os.Setenv(config.EnvDataPath, dir+"/data")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
