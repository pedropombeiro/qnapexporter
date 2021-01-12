package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitlab.com/pedropombeiro/qnapexporter/lib/exporter"
	"gitlab.com/pedropombeiro/qnapexporter/lib/exporter/prometheus"
)

func main() {
	port := flag.String("port", ":9094", "Port to serve at (e.g. :9094).")
	pingTarget := flag.String("ping-target", "1.1.1.1", "Host to periodically ping (e.g. 1.1.1.1).")
	logFile := flag.String("log", "", "Log file path (defaults to empty, i.e. STDOUT).")
	flag.Parse()

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

	err := serveHTTP(e, *port, logger, exitCh)
	if err != nil {
		log.Println(err.Error())
	}
	os.Exit(1)
}

func serveHTTP(e exporter.Exporter, port string, logger *log.Logger, exitCh chan os.Signal) error {
	defer e.Close()

	// handle route using handler function
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/plain")

		err := e.WriteMetrics(w)
		if err != nil {
			logger.Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	// listen to port
	server := &http.Server{Addr: port}
	server.ErrorLog = logger
	go func() {
		log.Printf("Listening to HTTP requests at %s\n", port)

		for {
			select {
			case <-exitCh:
				log.Println("Program aborted, exiting...")
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
				err := server.Shutdown(ctx)
				if err != nil {
					log.Println(err.Error())
				}
				cancel()
				return
			case <-time.After(1 * time.Second):
				break
			}
		}
	}()

	return server.ListenAndServe()
}
