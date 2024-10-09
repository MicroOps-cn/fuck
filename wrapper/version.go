package w

import (
	"fmt"
	"time"
)

var (
	GitCommit string
	BuildDate string
	GoVersion string
	Platform  string
	Version   string = "0.0.0"
)

var timeFormat = []string{
	time.RFC3339,
	time.RFC3339Nano,
	time.DateTime,
	time.ANSIC,
	time.UnixDate,
	time.RubyDate,
	time.RFC822,
	time.RFC822Z,
	time.RFC850,
	time.RFC1123,
	time.RFC1123Z,
	time.RFC822,
	time.RFC822Z,
	time.RFC850,
	time.RFC1123,
	time.RFC1123Z,
	"2006-01-02T15:04:05Z0700",
}

func AddVersionFlags(flagFunc func(shortVersion, fullVersion string)) {
	for _, tf := range timeFormat {
		buildDate, err := time.Parse(tf, BuildDate)
		if err == nil {
			BuildDate = buildDate.UTC().Format(time.RFC3339)
			break
		}
	}
	flagFunc(Version, fmt.Sprintf(
		"%s, build %s/%s [%s/%s]\n",
		Version, GitCommit, BuildDate, GoVersion, Platform,
	))
}
