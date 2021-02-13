package tagextractor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNotificationCenterTagExtractor(t *testing.T) {
	e := NewNotificationCenterTagExtractor()

	require.NotNil(t, e)
	assert.IsType(t, &notificationCenterTagExtractor{}, e)
}

func TestExtractTags(t *testing.T) {
	e := NewNotificationCenterTagExtractor()

	a, tags := e.Extract("[nas] [SecurityCounselor] Started running Security Checkup.")
	assert.Equal(t, "Started running Security Checkup.", a)
	assert.Equal(t, []string{"nas", "SecurityCounselor"}, tags)

	a, tags = e.Extract("Started running Security Checkup.")
	assert.Equal(t, "Started running Security Checkup.", a)
	assert.Empty(t, tags)
}
