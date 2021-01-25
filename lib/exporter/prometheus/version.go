package prometheus

import "fmt"

func (e *promExporter) getVersionMetrics() (metrics []metric, err error) {
	return []metric{
		{
			name:  "go_program",
			attr:  fmt.Sprintf("branch=%q,revision=%q,built=%q,version=%q", e.status.Branch, e.status.Revision, e.status.Built, e.status.Version),
			help:  "Information about qnapexporter",
			value: 1,
		},
	}, nil
}
