// +build !linux

package prometheus

import (
	"github.com/mackerelio/go-osstat/cpu"
)

func getCpuRatioMetrics() ([]metric, error) {
	s, err := cpu.Get()
	if err != nil {
		return nil, err
	}

	metrics := []metric{
		{
			name:  "node_cpu_ratio",
			attr:  `mode="user"`,
			value: float64(s.User) / float64(s.Total),
		},
		{
			name:  "node_cpu_ratio",
			attr:  `mode="nice"`,
			value: float64(s.Nice) / float64(s.Total),
		},
		{
			name:  "node_cpu_ratio",
			attr:  `mode="system"`,
			value: float64(s.System) / float64(s.Total),
		},
		{
			name:  "node_cpu_ratio",
			attr:  `mode="idle"`,
			value: float64(s.Idle) / float64(s.Total),
		},
	}

	return metrics, nil
}
