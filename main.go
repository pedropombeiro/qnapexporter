package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/pedropombeiro/qnapexporter/lib/exporter"
	"github.com/pedropombeiro/qnapexporter/lib/exporter/prometheus"
	"github.com/pedropombeiro/qnapexporter/lib/notifications"
	"github.com/pedropombeiro/qnapexporter/lib/notifications/tagextractor"
	"github.com/pedropombeiro/qnapexporter/lib/status"
	"github.com/pedropombeiro/qnapexporter/lib/utils"
)

const (
	metricsEndpoint      = "/metrics"
	notificationEndpoint = "/notification"
)

var (
	healthCheckExpiry   time.Time
	healthCheckValidity time.Duration = time.Duration(5 * time.Minute)
)

type httpServerArgs struct {
	exporter    exporter.Exporter
	port        string
	healthcheck string
	logger      *log.Logger
}

func main() {
	runtime.GOMAXPROCS(0)

	port := flag.String("port", ":9094", "Port to serve at (e.g. :9094).")
	pingTarget := flag.String("ping-target", "", "Host to periodically ping (e.g. 1.1.1.1).")
	healthcheck := flag.String("healthcheck", os.Getenv("HEALTHCHECK_CONFIG"), "Healthcheck service to ping every 5 minutes (currently supported: healthchecks.io:<check-id>).")
	grafanaURL := flag.String("grafana-url", os.Getenv("GRAFANA_URL"), "Grafana host (e.g.: https://grafana.example.com).")
	grafanaAuthToken := flag.String("grafana-auth-token", os.Getenv("GRAFANA_AUTH_TOKEN"), "Grafana authorization token.")
	grafanaTags := flag.String("grafana-tags", os.Getenv("GRAFANA_TAGS"), "Grafana annotation tags, separated by quotes (default: 'nas').")
	logFile := flag.String("log", "", "Log file path (defaults to empty, i.e. STDOUT).")
	defaultUsage := flag.Usage
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "qnapexporter version %s (%s-%s) built on %s\n", utils.VERSION, utils.REVISION, utils.BRANCH, utils.BUILT)
		fmt.Fprintln(flag.CommandLine.Output(), "")
		defaultUsage()
	}
	flag.Parse()

	healthCheckExpiry = time.Now()

	var logWriter io.Writer = os.Stderr
	if *logFile != "" {
		lf, err := os.OpenFile(*logFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatalf("Error creating log file: %v\n", err)
		}
		defer lf.Close()

		logWriter = lf
	}
	logger := log.New(logWriter, "", log.LstdFlags)

	serverStatus := &status.Status{
		MetricsEndpoint: metricsEndpoint,
		ExporterStatus: exporter.Status{
			Branch:   utils.BRANCH,
			Revision: utils.REVISION,
			Built:    utils.BUILT,
			Version:  utils.VERSION,
		},
	}
	if *grafanaURL != "" {
		serverStatus.NotificationEndpoint = notificationEndpoint
	}

	config := prometheus.ExporterConfig{
		PingTarget: *pingTarget,
		Logger:     logger,
	}
	e := prometheus.NewExporter(config, &serverStatus.ExporterStatus)

	args := httpServerArgs{
		exporter:    e,
		port:        *port,
		healthcheck: *healthcheck,
		logger:      logger,
	}
	notifCenterAnnotator := notifications.NewRegionMatchingAnnotator(
		*grafanaURL,
		*grafanaAuthToken,
		append(strings.Split(*grafanaTags, ","), "notification-center"),
		tagextractor.NewNotificationCenterTagExtractor(),
		notifications.NewRegionMatcher(20),
		&http.Client{Timeout: 5 * time.Second},
		logger,
	)
	dockerAnnotator := notifications.NewSimpleAnnotator(
		*grafanaURL,
		*grafanaAuthToken,
		append(strings.Split(*grafanaTags, ","), "docker"),
		&http.Client{Timeout: 5 * time.Second},
		logger,
	)

	ctx, cancelFn := context.WithCancel(context.Background())

	// Setup our Ctrl+C handler
	exitCh := make(chan os.Signal, 1)
	signal.Notify(exitCh, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		defer cancelFn()

		// Wait for program exit
		<-exitCh
	}()

	go func() { _ = handleDockerEvents(ctx, args, dockerAnnotator, &serverStatus.ExporterStatus) }()

	err := serveHTTP(ctx, args, notifCenterAnnotator, serverStatus)
	if err != nil {
		log.Println(err.Error())
	}
	os.Exit(1)
}

