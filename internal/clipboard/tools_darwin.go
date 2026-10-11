package clipboard

import "os"

var (
	copyTool  = []string{"pbcopy"}
	pasteTool = []string{"pbpaste"}
)

func toolEnv() ([]string, bool) {
	return os.Environ(), true
}
