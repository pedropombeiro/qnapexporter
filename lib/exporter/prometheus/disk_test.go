package prometheus

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDmCacheRawStats(t *testing.T) {
	input := `reads : 523268994
writes : 243137327
read hit : 500560334
read miss : 17996630
write hit : 240558830
write miss : 1465739
dirty write hit : 236879727
discard : 0
ssd reads : 503538895
ssd writes : 254208441
uncached reads : 4712030
uncached writes : 1112758
uncached seq reads : 56428
uncached seq writes : 414
page pool grow : 6
not a stat line
broken : value
`
	metrics := parseDmCacheRawStats(strings.Split(input, "\n"), `device="dm-3"`)

	values := make(map[string]float64, len(metrics))
	for _, m := range metrics {
		assert.Equal(t, `device="dm-3"`, m.attr)
		assert.Equal(t, "counter", m.metricType)
		assert.NotEmpty(t, m.help)
		values[m.name] = m.value
	}

	assert.Equal(t, map[string]float64{
		"node_dmcache_stats_reads_total":                      523268994,
		"node_dmcache_stats_writes_total":                     243137327,
		"node_dmcache_stats_read_hits_total":                  500560334,
		"node_dmcache_stats_read_misses_total":                17996630,
		"node_dmcache_stats_write_hits_total":                 240558830,
		"node_dmcache_stats_write_misses_total":               1465739,
		"node_dmcache_stats_dirty_write_hits_total":           236879727,
		"node_dmcache_stats_ssd_reads_total":                  503538895,
		"node_dmcache_stats_ssd_writes_total":                 254208441,
		"node_dmcache_stats_uncached_reads_total":             4712030,
		"node_dmcache_stats_uncached_writes_total":            1112758,
		"node_dmcache_stats_uncached_sequential_reads_total":  56428,
		"node_dmcache_stats_uncached_sequential_writes_total": 414,
	}, values)
}

func TestParseDmCacheRawStatsSkipsMissingFields(t *testing.T) {
	metrics := parseDmCacheRawStats([]string{"reads : 10", ""}, `device="dm-3"`)

	assert.Len(t, metrics, 1)
	assert.Equal(t, "node_dmcache_stats_reads_total", metrics[0].name)
	assert.Equal(t, 10.0, metrics[0].value)
}

func TestParseDmCacheCurrStats(t *testing.T) {
	input := `reads : 66870
writes : 13782
read hit : 58033
write hit : 12522
dirty write hit : 12431
read hit percent (%) : 86
write hit percent (%) : 90`
	readHits, readTotal, writeHits, writeTotal := parseDmCacheCurrStats(strings.Split(input, "\n"))

	assert.Equal(t, 58033.0, readHits)
	assert.Equal(t, 66870.0, readTotal)
	assert.Equal(t, 12522.0, writeHits)
	assert.Equal(t, 13782.0, writeTotal)
}

func TestParseDmCacheCurrStatsSkipsLinesThatDontParse(t *testing.T) {
	// A line without ":" used to panic, and a value that isn't a number dropped every value.
	readHits, readTotal, writeHits, writeTotal := parseDmCacheCurrStats(
		[]string{"reads : 10", "", "no separator", "writes : n/a", "read hit : 7"},
	)

	assert.Equal(t, 7.0, readHits)
	assert.Equal(t, 10.0, readTotal)
	assert.Zero(t, writeHits)
	assert.Zero(t, writeTotal)
}
