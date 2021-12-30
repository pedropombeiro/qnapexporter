package status

import (
	"html/template"
	"io"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/dustin/go-humanize/english"
	"gitlab.com/pedropombeiro/qnapexporter/lib/exporter"
)

const (
	statusHtmlTemplate = `
<head>
	<meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no">
	<style>
		body { font-family: helvetica; }
		th   {
			background: lightgrey;
			padding-left: 1em;
			padding-right: 1em;
		}
		td   {
			padding-left: 1em;
			padding-right: 1em;
		}
	</style>
	<title>qnapexporter status</title>
</head>

<body>
	<h1>Active endpoints</h1>
	<table>
		<tbody>
			{{ range . }}
			{{ if .Path }}
			<tr>
				<td>
					<hr/>
					<a href="{{ .Path }}">{{ .Path }}</a>
				</td>
			</tr>
			<tr>
				<td>
					<table style="margin-left: 2em; margin-top: 1em">
						<tbody>
							{{ range $key, $value := .Properties }}
							<tr>
								<th>{{ $key }}</th>
								<td>{{ $value }}</td>
							</tr>
							{{ else }}
							<tr>
								<th colspan="2">No properties</th>
							</tr>
							{{ end }}
						</tbody>
					</table>
				</td>
			</tr>
			{{ end }}
			{{ end }}
		</tbody>
	</table>
</body>
`
)

type endpointStatus struct {
	Path       string
	Properties map[string]string
}

type Status struct {
	MetricsEndpoint      string
	NotificationEndpoint string
	ExporterStatus       exporter.Status
	LastNotification     time.Time
}

func (s *Status) WriteHTML(w io.Writer) error {
	e := s.ExporterStatus
	ms := endpointStatus{
		Path: s.MetricsEndpoint,
		Properties: map[string]string{
			"Uptime":        humanizeTime(e.Uptime),
			"Last fetch":    humanizeTime(e.LastFetch),
			"Last duration": e.LastFetchDuration.String(),
			"Metrics":       humanize.Comma(int64(e.MetricCount)),
			"UPS":           humanizeList(e.Ups),
			"Devices":       humanizeList(e.Devices),
			"Volumes":       humanizeList(e.Volumes),
			"Interfaces":    humanizeList(e.Interfaces),
			"Enclosures":    humanizeList(e.Enclosures),
			"dm-caches":     humanizeList(e.DmCaches),
			"Docker":        e.Docker,
		},
	}
	endpoints := []endpointStatus{ms}
	endpoints = append(endpoints, endpointStatus{
		Path: s.NotificationEndpoint,
		Properties: map[string]string{
			"Last notification": humanizeTime(s.LastNotification),
		},
	})

	tmpl, err := template.New("html").Parse(statusHtmlTemplate)
	if err == nil {
		err = tmpl.Execute(w, endpoints)
		if err == nil {
			return nil
		}
	}

	return err
}

func humanizeList(a []string) string {
	if len(a) == 0 {
		return "N/A"
	}

	return english.OxfordWordSeries(a, "and")
}

func humanizeTime(t time.Time) string {
	if t.IsZero() {
		return "N/A"
	}

	return humanize.Time(t)
}
