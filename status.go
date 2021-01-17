package main

import (
	"fmt"
	"net/http"

	"github.com/dustin/go-humanize"
	"github.com/dustin/go-humanize/english"
)

const (
	rootHeadHtml = `
	<head>
		<meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no">
		<style>
			body { font-family: helvetica; }
			th   { background: lightgrey; }
		</style>
		<title>Active endpoints</title>
	</head>

	<body>
`
	rootMetricsHtmlFragment = `
		<p>
			<a href="%s">%s</a>
			<table style="margin-left: 2em; margin-top: 1em">
				<tr>
					<th>Property</th>
					<th>Value</th>
				</tr>
				<tbody>
					<tr>
						<th>Uptime</th>
						<td>%s</td>
					</tr>
					<tr>
						<th>Last fetch</th>
						<td>%s</td>
					</tr>
					<tr>
						<th>Last duration</th>
						<td>%v</td>
					</tr>
					<tr>
						<th>Metrics</th>
						<td>%s</td>
					</tr>
					<tr>
						<th>UPS</th>
						<td>%v</td>
					</tr>
					<tr>
						<th>Devices</th>
						<td>%v</td>
					</tr>
					<tr>
						<th>Volumes</th>
						<td>%v</td>
					</tr>
					<tr>
						<th>Interfaces</th>
						<td>%v</td>
					</tr>
				</tbody>
			</table>
		</p>
	`
	rootNotificationsHtmlFragment = `
	<p>
		<div>%s</div>
		<table style="margin-left: 2em; margin-top: 1em">
			<tr>
				<th>Property</th>
				<th>Value</th>
			</tr>
			<tbody>
				<tr>
					<th>Last notification</th>
					<td>%s</td>
				</tr>
			</tbody>
		</table>
	</p>
`
)

func handleRootHTTPRequest(w http.ResponseWriter, r *http.Request, args httpEndpointArgs) {
	w.Header().Add("Content-Type", "text/html")

	s := args.exporter.Status()

	lf := "N/A"
	if !s.LastFetch.IsZero() {
		lf = humanize.Time(s.LastFetch)
	}
	_, _ = w.Write([]byte(fmt.Sprintf(rootHeadHtml+rootMetricsHtmlFragment,
		metricsEndpoint, metricsEndpoint,
		humanize.Time(s.Uptime),
		lf,
		s.LastFetchDuration,
		humanize.Comma(int64(s.MetricCount)),
		english.OxfordWordSeries(s.Ups, "and"),
		english.OxfordWordSeries(s.Devices, "and"),
		english.OxfordWordSeries(s.Volumes, "and"),
		english.OxfordWordSeries(s.Interfaces, "and"))))
	if args.grafanaURL != "" {
		ln := "N/A"
		if !lastNotification.IsZero() {
			ln = humanize.Time(lastNotification)
		}
		_, _ = w.Write([]byte(fmt.Sprintf(rootNotificationsHtmlFragment, notificationEndpoint, ln)))
	}
	_, _ = w.Write([]byte(`
	<body>
	`))
}
