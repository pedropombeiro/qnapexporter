package prometheus

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pedropombeiro/qnapexporter/lib/utils"
	"github.com/shirou/gopsutil/v4/disk"
)

func (e *promExporter) getSysInfoHdMetrics() ([]metric, error) {
	if e.getsysinfo == "" {
		return nil, nil
	}

	metrics := make([]metric, 0, e.syshdnum)
	highestAvailable := 0

	for hdnum := 1; hdnum <= e.syshdnum; hdnum++ {
		hdnumStr := strconv.Itoa(hdnum)
		tempStr, err := utils.ExecCommand(e.getsysinfo, "hdtmp", hdnumStr)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(tempStr, "--") {
			continue
		}

		smart, err := utils.ExecCommand(e.getsysinfo, "hdsmart", hdnumStr)
		if err != nil {
			return nil, err
		}

		temp, err := strconv.ParseFloat(strings.SplitN(tempStr, " ", 2)[0], 64)
		if err != nil {
			return metrics, err
		}

		metrics = append(metrics, metric{
			name:  "node_hdtmp_C",
			attr:  fmt.Sprintf(`hd=%q,smart=%q`, hdnumStr, smart),
			value: temp,
		})
		highestAvailable = hdnum
	}

	// Do not ask for data next time on disks that do not report it
	e.syshdnum = highestAvailable

	return metrics, nil
}

func (e *promExporter) getFlashCacheStatsMetrics() ([]metric, error) {
	if e.kernelVersion >= 5 {
		return nil, nil
	}

	lines, err := utils.ReadFileLines(flashcacheStatsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Ignore if the file does not exist
			return nil, nil
		}

		return nil, err
	}

	metrics := make([]metric, 0, len(lines))
	for _, line := range lines {
		tokens := strings.SplitN(line, ":", 2)
		valueStr := strings.TrimSpace(tokens[1])
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			return nil, err
		}

		metrics = append(metrics, metric{
			name:  "node_flashcache_" + tokens[0],
			value: value,
		})
	}

	return metrics, nil
}

func (e *promExporter) getDmCacheStatsMetrics() ([]metric, error) {
	if len(e.dmCacheClients) == 0 {
		return nil, nil
	}

	args := append([]string{"status", "--noflush"}, e.dmCacheClients...)
	lines, err := utils.ExecCommandGetLines("dmsetup", args...)
	if err != nil {
		return nil, fmt.Errorf("get dm-cache status (dmsetup %s): %w", args, err)
	}

	metrics := make([]metric, 0, len(lines)+6*2)
	for index, line := range lines {
		tokens := strings.SplitN(line, " ", 13)
		cache := e.dmCacheClients[index]
		var allocationTokens []string
		for _, t := range tokens {
			if strings.Contains(t, "/") {
				allocationTokens = strings.SplitN(t, "/", 2)
				break
			}
		}
		attr := fmt.Sprintf("device=%q", cache)

		metrics = appendFloatMetric(metrics, "node_flashcache_cached_blocks", allocationTokens[0], 1, attr, "Number of blocks resident in the cache")
		metrics = appendFloatMetric(metrics, "node_flashcache_total_blocks", allocationTokens[1], 1, attr, "Total number of cache blocks")
		metrics = appendFloatMetric(metrics, "node_dmcache_used_bytes_total", allocationTokens[0], 1024*1024, attr, "Number of blocks resident in the cache")
		metrics = appendFloatMetric(metrics, "node_dmcache_bytes_total", allocationTokens[1], 1024*1024, attr, "Total number of cache blocks")
	}

	return e.appendDmCacheHitMetrics(metrics)
}

