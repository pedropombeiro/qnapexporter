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

	"gitlab.com/pedropombeiro/qnapexporter/lib/exporter"
	"gitlab.com/pedropombeiro/qnapexporter/lib/exporter/prometheus"
)

const (
	metricsEndpoint      = "/metrics"
	notificationEndpoint = "/notification"
)

var (
	healthCheckExpiry   time.Time
	healthCheckValidity time.Duration = time.Duration(5 * time.Minute)
	lastNotification    time.Time
)

type httpEndpointArgs struct {
	exporter                     exporter.Exporter
	port                         string
	grafanaURL, grafanaAuthToken string
	healthcheck                  string
	logger                       *log.Logger
}

func main() {
	runtime.GOMAXPROCS(0)

	port := flag.String("port", ":9094", "Port to serve at (e.g. :9094).")
	pingTarget := flag.String("ping-target", "1.1.1.1", "Host to periodically ping (e.g. 1.1.1.1).")
	healthcheck := flag.String("healthcheck", "", "Healthcheck service to ping every 5 minutes (currently supported: healthchecks.io:<check-id>).")
	grafanaURL := flag.String("grafana-url", os.Getenv("GRAFANA_URL"), "Grafana host (e.g.: https://grafana.example.com).")
	grafanaAuthToken := flag.String("grafana-auth-token", os.Getenv("GRAFANA_AUTH_TOKEN"), "Grafana authorization token.")
	logFile := flag.String("log", "", "Log file path (defaults to empty, i.e. STDOUT).")
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

	// Setup our Ctrl+C handler
	exitCh := make(chan os.Signal, 1)
	signal.Notify(exitCh, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	e := prometheus.NewExporter(*pingTarget, logger)

	args := httpEndpointArgs{
		exporter:         e,
		port:             *port,
		healthcheck:      *healthcheck,
		grafanaURL:       *grafanaURL,
		grafanaAuthToken: *grafanaAuthToken,
		logger:           logger,
	}

	err := serveHTTP(args, exitCh)
	if err != nil {
		log.Println(err.Error())
	}
	os.Exit(1)
}

func handleMetricsHTTPRequest(w http.ResponseWriter, r *http.Request, args httpEndpointArgs) {
	w.Header().Add("Content-Type", "text/plain")

	handleHealthcheckStart(args.healthcheck)

	err := args.exporter.WriteMetrics(w)
	if err != nil {
		args.logger.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}

	handleHealthcheckEnd(args.healthcheck, err)
}

func handleNotificationHTTPRequest(w http.ResponseWriter, r *http.Request, args httpEndpointArgs) {
	text := r.URL.Query().Get("text")
	if len(text) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	url := fmt.Sprintf("%s/api/annotations", args.grafanaURL)
	body := strings.NewReader(fmt.Sprintf(`{"tags":["nas"],"text":%q}`, text))

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		args.logger.Printf("Error creating Grafana annotation request: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if args.grafanaAuthToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", args.grafanaAuthToken))
	}

	c := http.Client{Timeout: 5 * time.Second}
	resp, err := c.Do(req)
	if err == nil {
		args.logger.Printf("Created Grafana annotation at %s: %s\n", url, resp.Status)
	} else {
		args.logger.Printf("Error creating Grafana annotation at %s: %v\n", url, err)
	}
}

func serveHTTP(args httpEndpointArgs, exitCh chan os.Signal) error {
	defer args.exporter.Close()

	// handle route using handler function
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { handleRootHTTPRequest(w, r, args) })
	http.HandleFunc(metricsEndpoint, func(w http.ResponseWriter, r *http.Request) { handleMetricsHTTPRequest(w, r, args) })
	if args.grafanaURL != "" {
		http.HandleFunc(notificationEndpoint, func(w http.ResponseWriter, r *http.Request) {
			handleNotificationHTTPRequest(w, r, args)
		})
	}

	// listen to port
	server := http.Server{Addr: args.port}
	server.ErrorLog = args.logger
	go func() {
		log.Printf("Listening to HTTP requests at %s\n", args.port)

		// Wait for program exit
		<-exitCh

		log.Println("Program aborted, exiting...")
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
		err := server.Shutdown(ctx)
		if err != nil {
			log.Println(err.Error())
		}
		cancel()
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
		_, err := client.Head(url)
		log.Printf("Sent %s healthcheck ping to %s: %v\n", endpoint, url, err)
	}

	if !start {
		healthCheckExpiry = healthCheckExpiry.Add(healthCheckValidity)
	}
}
