package version

// Commit is set via ldflags at build time:
//
//	go build -ldflags="-X github.com/myorg/myapp/internal/version.Commit=abc1234..."
var Commit = "dev"

// CommitShort returns the first 7 characters of the commit hash,
// or the full value if shorter (e.g. "dev").
func CommitShort() string {
	if len(Commit) > 7 {
		return Commit[:7]
	}
	return Commit
}
