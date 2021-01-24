package prometheus

import "fmt"

//nolint:deadcode
var (
	REVISION = "HEAD"
	BRANCH   = "HEAD"
	BUILT    = "unknown"
	VERSION  = "dev"
)

func (e *promExporter) getVersionMetrics() (metrics []metric, err error) {
	return []metric{
		{
			name:  "go_program",
			attr:  fmt.Sprintf("branch=%q,revision=%q,built=%q,version=%q", BRANCH, REVISION, BUILT, VERSION),
			help:  "Information about qnapexporter",
			value: 1,
		},
	}, nil
}
