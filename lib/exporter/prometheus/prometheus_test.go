package prometheus

import (
	"bytes"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/pedropombeiro/qnapexporter/lib/exporter"
)

func TestNewExporter(t *testing.T) {
	e := NewExporter("1.1.1.1", nil, log.New(ioutil.Discard, "", 0))

	require.NotNil(t, e)
	assert.IsType(t, &promExporter{}, e)
}

func TestWriteMetrics(t *testing.T) {
	var s exporter.Status
	startTime := time.Now()
	e := NewExporter("8.8.8.8", &s, log.New(ioutil.Discard, "", 0))
	b := new(bytes.Buffer)
	defer e.Close()

	err := e.WriteMetrics(b)
	assert.NoError(t, err)

	output := b.String()
	assert.Contains(t, output, "\nnode_time_seconds{node=\"")
	assert.Contains(t, output, "dial tcp 127.0.0.1:3493: connect: connection refused")
	assert.Contains(t, output, "listen ip4:icmp : socket: operation not permitted")
	assert.True(t, s.Uptime.After(startTime))
	assert.True(t, s.LastFetch.After(s.Uptime))
	assert.NotZero(t, s.LastFetchDuration.Microseconds())
	assert.NotZero(t, s.MetricCount)
}

func BenchmarkWriteMetrics(b *testing.B) {
	e := NewExporter("8.8.8.8", nil, log.New(ioutil.Discard, "", 0))
	defer e.Close()

	for i := 0; i < b.N; i++ {
		buf := new(bytes.Buffer)
		_ = e.WriteMetrics(buf)
	}
}
