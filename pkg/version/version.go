package version

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	sdkVersion "github.com/operator-framework/operator-sdk/version"
)

var (
	// Release returns the release version
	Release = "UNKNOWN"
	// Commit returns the short sha from git
	Commit = "UNKNOWN"
	// BuildDate is the build date
	BuildDate = ""
)

// Version ...
type Version struct {
	GitCommit  string
	BuildDate  string
	Release    string
	GoVersion  string
	Compiler   string
	Platform   string
	SdkVersion string
}

// String ...
func (v Version) String() string {
	return fmt.Sprintf(
		"%s/%s (%s/%s) Date/%s GitCommit/%s SDKVersion/%s",
		filepath.Base(os.Args[0]),
		v.Release,
		runtime.GOOS,
		runtime.GOARCH,
		v.BuildDate,
		v.GitCommit,
		v.SdkVersion,
	)
}

// GetVersion returns version
func GetVersion() Version {
	return Version{
		GitCommit:  Commit,
		BuildDate:  BuildDate,
		Release:    Release,
		GoVersion:  runtime.Version(),
		Compiler:   runtime.Compiler,
		Platform:   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		SdkVersion: sdkVersion.Version,
	}
}
