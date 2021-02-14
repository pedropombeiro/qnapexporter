package prometheus

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/disk"

	"gitlab.com/pedropombeiro/qnapexporter/lib/utils"
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

func getFlashCacheStatsMetrics() ([]metric, error) {
	lines, err := utils.ReadFileLines(flashcacheStatsPath)
	if err != nil {
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
