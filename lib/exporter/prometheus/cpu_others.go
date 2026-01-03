//go:build !linux
// +build !linux

package prometheus

import (
	"github.com/shirou/gopsutil/v4/cpu"
)

func getCpuRatioMetrics() ([]metric, error) {
	a, err := cpu.Times(false)
	if err != nil {
		return nil, err
	}
	s := a[0]

	counts, err := cpu.Counts(false)
	if err != nil {
		return nil, err
	}

	metrics := []metric{
		{
			name:       "node_cpu_seconds_total",
			attr:       `mode="user"`,
			metricType: "counter",
			value:      float64(s.User),
		},
		{
			name:       "node_cpu_seconds_total",
			attr:       `mode="nice"`,
			metricType: "counter",
			value:      float64(s.Nice),
		},
		{
			name:       "node_cpu_seconds_total",
			attr:       `mode="system"`,
			metricType: "counter",
			value:      float64(s.System),
		},
		{
			name:       "node_cpu_seconds_total",
			attr:       `mode="idle"`,
			metricType: "counter",
			value:      float64(s.Idle),
		},
		{
			name:  "node_cpu_count",
			value: float64(counts),
		},
	}

	return metrics, nil
}
