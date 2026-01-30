package prometheus

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pedropombeiro/qnapexporter/lib/utils"
)

// Regular expressions for parsing nvme smart-log output
var (
	nvmeTemperatureRe     = regexp.MustCompile(`(?m)^temperature\s*:\s*(\d+)\s*C`)
	nvmeAvailableSpareRe  = regexp.MustCompile(`(?m)^available_spare\s*:\s*(\d+)%`)
	nvmeSpareThresholdRe  = regexp.MustCompile(`(?m)^available_spare_threshold\s*:\s*(\d+)%`)
	nvmePercentageUsedRe  = regexp.MustCompile(`(?m)^percentage_used\s*:\s*(\d+)%`)
	nvmePowerOnHoursRe    = regexp.MustCompile(`(?m)^power_on_hours\s*:\s*([\d,]+)`)
	nvmePowerCyclesRe     = regexp.MustCompile(`(?m)^power_cycles\s*:\s*([\d,]+)`)
	nvmeUnsafeShutdownsRe = regexp.MustCompile(`(?m)^unsafe_shutdowns\s*:\s*([\d,]+)`)
	nvmeMediaErrorsRe     = regexp.MustCompile(`(?m)^media_errors\s*:\s*([\d,]+)`)
)

// nvmeSmartData holds parsed SMART log data for an NVMe device
type nvmeSmartData struct {
	Temperature             float64
	AvailableSpare          float64
	AvailableSpareThreshold float64
	PercentageUsed          float64
	PowerOnHours            float64
	PowerCycles             float64
	UnsafeShutdowns         float64
	MediaErrors             float64
}

// parseNvmeSmartLog parses the output of `nvme smart-log` command
func parseNvmeSmartLog(output string) (*nvmeSmartData, error) {
	data := &nvmeSmartData{}

	// Parse temperature
	if matches := nvmeTemperatureRe.FindStringSubmatch(output); len(matches) >= 2 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			data.Temperature = val
		}
	}

	// Parse available spare (convert percentage to ratio)
	if matches := nvmeAvailableSpareRe.FindStringSubmatch(output); len(matches) >= 2 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			data.AvailableSpare = val / 100.0
		}
	}

	// Parse available spare threshold (convert percentage to ratio)
	if matches := nvmeSpareThresholdRe.FindStringSubmatch(output); len(matches) >= 2 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			data.AvailableSpareThreshold = val / 100.0
		}
	}

	// Parse percentage used (convert percentage to ratio)
	if matches := nvmePercentageUsedRe.FindStringSubmatch(output); len(matches) >= 2 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			data.PercentageUsed = val / 100.0
		}
	}

	// Parse power on hours (remove commas from number)
	if matches := nvmePowerOnHoursRe.FindStringSubmatch(output); len(matches) >= 2 {
		if val, err := parseNumberWithCommas(matches[1]); err == nil {
			data.PowerOnHours = val
		}
	}

	// Parse power cycles
	if matches := nvmePowerCyclesRe.FindStringSubmatch(output); len(matches) >= 2 {
		if val, err := parseNumberWithCommas(matches[1]); err == nil {
			data.PowerCycles = val
		}
	}

	// Parse unsafe shutdowns
	if matches := nvmeUnsafeShutdownsRe.FindStringSubmatch(output); len(matches) >= 2 {
		if val, err := parseNumberWithCommas(matches[1]); err == nil {
			data.UnsafeShutdowns = val
		}
	}

	// Parse media errors
	if matches := nvmeMediaErrorsRe.FindStringSubmatch(output); len(matches) >= 2 {
		if val, err := parseNumberWithCommas(matches[1]); err == nil {
			data.MediaErrors = val
		}
	}

	return data, nil
}

// parseNumberWithCommas parses a number string that may contain commas as thousand separators
func parseNumberWithCommas(s string) (float64, error) {
	s = strings.ReplaceAll(s, ",", "")
	return strconv.ParseFloat(s, 64)
}

// getNvmeSmartMetrics retrieves SMART metrics for all NVMe devices
func (e *promExporter) getNvmeSmartMetrics() ([]metric, error) {
	if e.nvmePath == "" || len(e.nvmeDevices) == 0 {
		return nil, nil
	}

	// Pre-allocate space for metrics (8 metrics per device)
	metrics := make([]metric, 0, len(e.nvmeDevices)*8)

	for _, device := range e.nvmeDevices {
		devicePath := fmt.Sprintf("/dev/%s", device)
		output, err := utils.ExecCommand(e.nvmePath, "smart-log", devicePath)
		if err != nil {
			// Log the error but continue with other devices
			e.Logger.Printf("Failed to get NVMe SMART data for %s: %v", device, err)
			continue
		}

		data, err := parseNvmeSmartLog(output)
		if err != nil {
			e.Logger.Printf("Failed to parse NVMe SMART data for %s: %v", device, err)
			continue
		}

		attr := fmt.Sprintf(`device=%q`, device)

		// Temperature
		metrics = append(metrics, metric{
			name:       "node_nvme_temperature_celsius",
			attr:       attr,
			value:      data.Temperature,
			help:       "Current temperature of the NVMe device in Celsius",
			metricType: "gauge",
		})

		// Available spare
		metrics = append(metrics, metric{
			name:       "node_nvme_available_spare_ratio",
			attr:       attr,
			value:      data.AvailableSpare,
			help:       "Normalized percentage of remaining spare capacity available",
			metricType: "gauge",
		})

		// Available spare threshold
		metrics = append(metrics, metric{
			name:       "node_nvme_available_spare_threshold_ratio",
			attr:       attr,
			value:      data.AvailableSpareThreshold,
			help:       "Threshold at which spare capacity is considered critically low",
			metricType: "gauge",
		})

		// Percentage used
		metrics = append(metrics, metric{
			name:       "node_nvme_percentage_used_ratio",
			attr:       attr,
			value:      data.PercentageUsed,
			help:       "Vendor-specific estimate of the percentage of NVMe subsystem life used",
			metricType: "gauge",
		})

		// Power on hours
		metrics = append(metrics, metric{
			name:       "node_nvme_power_on_hours_total",
			attr:       attr,
			value:      data.PowerOnHours,
			help:       "Total number of power-on hours",
			metricType: "counter",
		})

		// Power cycles
		metrics = append(metrics, metric{
			name:       "node_nvme_power_cycles_total",
			attr:       attr,
			value:      data.PowerCycles,
			help:       "Total number of power cycles",
			metricType: "counter",
		})

		// Unsafe shutdowns
		metrics = append(metrics, metric{
			name:       "node_nvme_unsafe_shutdowns_total",
			attr:       attr,
			value:      data.UnsafeShutdowns,
			help:       "Total number of unsafe shutdowns",
			metricType: "counter",
		})

		// Media errors
		metrics = append(metrics, metric{
			name:       "node_nvme_media_errors_total",
			attr:       attr,
			value:      data.MediaErrors,
			help:       "Total number of unrecovered data integrity errors",
			metricType: "counter",
		})
	}

	return metrics, nil
}