func (e *promExporter) appendDmCacheHitMetrics(metrics []metric) ([]metric, error) {
	if e.dmCacheDeviceMinorNumber == "" {
		return metrics, nil
	}

	cache := fmt.Sprintf("dm-%s", e.dmCacheDeviceMinorNumber)
	dmCacheStatsFilePath := fmt.Sprintf(dmCacheStatsFilePathFormat, cache)

	lines, err := utils.ReadFileLines(dmCacheStatsFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Ignore if the file does not exist
			return metrics, nil
		}

		return nil, err
	}

	var readHits, readTotal, writeHits, writeTotal float64
	for _, line := range lines {
		tokens := strings.SplitN(line, ":", 2)
		id := strings.TrimSpace(tokens[0])
		valueStr := strings.TrimSpace(tokens[1])
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			return nil, err
		}

		switch id {
		case "read hit":
			readHits = value
		case "reads":
			readTotal = value
		case "write hit":
			writeHits = value
		case "writes":
			writeTotal = value
		}
	}

	attr := fmt.Sprintf("device=%q", cache)
	metrics = append(metrics, metric{
		name:       "node_flashcache_read_hits",
		attr:       attr,
		value:      readHits,
		help:       "Number of times a READ bio has been mapped to the cache",
		metricType: "counter",
	})
	metrics = append(metrics, metric{
		name:       "node_dmcache_read_hit_total",
		attr:       attr,
		value:      readHits,
		help:       "Number of times a READ bio has been mapped to the cache",
		metricType: "counter",
	})
	metrics = append(metrics, metric{
		name:       "node_flashcache_reads",
		attr:       attr,
		value:      readTotal,
		help:       "Number of times a READ bio has occurred",
		metricType: "counter",
	})
	metrics = append(metrics, metric{
		name:       "node_dmcache_read_total",
		attr:       attr,
		value:      readTotal,
		help:       "Number of times a READ bio has occurred",
		metricType: "counter",
	})
	if readTotal > 0 {
		metrics = append(metrics, metric{
			name:       "node_flashcache_read_hit_percent",
			attr:       attr,
			value:      readHits / readTotal * 100,
			metricType: "gauge",
		})
		metrics = append(metrics, metric{
			name:       "node_dmcache_read_hit_percent",
			attr:       attr,
			value:      readHits / readTotal * 100,
			metricType: "gauge",
		})
	}

	metrics = append(metrics, metric{
		name:       "node_flashcache_write_hits",
		attr:       attr,
		value:      writeHits,
		help:       "Number of times a WRITE bio has been mapped to the cache",
		metricType: "counter",
	})
	metrics = append(metrics, metric{
		name:       "node_dmcache_write_hit_total",
		attr:       attr,
		value:      writeHits,
		help:       "Number of times a WRITE bio has been mapped to the cache",
		metricType: "counter",
	})
	metrics = append(metrics, metric{
		name:       "node_flashcache_writes",
		attr:       attr,
		value:      writeTotal,
		help:       "Number of times a WRITE bio has occurred",
		metricType: "counter",
	})
	metrics = append(metrics, metric{
		name:       "node_dmcache_write_total",
		attr:       attr,
		value:      writeTotal,
		help:       "Number of times a WRITE bio has occurred",
		metricType: "counter",
	})
	if writeTotal > 0 {
		metrics = append(metrics, metric{
			name:       "node_flashcache_write_hit_percent",
			attr:       attr,
			value:      writeHits / writeTotal * 100,
			metricType: "gauge",
		})
		metrics = append(metrics, metric{
			name:       "node_dmcache_write_hit_percent",
			attr:       attr,
			value:      writeHits / writeTotal * 100,
			metricType: "gauge",
		})
	}

	return e.appendDmCacheRawStatsMetrics(metrics, cache)
}

func appendFloatMetric(metrics []metric, metricName string, valueStr string, factor float64, attr string, help string) []metric {
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return metrics
	}

	return append(metrics, metric{
		name:       metricName,
		attr:       attr,
		value:      value * factor,
		help:       help,
		metricType: "counter",
	})
}

