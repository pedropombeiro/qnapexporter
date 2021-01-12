// +build !linux

package prometheus

import (
	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/memory"
)

func getMemInfoMetrics() ([]metric, error) {
	s, err := memory.Get()
	if err != nil {
		return nil, err
	}

	metrics := []metric{
		{name: "node_memory_MemTotal_bytes", value: float64(s.Total)},
		{name: "node_memory_MemFree_bytes", value: float64(s.Free)},
		{name: "node_memory_Cached_bytes", value: float64(s.Cached)},
		{name: "node_memory_Active_bytes", value: float64(s.Active)},
		{name: "node_memory_Inactive_bytes", value: float64(s.Inactive)},
		{name: "node_memory_SwapTotal_bytes", value: float64(s.SwapTotal)},
		{name: "node_memory_SwapFree_bytes", value: float64(s.SwapFree)},
	}

	return metrics, nil
}

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
