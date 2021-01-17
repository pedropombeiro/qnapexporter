package main

import (
	"html/template"
	"net/http"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/dustin/go-humanize/english"
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
		</tbody>
	</table>
</body>
`
)

type EndpointStatus struct {
	Path       string
	Properties map[string]string
}

func handleRootHTTPRequest(w http.ResponseWriter, r *http.Request, args httpEndpointArgs) {
	w.Header().Add("Content-Type", "text/html")

	endpoints := getEndpoints(args)
	tmpl, err := template.New("html").Parse(statusHtmlTemplate)
	if err == nil {
		err = tmpl.Execute(w, endpoints)
		if err == nil {
			return
		}
	}

	args.logger.Println(err.Error())
	w.WriteHeader(http.StatusInternalServerError)
}

func getEndpoints(args httpEndpointArgs) []EndpointStatus {
	s := args.exporter.Status()
	ms := EndpointStatus{
		Path: metricsEndpoint,
		Properties: map[string]string{
			"Uptime":        humanizeTime(s.Uptime),
			"Last fetch":    humanizeTime(s.LastFetch),
			"Last duration": s.LastFetchDuration.String(),
			"Metrics":       humanize.Comma(int64(s.MetricCount)),
			"UPS":           humanizeList(s.Ups),
			"Devices":       humanizeList(s.Devices),
			"Volumes":       humanizeList(s.Volumes),
			"Interfaces":    humanizeList(s.Interfaces),
		},
	}
	endpoints := []EndpointStatus{ms}
	if args.grafanaURL != "" {
		endpoints = append(endpoints, EndpointStatus{
			Path: notificationEndpoint,
			Properties: map[string]string{
				"Last notification": humanizeTime(lastNotification),
			},
		})
	}

	return endpoints
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
