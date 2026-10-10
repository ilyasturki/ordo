package app

import (
	"os"
	"testing"

	"phasionary/internal/clipboard"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "phasionary-app-test-")
	if err != nil {
		panic(err)
	}
	os.Setenv(clipboard.EnvFile, dir+"/clipboard")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
