// Package version is a convenience utility that provides SDK
// consumers with a ready-to-use version command that
// produces app versioning information based on flags
// passed at compile time.
//
// # Configure the version command
//
// The version command can be just added to your cobra root command.
// At build time, the variables Name, Version, Commit, and BuildTags
// can be passed as build flags as shown in the following example:
//
//	go build -X github.com/cosmos/cosmos-sdk/version.Name=heimdall \
//	 -X github.com/cosmos/cosmos-sdk/version.ServerName=heimdalld \
//	 -X github.com/cosmos/cosmos-sdk/version.Version=1.0 \
//	 -X github.com/cosmos/cosmos-sdk/version.Commit=f0f7b7dab7e36c20b757cebce0e8f4fc5b95de60 \
//	 -X "github.com/cosmos/cosmos-sdk/version.BuildTags=linux darwin amd64"
package version

import (
	"fmt"
	"runtime"
)

var (
	// Name is the application's name
	Name = ""
	// ServerName is the server binary name
	ServerName = "heimdalld"
	// ClientName is the client binary name
	ClientName = "heimdalld"
	// Version is the app's version string
	Version = ""
	// Commit is the app's commit hash
	Commit = ""
)

// Info defines the application version information.
type Info struct {
	Name       string `json:"name" yaml:"name"`
	ServerName string `json:"server_name" yaml:"server_name"`
	ClientName string `json:"client_name" yaml:"client_name"`
	Version    string `json:"version" yaml:"version"`
	GitCommit  string `json:"commit" yaml:"commit"`
	GoVersion  string `json:"go" yaml:"go"`
}

func NewInfo() Info {
	return Info{
		Name:       Name,
		ServerName: ServerName,
		ClientName: ClientName,
		Version:    Version,
		GitCommit:  Commit,
		GoVersion:  fmt.Sprintf("go version %s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH),
	}
}

func (vi Info) String() string {
	return fmt.Sprintf(`%s: %s
git commit: %s
%s`,
		vi.Name, vi.Version, vi.GitCommit, vi.GoVersion,
	)
}