func handleMetricsHTTPRequest(w http.ResponseWriter, r *http.Request, args httpServerArgs) {
	w.Header().Add("Content-Type", "text/plain")

	handleHealthcheckStart(args.healthcheck)

	err := args.exporter.WriteMetrics(w)
	if err != nil {
		args.logger.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}

	handleHealthcheckEnd(args.healthcheck, err)
}

func handleNotificationHTTPRequest(w http.ResponseWriter, r *http.Request, annotator notifications.Annotator) {
	notification := r.URL.Query().Get("text")
	if len(notification) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, _ = annotator.Post(notification, time.Now())
}

func handleRootHTTPRequest(w http.ResponseWriter, r *http.Request, serverStatus *status.Status, logger *log.Logger) {
	w.Header().Add("Content-Type", "text/html")
	w.Header().Add("Cache-Control", "no-cache")

	err := serverStatus.WriteHTML(w)
	if err != nil {
		logger.Println(err.Error())

		w.WriteHeader(http.StatusInternalServerError)
	}
}

func serveHTTP(ctx context.Context, args httpServerArgs, annotator notifications.Annotator, serverStatus *status.Status) error {
	defer args.exporter.Close()

	// handle route using handler function
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleRootHTTPRequest(w, r, serverStatus, args.logger)
	})
	http.HandleFunc(metricsEndpoint, func(w http.ResponseWriter, r *http.Request) {
		handleMetricsHTTPRequest(w, r, args)
	})
	if serverStatus.NotificationEndpoint != "" {
		http.HandleFunc(notificationEndpoint, func(w http.ResponseWriter, r *http.Request) {
			serverStatus.LastNotification = time.Now()
			handleNotificationHTTPRequest(w, r, annotator)
		})
	}

	// listen to port
	server := http.Server{Addr: args.port}
	server.ErrorLog = args.logger
	go func() {
		log.Printf("Listening to HTTP requests at %s\n", args.port)

		// Wait for program exit
		<-ctx.Done()

		log.Println("Program aborted, exiting...")
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
		defer cancel()
		err := server.Shutdown(ctx)
		if err != nil {
			log.Println(err.Error())
		}
	}()

	return server.ListenAndServe()
}

func handleHealthcheckStart(healthcheck string) {
	handleHealthcheck(healthcheck, true, nil)
}

func handleHealthcheckEnd(healthcheck string, err error) {
	handleHealthcheck(healthcheck, false, err)
}

func handleHealthcheck(healthcheck string, start bool, err error) {
	if healthcheck == "" {
		return
	}

	if !time.Now().After(healthCheckExpiry) {
		return
	}

	parts := strings.SplitN(healthcheck, ":", 2)
	if len(parts) < 2 {
		log.Printf("Configuration error in healthcheck: %s\n", healthcheck)
		return
	}

	switch parts[0] {
	case "healthchecks.io":
		client := http.Client{Timeout: 5 * time.Second}
		endpoint := ""
		switch {
		case start:
			endpoint = "start"
		case err != nil:
			endpoint = "fail"
		}

		url := fmt.Sprintf("https://hc-ping.com/%s", parts[1])
		if endpoint != "" {
			url += "/" + endpoint
		}
		if err != nil {
			_, err = client.Post(url, "text/plain", strings.NewReader(err.Error()))
		} else {
			_, err = client.Head(url)
		}
		log.Printf("Sent %s healthcheck ping to %s: %v\n", endpoint, url, err)
	}

	if !start {
		healthCheckExpiry = healthCheckExpiry.Add(healthCheckValidity)
	}
}
