package prometheus

import (
	"fmt"
	"strconv"
	"strings"

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
	if e.iostat == "" {
		return nil, nil
	}

	args := []string{"-k", "-d"}
	args = append(args, e.devices...)
	lines, err := utils.ExecCommandGetLines(e.iostat, args...)
	if err != nil {
		return nil, err
	}

	if len(lines) < 4 {
		return nil, fmt.Errorf("iostat output missing expected lines - found %d lines", len(lines))
	}

	metrics := make([]metric, 0, len(e.devices)*2)
	for _, line := range lines[3:] {
		readMetric, err := e.getDiskStatMetric("node_disk_read_kbytes_total", "Total number of kilobytes read", line, 5)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, readMetric)

		writeMetric, err := e.getDiskStatMetric("node_disk_written_kbytes_total", "Total number of kilobytes written", line, 6)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, writeMetric)
	}

	return metrics, nil
}

func (e *promExporter) getDiskStatMetric(name string, help string, line string, field int) (metric, error) {
	fields := strings.Fields(line)
	if field >= len(fields) {
		return metric{}, fmt.Errorf("disk stat metric %q: field %d missing in %d total fields", name, field, len(fields))
	}

	value, err := strconv.ParseFloat(fields[field], 64)
	if err != nil {
		return metric{}, err
	}

	dev := fields[0]
	return metric{
		name:       name,
		attr:       fmt.Sprintf(`device=%q`, dev),
		value:      value,
		help:       help,
		metricType: "counter",
	}, nil
}
