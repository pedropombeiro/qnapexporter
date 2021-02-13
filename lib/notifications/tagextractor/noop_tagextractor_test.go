package tagextractor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNoOpTagExtractor(t *testing.T) {
	e := NewNoOpTagExtractor()

	require.NotNil(t, e)
	assert.IsType(t, &noOpTagExtractor{}, e)
}

func TestNoOpExtractTags(t *testing.T) {
	e := NewNoOpTagExtractor()

	a, tags := e.Extract("[nas] [SecurityCounselor] Started running Security Checkup.")
	assert.Equal(t, "[nas] [SecurityCounselor] Started running Security Checkup.", a)
	assert.Nil(t, tags)

	a, tags = e.Extract("Started running Security Checkup.")
	assert.Equal(t, "Started running Security Checkup.", a)
	assert.Nil(t, tags)
}
