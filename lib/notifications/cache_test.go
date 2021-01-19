package notifications

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNoOpAnnotationCache(t *testing.T) {
	c := NewNoOpAnnotationCache()

	require.NotNil(t, c)
	assert.IsType(t, &noOpAnnotationCache{}, c)
}

func TestNewMatcherAnnotationCache(t *testing.T) {
	c := NewMatcherAnnotationCache(20)

	require.NotNil(t, c)
	assert.IsType(t, &matcherAnnotationCache{}, c)
}

func TestMatcherAnnotationCacheWithSmallCacheSize(t *testing.T) {
	c := NewMatcherAnnotationCache(2)

	c.Add(1, `[nas] [Storage & Snapshots] Started ext4lazyinit. Volume: ForeignMedia_Vol, Storage pool: "1".`)
	c.Add(2, "message 2")
	c.Add(3, "[nas] [Storage & Snapshots] Started creating scheduled snapshot. Volume: System_Vol.")

	id := c.Match(`[nas] [Storage & Snapshots] Finished ext4lazyinit. Volume: ForeignMedia_Vol, Storage pool: "1".`)
	assert.Equal(t, -1, id)

	id = c.Match("[nas] [Storage & Snapshots] Finished creating scheduled snapshot. Volume: System_Vol.")
	assert.Equal(t, 3, id)
}

func TestMatcherAnnotationCache(t *testing.T) {
	c := NewMatcherAnnotationCache(20)

	id1 := c.Match("[nas] [Malware Remover] Started scanning.")
	require.Equal(t, -1, id1)

	c.Add(1, "[nas] [Malware Remover] Started scanning.")
	c.Add(3, "[nas] [Storage & Snapshots] Started creating scheduled snapshot. Volume: System_Vol.")
	c.Add(4, `[nas] [Storage & Snapshots] Started ext4lazyinit. Volume: ForeignMedia_Vol, Storage pool: "1".`)
	c.Add(5, "[nas] [Firmware Update] Started downloading firmware 4.5.1.1540 Build 20210107.")

	id1 = c.Match("[nas] [Malware Remover] Scan completed.")
	require.Equal(t, 1, id1)

	id2 := c.Match("[nas] [Disk scanner] Scan completed.")
	require.Equal(t, -1, id2)

	id5 := c.Match("[nas] [Firmware Update] Started updating firmware 4.5.1.1540 Build 20210107.")
	require.Equal(t, 5, id5)

	c.Add(6, "[nas] [Firmware Update] Started updating firmware.")
	c.Add(7, "[nas] [Disk S.M.A.R.T.] QM2 (PCIe1): PCIe 1 M.2 SSD 2 Rapid Test started.")
	c.Add(8, `[nas] [Antivirus] Started scan job "User data".`)
	c.Add(9, `[RunLast] begin "start" scripts ...`)
	c.Add(10, `[nas] [SortMyQPKGs] 'autofix' requested`)
	c.Add(11, `[nas] [SecurityCounselor] Started running Security Checkup.`)

	id3 := c.Match("[nas] [Storage & Snapshots] Finished creating scheduled snapshot. Volume: System_Vol.")
	require.Equal(t, 3, id3)

	id4 := c.Match(`[nas] [Storage & Snapshots] Finished ext4lazyinit. Volume: ForeignMedia_Vol, Storage pool: "1".`)
	require.Equal(t, 4, id4)

	id11 := c.Match("[nas] [SecurityCounselor] Finished running Security Checkup.")
	require.Equal(t, 11, id11)

	id10 := c.Match(`[nas] [SortMyQPKGs] 'autofix' completed`)
	require.Equal(t, 10, id10)

	id9 := c.Match(`[RunLast] end "start" scripts`)
	require.Equal(t, 9, id9)

	id7 := c.Match("[nas] [Disk S.M.A.R.T.] QM2 (PCIe1): PCIe 1 M.2 SSD 2 Rapid Test result: Completed without error.")
	require.Equal(t, 7, id7)

	id8 := c.Match(`[nas] [Antivirus] Completed scan job "User data". Go to "Antivirus" > "Reports" to review the full report.`)
	require.Equal(t, 8, id8)

	id6 := c.Match("[nas] [Firmware Update] Updated system.")
	require.Equal(t, 6, id6)
}
