package exporter

import (
	"io"
	"time"
)

// Exporter defines an interface for capturing and writing out a set of metrics
type Exporter interface {
	WriteMetrics(w io.Writer) error
	Close()
}

type Status struct {
	Branch, Revision, Built, Version string

	Uptime            time.Time
	LastFetch         time.Time
	LastFetchDuration time.Duration
	MetricCount       int
	Ups               []string
	Interfaces        []string
	Devices           []string
	Volumes           []string
}
