package exporter

import "io"

// Exporter defines an interface for capturing and writing out a set of metrics
type Exporter interface {
	WriteMetrics(w io.Writer) error
}
