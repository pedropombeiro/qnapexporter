package prometheus

import (
	"bytes"
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/pedropombeiro/qnapexporter/lib/exporter"
)

func TestNewExporter(t *testing.T) {
	config := ExporterConfig{
		PingTarget: "1.1.1.1",
		Logger:     log.New(io.Discard, "", 0),
	}
	e := NewExporter(config, nil)

	require.NotNil(t, e)
	assert.IsType(t, &promExporter{}, e)
}

func TestWriteMetrics(t *testing.T) {
	var s exporter.Status
	startTime := time.Now()
	config := ExporterConfig{
		PingTarget: "8.8.8.8",
		Logger:     log.New(io.Discard, "", 0),
	}
	e := NewExporter(config, &s)
	b := new(bytes.Buffer)
	defer e.Close()

	err := e.WriteMetrics(b)
	require.Error(t, err)

	output := b.String()
	assert.Contains(t, output, "\nnode_time_seconds{node=\"")
	assert.Contains(t, output, "dial tcp 127.0.0.1:3493: connect: connection refused")
	assert.True(t, s.Uptime.After(startTime))
	assert.True(t, s.LastFetch.After(s.Uptime))
	assert.NotZero(t, s.LastFetchDuration.Microseconds())
	assert.NotZero(t, s.MetricCount)
}

func BenchmarkWriteMetrics(b *testing.B) {
	config := ExporterConfig{
		PingTarget: "8.8.8.8",
		Logger:     log.New(io.Discard, "", 0),
	}
	e := NewExporter(config, nil)
	defer e.Close()

	for i := 0; i < b.N; i++ {
		buf := new(bytes.Buffer)
		_ = e.WriteMetrics(buf)
	}
}
