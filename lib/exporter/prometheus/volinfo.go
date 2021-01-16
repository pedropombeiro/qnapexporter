package prometheus

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gitlab.com/pedropombeiro/qnapexporter/lib/utils"
)

type volumeInfo struct {
	fileSystem                    string
	description                   string
	freeSizeBytes, totalSizeBytes float64
}

func (e *promExporter) readSysVolInfo() {
	volCount := 0
	sysvolnumOutput, err := utils.ExecCommand(e.getsysinfo, "sysvolnum")
	if err == nil {
		volCount, err = strconv.Atoi(sysvolnumOutput)
		if err != nil {
			volCount = 0
		}
	}

	e.volumes = make([]volumeInfo, 0, volCount)

	for idx := 0; idx < volCount; idx++ {
		volIdx := strconv.FormatUint(uint64(idx), 10)

		desc, err := utils.ExecCommand(e.getsysinfo, "vol_desc", volIdx)
		if err != nil {
			e.logger.Printf("Error fetching volume %d description: %v", idx, err)
			continue
		}

		fileSystem, err := utils.ExecCommand(e.getsysinfo, "vol_fs", volIdx)
		if err != nil {
			e.logger.Printf("Error fetching volume %d file system: %v", idx, err)
			continue
		}
		if fileSystem == "Unknown" {
			e.logger.Printf("Ignoring %q volume with %s file system", desc, fileSystem)
			continue
		}

		volsizeStr, err := utils.ExecCommand(e.getsysinfo, "vol_totalsize", volIdx)
		if err != nil {
			e.logger.Printf("Error fetching volume %d size: %v", idx, err)
			continue
		}
		volsizeBytes, err := parseVolSize(volsizeStr)
		if err != nil {
			continue
		}

		e.volumes = append(
			e.volumes,
			volumeInfo{
				description:    parseVolDesc(desc),
				fileSystem:     fileSystem,
				totalSizeBytes: volsizeBytes,
			},
		)
	}

	e.volumeExpiry = time.Now()
	e.logger.Printf("Found volumes %v", e.volumes)
}

func (e *promExporter) getSysInfoVolMetrics() ([]metric, error) {
	if e.getsysinfo == "" {
		return nil, nil
	}

	metrics := make([]metric, 0, 2*len(e.volumes))

	expired := time.Now().After(e.volumeExpiry)
	for idx, v := range e.volumes {
		volnumStr := strconv.Itoa(idx)

		if expired {
			freesizeStr, err := utils.ExecCommand(e.getsysinfo, "vol_freesize", volnumStr)
			if err != nil {
				return nil, err
			}
			v.freeSizeBytes, err = parseVolSize(freesizeStr)
			if err != nil {
				return nil, err
			}

			e.volumes[idx] = v
		}

		attr := fmt.Sprintf("volume=%q,filesystem=%q", v.description, v.fileSystem)
		newMetrics := []metric{
			{
				name:  "node_volume_avail_bytes",
				attr:  attr,
				value: v.freeSizeBytes,
			},
			{
				name:  "node_volume_size_bytes",
				attr:  attr,
				value: v.totalSizeBytes,
			},
		}
		metrics = append(metrics, newMetrics...)
	}

	if expired {
		e.volumeExpiry = e.volumeExpiry.Add(volumeValidity)
	}

	return metrics, nil
}

func parseVolDesc(desc string) string {
	var index int
	switch {
	case strings.HasPrefix(desc, "[Volume"):
		index = 8
	case strings.HasPrefix(desc, "[Single Disk Volume:"):
		index = 21
	default:
		return desc
	}

	return strings.SplitN(strings.TrimSpace(desc[index:]), ",", 2)[0]
}

func parseVolSize(s string) (float64, error) {
	fields := strings.Fields(s)
	size, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("parse volume size (%s): %w", s, err)
	}

	var factor float64
	switch fields[1] {
	case "TB":
		factor = 1024 * 1024 * 1024 * 1024
	case "GB":
		factor = 1024 * 1024 * 1024
	case "MB":
		factor = 1024 * 1024
	case "KB":
		factor = 1024
	case "B":
		factor = 1
	}

	return size * factor, nil
}
