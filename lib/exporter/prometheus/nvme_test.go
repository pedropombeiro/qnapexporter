package prometheus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNvmeSmartLog(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *nvmeSmartData
	}{
		{
			name: "typical nvme smart-log output",
			input: `Smart Log for NVME device:nvme0 namespace-id:ffffffff
critical_warning                    : 0
temperature                         : 35 C
available_spare                     : 100%
available_spare_threshold           : 10%
percentage_used                     : 12%
data_units_read                     : 1,154,955,542
data_units_written                  : 167,093,012
host_read_commands                  : 9,762,913,581
host_write_commands                 : 4,423,574,741
controller_busy_time                : 159,895
power_cycles                        : 35
power_on_hours                      : 44,891
unsafe_shutdowns                    : 11
media_errors                        : 0
num_err_log_entries                 : 0
Warning Temperature Time            : 0
Critical Composite Temperature Time : 0
Thermal Management T1 Trans Count   : 0
Thermal Management T2 Trans Count   : 0
Thermal Management T1 Total Time    : 0
Thermal Management T2 Total Time    : 0`,
			expected: &nvmeSmartData{
				Temperature:             35,
				AvailableSpare:          1.0,
				AvailableSpareThreshold: 0.10,
				PercentageUsed:          0.12,
				PowerOnHours:            44891,
				PowerCycles:             35,
				UnsafeShutdowns:         11,
				MediaErrors:             0,
			},
		},
		{
			name: "high usage device",
			input: `Smart Log for NVME device:nvme1 namespace-id:ffffffff
critical_warning                    : 0
temperature                         : 48 C
available_spare                     : 85%
available_spare_threshold           : 5%
percentage_used                     : 45%
data_units_read                     : 2,500,000,000
data_units_written                  : 3,000,000,000
host_read_commands                  : 15,000,000,000
host_write_commands                 : 8,000,000,000
controller_busy_time                : 500,000
power_cycles                        : 150
power_on_hours                      : 87,600
unsafe_shutdowns                    : 25
media_errors                        : 3
num_err_log_entries                 : 5`,
			expected: &nvmeSmartData{
				Temperature:             48,
				AvailableSpare:          0.85,
				AvailableSpareThreshold: 0.05,
				PercentageUsed:          0.45,
				PowerOnHours:            87600,
				PowerCycles:             150,
				UnsafeShutdowns:         25,
				MediaErrors:             3,
			},
		},
		{
			name: "new device with zeros",
			input: `Smart Log for NVME device:nvme0 namespace-id:ffffffff
temperature                         : 25 C
available_spare                     : 100%
available_spare_threshold           : 10%
percentage_used                     : 0%
power_cycles                        : 1
power_on_hours                      : 0
unsafe_shutdowns                    : 0
media_errors                        : 0`,
			expected: &nvmeSmartData{
				Temperature:             25,
				AvailableSpare:          1.0,
				AvailableSpareThreshold: 0.10,
				PercentageUsed:          0,
				PowerOnHours:            0,
				PowerCycles:             1,
				UnsafeShutdowns:         0,
				MediaErrors:             0,
			},
		},
		{
			name:  "empty output",
			input: "",
			expected: &nvmeSmartData{
				Temperature:             0,
				AvailableSpare:          0,
				AvailableSpareThreshold: 0,
				PercentageUsed:          0,
				PowerOnHours:            0,
				PowerCycles:             0,
				UnsafeShutdowns:         0,
				MediaErrors:             0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseNvmeSmartLog(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.Temperature, result.Temperature, "Temperature mismatch")
			assert.Equal(t, tt.expected.AvailableSpare, result.AvailableSpare, "AvailableSpare mismatch")
			assert.Equal(t, tt.expected.AvailableSpareThreshold, result.AvailableSpareThreshold, "AvailableSpareThreshold mismatch")
			assert.Equal(t, tt.expected.PercentageUsed, result.PercentageUsed, "PercentageUsed mismatch")
			assert.Equal(t, tt.expected.PowerOnHours, result.PowerOnHours, "PowerOnHours mismatch")
			assert.Equal(t, tt.expected.PowerCycles, result.PowerCycles, "PowerCycles mismatch")
			assert.Equal(t, tt.expected.UnsafeShutdowns, result.UnsafeShutdowns, "UnsafeShutdowns mismatch")
			assert.Equal(t, tt.expected.MediaErrors, result.MediaErrors, "MediaErrors mismatch")
		})
	}
}

func TestParseNumberWithCommas(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"0", 0},
		{"123", 123},
		{"1,234", 1234},
		{"1,234,567", 1234567},
		{"9,762,913,581", 9762913581},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseNumberWithCommas(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
