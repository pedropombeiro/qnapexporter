package prometheus

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pedropombeiro/qnapexporter/lib/utils"
	"github.com/shirou/gopsutil/v3/disk"
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
		allocationRatioStr := strings.TrimSpace(tokens[3])
		allocationTokens := strings.SplitN(allocationRatioStr, "/", 2)
		attr := fmt.Sprintf("device=%q", cache)

		metrics = appendFloatMetric(metrics, "node_flashcache_cached_blocks", allocationTokens[0], 1, "", "")
		metrics = appendFloatMetric(metrics, "node_flashcache_total_blocks", allocationTokens[1], 1, "", "")
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
		help:       "Number of times a READ bio has ocurred",
		metricType: "counter",
	})
	metrics = append(metrics, metric{
		name:       "node_dmcache_read_total",
		attr:       attr,
		value:      readTotal,
		help:       "Number of times a READ bio has ocurred",
		metricType: "counter",
	})
	if readTotal > 0 {
		metrics = append(metrics, metric{
			name:       "node_flashcache_read_hit_percent",
			attr:       attr,
			value:      readHits / readTotal * 100,
			metricType: "counter",
		})
		metrics = append(metrics, metric{
			name:       "node_dmcache_read_hit_percent",
			attr:       attr,
			value:      readHits / readTotal * 100,
			metricType: "counter",
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
		help:       "Number of times a WRITE bio has ocurred",
		metricType: "counter",
	})
	metrics = append(metrics, metric{
		name:       "node_dmcache_write_total",
		attr:       attr,
		value:      writeTotal,
		help:       "Number of times a WRITE bio has ocurred",
		metricType: "counter",
	})
	if writeTotal > 0 {
		metrics = append(metrics, metric{
			name:       "node_flashcache_write_hit_percent",
			attr:       attr,
			value:      writeHits / writeTotal * 100,
			metricType: "counter",
		})
		metrics = append(metrics, metric{
			name:       "node_dmcache_write_hit_percent",
			attr:       attr,
			value:      writeHits / writeTotal * 100,
			metricType: "counter",
		})
	}

	return metrics, nil
}

func getTokenValue(token string) float64 {
	token = strings.TrimSpace(token)
	value, err := strconv.ParseFloat(token, 64)
	if err != nil {
		return 0.0
	}
	return value
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