func (e *promExporter) getDiskStatsMetrics() ([]metric, error) {
	stats, err := disk.IOCounters(e.devices...)
	if err != nil {
		return nil, err
	}

	metrics := make([]metric, 0, len(e.devices)*2)
	for _, s := range stats {
		attr := fmt.Sprintf(`device=%q`, s.Name)

		metrics = append(
			metrics,
			metric{
				name:       "node_disk_read_bytes_total",
				attr:       attr,
				value:      float64(s.ReadBytes),
				help:       "Total number of bytes read",
				metricType: "counter",
			},
			metric{
				name:       "node_disk_written_bytes_total",
				attr:       attr,
				value:      float64(s.WriteBytes),
				help:       "Total number of bytes written",
				metricType: "counter",
			},
			metric{
				name:       "node_disk_read_ops_total",
				attr:       attr,
				value:      float64(s.ReadCount),
				help:       "Total number of read operations",
				metricType: "counter",
			},
			metric{
				name:       "node_disk_write_ops_total",
				attr:       attr,
				value:      float64(s.WriteCount),
				help:       "Total number of write operations",
				metricType: "counter",
			},
			metric{
				name:       "node_disk_read_time_msec",
				attr:       attr,
				value:      float64(s.ReadTime),
				help:       "# of milliseconds spent reading",
				metricType: "counter",
			},
			metric{
				name:       "node_disk_write_time_msec",
				attr:       attr,
				value:      float64(s.WriteTime),
				help:       "# of milliseconds spent writing",
				metricType: "counter",
			},
			metric{
				name:       "node_disk_iops_in_progress",
				attr:       attr,
				value:      float64(s.IopsInProgress),
				help:       "# of I/Os currently in progress",
				metricType: "gauge",
			},
			metric{
				name:       "node_disk_iotime_msec",
				attr:       attr,
				value:      float64(s.IoTime),
				help:       "# of milliseconds spent doing I/Os",
				metricType: "counter",
			},
		)
	}

	return metrics, nil
}

// dmCacheRawStats maps the fields of raw_stats to the names they're exported under, with their help text.
var dmCacheRawStats = []struct {
	field string
	name  string
	help  string
}{
	{"reads", "reads", "Number of READ bios since boot"},
	{"writes", "writes", "Number of WRITE bios since boot"},
	{"read hit", "read_hits", "Number of READ bios served from the cache since boot"},
	{"read miss", "read_misses", "Number of READ bios that missed the cache since boot"},
	{"write hit", "write_hits", "Number of WRITE bios mapped to the cache since boot"},
	{"write miss", "write_misses", "Number of WRITE bios that missed the cache since boot"},
	{"dirty write hit", "dirty_write_hits", "Number of WRITE bios that hit a dirty cache block since boot"},
	{"ssd reads", "ssd_reads", "Number of reads from the cache SSDs since boot"},
	{"ssd writes", "ssd_writes", "Number of writes to the cache SSDs since boot"},
	{"uncached reads", "uncached_reads", "Number of READ bios that bypassed the cache since boot"},
	{"uncached writes", "uncached_writes", "Number of WRITE bios that bypassed the cache since boot"},
	{"uncached seq reads", "uncached_sequential_reads", "Number of sequential READ bios that bypassed the cache since boot"},
	{"uncached seq writes", "uncached_sequential_writes", "Number of sequential WRITE bios that bypassed the cache since boot"},
}

// appendDmCacheRawStatsMetrics adds the counters of raw_stats as node_dmcache_stats_*_total. QTS resets
// curr_stats, the source of the other dm-cache hit metrics, about once a minute. raw_stats counts from boot,
// so rate() and increase() see every access. A missing file is ignored, as on QTS versions without it.
func (e *promExporter) appendDmCacheRawStatsMetrics(metrics []metric, cache string) ([]metric, error) {
	lines, err := utils.ReadFileLines(fmt.Sprintf(dmCacheRawStatsFilePathFormat, cache))
	if err != nil {
		if os.IsNotExist(err) {
			return metrics, nil
		}

		return nil, err
	}

	return append(metrics, parseDmCacheRawStats(lines, fmt.Sprintf("device=%q", cache))...), nil
}

// parseDmCacheRawStats turns "field : value" lines into counters. Unknown fields and lines that don't parse are skipped.
func parseDmCacheRawStats(lines []string, attr string) []metric {
	values := make(map[string]float64, len(lines))
	for _, line := range lines {
		field, valueStr, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(valueStr), 64)
		if err != nil {
			continue
		}
		values[strings.TrimSpace(field)] = value
	}

	metrics := make([]metric, 0, len(dmCacheRawStats))
	for _, stat := range dmCacheRawStats {
		value, ok := values[stat.field]
		if !ok {
			continue
		}
		metrics = append(metrics, metric{
			name:       "node_dmcache_stats_" + stat.name + "_total",
			attr:       attr,
			value:      value,
			help:       stat.help,
			metricType: "counter",
		})
	}

	return metrics
}
