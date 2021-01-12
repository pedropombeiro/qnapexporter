package prometheus

import (
	"bytes"
	"io/ioutil"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExporter(t *testing.T) {
	e := NewExporter("1.1.1.1", log.New(ioutil.Discard, "", 0))

	require.NotNil(t, e)
	assert.IsType(t, &promExporter{}, e)
}

func TestWriteMetrics(t *testing.T) {
	e := NewExporter("8.8.8.8", log.New(ioutil.Discard, "", 0))
	b := new(bytes.Buffer)
	defer e.Close()

	err := e.WriteMetrics(b)
	assert.NoError(t, err)

	output := b.String()
	assert.Contains(t, output, "\nnode_time_seconds{node=\"")
	assert.Contains(t, output, "dial tcp 127.0.0.1:3493: connect: connection refused")
	assert.Contains(t, output, "listen ip4:icmp : socket: operation not permitted")
}
