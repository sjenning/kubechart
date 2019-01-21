package version

import "fmt"

var (
	// Version is the current version of kubechart, set by the go linker's -X flag at build time.
	Version string

	// GitSHA is the actual commit that is being built, set by the go linker's -X flag at build time.
	GitSHA string

	// GitTreeState indicates if the git tree is clean or dirty, set by the go linker's -X flag at build
	// time.
	GitTreeState string
)

// FormattedGitSHA renders the Git SHA with an indicator of the tree state.
func FormattedGitSHA() string {
	if GitTreeState != "clean" {
		return fmt.Sprintf("%s-%s", GitSHA, GitTreeState)
	}
	return GitSHA
}
