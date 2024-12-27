package version

import "fmt"

var (
	Version   = "v0.0.0-dev"
	GitCommit = "HEAD"
)

// FriendlyVersion outputs a version that will be displayed on running --version on the binary
func FriendlyVersion() string {
	return fmt.Sprintf("%s (%s)", Version, GitCommit)
}
