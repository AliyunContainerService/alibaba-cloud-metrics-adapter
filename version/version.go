package version

import "fmt"

var version, GitCommit string

func VersionInfo() string {
	return fmt.Sprintf("version: %s\ncommit: %s", version, GitCommit)
}
