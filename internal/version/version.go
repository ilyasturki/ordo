package version

import "runtime/debug"

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		Version, Commit = fromBuildInfo(info, Version, Commit)
	}
}

// fromBuildInfo fills what -ldflags left unset: `go install …@vX.Y.Z` stamps
// the module version, and a build inside a checkout the VCS revision.
func fromBuildInfo(info *debug.BuildInfo, version, commit string) (string, string) {
	if version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}
	if commit == "none" {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && len(s.Value) >= 7 {
				commit = s.Value[:7]
			}
		}
	}
	return version, commit
}
