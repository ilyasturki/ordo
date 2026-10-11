package version

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromBuildInfo_TakesTheModuleVersionOfAGoInstall(t *testing.T) {
	info := &debug.BuildInfo{Main: debug.Module{Version: "v0.10.0"}}

	version, commit := fromBuildInfo(info, "dev", "none")

	assert.Equal(t, "v0.10.0", version)
	assert.Equal(t, "none", commit)
}

func TestFromBuildInfo_TakesTheRevisionOfACheckoutBuild(t *testing.T) {
	info := &debug.BuildInfo{
		Main:     debug.Module{Version: "(devel)"},
		Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "05bc1cc8e2f1a2b3c4d5e6f708192a3b4c5d6e7f"}},
	}

	version, commit := fromBuildInfo(info, "dev", "none")

	assert.Equal(t, "dev", version)
	assert.Equal(t, "05bc1cc", commit)
}

func TestFromBuildInfo_KeepsWhatLdflagsSet(t *testing.T) {
	info := &debug.BuildInfo{
		Main:     debug.Module{Version: "v0.10.1-0.20261011102921-05bc1cc8e2f1"},
		Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "05bc1cc8e2f1a2b3c4d5e6f708192a3b4c5d6e7f"}},
	}

	version, commit := fromBuildInfo(info, "v0.10.0", "abc1234")

	assert.Equal(t, "v0.10.0", version)
	assert.Equal(t, "abc1234", commit)
}
