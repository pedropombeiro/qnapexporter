package status

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWriteHTML(t *testing.T) {
	s := Status{NotificationEndpoint: "/notifications", LastNotification: time.Now()}
	err := s.WriteHTML(os.Stderr)
	require.NoError(t, err)
}
