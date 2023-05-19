package prometheus

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pedropombeiro/qnapexporter/lib/utils"
)

type volumeInfo struct {
	index                         string
	fileSystem                    string
	description                   string
	status                        string
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
	e.Logger.Printf("Retrieved volCount: %d", volCount)

	e.volumes = make([]volumeInfo, 0, volCount)

	idx := uint64(0)
	for parsedVolCount := 0; parsedVolCount < volCount; idx++ {
		volIdx := strconv.FormatUint(idx, 10)

		desc, err := utils.ExecCommand(e.getsysinfo, "vol_desc", volIdx)
		if err != nil {
			e.Logger.Printf("Error fetching volume %d description: %v", idx, err)
			continue
		}
		description := parseVolDesc(desc)
		e.Logger.Printf("Retrieved vol_desc %q, parsed to %q", desc, description)

		parsedVolCount++
		if description == "" {
			continue
		}

		fileSystem, err := utils.ExecCommand(e.getsysinfo, "vol_fs", volIdx)
		if err != nil {
			e.Logger.Printf("Error fetching volume %q file system: %v", description, err)
			continue
		}
		e.Logger.Printf("Retrieved volume %q vol_fs %q", description, fileSystem)
		if fileSystem == "Unknown" {
			e.Logger.Printf("Ignoring %q volume with %s file system", description, fileSystem)
			continue
		}

		volsizeStr, err := utils.ExecCommand(e.getsysinfo, "vol_totalsize", volIdx)
		if err != nil {
			e.Logger.Printf("Error fetching volume %q size: %v", description, err)
			continue
		}
		e.Logger.Printf("Retrieved volume %q vol_totalsize %q", description, volsizeStr)

		volsizeBytes, err := parseVolSize(volsizeStr)
		if err != nil {
			continue
		}

		status, err := utils.ExecCommand(e.getsysinfo, "vol_status", volIdx)
		if err != nil {
			e.Logger.Printf("Error fetching volume %q status: %v", description, err)
			continue
		}
		e.Logger.Printf("Retrieved volume %q vol_status %q", description, status)

		e.volumes = append(
			e.volumes,
			volumeInfo{
				index:          volIdx,
				description:    description,
				fileSystem:     fileSystem,
				status:         status,
				totalSizeBytes: volsizeBytes,
			},
		)
	}

	e.Logger.Printf("Found volumes %v", e.volumes)
}

func (e *promExporter) getSysInfoVolMetrics() ([]metric, error) {
	if e.getsysinfo == "" {
		return nil, nil
	}

	metrics := make([]metric, 0, 2*len(e.volumes))
	e.status.Volumes = []string{}

	expired := e.volumeLastFetch.IsZero() || time.Now().After(e.volumeLastFetch.Add(volumeValidity))
	if expired {
		e.volumeLastFetch = time.Now()
	}

	for idx, v := range e.volumes {
		e.status.Volumes = append(e.status.Volumes, v.description)

		if expired || v.freeSizeBytes == 0 {
			freesizeStr, err := utils.ExecCommand(e.getsysinfo, "vol_freesize", v.index)
			if err != nil {
				return nil, err
			}
			freeSizeBytes, err := parseVolSize(freesizeStr)
			if err != nil {
				return nil, err
			}

			v.freeSizeBytes = freeSizeBytes
			e.volumes[idx] = v
		}

		attr := fmt.Sprintf("volume=%q,filesystem=%q,status=%q", v.description, v.fileSystem, v.status)
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

	return metrics, nil
}

func parseVolDesc(desc string) string {
	var index int
	switch {
	case strings.HasPrefix(desc, "[Volume"):
		index = 8
	case strings.HasPrefix(desc, "[Single Disk Volume:"):
		return ""
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
