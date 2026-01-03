//go:build !linux
// +build !linux

package prometheus

import (
	"github.com/shirou/gopsutil/v4/mem"
)

func getMemInfoMetrics() ([]metric, error) {
	s, err := mem.VirtualMemory()
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
