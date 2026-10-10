package cli

import (
	"os"
	"testing"

	"phasionary/internal/config"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "phasionary-cli-test-")
	if err != nil {
		panic(err)
	}
	os.Setenv(config.EnvConfigPath, dir+"/config")
	os.Setenv(config.EnvDataPath, dir+"/data")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
