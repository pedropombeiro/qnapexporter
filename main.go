package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"gitlab.com/pedropombeiro/qnapexporter/lib/exporter"
	"gitlab.com/pedropombeiro/qnapexporter/lib/exporter/prometheus"
)

const metricsDir = "/share/CACHEDEV1_DATA/Container/data/grafana/qnapnodeexporter"

func main() {
	// Setup our Ctrl+C handler
	exitCh := make(chan os.Signal, 1)
	signal.Notify(exitCh, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	e := prometheus.NewExporter()

	for {
		err := writeMetrics(e)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
		}

		select {
		case <-exitCh:
			fmt.Fprintln(os.Stderr, "Program aborted, exiting")
			os.Exit(1)
		case <-time.After(5 * time.Second):
			break
		}
	}
}

func writeMetrics(e exporter.Exporter) error {
	var err error
	var tmpFile *os.File

	tmpFile, err = ioutil.TempFile(metricsDir, "qnapexporter-")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	w := bufio.NewWriter(tmpFile)

	_ = e.WriteMetrics(w)

	// Close the file
	w.Flush()
	if err := tmpFile.Chmod(0644); err != nil {
		return fmt.Errorf("change temporary file permissions: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}

	err = os.Rename(tmpFile.Name(), path.Join(metricsDir, "metrics"))
	if err != nil {
		return fmt.Errorf("move temporary file: %w", err)
	}

	return nil
}
