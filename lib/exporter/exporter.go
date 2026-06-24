// Package exporter defines the Exporter interface and the runtime status shared
// across metric exporter implementations.
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

// Status captures the runtime state of an exporter, including build metadata
// and the inventory of devices discovered during metric collection.
type Status struct {
	Branch, Revision, Built, Version string

	Uptime            time.Time
	LastFetch         time.Time
	LastFetchDuration time.Duration
	MetricCount       int
	Ups               []string
	Interfaces        []string
	Devices           []string
	NvmeDevices       []string
	Volumes           []string
	Enclosures        []string
	DmCaches          []string
	DmCacheDevice     string
	Docker            string
}
